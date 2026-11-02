import os
import sys
from pathlib import Path
from fastapi import FastAPI
from pydantic import BaseModel
import uvicorn

_CURRENT_DIR = Path(__file__).resolve().parent
_REPO_ROOT = _CURRENT_DIR.parent
for _p in (str(_REPO_ROOT), str(_CURRENT_DIR)):
    if _p not in sys.path:
        sys.path.insert(0, _p)

try:
    from ai.chat import step_oracle
except (ImportError, ModuleNotFoundError):
    from chat import step_oracle

app = FastAPI(title="OmniKernel AI Brain")

class StepRequest(BaseModel):
    query: str | None = None
    tool_call_id: str | None = None
    tool_result: str | None = None

@app.post("/step")
def step_ai(request: StepRequest):
    """Takes a step in the conversation (either a new query or a tool result)."""
    return step_oracle(request.query, request.tool_call_id, request.tool_result)

if __name__ == "__main__":
    # Ensure ~/.omnikernel directory exists
    sock_dir = Path.home() / ".omnikernel"
    sock_dir.mkdir(parents=True, exist_ok=True)
    socket_path = str(sock_dir / "omni.sock")
    
    # Clean up old socket if it exists
    if os.path.exists(socket_path):
        try:
            os.remove(socket_path)
        except OSError:
            pass
        
    print(f"Starting Python AI Brain on socket: {socket_path}")
    uvicorn.run(app, uds=socket_path)
