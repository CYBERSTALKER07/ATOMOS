import os
import chromadb
from chromadb.utils import embedding_functions
from mcp.server.fastmcp import FastMCP

# Initialize FastMCP Server
mcp = FastMCP("PegasusX-Agent-RAG")

# Initialize ChromaDB client
MEMORY_DIR = os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(__file__))), ".agents", "memory")
client = chromadb.PersistentClient(path=MEMORY_DIR)
emb_fn = embedding_functions.SentenceTransformerEmbeddingFunction(model_name="all-MiniLM-L6-v2")
collection = client.get_or_create_collection(name="pegasusx_codebase", embedding_function=emb_fn)

@mcp.tool()
def semantic_code_search(query: str, layer: str = None, limit: int = 5) -> str:
    """
    Search the PegasusX codebase semantically using vector embeddings.
    
    Args:
        query: The natural language or code query to search for (e.g., "Supplier WebSocket payload definition").
        layer: Optional filter to restrict search to a specific layer (e.g., "backend", "infra", "client-web", "mobile-android", "mobile-ios", "shared-packages").
        limit: Number of code chunks to return (default: 5).
    """
    where_filter = {"layer": layer} if layer else None
    
    results = collection.query(
        query_texts=[query],
        n_results=limit,
        where=where_filter
    )
    
    if not results['documents'] or not results['documents'][0]:
        return "No relevant code found in the RAG database for this query."
        
    formatted_results = []
    for i in range(len(results['documents'][0])):
        doc = results['documents'][0][i]
        meta = results['metadatas'][0][i]
        dist = results['distances'][0][i]
        
        filepath = meta.get("file", "unknown_file")
        language = meta.get("language", "text")
        layer_name = meta.get("layer", "unknown_layer")
        
        formatted_results.append(
            f"### Result {i+1} (Score: {dist:.4f})\n"
            f"**File:** `{filepath}`\n"
            f"**Layer:** `{layer_name}`\n"
            f"```{language}\n{doc}\n```\n"
        )
        
    return "\n---\n".join(formatted_results)

if __name__ == "__main__":
    import sys
    print("Starting PegasusX Agent RAG MCP Server on stdio...", file=sys.stderr, flush=True)
    mcp.run()
