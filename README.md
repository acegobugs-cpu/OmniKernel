# 🧠 OmniKernel

OmniKernel is a local-first, autonomous system intelligence engine and live hardware monitor for Linux. It combines a high-performance, responsive Terminal User Interface (TUI) written in **Go** with a specialized **Python** AI reasoning engine communicating over low-latency **Unix Domain Sockets**.

---

## 🗺️ System Architecture

OmniKernel operates as a co-operating dual-process runtime:

```text
┌────────────────────────────────────────────────────────────────────────┐
│                       Go Terminal Frontend (TUI)                       │
│  - Charm Bubble Tea Event Loop & Lipgloss Dashboard                   │
│  - Live Hardware Telemetry (gopsutil CPU/RAM/Net/Disk/Temp/Load)       │
│  - Native, Sandboxed Tool Execution Engine (actions.go, monitor.go)    │
└───────────────────────────────────▲────────────────────────────────────┘
                                    │
                         HTTP over Unix Domain Socket
                         (~/.omnikernel/omni.sock)
                                    │
┌───────────────────────────────────▼────────────────────────────────────┐
│                      Python AI Brain (ai/ directory)                   │
│  - FastAPI / Uvicorn Server (api.py)                                   │
│  - Multi-Turn Conversation & Tool Routing (chat.py)                    │
│  - Global Conversation Memory (~/.omnikernel/agent_memory.json)        │
│  - Local Vector Codebase Oracle (ChromaDB + LangChain Chunking)        │
└────────────────────────────────────────────────────────────────────────┘
```

### 1. Go TUI Frontend
* **Process Manager**: Automatically provisions and manages the lifecycle of the Python background daemon on startup and guarantees clean socket teardown on exit.
* **Hardware Telemetry**: Continuously samples system hardware every second (multi-core CPU load, memory breakdown, disk mountpoints, temperature sensors, network I/O, load averages, and uptime).
* **Native Tool Execution**: Executes system commands and queries requested by the AI with hard timeout constraints, permission checks, and safety guardrails.

### 2. Python AI Brain
* **Unix Socket Server**: Hosts a lightweight FastAPI service over `~/.omnikernel/omni.sock`.
* **Autonomous Decision Loop**: Evaluates user prompts, selects appropriate native system tools, interprets tool output, and streams answers back to Go.
* **Persistent Memory**: Retains conversation history across turns in `~/.omnikernel/agent_memory.json`.

### 3. Codebase Oracle
* **Structural Chunking**: Crawls and splits code using `RecursiveCharacterTextSplitter` configured with language-aware separators.
* **Vector Store Persistence**: Embeds and indexes code fragments into a local `ChromaDB` vector database (`./.omnikernel_db`).
* **Hot Re-indexing Watcher**: A `watchdog`-based file watcher that detects file modifications, creations, or deletions and performs delta re-indexing in real time.

---

## 🖥️ Terminal Dashboard Layout

The terminal interface dynamically adapts to your window dimensions:

```text
┌──────────────────────────────────────────────┬─────────────────────────┐
│ 💻 System Overview                           │ 🔥 Cores & Temp         │
│ CPU Load: 14.2%                              │ Core 0: 12.0%           │
│ RAM Used: 42.1% (6.74 GB / 16.00 GB)         │ Core 1: 15.5%           │
│ Disks: /: 54.2%                              │ Temp: 48.0°C            │
├──────────────┬──────────────┬────────────────┴─────────────────────────┤
│ 🌐 Net I/O   │ 🖥️ Host      │ 🤖 Agent Chat                            │
│ Sent: 1.2 MB │ Ubuntu 24.04 │ User 💻: check my top processes          │
│ Recv: 8.4 MB │ Up: 2d 4h    │ ⚙️ Agent executed Go tool: check_processes│
├──────────────┼──────────────┤ OmniKernel 🧠:                           │
│ 🧠 Memory    │ ⚖️ Load Avg  │ Top CPU processes:                       │
│ Used: 6.7 GB │ 1m:  0.45    │ 1. omni (PID 12431) - 4.2% CPU           │
│ Free: 9.3 GB │ 5m:  0.62    │ 2. python (PID 12410) - 2.1% CPU         │
│ Total: 16 GB │ 15m: 0.58    │ > Ask the agent a question...            │
└──────────────┴──────────────┴──────────────────────────────────────────┘
```

---

## ⚙️ Native System Tools & Guardrails

The Python AI backend never executes operating system commands directly; instead, it delegates structured requests to Go. The Go engine enforces strict security boundaries:

