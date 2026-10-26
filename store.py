import os
import chromadb
from ingest import crawl_codebase, chunk_documents

def is_database_populated():
    """Checks if the vector database already contains indexed files."""
    chroma_client = chromadb.PersistentClient(path="./.omnikernel_db")
    collection = chroma_client.get_or_create_collection(name="codebase_chunks")
    
    # .count() returns the number of chunks currently in the database
    return collection.count() > 0

def seed_vector_database(chunks):
    """Initializes local ChromaDB and saves the code chunks using local CPU embedding."""
    print("\n📦 Initializing local vector database...")
    
    # 1. Create a persistent client that saves data to a hidden folder on disk
    chroma_client = chromadb.PersistentClient(path="./.omnikernel_db")
    
    # 2. Get or create a collection. 
    # By NOT passing an embedding function, ChromaDB defaults to a built-in, 
    # lightweight local CPU math model (all-MiniLM-L6-v2).
    collection = chroma_client.get_or_create_collection(name="codebase_chunks")
    
    print(f"🧬 Indexing {len(chunks)} chunks into ChromaDB. Please wait...")
    
    # We can batch insert into ChromaDB to make it much faster!
    ids = []
    documents = []
    metadatas = []
    
    for idx, chunk in enumerate(chunks):
        chunk_id = f"chunk_{chunk['metadata']['relative_path']}_{idx}"
        ids.append(chunk_id)
        documents.append(chunk["text"])
        metadatas.append(chunk["metadata"])
        
    # 3. Add everything to the collection at once
    if ids:
        collection.add(
            ids=ids,
            documents=documents,
            metadatas=metadatas
        )
        
    print(f"✅ Successfully indexed {len(chunks)} fragments into ./.omnikernel_db")


def update_file_in_database(relative_path, new_chunks):
    """Deletes old chunks for a specific file and inserts the new ones."""
    print(f"♻️ Updating database for: {relative_path}")
    
    chroma_client = chromadb.PersistentClient(path="./.omnikernel_db")
    collection = chroma_client.get_or_create_collection(name="codebase_chunks")
    
    # 1. Delete all existing chunks for this specific file
    collection.delete(where={"relative_path": relative_path})
    
    # 2. Add the newly saved chunks
    if new_chunks:
        seed_vector_database(new_chunks)
    else:
        print(f"🗑️ File {relative_path} is empty. Removed from index.")
        

def remove_file_from_database(relative_path):
    """Deletes all chunks for a file that was removed from the workspace."""
    print(f"🗑️ Purging deleted file from database: {relative_path}")
    
    chroma_client = chromadb.PersistentClient(path="./.omnikernel_db")
    collection = chroma_client.get_or_create_collection(name="codebase_chunks")
    
    collection.delete(where={"relative_path": relative_path})


if __name__ == "__main__":
    current_dir = os.path.dirname(os.path.abspath(__file__))
    
    # 1. Crawl files
    raw_docs = crawl_codebase(current_dir)
    
    # 2. Chop into paragraphs
    all_chunks = chunk_documents(raw_docs)
    
    # 3. Save to database using local, zero-lag math processing
    seed_vector_database(all_chunks)