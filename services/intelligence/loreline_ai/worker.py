from __future__ import annotations

import json
import logging
import signal
import socket
import threading
import time
from typing import Any

from psycopg import Connection, OperationalError
from psycopg.rows import dict_row

from .config import settings
from .ingestion import embedding, fingerprint, normalize
from .storage import ObjectStore
from .connectors import build_connector

logging.basicConfig(level=logging.INFO,format='{"time":"%(asctime)s","level":"%(levelname)s","message":"%(message)s"}')
log=logging.getLogger("loreline.worker")
stop=threading.Event()
VERSION="1.1.0"


def claim(conn: Connection[Any], worker_id: str) -> dict[str, Any] | None:
    with conn.transaction():
        row=conn.execute("""SELECT id::text,tenant_id::text,kind,payload,attempts,max_attempts FROM jobs WHERE status IN ('queued','failed') AND run_after<=now() AND attempts<max_attempts ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1""").fetchone()
        if not row: return None
        conn.execute("UPDATE jobs SET status='running',attempts=attempts+1,locked_at=now(),locked_by=%s WHERE id=%s",(worker_id,row["id"]))
        return row


def process_ingest(conn: Connection[Any], job: dict[str, Any]) -> None:
    cfg=settings(); payload=job["payload"]; document_id=payload["document_id"]
    doc=conn.execute("SELECT id::text,tenant_id::text,title,media_type,content_sha256 FROM documents WHERE id=%s",(document_id,)).fetchone()
    if not doc: raise ValueError("document not found")
    raw=payload["content"]
    result=normalize(doc["title"],raw,doc["media_type"],cfg.chunk_chars,cfg.chunk_overlap)
    object_key=ObjectStore(cfg).put(f"{doc['tenant_id']}/{document_id}/{doc['content_sha256']}",raw.encode("utf-8"),doc["media_type"])
    with conn.transaction():
        duplicate=conn.execute("SELECT id::text FROM candidates WHERE tenant_id=%s AND md5(content)=md5(%s) LIMIT 1",(doc["tenant_id"],result.content)).fetchone()
        candidate=conn.execute("""INSERT INTO candidates(tenant_id,document_id,title,content,confidence,duplicate_of) VALUES(%s,%s,%s,%s,%s,%s) RETURNING id::text""",(doc["tenant_id"],document_id,result.title,result.content,result.confidence,duplicate["id"] if duplicate else None)).fetchone()["id"]
        for ordinal,part in enumerate(result.chunks):
            vector=embedding(part)
            conn.execute("INSERT INTO chunks(tenant_id,document_id,candidate_id,ordinal,content,content_sha256,embedding) VALUES(%s,%s,%s,%s,%s,%s,%s)",(doc["tenant_id"],document_id,candidate,ordinal,part,fingerprint(part),vector))
        conn.execute("UPDATE documents SET state='review',processed_at=now(),object_key=COALESCE(%s,object_key) WHERE id=%s",(object_key,document_id))


def process_index(conn: Connection[Any], job: dict[str, Any]) -> None:
    candidate=job["payload"]["candidate_id"]
    with conn.transaction(): conn.execute("UPDATE chunks SET active=true WHERE candidate_id=%s",(candidate,))

def process_retire(conn: Connection[Any], job: dict[str, Any]) -> None:
    document=job["payload"]["document_id"]
    with conn.transaction():
        conn.execute("UPDATE chunks SET active=false WHERE document_id=%s",(document,)); conn.execute("UPDATE documents SET state='retired',retired_at=now() WHERE id=%s",(document,))

def process_sync(conn:Connection[Any],job:dict[str,Any])->None:
    source_id=job["payload"]["source_id"]
    source=conn.execute("SELECT id::text,tenant_id::text,config,cursor FROM sources WHERE id=%s AND status='active'",(source_id,)).fetchone()
    if not source:raise ValueError("active source not found")
    connector=build_connector(source["config"]);records,cursor=connector.fetch(source["cursor"])
    with conn.transaction():
        for record in records:
            digest=fingerprint(record.content);existing=conn.execute("SELECT id FROM documents WHERE tenant_id=%s AND source_id=%s AND external_id=%s AND content_sha256=%s LIMIT 1",(source["tenant_id"],source_id,record.external_id,digest)).fetchone()
            if existing:continue
            version=conn.execute("SELECT COALESCE(max(version),0)+1 AS version FROM documents WHERE tenant_id=%s AND source_id=%s AND external_id=%s",(source["tenant_id"],source_id,record.external_id)).fetchone()["version"]
            document=conn.execute("INSERT INTO documents(tenant_id,source_id,external_id,title,media_type,content_sha256,metadata,version) VALUES(%s,%s,%s,%s,%s,%s,%s,%s) RETURNING id::text",(source["tenant_id"],source_id,record.external_id,record.title,record.media_type,digest,record.metadata or {},version)).fetchone()["id"]
            conn.execute("INSERT INTO jobs(tenant_id,kind,payload) VALUES(%s,'ingest',jsonb_build_object('document_id',(%s)::text,'content',%s))",(source["tenant_id"],document,record.content))
        conn.execute("UPDATE sources SET cursor=%s,last_synced_at=now(),last_error=NULL WHERE id=%s",(json.dumps(cursor),source_id))

