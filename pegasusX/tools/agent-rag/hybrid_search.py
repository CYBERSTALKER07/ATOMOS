#!/usr/bin/env python3
"""
PegasusX Hybrid Search Engine
Combines SQLite FTS5 BM25 Sparse Search + ChromaDB Dense Vector Search via Reciprocal Rank Fusion (RRF).
"""

import os
import sys
import re
import sqlite3
from typing import List, Dict, Any, Optional

os.environ["HF_HUB_DISABLE_SYMLINKS_WARNING"] = "1"
os.environ["TOKENIZERS_PARALLELISM"] = "false"
os.environ["TRANSFORMERS_VERBOSITY"] = "error"
os.environ["TQDM_DISABLE"] = "1"
os.environ["HF_HUB_OFFLINE"] = "1"

import chromadb
from chromadb.utils import embedding_functions

WORKSPACE_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
MEMORY_DIR = os.path.join(WORKSPACE_ROOT, ".agents", "memory")
SQLITE_DB = os.path.join(MEMORY_DIR, "chroma.sqlite3")

_client = None
_collection = None

def get_chroma_collection():
    global _client, _collection
    if _collection is None:
        _client = chromadb.PersistentClient(path=MEMORY_DIR)
        emb_fn = embedding_functions.SentenceTransformerEmbeddingFunction(model_name="all-MiniLM-L6-v2")
        _collection = _client.get_or_create_collection(name="pegasusx_codebase", embedding_function=emb_fn)
    return _collection

def sanitize_fts_query(query: str) -> str:
    """Sanitize user query for SQLite FTS5 MATCH expression."""
    clean = re.sub(r'[^\w\s\_\-\.]', ' ', query)
    tokens = [t.strip() for t in clean.split() if t.strip()]
    if not tokens:
        return ""
    # If single exact code token (e.g. defaultOrderID or OrderStockReservationMarkers), match exact or prefix
    if len(tokens) == 1:
        token = tokens[0]
        return f'"{token}" OR {token}*'
    # Multiple tokens: match either full phrase or individual tokens
    phrase = " ".join(tokens)
    token_match = " AND ".join(tokens)
    return f'"{phrase}" OR ({token_match})'

def is_code_identifier(query: str) -> bool:
    """Detect if query looks like a specific code symbol / identifier."""
    q = query.strip()
    if ' ' in q:
        return False
    # Check camelCase, PascalCase, snake_case, or dotted identifier
    if re.match(r'^[A-Za-z0-9_]+(\.[A-Za-z0-9_]+)*$', q) and len(q) > 3:
        return True
    return False

def search_bm25(query: str, layer: Optional[str] = None, limit: int = 25) -> List[Dict[str, Any]]:
    """Execute sparse BM25 search over ChromaDB SQLite FTS5 table."""
    fts_expr = sanitize_fts_query(query)
    if not fts_expr or not os.path.exists(SQLITE_DB):
        return []

    conn = sqlite3.connect(SQLITE_DB)
    cur = conn.cursor()
    
    try:
        if layer:
            sql = """
            SELECT e.embedding_id, bm25(embedding_fulltext_search) as bm25_score, fts.string_value
            FROM embedding_fulltext_search fts
            JOIN embeddings e ON fts.rowid = e.id
            JOIN embedding_metadata m ON e.id = m.id AND m.key = 'layer' AND m.string_value = ?
            WHERE embedding_fulltext_search MATCH ?
            ORDER BY bm25_score ASC
            LIMIT ?
            """
            cur.execute(sql, (layer, fts_expr, limit))
        else:
            sql = """
            SELECT e.embedding_id, bm25(embedding_fulltext_search) as bm25_score, fts.string_value
            FROM embedding_fulltext_search fts
            JOIN embeddings e ON fts.rowid = e.id
            WHERE embedding_fulltext_search MATCH ?
            ORDER BY bm25_score ASC
            LIMIT ?
            """
            cur.execute(sql, (fts_expr, limit))

        rows = cur.fetchall()
        
        # Batch load metadata for matched embeddings
        results = []
        for embedding_id, bm25_score, doc_text in rows:
            # Load metadata keys
            cur.execute("""
                SELECT m.key, m.string_value, m.int_value 
                FROM embeddings e 
                JOIN embedding_metadata m ON e.id = m.id 
                WHERE e.embedding_id = ?
            """, (embedding_id,))
            meta_rows = cur.fetchall()
            meta = {}
            for k, s_val, i_val in meta_rows:
                if s_val is not None:
                    meta[k] = s_val
                elif i_val is not None:
                    meta[k] = i_val
                    
            results.append({
                "id": embedding_id,
                "document": doc_text,
                "metadata": meta,
                "bm25_score": bm25_score
            })
        return results
    except Exception as exc:
        return []
    finally:
        conn.close()

def search_dense(query: str, layer: Optional[str] = None, limit: int = 25) -> List[Dict[str, Any]]:
    """Execute dense vector similarity search via ChromaDB."""
    coll = get_chroma_collection()
    where_filter = {"layer": layer} if layer else None
    
    try:
        res = coll.query(
            query_texts=[query],
            n_results=limit,
            where=where_filter
        )
    except Exception:
        return []
        
    if not res['documents'] or not res['documents'][0]:
        return []
        
    results = []
    for i in range(len(res['documents'][0])):
        raw_meta = res['metadatas'][0][i] if res['metadatas'] else {}
        results.append({
            "id": res['ids'][0][i],
            "document": res['documents'][0][i],
            "metadata": raw_meta or {},
            "distance": res['distances'][0][i] if res['distances'] else 0.0
        })
    return results

