import os
import sys
from pathlib import Path

_CURRENT_DIR = Path(__file__).resolve().parent
_REPO_ROOT = _CURRENT_DIR.parent
for _p in (str(_REPO_ROOT), str(_CURRENT_DIR)):
    if _p not in sys.path:
        sys.path.insert(0, _p)

try:
    import ai.conf.env  # loads dotenv
except (ImportError, ModuleNotFoundError):
    import conf.env  # loads dotenv

from openai import OpenAI

client = OpenAI(
    base_url=os.getenv("BASE_URL", "https://api.groq.com/openai/v1"),
    api_key=os.getenv("API_KEY", "")
)
