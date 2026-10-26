import os
import signal
import subprocess

def kill_process(pid: int) -> str:
    """Kills a process by its PID."""
    try:
        # SIGTERM (15) asks the process to terminate gracefully. 
        # You could use SIGKILL (9) to force kill it.
        os.kill(pid, signal.SIGTERM)
        return f"✅ Successfully sent termination signal to PID {pid}."
    except ProcessLookupError:
        return f"❌ PID {pid} not found. It may have already closed."
    except PermissionError:
        return f"❌ Permission denied to kill PID {pid}. (Try running with sudo if necessary)"
    except Exception as e:
        return f"❌ Failed to kill PID {pid}: {e}"

def manage_docker(action: str, container_id: str = "") -> str:
    """Runs basic docker commands (list, restart, stop)."""
    try:
        if action == "list":
            result = subprocess.run(["docker", "ps", "--format", "table {{.ID}}\t{{.Names}}\t{{.Status}}"], capture_output=True, text=True)
            return result.stdout.strip() if result.stdout else "No running Docker containers."
            
        elif action in ["stop", "restart"] and container_id:
            result = subprocess.run(["docker", action, container_id], capture_output=True, text=True)
            if result.returncode == 0:
                return f"✅ Successfully executed '{action}' on container {container_id}."
            else:
                return f"❌ Docker command failed: {result.stderr.strip()}"
        else:
            return "❌ Invalid action or missing container_id."
            
    except FileNotFoundError:
        return "❌ Docker is not installed or not running on this system."

def read_system_file(filepath: str) -> str:
    """Reads the content of any file on the system."""
    if not os.path.exists(filepath):
        return f"❌ File not found: {filepath}"
    try:
        with open(filepath, 'r', encoding='utf-8') as f:
            # Read only the first 2000 characters to prevent overwhelming the AI's memory
            content = f.read(2000) 
            return f"--- Content of {filepath} ---\n{content}\n...(truncated if too long)"
    except Exception as e:
        return f"❌ Failed to read file: {e}"

def list_python_packages() -> str:
    """Lists installed pip packages to check dependencies."""
    try:
        result = subprocess.run(["pip", "freeze"], capture_output=True, text=True)
        return result.stdout.strip() if result.stdout else "No packages found."
    except Exception as e:
        return f"❌ Failed to list packages: {e}"