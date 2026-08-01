from __future__ import annotations
import json,os,time,urllib.error,urllib.request,uuid

BASE=os.environ.get("LORELINE_URL","http://localhost:8088").rstrip("/")
KEY=os.environ["LORELINE_API_KEY"]

def call(method:str,path:str,payload:dict|None=None,headers:dict|None=None):
    body=json.dumps(payload).encode() if payload is not None else None
    request=urllib.request.Request(BASE+path,data=body,method=method,headers={"Authorization":"Bearer "+KEY,"Content-Type":"application/json",**(headers or {})})
    with urllib.request.urlopen(request,timeout=15) as response:return response.status,json.loads(response.read() or b"{}")

assert call("GET","/livez")[0]==200
_,source=call("POST","/v1/sources",{"kind":"api","name":"Smoke test source","config":{}})
external="smoke-"+str(uuid.uuid4());_,queued=call("POST","/v1/documents",{"source_id":source["id"],"external_id":external,"title":"Smoke test access policy","media_type":"text/plain","content":"Smoke test users restore access by waiting fifteen minutes and retrying the identity portal.","metadata":{"smoke":True}},{"Idempotency-Key":external})
deadline=time.time()+30;candidate=None
while time.time()<deadline:
    _,result=call("GET","/v1/candidates?status=pending&limit=100")
    candidate=next((item for item in result["items"] if item["document_id"]==queued["document_id"]),None)
    if candidate:break
    time.sleep(.5)
assert candidate,"worker did not produce a candidate"
call("POST",f"/v1/candidates/{candidate['id']}/decision",{"action":"approve","note":"production smoke test"})
_,answer=call("POST","/v1/assistant/ask",{"question":"How do smoke test users restore access?"})
assert answer["grounded"] and answer["citations"],answer
call("POST","/v1/feedback",{"message_id":answer["message_id"],"rating":1})
assert call("GET","/v1/audit?limit=20")[0]==200
print(json.dumps({"status":"passed","document_id":queued["document_id"],"message_id":answer["message_id"]}))
