from __future__ import annotations
from typing import Any
from .base import Connector
from .confluence import ConfluenceConnector
from .http_json import HTTPJSONConnector
from .zendesk import ZendeskConnector

def build_connector(config:dict[str,Any])->Connector:
    provider=config.get("provider","http_json")
    if provider=="http_json":return HTTPJSONConnector(config)
    if provider=="zendesk":return ZendeskConnector(config)
    if provider=="confluence":return ConfluenceConnector(config)
    raise ValueError(f"unsupported connector provider: {provider}")

__all__=["build_connector","Connector"]
