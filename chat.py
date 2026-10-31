import os
import json
from clientAi import client
from mem import MEMORY_FILE
from monitor import get_system_metrics_text, get_top_processes_text, get_system_logs, find_process_by_name
from actions import kill_process, manage_docker, read_system_file, run_shell_command, list_python_packages

from rich.console import Console
from rich.markdown import Markdown

# Initialize the rich console
console = Console()

# 1. Define the "Menu" of tools the AI is allowed to use
AVAILABLE_TOOLS = [
    # ... (keep your existing check_hardware and check_processes tools here) ...
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
    # {
    #     "type": "function",
    #     "function": {
    #         "name": "list_python_packages",
    #         "description": "List all installed Python dependencies and pip packages.",
    #         "parameters": {"type": "object", "properties": {}}
    #     }
    # }
]


def load_memory():
    """Loads the previous conversation history from disk."""
    if os.path.exists(MEMORY_FILE):
        try:
            with open(MEMORY_FILE, 'r', encoding='utf-8') as f:
                # Load history, but only keep the last 20 messages 
                # to prevent hitting the API token limit on long conversations.
                history = json.load(f)
                return history[-20:] 
        except Exception as e:
            console.print(f"⚠️ Could not load memory: {e}")
    return []

def save_memory(history):
    """Saves the current conversation history to disk."""
    try:
        with open(MEMORY_FILE, 'w', encoding='utf-8') as f:
            json.dump(history, f, indent=4)
    except Exception as e:
        console.print(f"⚠️ Could not save memory: {e}")

def ask_oracle(query, chat_history):
    # # --- PHASE 1: Codebase RAG ---
    # results = collection.query(query_texts=[query], n_results=3)
    # retrieved_context = ""
    # if results and results['documents']:
    #     for doc, metadata in zip(results['documents'][0], results['metadatas'][0]):
    #         retrieved_context += f"File: {metadata['relative_path']}\nCode:\n{doc}\n---\n"

    # augmented_query = f"Code Context:\n{retrieved_context}\nUser Question: {query}"

    api_messages = [
        {"role": "system", "content": "You are OmniKernel, an autonomous system monitor and codebase oracle. Use your tools to check system health if the user asks about it."}
    ]
    api_messages.extend(chat_history)
    api_messages.append({"role": "user", "content": query})
    # api_messages.append({"role": "user", "content": augmented_query})

    # --- PHASE 2: The Agentic Loop ---
    # The agent will stay in this loop until all tasks are finished.
    final_answer = ""
    
    while True:
        response = client.chat.completions.create(
            model="openai/gpt-oss-120b",
            messages=api_messages,
            tools=AVAILABLE_TOOLS,
            tool_choice="auto"
        )
        
        response_message = response.choices[0].message
        
        # If the AI decides it needs to use a tool...
        if response_message.tool_calls:
            # 1. Save the AI's tool request to the message history
            api_messages.append(response_message)
            
            # 2. Execute every tool the AI asked for
            for tool_call in response_message.tool_calls:
                function_name = tool_call.function.name
                args = json.loads(tool_call.function.arguments) if tool_call.function.arguments else {}
                
                if function_name == "check_hardware":
                    console.print("⚙️ Agent triggered: [check_hardware]")
                    tool_result = get_system_metrics_text()
                    
                elif function_name == "check_processes":
                    console.print("⚙️ Agent triggered: [check_processes]")
                    tool_result = get_top_processes_text(limit=args.get("limit", 5))
                    
                elif function_name == "kill_process":
                    console.print(f"⚠️ Agent actively killing PID {args.get('pid')}...")
                    tool_result = kill_process(pid=args.get("pid"))
                    
                elif function_name == "manage_docker":
                    console.print(f"🐳 Agent triggered Docker action: {args.get('action')}")
                    tool_result = manage_docker(action=args.get("action"), container_id=args.get("container_id", ""))
                    
                elif function_name == "read_system_file":
                    console.print(f"📂 Agent reading file: {args.get('filepath')}")
                    tool_result = read_system_file(filepath=args.get("filepath"))
                    
                elif function_name == "find_process_by_name":
                    console.print(f"🔎 Agent searching for process: {args.get('search_term')}")
                    tool_result = find_process_by_name(search_term=args.get("search_term"))

                elif function_name == "check_system_logs":
                    console.print(f"📖 Agent reading system logs...")
                    
                    # If the AI sends null, Python reads it as None. Convert it back to an empty string.
                    raw_service = args.get("service")
                    safe_service = raw_service if raw_service is not None else ""
                    
                    tool_result = get_system_logs(
                        lines=args.get("lines", 50), 
                        service=safe_service
                    )

                elif function_name == "run_shell_command":
                    console.print(f"💻 Agent running shell command: {args.get('command')}")
                    tool_result = run_shell_command(command=args.get("command"))

                # elif function_name == "list_python_packages":
                #     console.print("📦 Agent checking installed packages...")
                #     tool_result = list_python_packages()  

                else:
                    tool_result = f"Error: Tool {function_name} not found."
                
                # 3. Feed the result back into the history so the AI can read it
                # Convert the result to a string and hard-cap it at 4000 characters
                safe_result = str(tool_result)
                if len(safe_result) > 4000:
                    safe_result = safe_result[:4000] + "\n...[SYSTEM: OUTPUT TRUNCATED TO SAVE TOKENS]"
                
                api_messages.append({
                    "role": "tool",
                    "tool_call_id": tool_call.id,
                    "content": safe_result
                })
            
            # The loop automatically repeats here! The AI reads the tool results 
            # and decides if it needs to call another tool (like kill_process).
            
        else:
            # If no tools were called, the AI is done working and gave us text.
            console.print("\n[bold cyan]OmniKernel 🧠:[/bold cyan]")
            console.print(Markdown(response_message.content))
            print("\n")
            final_answer = response_message.content
            break

    # Save the final interaction to conversation memory
    chat_history.append({"role": "user", "content": query})
    chat_history.append({"role": "assistant", "content": final_answer})

    # NEW: Persist the updated list to the hard drive
    save_memory(chat_history)

def start_chat():
    print("\n====================================================")
    print("   🌐 OmniKernel Phase 2: Autonomous Agent Active   ")
    print("====================================================")
    print("Ask about your code, or your system health. Type 'exit' to quit.\n")
    
    history = load_memory()
    if history:
        console.print(f"🧠 Memory loaded. I remember our last {len(history) // 2} interactions.\n")

    while True:
        user_input = input("User 💻 >> ")
        if user_input.strip().lower() == "exit":
            break
        if not user_input.strip():
            continue
            
        try:
            ask_oracle(user_input, history)
        except Exception as e:
            console.print(f"\n❌ Pipeline Error: {e}\n")

if __name__ == "__main__":
    start_chat()