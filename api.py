import os
from pathlib import Path
from fastapi import FastAPI
from pydantic import BaseModel
import uvicorn
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
    # Define the socket path
    socket_path = str(Path.home() / ".omnikernel" / "omni.sock")
    
    # Clean up old socket if it exists
    if os.path.exists(socket_path):
        os.remove(socket_path)
        
    print(f"Starting Python AI Brain on socket: {socket_path}")
    uvicorn.run(app, uds=socket_path)