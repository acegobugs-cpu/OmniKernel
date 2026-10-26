# AI Engineer: Project Blueprint

## Project Identity
**Project Name:** **OmniKernel**  
*Concept:* A localized intelligence engine that starts by mastering codebase structures (Phase 1) and progressively branches outward to consume and manage your entire Linux ecosystem (Phase 2 & 3).

---

## 🗺️ Master Architecture & Project Evolution (The Tree)

```text
OmniKernel (Root Project)
└── Phase 1: The Codebase Oracle (Standalone CLI Application) [Target 1]
    │   ├── Feature: Local Multi-File Ingestion Pipeline
    │   ├── Feature: Code-Specific Semantic Chunking
    │   ├── Feature: Local Vector Seeding & Vector Store Persistence
    │   └── Feature: Context-Constrained Local LLM Chat Loop
    │
    └── Phase 2: The Desktop Semantic Layer (Extension of Phase 1 Core) [Target 2]
        │   ├── Feature: OS File-System Watcher (~/.config, ~/.bashrc, ~/Documents)
        │   ├── Feature: Automated Delta Ingestion & Re-indexing
        │   └── Feature: Multi-Collection Routing Logic (Switch between Code / System Docs)
        │
        └── Phase 3: The Agentic System Executor (Full OS Integration) [Target 3]
            ├── Feature: Function-Calling Interface (JSON Command Schemes)
            ├── Feature: Read-Only System Execution Sandbox (bash/subprocess)
            └── Feature: Dual RAG/Agentic Self-Correction Control Loop
```

---

## 🛠️ Phase 1 Detailed Specification: The Codebase Oracle

Phase 1 delivers a fully local, self-contained Command Line Interface (CLI) tool. You point it at any local Git repository or directory containing code, it indexes the directory into a local vector database, and it starts an interactive chat terminal session. The LLM answers software architecture, debugging, and implementation questions using *only* your local files as context.

### 1. Technical Stack Selection
*   **Host Language:** Python 3.10+ (Recommended for seamless integration with AI packages and standard OS data-wrangling).
*   **AI Orchestration Framework:** `LangChain` or `LlamaIndex` (Choose **LangChain** for explicit control, or write standard HTTP requests to Ollama if you prefer no framework magic).
*   **Local Inference Engine:** **Ollama** running locally on your hardware.
*   **Local LLM Models:**
    *   *Text Embeddings:* `nomic-embed-text` (State-of-the-art local embeddings with variable context windows).
    *   *Text Generation:* `llama3` (8B) or `mistral` (7B) (Highly responsive, localized, parameter-dense models).
*   **Vector Storage Layer:** **ChromaDB** (Lightweight, embedded, file-system persistent database requiring zero external daemons).

### 2. Step-by-Step Execution Plan

#### Step 1.1: Project Setup & Inference Verification
1. Install dependencies via pip or poetry: `chromadb`, `ollama`, `langchain` (optional), and `tqdm` (for ingestion progress bars).
2. Download and launch Ollama on your machine. Run `ollama pull nomic-embed-text` and `ollama pull llama3`.
3. Write a small sanity-check script (`test_infra.py`) to hit Ollama's local embedding endpoint with a string and verify it returns a 768-dimension array of floating-point numbers.

#### Step 1.2: Code Ingestion & Structural Chunking Pipeline
1. Build a local path crawler script. Implement custom exclusion rules to filter out binary assets and structural noise (`.git/`, `node_modules/`, `dist/`, `__pycache__/`, lockfiles).
2. Read allowed code files (`.py`, `.js`, `.ts`, `.rs`, `.go`, etc.) as raw strings.
3. **The Chunking Strategy:** Use a `RecursiveCharacterTextSplitter` configured with code-specific separators (`class `, `def `, `function `, `

`, `
`). Set `chunk_size=1000` tokens and `chunk_overlap=150` tokens. 
4. **Metadata Attachment:** Every chunk must retain its absolute file path, relative file path, language extension, and chunk index number as a metadata dictionary.

#### Step 1.3: Vector Seeding & Persistence
1. Initialize a local database instance pointing to a persistent directory: `chromadb.PersistentClient(path="./.omnikernel_db")`.
2. Batch-process your code chunks. For each batch, extract text elements, request embedding vectors from Ollama, and save the data to ChromaDB alongside the metadata keys.

#### Step 1.4: Context-Augmented Query and Generation Loop
1. Build an interactive CLI loop: `while True: query = input("OmniKernel >> ")`.
2. Convert the incoming user prompt string into a vector embedding.
3. Perform a $K$-Nearest Neighbors ($K$-NN) vector similarity query inside ChromaDB to fetch the top 3 or 4 code chunks containing relevant algorithmic or business logic.
4. Construct the **System Prompt** injected directly into the LLM context window:

```text
You are an expert software architecture and code comprehension engine. 
Answer the user's question USING ONLY the relevant code context chunks provided below. 
Do not assume, invent, or project functionality not present in the text context. 
If the answer cannot be confidently deduced from the given source material, states explicitly: "Target context not found within current indexing boundaries."

---
ACCESSIBLE CODE CONTEXT:
[For each chunk retrieved, render]:
Source File: {relative_path}
Source Code:
{chunk_text}
---

USER PROMPT: {user_query}
RESPONSE:
```

5. Pipe this full compiled string to Ollama's generation endpoint, stream the tokens directly to stdout for real-time printing, and maintain standard conversational history records for subsequent queries.

---

## 🧠 Prior Knowledge Requirements (What you need on day one)

Since you are already an applications developer, you do *not* need to review loops, package managers, or language syntax. You only need to verify the following developer prerequisites:
1. **Asynchronous I/O operations:** You must feel comfortable parsing systemic directories, reading file encodings (safely catching `UnicodeDecodeError`), and piping streaming streams.
2. **REST API Consumption:** Understanding standard JSON request/response bodies, HTTP authorization parameters, and long-polling stream behaviors (SSE / Server-Sent Events).

---

## 🎯 Phase 1 Gained Experience (XP Profile)

Completing Phase 1 safely transitions your engineering skill set into the foundational AI tier:

*   **Semantic Data Representation ($+500\ XP$):** Moving past basic keyword searches (`grep`) to mathematical proximity scoring. You will gain a tactile understanding of how high-dimensional token vector spaces operate.
*   **Context Window Engineering ($+400\ XP$):** Gaining real execution experience dealing with LLM context window thresholds, tracking token budgets, managing information density, and eliminating system hallucinations.
*   **Deterministic Prompt Structuring ($+300\ XP$):** Mastering how to command an LLM to follow strict logic parameters, enforce behavioral boundaries, and cleanly separate code data boundaries from instructional spaces.
*   **Unstructured Data Pipeline Automation ($+400\ XP$):** Transforming raw, unstructured text files into searchable, optimized structural indices mapping cleanly to multi-modal database tables.
*   **Function Calling & Tool Use (+600 XP):** You will learn how to make an LLM reliably output structured JSON commands instead of conversational text so your code can trigger system functions.

*   **Agentic Workflows (+500 XP):** You'll build a loop where the AI can think, act (run a command), look at the error output, and fix its command if it fails.

*   **Security & Guardrails (+700 XP):** You will face a major software design challenge: How do I give an AI access to my terminal without accidentally letting it run rm -rf /? (Hint: You strictly parse and validate string inputs, or run commands through a safe, read-only user profile).
