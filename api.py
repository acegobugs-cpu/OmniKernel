import os
from pathlib import Path
from fastapi import FastAPI
from pydantic import BaseModel
import uvicorn
from chat import ask_oracle, load_memory, save_memory

app = FastAPI(title="OmniKernel AI Brain")

class QueryRequest(BaseModel):
    query: str

@app.post("/ask")
def ask_ai(request: QueryRequest):
    history = load_memory()
    response_text = ask_oracle(request.query, history) 
    save_memory(history)
    return {"response": response_text}

if __name__ == "__main__":
    # Define the socket path
    socket_path = str(Path.home() / ".omnikernel" / "omni.sock")
    
    # Clean up old socket if it exists
    if os.path.exists(socket_path):
        os.remove(socket_path)
        
    print(f"Starting Python AI Brain on socket: {socket_path}")
    uvicorn.run(app, uds=socket_path)