def finish(conn: Connection[Any], job: dict[str, Any], error: Exception | None) -> None:
    with conn.transaction():
        if error is None: conn.execute("UPDATE jobs SET status='succeeded',finished_at=now(),locked_at=NULL,locked_by=NULL,payload=payload-'content' WHERE id=%s",(job["id"],))
        else:
            dead=job["attempts"]+1>=job["max_attempts"]; delay=min(3600,2**(job["attempts"]+1))
            conn.execute("UPDATE jobs SET status=%s,last_error=%s,run_after=now()+(%s*interval '1 second'),locked_at=NULL,locked_by=NULL WHERE id=%s",("dead" if dead else "failed",str(error)[:2000],delay,job["id"]))

def maintenance(conn:Connection[Any])->None:
    with conn.transaction():
        conn.execute("UPDATE jobs SET status='failed',last_error='worker lease expired',locked_at=NULL,locked_by=NULL,run_after=now() WHERE status='running' AND locked_at<now()-interval '15 minutes'")
        conn.execute("DELETE FROM idempotency_keys WHERE expires_at<now()")
        conn.execute("DELETE FROM worker_heartbeats WHERE heartbeat_at<now()-interval '7 days'")
        conn.execute("DELETE FROM jobs WHERE status='succeeded' AND finished_at<now()-interval '30 days'")
        conn.execute("""INSERT INTO jobs(tenant_id,kind,payload) SELECT d.tenant_id,'retire',jsonb_build_object('document_id',d.id::text) FROM documents d JOIN tenants t ON t.id=d.tenant_id WHERE d.state='approved' AND d.processed_at<now()-(COALESCE((t.settings->>'freshness_days')::int,365)*interval '1 day') AND NOT EXISTS(SELECT 1 FROM jobs j WHERE j.kind='retire' AND j.payload->>'document_id'=d.id::text AND j.status IN ('queued','running')) LIMIT 500""")

def heartbeat(conn:Connection[Any],worker_id:str)->None:
    conn.execute("""INSERT INTO worker_heartbeats(worker_id,hostname,version) VALUES(%s,%s,%s) ON CONFLICT(worker_id) DO UPDATE SET hostname=excluded.hostname,version=excluded.version,heartbeat_at=now()""",(worker_id,socket.gethostname(),VERSION))

def process_job(conn:Connection[Any],job:dict[str,Any])->None:
    handlers={"ingest":process_ingest,"index":process_index,"retire":process_retire,"sync":process_sync}
    handler=handlers.get(job["kind"])
    if handler is None:raise ValueError(f"unsupported job kind: {job['kind']}")
    handler(conn,job)

def work_connection(cfg:Any,worker_id:str)->None:
    with Connection.connect(cfg.database_url,row_factory=dict_row,autocommit=True,connect_timeout=10) as conn:
        last_maintenance=0.0
        while not stop.is_set():
            if time.monotonic()-last_maintenance>15:
                maintenance(conn);heartbeat(conn,worker_id);last_maintenance=time.monotonic()
            job=claim(conn,worker_id)
            if not job:stop.wait(cfg.worker_poll_seconds);continue
            error=None
            try:process_job(conn,job)
            except Exception as exc:
                error=exc;log.exception("job failed: %s",job["id"])
                if job["kind"]=="sync":conn.execute("UPDATE sources SET last_error=%s WHERE id=%s",(str(exc)[:2000],job["payload"].get("source_id")))
            finish(conn,job,error)

def run() -> None:
    cfg=settings()
    if not cfg.database_url: raise RuntimeError("LORELINE_DATABASE_URL is required")
    store=ObjectStore(cfg)
    if not store.enabled and not cfg.allow_database_only:raise RuntimeError("object storage is required; set LORELINE_ALLOW_DATABASE_ONLY=true only for tests")
    worker_id=cfg.worker_id+"@"+socket.gethostname(); log.info("worker started: %s",worker_id)
    delay=1.0
    while not stop.is_set():
        try:work_connection(cfg,worker_id);delay=1.0
        except OperationalError:
            log.exception("database connection lost; retrying in %.1f seconds",delay)
            stop.wait(delay);delay=min(delay*2,30.0)
    log.info("worker stopped")

def main() -> None:
    signal.signal(signal.SIGTERM,lambda *_:stop.set());signal.signal(signal.SIGINT,lambda *_:stop.set());run()

if __name__=="__main__":main()
