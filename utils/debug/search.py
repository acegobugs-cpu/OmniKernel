import chromadb

def _load_collection():
    # Reuse your existing local vector store directory
    client = chromadb.PersistentClient(path="./.omnikernel_db")
    return client.get_or_create_collection(name="codebase_chunks")

def search_codebase(query: str, top_k: int = 3):
    collection = _load_collection()
    
    # FIX: By using query_texts instead of query_embeddings, 
    # ChromaDB automatically handles the MiniLM math transformation for us!
    results = collection.query(
        query_texts=[query],
        n_results=top_k,
        include=["documents", "metadatas", "distances"]
    )
    
    # Clean transform into a user-friendly dictionary format
    hits = []
    if results and results["documents"] and len(results["documents"][0]) > 0:
        for doc, meta, dist in zip(
            results["documents"][0],
            results["metadatas"][0],
            results["distances"][0],
        ):
            hits.append({
                "text": doc,
                "metadata": meta,
                "distance": dist,  # Lower distance = closer mathematical match
            })
    return hits

if __name__ == "__main__":
    print("====================================================")
    print("   🌐 OmniKernel Phase 1: Codebase Oracle SearchTab   ")
    print("====================================================")
    print("Enter a concept to see matching code snippets instantly.\n")
    
    while True:
        user_input = input("User 💻 >> ")
        if user_input.strip().lower() == "exit":
            print("Shutting down core link. Goodbye.")
            break
        if not user_input.strip():
            continue
            
        try:
            search_results = search_codebase(user_input)
            
            if not search_results:
                print("\n🔍 No matching code segments found in index.\n")
                continue
                
            # Pretty-print the results so it looks like a real dev tool
            for idx, hit in enumerate(search_results):
                print(f"\n🎯 [MATCH #{idx+1}] Match Score (Distance): {hit['distance']:.4f}")
                print(f"📂 File: {hit['metadata']['relative_path']} (Chunk: {hit['metadata']['chunk_index']})")
                print("-" * 50)
                print(hit['text'])
                print("=" * 50)
                
        except Exception as e:
            print(f"\n❌ Pipeline Error: {e}\n")