| Tool Name | Parameters | Description | Security Controls |
| :--- | :--- | :--- | :--- |
| `check_hardware` | *None* | Live CPU, RAM, disk, and network statistics | Read-only |
| `check_processes` | `limit: int` | Highest CPU & memory consuming processes | Read-only, sanitized limits |
| `find_process_by_name` | `search_term: str` | Search active processes by application name | Input sanitization against shell injection |
| `kill_process` | `pid: int` | Terminate unresponsive process | Protected against PID ≤ 1; user privilege verification |
| `manage_docker` | `action: str, container_id: str` | List, stop, or restart Docker containers | Parameterized validation (list/stop/restart only) |
| `read_system_file` | `filepath: str` | Read file contents from disk | Safe UTF-8 reading |
| `write_system_file` | `filepath: str, content: str` | Create or overwrite files | **Forbidden Paths**: `/etc`, `/bin`, `/sbin`, `/usr`, `/boot`, `/sys`, `/proc`, `/dev` |
| `check_system_logs` | `lines: int, service: str` | Read system logs via `journalctl` | Hard capped at 100 lines / 3000 chars to protect LLM context windows |
| `run_shell_command` | `command: str` | Execute arbitrary terminal commands | **10-Second Hard Timeout**; blacklists `rm -rf /`, `mkfs`, `dd if=`, and fork bombs |

---

## 📂 Repository Structure

```text
OmniKernel/
├── actions.go             # Native Go execution for all agent tools & security checks
├── actions_test.go        # Unit tests for Go tool execution and safety guards
├── dashboard.go           # Lipgloss UI layout, responsive grid boxes, and formatting
├── go.mod                 # Go module definition (omnikernel-ui)
├── go.sum                 # Go dependencies checksum
├── main.go                # Bubble Tea entrypoint, telemetry sampler, and IPC manager
├── monitor.go             # Hardware telemetry gathering and system diagnostic utilities
├── omni                   # Compiled executable binary (git-ignored)
├── omnikernel.md          # Architectural blueprint & specification
├── README.md              # Project documentation
│
└── ai/                    # Python AI Reasoning Engine & Codebase Oracle
    ├── __init__.py        # Package initialization
    ├── api.py             # FastAPI UDS server (~/.omnikernel/omni.sock)
    ├── chat.py            # AI decision loop, tool definitions & memory loader
    ├── clientAi.py        # OpenAI/Groq API client initialization
    ├── health.py          # Cloud LLM provider connectivity check
    ├── pyproject.toml     # Python package metadata and dependencies
    ├── uv.lock            # Locked Python dependency versions
    │
    ├── code_assist/       # Semantic Code Ingestion & Vector Indexing
    │   ├── __init__.py    # Code assist package initialization
    │   ├── ingest.py      # Multi-file crawler and code chunker
    │   ├── store.py       # ChromaDB vector store seeding & document updates
    │   └── watcher.py     # Real-time filesystem observer for hot re-indexing
    │
    ├── conf/              # Configuration & State Management
    │   ├── __init__.py
    │   ├── env.py         # Priority-based .env loader
    │   └── mem.py         # Global agent memory path resolution
    │
    └── utils/             # Debugging & Inspection Utilities
        └── debug/
            ├── __init__.py
            └── search.py  # Interactive CLI for testing ChromaDB vector queries
```

---

## 🚀 Getting Started

### Prerequisites

* **Linux**: Ubuntu 22.04+, Debian 12+, or compatible Linux distribution
* **Go**: Version 1.22 or higher
* **Python**: Version 3.12 or higher
* **uv**: Astral's [`uv`](https://docs.astral.sh/uv/) package manager (recommended)

### 1. Configuration

OmniKernel searches for your environment configuration in `ai/.env` or globally in `~/.omnikernel/.env`:

```bash
# ~/.omnikernel/.env or ai/.env
BASE_URL="https://api.groq.com/openai/v1"
API_KEY="gsk_your_groq_or_openai_api_key_here"
```

> [!TIP]
> You can also point OmniKernel to any OpenAI-compatible API endpoint, including local inference via **Ollama** (`http://localhost:11434/v1`).

### 2. Install Python Dependencies

Using `uv`:
```bash
cd ai
uv sync
cd ..
```

### 3. Build & Run OmniKernel

Compile and launch the native application:

```bash
go build -o omni .
./omni
```

The Go frontend will automatically start the Python AI backend, connect to `~/.omnikernel/omni.sock`, and present the live monitoring dashboard.

---

## 🧪 Testing & Verification

### Run Go Unit Tests
Run the test suite verifying shell timeouts, security guards, file operations, and uptime calculations:
```bash
go test -v ./...
```

### Verify AI Cloud Connectivity
Test connectivity to your configured LLM provider:
```bash
cd ai
uv run python health.py
```

### Test Semantic Vector Search
Seed and query your local vector store manually:
```bash
# Index codebase
cd ai
uv run python -m code_assist.store

# Interactive semantic search
uv run python -m utils.debug.search
```

### Launch the Hot Re-indexing Watcher
Monitor workspace changes and automatically update the vector store:
```bash
cd ai
uv run python -m code_assist.watcher
```

---

## 🔒 Security Summary

* **Read-Only by Default**: Telemetry, logs, and process inspectors operate in read-only mode.
* **Non-Interactive Privilege Guard**: Prevents hanging on `sudo` authentication prompts.
* **Path Traversal Protection**: System directories are rejected for write actions.
* **Execution Deadlines**: All shell executions are enforced with a strict 10-second timeout context.
* **Isolated IPC**: All communication between the frontend and AI occurs locally over protected Unix Domain Sockets.
