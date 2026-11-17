#!/usr/bin/env python3
import os
import sys

os.environ["HF_HUB_DISABLE_SYMLINKS_WARNING"] = "1"
os.environ["TOKENIZERS_PARALLELISM"] = "false"
os.environ["TRANSFORMERS_VERBOSITY"] = "error"
os.environ["TQDM_DISABLE"] = "1"
os.environ["HF_HUB_OFFLINE"] = "1"

from hybrid_search import hybrid_search, find_symbol

def search(query: str, layer: str = None, limit: int = 5):
    hits = hybrid_search(query, layer=layer, limit=limit)
    if not hits:
        print("No relevant code found in the RAG database for query:", query)
        return
        
    for i, hit in enumerate(hits, start=1):
        doc = hit['document']
        meta = hit.get('metadata', {})
        score = hit.get('rrf_score', 0.0)
        
        filepath = meta.get("file", "unknown_file")
        language = meta.get("language", "text")
        layer_name = meta.get("layer", "unknown_layer")
        symbol = meta.get("symbol", "")
        
        sym_header = f" | Symbol: `{symbol}`" if symbol else ""
        print(f"### Result {i} (RRF Score: {score:.5f})")
        print(f"**File:** `{filepath}` | **Layer:** `{layer_name}`{sym_header}")
        print(f"```{language}\n{doc}\n```\n")
        print("-" * 60)

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python3 query.py <search_query> [layer] [limit]")
        print("   or: python3 query.py --symbol <symbol_name> [layer] [limit]")
        sys.exit(1)
        
    if sys.argv[1] == "--symbol":
        sym = sys.argv[2]
        layer_filter = sys.argv[3] if len(sys.argv) > 3 and sys.argv[3] != "all" else None
        limit_count = int(sys.argv[4]) if len(sys.argv) > 4 else 5
        hits = find_symbol(sym, layer=layer_filter, limit=limit_count)
        for i, hit in enumerate(hits, start=1):
            print(f"### Symbol Match {i}: {hit['metadata'].get('file')}")
            print(f"```{hit['metadata'].get('language')}\n{hit['document']}\n```")
    else:
        query_text = sys.argv[1]
        layer_filter = sys.argv[2] if len(sys.argv) > 2 and sys.argv[2] != "all" else None
        limit_count = int(sys.argv[3]) if len(sys.argv) > 3 else 5
        search(query_text, layer_filter, limit_count)
