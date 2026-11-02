import os
import sys
import json
from pathlib import Path

_CURRENT_DIR = Path(__file__).resolve().parent
_REPO_ROOT = _CURRENT_DIR.parent
for _p in (str(_REPO_ROOT), str(_CURRENT_DIR)):
    if _p not in sys.path:
        sys.path.insert(0, _p)

try:
    from ai.clientAi import client
    from ai.conf.mem import MEMORY_FILE
except (ImportError, ModuleNotFoundError):
    from clientAi import client
    from conf.mem import MEMORY_FILE

# 1. Define the "Menu" of tools the AI is allowed to use
AVAILABLE_TOOLS = [
    {
        "type": "function",
        "function": {
            "name": "check_hardware",
            "description": "Get live hardware metrics: CPU, RAM, Battery, and Temperature.",
            "parameters": {"type": "object", "properties": {}}
        }
    },
    {
        "type": "function",
        "function": {
            "name": "check_processes",
            "description": "Get the top running applications consuming the most CPU and RAM.",
            "parameters": {
                "type": "object",
                "properties": {
                    "limit": {
                        "type": "integer",
                        "description": "Number of top processes to return (default 5)"
                    }
                }
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "find_process_by_name",
            "description": "Search for a specific running application or process by its name (e.g., 'firefox', 'chrome', 'code') to find its PID.",
            "parameters": {
                "type": "object",
                "properties": {
                    "search_term": {"type": "string", "description": "The name of the application to search for."}
                },
                "required": ["search_term"]
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "kill_process",
            "description": "Kill a rogue or frozen application using its PID.",
            "parameters": {
                "type": "object",
                "properties": {
                    "pid": {"type": "integer", "description": "The Process ID (PID) to kill."}
                },
                "required": ["pid"]
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "manage_docker",
            "description": "List running docker containers, or restart/stop a specific container.",
            "parameters": {
                "type": "object",
                "properties": {
                    "action": {"type": "string", "enum": ["list", "restart", "stop"]},
                    "container_id": {"type": "string", "description": "The ID or name of the container (only if restarting or stopping)."}
                },
                "required": ["action"]
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "read_system_file",
            "description": "Read the contents of a specific file on the hard drive by providing its absolute or relative path.",
            "parameters": {
                "type": "object",
                "properties": {
                    "filepath": {"type": "string", "description": "The path to the file to read."}
                },
                "required": ["filepath"]
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "write_system_file",
            "description": "Write or overwrite a file with new content. System directories like /etc and /bin are protected.",
            "parameters": {
                "type": "object",
                "properties": {
                    "filepath": {"type": "string", "description": "The destination path of the file."},
                    "content": {"type": "string", "description": "The content to write into the file."}
                },
                "required": ["filepath", "content"]
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "check_system_logs",
            "description": "Read the recent Ubuntu system logs using journalctl. Useful for diagnosing crashes or service failures.",
            "parameters": {
                "type": "object",
                "properties": {
                    "lines": {"type": "integer", "description": "Number of recent log lines to fetch (default 50)."},
                    "service": {
                        "type": ["string", "null"], 
                        "description": "Optional: Filter logs by a specific service name (e.g., 'docker'). Pass null if not filtering."
                    }
                }
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "run_shell_command",
            "description": "Execute a standard Linux terminal command (e.g., ping, netstat, ls, grep, apt). Use this for general system diagnostics.",
            "parameters": {
                "type": "object",
                "properties": {
                    "command": {"type": "string", "description": "The bash command to execute."}
                },
                "required": ["command"]
            }
        }
    },
]


def load_memory():
    if os.path.exists(MEMORY_FILE):
        try:
            with open(MEMORY_FILE, 'r', encoding='utf-8') as f:
                return json.load(f)[-20:]
        except (json.JSONDecodeError, OSError):
            return []
    return []

def save_memory(history):
    try:
        with open(MEMORY_FILE, 'w', encoding='utf-8') as f:
            json.dump(history, f, indent=4)
    except OSError as e:
        print(f"⚠️ Failed to save memory: {e}")

def step_oracle(query=None, tool_call_id=None, tool_result=None):
    """Processes a single AI turn and either returns a final answer or requests a tool execution."""
    api_messages = [{"role": "system", "content": "You are OmniKernel, an autonomous Ubuntu system monitor."}]
    history = load_memory()

    # Append new context based on what Go just sent us
    if tool_result and tool_call_id:
        history.append({
            "role": "tool",
            "tool_call_id": tool_call_id,
            "content": str(tool_result)
        })
    elif query:
        history.append({"role": "user", "content": query})

    api_messages.extend(history)

    # Ask the AI what to do next
    try:
        response = client.chat.completions.create(
            model="openai/gpt-oss-120b",
            messages=api_messages,
            tools=AVAILABLE_TOOLS,
            tool_choice="auto"
        )
    except Exception as e:
        return {
            "status": "done",
            "response": f"❌ Error communicating with AI provider: {e}"
        }

    msg = response.choices[0].message

    # DECISION A: The AI wants to run a tool. Ask Go to run it.
    if msg.tool_calls:
        tool = msg.tool_calls[0]  # We process one at a time for simplicity
        tool_id = tool.id or "call_default"
        
        # Save the AI's intent to memory so we remember why we are running this tool
        history.append({
            "role": "assistant", 
            "tool_calls": [{"id": tool_id, "type": "function", "function": {"name": tool.function.name, "arguments": tool.function.arguments}}]
        })
        save_memory(history)
        
        return {
            "status": "tool_call",
            "tool_call_id": tool_id,
            "name": tool.function.name,
            "arguments": tool.function.arguments
        }
        
    # DECISION B: The AI is done thinking and has a final text answer.
    else:
        history.append({"role": "assistant", "content": msg.content or ""})
        save_memory(history)
        
        return {
            "status": "done",
            "response": msg.content or ""
        }