def hybrid_search(query: str, layer: Optional[str] = None, limit: int = 5, k_rrf: int = 60) -> List[Dict[str, Any]]:
    """
    Combines BM25 and Dense Vector search using Reciprocal Rank Fusion.
    Score = (1 - alpha)/(k + dense_rank) + alpha/(k + bm25_rank) + symbol_bonus.
    """
    alpha = 0.7 if is_code_identifier(query) else 0.5
    fetch_k = max(25, limit * 3)
    
    bm25_candidates = search_bm25(query, layer=layer, limit=fetch_k)
    dense_candidates = search_dense(query, layer=layer, limit=fetch_k)
    
    rrf_scores: Dict[str, float] = {}
    doc_store: Dict[str, Dict[str, Any]] = {}
    
    # 1. Score Dense Ranks
    for rank, item in enumerate(dense_candidates, start=1):
        doc_id = item["id"]
        doc_store[doc_id] = item
        score = (1.0 - alpha) / (k_rrf + rank)
        rrf_scores[doc_id] = rrf_scores.get(doc_id, 0.0) + score
        
    # 2. Score BM25 Ranks
    for rank, item in enumerate(bm25_candidates, start=1):
        doc_id = item["id"]
        if doc_id not in doc_store:
            doc_store[doc_id] = item
        else:
            # Merge metadata if missing
            cur_meta = doc_store[doc_id].get("metadata") or {}
            cur_meta.update(item.get("metadata") or {})
            doc_store[doc_id]["metadata"] = cur_meta
        score = alpha / (k_rrf + rank)
        rrf_scores[doc_id] = rrf_scores.get(doc_id, 0.0) + score

    # 3. Apply exact symbol match bonus
    query_clean = query.strip().lower()
    for doc_id, score in list(rrf_scores.items()):
        meta = doc_store[doc_id].get("metadata") or {}
        symbol = str(meta.get("symbol", "")).lower()
        if symbol and query_clean in symbol:
            rrf_scores[doc_id] += 0.05
            if symbol == query_clean:
                rrf_scores[doc_id] += 0.10

    # 4. Sort and return top-limit
    ranked_ids = sorted(rrf_scores.keys(), key=lambda x: rrf_scores[x], reverse=True)[:limit]
    
    output = []
    for doc_id in ranked_ids:
        item = doc_store[doc_id]
        output.append({
            "id": doc_id,
            "rrf_score": rrf_scores[doc_id],
            "document": item["document"],
            "metadata": item["metadata"]
        })
    return output

def find_symbol(symbol_name: str, layer: Optional[str] = None, limit: int = 5) -> List[Dict[str, Any]]:
    """Direct fast-path lookup for an exact symbol declaration in metadata."""
    if not os.path.exists(SQLITE_DB):
        return []
        
    conn = sqlite3.connect(SQLITE_DB)
    cur = conn.cursor()
    try:
        if layer:
            sql = """
            SELECT e.embedding_id, fts.string_value
            FROM embedding_metadata m
            JOIN embeddings e ON m.id = e.id
            JOIN embedding_fulltext_search fts ON e.id = fts.rowid
            JOIN embedding_metadata ml ON e.id = ml.id AND ml.key = 'layer' AND ml.string_value = ?
            WHERE m.key = 'symbol' AND m.string_value LIKE ?
            LIMIT ?
            """
            cur.execute(sql, (layer, f"%{symbol_name}%", limit))
        else:
            sql = """
            SELECT e.embedding_id, fts.string_value
            FROM embedding_metadata m
            JOIN embeddings e ON m.id = e.id
            JOIN embedding_fulltext_search fts ON e.id = fts.rowid
            WHERE m.key = 'symbol' AND m.string_value LIKE ?
            LIMIT ?
            """
            cur.execute(sql, (f"%{symbol_name}%", limit))
            
        rows = cur.fetchall()
        results = []
        for embedding_id, doc_text in rows:
            cur.execute("""
                SELECT m.key, m.string_value, m.int_value 
                FROM embeddings e 
                JOIN embedding_metadata m ON e.id = m.id 
                WHERE e.embedding_id = ?
            """, (embedding_id,))
            meta = {k: (s if s is not None else i) for k, s, i in cur.fetchall()}
            results.append({
                "id": embedding_id,
                "document": doc_text,
                "metadata": meta,
                "symbol_match": True
            })
        return results
    finally:
        conn.close()

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python3 hybrid_search.py <query> [layer] [limit]")
        sys.exit(1)
        
    q = sys.argv[1]
    l = sys.argv[2] if len(sys.argv) > 2 and sys.argv[2] != "all" else None
    lim = int(sys.argv[3]) if len(sys.argv) > 3 else 5
    
    hits = hybrid_search(q, layer=l, limit=lim)
    print(f"\n--- Hybrid Search Results for: {q} (Found: {len(hits)}) ---")
    for idx, hit in enumerate(hits, start=1):
        meta = hit["metadata"]
        print(f"\n[{idx}] Score: {hit['rrf_score']:.5f} | File: {meta.get('file')} | Layer: {meta.get('layer')}")
        if meta.get("symbol"):
            print(f"    Symbol: {meta.get('symbol')} ({meta.get('symbol_type', 'unknown')})")
        print("```" + str(meta.get("language", "")) + "\n" + hit["document"][:400] + "\n```")
