#!/usr/bin/env python3
import os
import sys

os.environ["HF_HUB_DISABLE_SYMLINKS_WARNING"] = "1"
os.environ["TOKENIZERS_PARALLELISM"] = "false"
os.environ["TRANSFORMERS_VERBOSITY"] = "error"
os.environ["TQDM_DISABLE"] = "1"

import chromadb
from chromadb.utils import embedding_functions

MEMORY_DIR = os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))), ".agents", "memory")

def search(query: str, layer: str = None, limit: int = 5):
    client = chromadb.PersistentClient(path=MEMORY_DIR)
    emb_fn = embedding_functions.SentenceTransformerEmbeddingFunction(model_name="all-MiniLM-L6-v2")
    collection = client.get_or_create_collection(name="pegasusx_codebase", embedding_function=emb_fn)
    
    where_filter = {"layer": layer} if layer else None
    results = collection.query(
        query_texts=[query],
        n_results=limit,
        where=where_filter
    )
    
    if not results['documents'] or not results['documents'][0]:
        print("No relevant code found in the RAG database for query:", query)
        return
        
    for i in range(len(results['documents'][0])):
        doc = results['documents'][0][i]
        meta = results['metadatas'][0][i]
        dist = results['distances'][0][i]
        
        filepath = meta.get("file", "unknown_file")
        language = meta.get("language", "text")
        layer_name = meta.get("layer", "unknown_layer")
        
        print(f"### Result {i+1} (Score: {dist:.4f})")
        print(f"**File:** `{filepath}`")
        print(f"**Layer:** `{layer_name}`")
        print(f"```{language}\n{doc}\n```\n")
        print("-" * 60)

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python3 query.py <search_query> [layer] [limit]")
        sys.exit(1)
        
    q = sys.argv[1]
    l = sys.argv[2] if len(sys.argv) > 2 else None
    lim = int(sys.argv[3]) if len(sys.argv) > 3 else 5
    search(q, l, lim)
