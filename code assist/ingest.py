import os
from langchain_text_splitters import RecursiveCharacterTextSplitter

# 1. Define allowed file extensions we want our oracle to read
ALLOWED_EXTENSIONS = {'.py', '.js', '.ts', '.jsx', '.tsx', '.go', '.rs', '.java', '.cpp', '.h'}

# 2. Define directories we strictly want to ignore
IGNORED_DIRS = {'node_modules', '.git', '__pycache__', 'dist', 'build', '.venv', '.env'}

def crawl_codebase(root_dir):
    """Walks the directory and reads valid code files into memory."""
    raw_documents = []
    
    print(f"📁 Scanning directory: {root_dir}")
    
    for dirpath, dirnames, filenames in os.walk(root_dir):
        # Modifying dirnames in-place allows os.walk to skip ignored directories completely
        dirnames[:] = [d for d in dirnames if d not in IGNORED_DIRS]
        
        for filename in filenames:
            ext = os.path.splitext(filename)[1].lower()
            if ext in ALLOWED_EXTENSIONS:
                absolute_path = os.path.abspath(os.path.join(dirpath, filename))
                relative_path = os.path.relpath(absolute_path, root_dir)
                
                try:
                    with open(absolute_path, 'r', encoding='utf-8') as f:
                        content = f.read()
                    
                    raw_documents.append({
                        "content": content,
                        "metadata": {
                            "absolute_path": absolute_path,
                            "relative_path": relative_path,
                            "language": ext.strip('.')
                        }
                    })
                except UnicodeDecodeError:
                    # Ignore binaries or weirdly encoded files safely
                    continue
                except Exception as e:
                    print(f"⚠️ Error reading {relative_path}: {e}")
                    
    for i in raw_documents:                    
        print(f"✅ Found {i["metadata"]} code files.")
    return raw_documents

def chunk_documents(documents):
    """Splits raw file text into structural, token-friendly segments."""
    print("✂️ Chunking source files into segments...")
    
    # We configure a splitter with typical code structural characters
    splitter = RecursiveCharacterTextSplitter(
        chunk_size=1000,
        chunk_overlap=150,
        separators=["\nclass ", "\ndef ", "\nfunction ", "\nconst ", "\n\n", "\n", " "]
    )
    
    all_chunks = []
    
    for doc in documents:
        # Split the text string into an array of smaller strings
        chunks = splitter.split_text(doc["content"])
        
        for idx, chunk_text in enumerate(chunks):
            # Maintain an exact reference map back to the origin file
            chunk_data = {
                "text": chunk_text,
                "metadata": {
                    **doc["metadata"],
                    "chunk_index": idx
                }
            }
            all_chunks.append(chunk_data)
            
    print(f"🧩 Created {len(all_chunks)} total chunks.")
    return all_chunks

if __name__ == "__main__":
    # Test it on your current project directory (which has test_infra.py and ingest.py)
    current_dir = os.path.dirname(os.path.abspath(__file__))
    
    source_files = crawl_codebase(current_dir)
    processed_chunks = chunk_documents(source_files)
    
    # Let's print out a sample chunk to verify everything works
    if processed_chunks:
        print("\n--- SAMPLE CHUNK METADATA ---")
        print(processed_chunks[0]["metadata"])
        print("\n--- SAMPLE CHUNK TEXT ---")
        print(processed_chunks[0]["text"])