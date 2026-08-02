from __future__ import annotations
import boto3
from botocore.config import Config
from .config import Settings

class ObjectStore:
    def __init__(self, cfg: Settings):
        self.bucket=cfg.object_bucket
        self.enabled=bool(cfg.object_endpoint and cfg.object_access_key and cfg.object_secret_key)
        self.client=boto3.client("s3",endpoint_url=cfg.object_endpoint,aws_access_key_id=cfg.object_access_key,aws_secret_access_key=cfg.object_secret_key,config=Config(signature_version="s3v4",connect_timeout=5,read_timeout=30,retries={"mode":"standard","max_attempts":4}),region_name="us-east-1") if self.enabled else None
    def ensure(self)->None:
        if not self.enabled:return
        try:self.client.head_bucket(Bucket=self.bucket)
        except Exception:self.client.create_bucket(Bucket=self.bucket)
    def put(self,key:str,content:bytes,media_type:str)->str|None:
        if not self.enabled:return None
        self.ensure();self.client.put_object(Bucket=self.bucket,Key=key,Body=content,ContentType=media_type,ServerSideEncryption="AES256");return key
