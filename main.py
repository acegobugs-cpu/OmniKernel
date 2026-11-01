import os
import sys
import threading
import subprocess

from health import test_generation
from chat import start_chat, ask_oracle, run_daemon_check, load_memory

def send_desktop_notification(message):
    """Triggers a native Ubuntu desktop popup."""
    try:
        # '-u critical' ensures the popup stays on screen until you manually click it away
        subprocess.run(["notify-send", "-u", "critical", "OmniKernel Alert", message])
    except Exception as e:
        print(f"Failed to send notification: {e}")

def main():
    print("🚀 Starting OmniKernel Master Pipeline...")
    
    # Step 1: Health Check
    print("\n--- Phase 1: Cloud Health Check ---")
    try:
        test_generation()
    except Exception as e:
        print(f"❌ Critical Error: Cloud API is unreachable. {e}")
        sys.exit(1)

    # Step 2: Check if the user passed arguments
    if len(sys.argv) > 1:
        command = sys.argv[1].lower()
        
        # --- DAEMON MODE ---
        if command in ["daemon", "--daemon"]:
            result = run_daemon_check()
            if "ALL_GOOD" not in result:
                send_desktop_notification(result)
            sys.exit(0)
            
        # --- ONE-SHOT MODE ---
        query = " ".join(sys.argv[1:])
        history = load_memory()
        
        from rich.console import Console
        console = Console()
        console.print(f"[dim]Executing one-shot command: '{query}'...[/dim]")
        
        ask_oracle(query, history)
        sys.exit(0)
    
    # Step 3: INTERACTIVE MODE (No arguments)
    print("🚀 Starting OmniKernel System Monitor...")
    try:
        test_generation()
    except Exception as e:
        print(f"❌ Critical Error: Cloud API is unreachable. {e}")
        sys.exit(1)

    print("\n✅ System primed. Booting Agentic Interface...")
    start_chat()
    start_chat()

if __name__ == "__main__":
    main()