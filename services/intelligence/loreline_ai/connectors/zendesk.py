from __future__ import annotations
import json,os
from typing import Any
import httpx
from .base import SourceRecord

class ZendeskConnector:
    def __init__(self,config:dict[str,Any]):
        self.base=str(config["base_url"]).rstrip("/");raw=os.environ.get(str(config.get("credential_env","")),"{}");creds=json.loads(raw);self.auth=(str(creds["email"])+"/token",str(creds["token"]))
    def fetch(self,cursor:dict[str,Any])->tuple[list[SourceRecord],dict[str,Any]]:
        url=str(cursor.get("next_page") or self.base+"/api/v2/search.json?query=type:ticket status:solved");response=httpx.get(url,auth=self.auth,headers={"Accept":"application/json"},timeout=30,follow_redirects=False);response.raise_for_status();data=response.json();records=[]
        for ticket in data.get("results",[]):
            body=str(ticket.get("description") or "")
            if body:records.append(SourceRecord(str(ticket["id"]),str(ticket.get("subject") or "Resolved ticket"),body,metadata={"url":ticket.get("url"),"updated_at":ticket.get("updated_at"),"connector":"zendesk"}))
        return records,{"next_page":data.get("next_page")} if data.get("next_page") else {}
