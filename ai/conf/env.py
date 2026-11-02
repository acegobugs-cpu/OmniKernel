from pathlib import Path
import os
from dotenv import load_dotenv

def init_env():
    """Load environment variables from ~/.omnikernel/.env, local .env, or working directory."""
    config_path = Path.home() / ".omnikernel" / ".env"
    local_env = Path(__file__).resolve().parent.parent / ".env"

    if config_path.exists():
        load_dotenv(dotenv_path=config_path)
    elif local_env.exists():
        load_dotenv(dotenv_path=local_env)
    else:
        load_dotenv()

init_env()
