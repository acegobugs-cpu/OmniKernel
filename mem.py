from pathlib import Path

# Create the ~/.omnikernel directory if it doesn't exist
CONFIG_DIR = Path.home() / ".omnikernel"
CONFIG_DIR.mkdir(parents=True, exist_ok=True)

# Save memory globally
MEMORY_FILE = ".agent_memory.json"