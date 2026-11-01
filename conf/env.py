from pathlib import Path
from dotenv import load_dotenv

# Force the app to look in ~/.omnikernel/.env no matter where it is executed from
config_path = Path.home() / ".omnikernel" / ".env"
load_dotenv()