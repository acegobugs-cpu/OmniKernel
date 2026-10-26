#some thing in the way
import os
import time
from watchdog.observers import Observer
from watchdog.events import FileSystemEventHandler
from ingest import crawl_codebase, chunk_documents
from store import update_file_in_database, remove_file_from_database
# We will import our database updater logic here in a moment

# Reuse the same configuration from your ingestion step
ALLOWED_EXTENSIONS = {'.py', '.js', '.ts', '.jsx', '.tsx', '.go', '.rs', '.java', '.cpp', '.h'}
IGNORED_DIRS = {'node_modules', '.git', '__pycache__', 'dist', 'build', 'venv', '.env', '.omnikernel_db'}

class CodebaseHandler(FileSystemEventHandler):
    """Listens for file changes and filters them based on our project rules."""
    
    def on_modified(self, event):
        if event.is_directory: return
        self._route_event(event.src_path, "modified")

    def on_created(self, event):
        if event.is_directory: return
        self._route_event(event.src_path, "created")

    def on_deleted(self, event):
        if event.is_directory: return
        self._route_event(event.src_path, "deleted")

    def _route_event(self, file_path, event_type):
        """Helper to filter paths and route the event to the right database action."""
        filename = os.path.basename(file_path)
        ext = os.path.splitext(filename)[1].lower()
        
        path_parts = set(file_path.split(os.sep))
        if path_parts.intersection(IGNORED_DIRS):
            return 
            
        if ext in ALLOWED_EXTENSIONS:
            current_dir = os.path.dirname(os.path.abspath(__file__))
            relative_path = os.path.relpath(file_path, current_dir)
            
            if event_type == "deleted":
                print(f"\n🚨 File Deletion Detected: {filename}")
                remove_file_from_database(relative_path)
            else:
                print(f"\n⚡ File {event_type.capitalize()} Detected: {filename}")
                self.process_updated_file(file_path)

    def process_updated_file(self, absolute_path):
        """This function runs every time you hit save on a valid code file."""
        print(f"🔄 Hot-reindexing: {absolute_path}")
        
        # Read the updated file content
        try:
            with open(absolute_path, 'r', encoding='utf-8') as f:
                content = f.read()
                
            # Create a document structure matching what store.py expects
            current_dir = os.path.dirname(os.path.abspath(__file__))
            relative_path = os.path.relpath(absolute_path, current_dir)
            ext = os.path.splitext(absolute_path)[1].lower()
            
            doc = {
                "content": content,
                "metadata": {
                    "absolute_path": absolute_path,
                    "relative_path": relative_path,
                    "language": ext.strip('.') if 'ext' in locals() else absolute_path.split('.')[-1]
                }
            }
            
            # Phase 2 Goal: We will pass this to an automated delta update module next!
            new_chunks = chunk_documents([doc])
            update_file_in_database(relative_path, new_chunks)

            print(f"✅ Successfully read and rebuilt payload for {relative_path}")
            
        except Exception as e:
            print(f"⚠️ Watcher update failure: {e}")

def start_watching(target_directory):
    event_handler = CodebaseHandler()
    observer = Observer()
    observer.schedule(event_handler, path=target_directory, recursive=True)
    observer.start()
    
    print(f"👁️  OmniKernel File System Watcher active on: {target_directory}")
    print("Press Ctrl+C to stop.")
    
    try:
        while True:
            time.sleep(1) # Keep the background thread alive smoothly
    except KeyboardInterrupt:
        print("\nStopping file system watcher...")
        observer.stop()
    observer.join()

if __name__ == "__main__":
    # Watch our current development environment directory
    current_workspace = os.path.dirname(os.path.abspath(__file__))
    start_watching(current_workspace)