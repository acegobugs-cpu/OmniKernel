import os
import signal
import subprocess

def kill_process(pid: int) -> str:
    """Forcefully kills a process by its PID."""
    try:
        # We use subprocess to execute a native Linux kill -9 command.
        # This is much more aggressive than Python's os.kill
        result = subprocess.run(["sudo", "kill", "-9", str(pid)], capture_output=True, text=True)
        
        if result.returncode == 0:
            return f"✅ Successfully executed SIGKILL on PID {pid}."
        else:
            return f"❌ Failed to kill PID {pid}: {result.stderr.strip()}"
            
    except Exception as e:
        return f"❌ Error executing kill command: {e}"

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

def run_shell_command(command: str) -> str:
    """Executes an arbitrary shell command safely with a timeout."""
    # 1. Hardcoded blacklist to prevent catastrophic AI mistakes
    blacklist = ["rm -rf /", "mkfs", "dd if="]
    if any(bad_cmd in command for bad_cmd in blacklist):
        return "❌ Security Block: That command is highly destructive and blacklisted."
        
    try:
        # 2. Run the command with a strict 10-second timeout
        result = subprocess.run(
            command, 
            shell=True, 
            capture_output=True, 
            text=True, 
            timeout=10
        )
        
        # 3. Combine stdout and stderr
        output = result.stdout.strip()
        error = result.stderr.strip()
        
        final_output = output if output else error
        if not final_output:
            return f"✅ Command '{command}' executed successfully with no output."
            
        return f"--- Output of '{command}' ---\n{final_output}"
        
    except subprocess.TimeoutExpired:
        return f"❌ Error: Command '{command}' timed out after 10 seconds."
    except Exception as e:
        return f"❌ Error executing command: {e}"