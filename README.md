# 🧠 OmniKernel

**OmniKernel** is a local-first, autonomous system intelligence engine and monitor for Linux. It combines a high-performance, real-time terminal UI (TUI) written in **Go** with a specialized **Python** AI reasoning engine communicating seamlessly over low-latency **Unix Domain Sockets**.

---

## 🗺️ Architecture Overview

OmniKernel operates as a unified dual-process system:

```text
┌────────────────────────────────────────────────────────┐
│               Go Terminal Frontend (TUI)               │
│  - Charm Bubble Tea & Lipgloss Dashboard              │
│  - Live Hardware Sensors (gopsutil CPU/RAM/Net/Disk)   │
│  - Secure Native Tool Execution Engine                 │
└───────────────────────────▲────────────────────────────┘
                            │  HTTP over Unix Socket
                            │  (~/.omnikernel/omni.sock)
┌───────────────────────────▼────────────────────────────┐
│              Python AI Reasoning Brain (ai/)           │
│  - FastAPI / Uvicorn Server                            │
│  - Tool Selection & Conversational Memory Routing     │
│  - Local Vector Codebase Oracle (ChromaDB + LangChain) │
└────────────────────────────────────────────────────────┘
```

1. **Go TUI Frontend (`main.go`, `dashboard.go`)**:
   - Manages interactive user chat and terminal resize events.
   - Monitors real-time hardware telemetry: overall CPU, per-core utilization, RAM breakdown, disk mounts, temperatures, network throughput, system uptime, and load averages.
   - Dispatches and enforces security guardrails for AI tool executions natively in Go (`actions.go`, `monitor.go`).

2. **Python AI Brain (`ai/api.py`, `ai/chat.py`)**:
   - Boots automatically as a background daemon listening on `~/.omnikernel/omni.sock`.
   - Interacts with LLM providers (supporting OpenAI-compatible cloud endpoints such as Groq, or local Ollama).
   - Maintains conversational history in persistent memory (`~/.omnikernel/agent_memory.json`).

3. **Codebase Oracle (`ai/code_assist/`)**:
   - Semantic code chunking via `langchain-text-splitters`.
   - Local vector database persistence powered by `ChromaDB`.
   - Continuous file-system watcher (`watchdog`) for hot delta re-indexing upon file edits.

---

## ⚙️ Built-In Native Agent Tools

OmniKernel executes system tools natively with strict guardrails:

| Tool Name | Description | Safety Guardrails |
| :--- | :--- | :--- |
| `check_hardware` | Inspect live CPU, RAM, disk, and network stats | Read-only |
| `check_processes` | List top CPU/memory consuming processes | Read-only |
| `find_process_by_name` | Locate running processes by name to resolve PIDs | Sanitized query strings |
| `kill_process` | Terminate frozen or rogue processes by PID | Protects PID ≤ 1; user privilege enforcement |
| `manage_docker` | List, stop, or restart Docker containers | Strict parameter validation |
| `read_system_file` | Read arbitrary configuration or system files | Safe UTF-8 file reading |
| `write_system_file` | Write or update configuration files | Blocks write attempts to `/etc`, `/bin`, `/sbin`, `/usr`, `/boot`, `/sys`, `/proc`, `/dev` |
| `check_system_logs` | Retrieve `journalctl` logs for specific services | Hard capped at 100 lines / 3000 chars to protect LLM context windows |
| `run_shell_command` | Execute arbitrary bash commands | 10-second hard execution timeout; blacklists destructive commands (`rm -rf /`, `mkfs`, fork bombs) |

---

## 🚀 Getting Started

### Prerequisites

- **Linux** (Ubuntu / Debian or compatible distribution)
- **Go** (version 1.22+)
- **Python** (version 3.12+) and [`uv`](https://docs.astral.sh/uv/) (recommended)

### Environment Configuration

Configure your environment variables in `ai/.env` or `~/.omnikernel/.env`:

```bash
BASE_URL="https://api.groq.com/openai/v1"
API_KEY="your-api-key-here"
```

### Installation & Launch

1. **Install Python backend dependencies:**
   ```bash
   cd ai
   uv sync
   cd ..
   ```

2. **Build and run the OmniKernel application:**
   ```bash
   go build -o omni .
   ./omni
   ```

The Go frontend will automatically boot the Python AI backend, link to the Unix domain socket, and display the live dashboard.

---

## 🧪 Testing

Run native Go unit tests:
```bash
go test -v ./...
```

Verify the Python AI health check:
```bash
cd ai
uv run python health.py
```

Index the local codebase into the ChromaDB vector database:
```bash
cd ai
uv run python -m code_assist.store
```
