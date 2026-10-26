import os
import sys
import threading

# Import from your existing files
from health import test_generation
from ingest import crawl_codebase, chunk_documents
from store import seed_vector_database, is_database_populated
from watcher import start_watching
from chat import start_chat

def main():
    print("🚀 Starting OmniKernel Master Pipeline...")
    
    # Step 1: Health Check
    print("\n--- Phase 1: Cloud Health Check ---")
    try:
        test_generation()
    except Exception as e:
        print(f"❌ Critical Error: Cloud API is unreachable. {e}")
        sys.exit(1)

    # # Step 2: Ingest and Store Codebase
    # print("\n--- Phase 2: Indexing Workspace ---")
    # current_workspace = os.path.dirname(os.path.abspath(__file__))
    # if is_database_populated():
    #     print("⚡ Database already exists. Skipping full index to save time.")
    # else:
    #     source_files = crawl_codebase(current_workspace)
    #     if not source_files:
    #         print("⚠️ No valid code files found.")
    #     else:
    #         processed_chunks = chunk_documents(source_files)
    #         seed_vector_database(processed_chunks)
        
    # # Step 3: Start Watcher in the Background
    # print("\n--- Phase 3: Starting Background File Watcher ---")
    # # We set daemon=True so this background thread automatically dies when you type 'exit'
    # watcher_thread = threading.Thread(
    #     target=start_watching, 
    #     args=(current_workspace,), 
    #     daemon=True
    # )
    # watcher_thread.start()#sfdklakj
    
    # Step 4: Start Interactive Chat Loop in the Foreground
    print("\n--- Phase 4: Chat Interface Online ---")
    print("Ask questions about your indexed files. Type 'exit' to quit.\n")
    
    start_chat()

if __name__ == "__main__":
    main()