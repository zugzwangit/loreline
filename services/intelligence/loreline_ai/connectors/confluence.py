from __future__ import annotations
import os
from typing import Any
import httpx
from .base import SourceRecord

class ConfluenceConnector:
    def __init__(self,config:dict[str,Any]):self.base=str(config["base_url"]).rstrip("/");self.space=str(config.get("space_key",""));self.token=os.getenv(str(config.get("credential_env","")),"")
    def fetch(self,cursor:dict[str,Any])->tuple[list[SourceRecord],dict[str,Any]]:
        start=int(cursor.get("start",0));params={"start":start,"limit":100,"expand":"body.storage,version","type":"page"}
        if self.space:params["spaceKey"]=self.space
        response=httpx.get(self.base+"/rest/api/content",params=params,headers={"Authorization":"Bearer "+self.token,"Accept":"application/json"},timeout=30,follow_redirects=False);response.raise_for_status();data=response.json();records=[SourceRecord(str(page["id"]),str(page.get("title") or "Wiki page"),str(page.get("body",{}).get("storage",{}).get("value", "")),"text/html",{"version":page.get("version",{}).get("number"),"connector":"confluence"}) for page in data.get("results",[])];next_start=start+len(records);return records,{"start":next_start} if data.get("_links",{}).get("next") else {}
