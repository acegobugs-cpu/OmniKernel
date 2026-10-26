import os
import json
import chromadb
from clientAi import client
from monitor import get_system_metrics_text, get_top_processes_text, find_process_by_name
from actions import kill_process, manage_docker, read_system_file, list_python_packages

chroma_client = chromadb.PersistentClient(path="./.omnikernel_db")
collection = chroma_client.get_or_create_collection(name="codebase_chunks")

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
    # {
    #     "type": "function",
    #     "function": {
    #         "name": "list_python_packages",
    #         "description": "List all installed Python dependencies and pip packages.",
    #         "parameters": {"type": "object", "properties": {}}
    #     }
    # }
]

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
                    print("⚙️ Agent triggered: [check_hardware]")
                    tool_result = get_system_metrics_text()
                    
                elif function_name == "check_processes":
                    print("⚙️ Agent triggered: [check_processes]")
                    tool_result = get_top_processes_text(limit=args.get("limit", 5))
                    
                elif function_name == "kill_process":
                    print(f"⚠️ Agent actively killing PID {args.get('pid')}...")
                    tool_result = kill_process(pid=args.get("pid"))
                    
                elif function_name == "manage_docker":
                    print(f"🐳 Agent triggered Docker action: {args.get('action')}")
                    tool_result = manage_docker(action=args.get("action"), container_id=args.get("container_id", ""))
                    
                elif function_name == "read_system_file":
                    print(f"📂 Agent reading file: {args.get('filepath')}")
                    tool_result = read_system_file(filepath=args.get("filepath"))
                    
                elif function_name == "find_process_by_name":
                    print(f"🔎 Agent searching for process: {args.get('search_term')}")
                    tool_result = find_process_by_name(search_term=args.get("search_term"))

                # elif function_name == "list_python_packages":
                #     print("📦 Agent checking installed packages...")
                #     tool_result = list_python_packages()  

                else:
                    tool_result = f"Error: Tool {function_name} not found."
                
                # 3. Feed the result back into the history so the AI can read it
                api_messages.append({
                    "role": "tool",
                    "tool_call_id": tool_call.id,
                    "content": str(tool_result)
                })
            
            # The loop automatically repeats here! The AI reads the tool results 
            # and decides if it needs to call another tool (like kill_process).
            
        else:
            # If no tools were called, the AI is done working and gave us text.
            print(f"\nOmniKernel 🧠: {response_message.content}\n")
            final_answer = response_message.content
            break

    # Save the final interaction to conversation memory
    chat_history.append({"role": "user", "content": query})
    chat_history.append({"role": "assistant", "content": final_answer})

def start_chat():
    print("\n====================================================")
    print("   🌐 OmniKernel Phase 2: Autonomous Agent Active   ")
    print("====================================================")
    print("Ask about your code, or your system health. Type 'exit' to quit.\n")
    
    history = []
    while True:
        user_input = input("User 💻 >> ")
        if user_input.strip().lower() == "exit":
            break
        if not user_input.strip():
            continue
            
        try:
            ask_oracle(user_input, history)
        except Exception as e:
            print(f"\n❌ Pipeline Error: {e}\n")

if __name__ == "__main__":
    start_chat()