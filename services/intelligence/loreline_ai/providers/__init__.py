from loreline_ai.config import Settings
from .base import Provider
from .extractive import ExtractiveProvider
from .openai_compatible import OpenAICompatibleProvider

def build_provider(config: Settings) -> Provider:
    if config.provider == "extractive": return ExtractiveProvider()
    if config.provider == "openai-compatible": return OpenAICompatibleProvider(config.provider_base_url,config.provider_api_key,config.provider_model)
    raise ValueError(f"unsupported provider: {config.provider}")

__all__=["Provider","build_provider"]
