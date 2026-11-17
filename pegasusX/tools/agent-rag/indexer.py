import os
import sys
import time
import re
import json
import hashlib
from typing import List, Dict, Any, Tuple, Optional

os.environ["HF_HUB_DISABLE_SYMLINKS_WARNING"] = "1"
os.environ["TOKENIZERS_PARALLELISM"] = "false"
os.environ["TRANSFORMERS_VERBOSITY"] = "error"
os.environ["TQDM_DISABLE"] = "1"
os.environ["HF_HUB_OFFLINE"] = "1"

import chromadb
from chromadb.utils import embedding_functions

WORKSPACE_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
MEMORY_DIR = os.path.join(WORKSPACE_ROOT, ".agents", "memory")
HASH_CACHE_FILE = os.path.join(MEMORY_DIR, "file_hashes.json")
os.makedirs(MEMORY_DIR, exist_ok=True)

client = chromadb.PersistentClient(path=MEMORY_DIR)
emb_fn = embedding_functions.SentenceTransformerEmbeddingFunction(model_name="all-MiniLM-L6-v2")
collection = client.get_or_create_collection(name="pegasusx_codebase", embedding_function=emb_fn)

IGNORE_DIRS = {
    'node_modules', '.next', 'dist', 'build', '.git', '.venv', 'venv',
    'target', '.turbo', '.gradle', 'Pods', 'DerivedData', '__pycache__'
}

# Initialize Tree-Sitter Parsers if available
ts_parsers = {}
try:
    import tree_sitter
    try:
        import tree_sitter_go
        go_lang = tree_sitter.Language(tree_sitter_go.language())
        ts_parsers['go'] = tree_sitter.Parser(go_lang)
    except Exception:
        pass
        
    try:
        import tree_sitter_typescript
        ts_lang = tree_sitter.Language(tree_sitter_typescript.language_typescript())
        tsx_lang = tree_sitter.Language(tree_sitter_typescript.language_tsx())
        ts_parsers['typescript'] = tree_sitter.Parser(ts_lang)
        ts_parsers['tsx'] = tree_sitter.Parser(tsx_lang)
    except Exception:
        pass
except Exception:
    pass

def compute_file_hash(filepath: str) -> str:
    """Computes SHA-256 hash of a file."""
    h = hashlib.sha256()
    try:
        with open(filepath, 'rb') as f:
            while chunk := f.read(65536):
                h.update(chunk)
        return h.hexdigest()
    except Exception:
        return ""

def load_hash_cache() -> Dict[str, Dict[str, Any]]:
    if os.path.exists(HASH_CACHE_FILE):
        try:
            with open(HASH_CACHE_FILE, 'r', encoding='utf-8') as f:
                return json.load(f)
        except Exception:
            return {}
    return {}

def save_hash_cache(cache: Dict[str, Dict[str, Any]]):
    try:
        with open(HASH_CACHE_FILE, 'w', encoding='utf-8') as f:
            json.dump(cache, f, indent=2)
    except Exception:
        pass

def collect_files(base_dir: str, extensions: List[str], path_filter=None) -> List[str]:
    """Fast file collection using os.walk with directory pruning."""
    collected = []
    target_dir = os.path.join(WORKSPACE_ROOT, base_dir) if not os.path.isabs(base_dir) else base_dir
    if not os.path.exists(target_dir):
        return collected
        
    for root, dirs, files in os.walk(target_dir):
        dirs[:] = [d for d in dirs if d not in IGNORE_DIRS and not d.startswith('.')]
        for f in files:
            if any(f.endswith(ext) for ext in extensions):
                full_path = os.path.join(root, f)
                if path_filter is None or path_filter(full_path):
                    collected.append(full_path)
    return sorted(collected)

def chunk_code_ast(filepath: str, content: str, language: str, layer_name: str) -> List[Dict[str, Any]]:
    """Semantically chunks code by AST symbol boundaries with contextual header injection."""
    if not content.strip():
        return []
        
    rel_path = os.path.relpath(filepath, WORKSPACE_ROOT)
    code_bytes = content.encode("utf-8")
    lines = content.splitlines(keepends=True)
    chunks = []

    # 1. Tree-Sitter parsing for Go
    if language == 'go' and 'go' in ts_parsers:
        try:
            tree = ts_parsers['go'].parse(code_bytes)
            for node in tree.root_node.children:
                if node.type in ('function_declaration', 'method_declaration', 'type_declaration'):
                    start_line = node.start_point[0] + 1
                    end_line = node.end_point[0] + 1
                    chunk_text = code_bytes[node.start_byte:node.end_byte].decode("utf-8", errors="ignore").strip()
                    if len(chunk_text) < 30:
                        continue
                    
                    symbol_name = ""
                    symbol_type = node.type
                    if node.type == 'function_declaration':
                        n = node.child_by_field_name('name')
                        symbol_name = code_bytes[n.start_byte:n.end_byte].decode("utf-8") if n else ""
                    elif node.type == 'method_declaration':
                        r = node.child_by_field_name('receiver')
                        n = node.child_by_field_name('name')
                        recv_text = code_bytes[r.start_byte:r.end_byte].decode("utf-8") if r else ""
                        name_text = code_bytes[n.start_byte:n.end_byte].decode("utf-8") if n else ""
                        symbol_name = f"{recv_text} {name_text}".strip()
                    elif node.type == 'type_declaration':
                        for c in node.children:
                            if c.type == 'type_spec':
                                n = c.child_by_field_name('name')
                                symbol_name = code_bytes[n.start_byte:n.end_byte].decode("utf-8") if n else ""
                                symbol_type = "type_spec"

                    header = f"// Symbol: {symbol_name} | Lines: {start_line}-{end_line} | File: {rel_path} | Layer: {layer_name}\n"
                    chunks.append({
                        "text": header + chunk_text,
                        "symbol": symbol_name,
                        "symbol_type": symbol_type,
                        "start_line": start_line,
                        "end_line": end_line
                    })
            if chunks:
                return chunks
        except Exception:
            pass

    # 2. Tree-Sitter parsing for TypeScript / TSX
    if language in ('typescript', 'tsx') and language in ts_parsers:
        try:
            tree = ts_parsers[language].parse(code_bytes)
            for node in tree.root_node.children:
                target = node
                if node.type == 'export_statement':
                    for c in node.children:
                        if c.type in ('function_declaration', 'class_declaration', 'interface_declaration', 'type_alias_declaration', 'lexical_declaration'):
                            target = c
                            break
                            
                if target.type in ('function_declaration', 'class_declaration', 'interface_declaration', 'type_alias_declaration', 'lexical_declaration'):
                    start_line = target.start_point[0] + 1
                    end_line = target.end_point[0] + 1
                    chunk_text = code_bytes[node.start_byte:node.end_byte].decode("utf-8", errors="ignore").strip()
                    if len(chunk_text) < 30:
                        continue
                        
                    n = target.child_by_field_name('name')
                    symbol_name = code_bytes[n.start_byte:n.end_byte].decode("utf-8") if n else ""
                    header = f"// Symbol: {symbol_name} | Lines: {start_line}-{end_line} | File: {rel_path} | Layer: {layer_name}\n"
                    chunks.append({
                        "text": header + chunk_text,
                        "symbol": symbol_name,
                        "symbol_type": target.type,
                        "start_line": start_line,
                        "end_line": end_line
                    })
            if chunks:
                return chunks
        except Exception:
            pass

    # 3. SQL / DDL Statements
    if language == 'sql':
        pattern = re.compile(r'(?i)(CREATE\s+(?:OR\s+REPLACE\s+)?(?:TABLE|INDEX|VIEW|PROCEDURE)|ALTER\s+TABLE)\s+([`"\w\.]+)')
        pos = 0
        matches = list(pattern.finditer(content))
        if matches:
            for i, m in enumerate(matches):
                start = m.start()
                end = matches[i+1].start() if i + 1 < len(matches) else len(content)
                chunk_text = content[start:end].strip()
                if len(chunk_text) > 25:
                    symbol_type = m.group(1).upper()
                    symbol_name = m.group(2).replace('`', '').replace('"', '')
                    start_line = content[:start].count('\n') + 1
                    end_line = content[:end].count('\n') + 1
                    header = f"-- Symbol: {symbol_type} {symbol_name} | Lines: {start_line}-{end_line} | File: {rel_path} | Layer: {layer_name}\n"
                    chunks.append({
                        "text": header + chunk_text,
                        "symbol": symbol_name,
                        "symbol_type": symbol_type,
                        "start_line": start_line,
                        "end_line": end_line
                    })
            if chunks:
                return chunks

    # 4. Swift & Kotlin: Pattern-based extraction
    if language in ('swift', 'kotlin'):
        pat = re.compile(r'(?m)^(?:(?:\@\w+\s+)*(?:public\s+|private\s+|internal\s+|fileprivate\s+|open\s+|override\s+)?(?:class|struct|enum|actor|protocol|func|fun|interface)\s+([A-Za-z0-9_]+))')
        matches = list(pat.finditer(content))
        if matches:
            for i, m in enumerate(matches):
                start = m.start()
                end = matches[i+1].start() if i + 1 < len(matches) else len(content)
                chunk_text = content[start:end].strip()
                if len(chunk_text) > 30:
                    symbol_name = m.group(1)
                    start_line = content[:start].count('\n') + 1
                    end_line = content[:end].count('\n') + 1
                    header = f"// Symbol: {symbol_name} | Lines: {start_line}-{end_line} | File: {rel_path} | Layer: {layer_name}\n"
                    chunks.append({
                        "text": header + chunk_text,
                        "symbol": symbol_name,
                        "symbol_type": "declaration",
                        "start_line": start_line,
                        "end_line": end_line
                    })
            if chunks:
                return chunks

    # 5. Python def/class extraction
    if language == 'python':
        pat = re.compile(r'(?m)^(?:\s*(?:def|class|async\s+def)\s+([A-Za-z0-9_]+))')
        matches = list(pat.finditer(content))
        if matches:
            for i, m in enumerate(matches):
                start = m.start()
                end = matches[i+1].start() if i + 1 < len(matches) else len(content)
                chunk_text = content[start:end].strip()
                if len(chunk_text) > 30:
                    symbol_name = m.group(1)
                    start_line = content[:start].count('\n') + 1
                    end_line = content[:end].count('\n') + 1
                    header = f"# Symbol: {symbol_name} | Lines: {start_line}-{end_line} | File: {rel_path} | Layer: {layer_name}\n"
                    chunks.append({
                        "text": header + chunk_text,
                        "symbol": symbol_name,
                        "symbol_type": "python_symbol",
                        "start_line": start_line,
                        "end_line": end_line
                    })
            if chunks:
                return chunks

    # 6. Universal Sliding Window Fallback (1200 chars with 200 char overlap)
    window_size = 1200
    overlap = 200
    step = window_size - overlap
    comment_sym = "--" if language == 'sql' else ("#" if language == 'python' else "//")
    
    for i in range(0, len(content), step):
        chunk_raw = content[i:i + window_size].strip()
        if len(chunk_raw) > 30:
            start_line = content[:i].count('\n') + 1
            end_line = content[:i + len(chunk_raw)].count('\n') + 1
            header = f"{comment_sym} Lines: {start_line}-{end_line} | File: {rel_path} | Layer: {layer_name}\n"
            chunks.append({
                "text": header + chunk_raw,
                "symbol": "",
                "symbol_type": "text_block",
                "start_line": start_line,
                "end_line": end_line
            })
            
    return chunks if chunks else [{
        "text": f"{comment_sym} File: {rel_path} | Layer: {layer_name}\n" + content[:2000],
        "symbol": "",
        "symbol_type": "full_file",
        "start_line": 1,
        "end_line": len(lines)
    }]

def index_layer_files(layer_name: str, files: List[str], language_hint: str, hash_cache: Dict[str, Dict[str, Any]], incremental: bool = False) -> int:
    t0 = time.time()
    
    docs = []
    metadatas = []
    ids = []
    total_chunks = 0
    batch_size = 150
    files_to_process = []

    for filepath in files:
        rel_path = os.path.relpath(filepath, WORKSPACE_ROOT)
        fhash = compute_file_hash(filepath)
        cached = hash_cache.get(rel_path)
        
        if incremental and cached and cached.get("hash") == fhash:
            continue
            
        files_to_process.append((filepath, rel_path, fhash))

    if not files_to_process:
        return 0

    print(f"[{layer_name}] Processing {len(files_to_process)} changed/new files...")

    for filepath, rel_path, fhash in files_to_process:
        try:
            with open(filepath, 'r', encoding='utf-8', errors='ignore') as f:
                content = f.read()
        except Exception:
            continue
            
        chunks = chunk_code_ast(filepath, content, language_hint, layer_name)
        
        # If updating an existing file in incremental mode, delete old chunks first
        if incremental and rel_path in hash_cache:
            try:
                old_count = hash_cache[rel_path].get("chunks_count", 0)
                old_ids = [f"{rel_path}_{ci}" for ci in range(old_count + 15)]
                collection.delete(ids=old_ids)
            except Exception:
                pass

        for chunk_idx, chunk_item in enumerate(chunks):
            chunk_id = f"{rel_path}_{chunk_idx}"
            docs.append(chunk_item["text"])
            metadatas.append({
                "layer": layer_name,
                "file": rel_path,
                "language": language_hint,
                "symbol": chunk_item["symbol"],
                "symbol_type": chunk_item["symbol_type"],
                "start_line": chunk_item["start_line"],
                "end_line": chunk_item["end_line"]
            })
            ids.append(chunk_id)
            total_chunks += 1
            
            if len(docs) >= batch_size:
                collection.upsert(documents=docs, metadatas=metadatas, ids=ids)
                docs, metadatas, ids = [], [], []

        hash_cache[rel_path] = {
            "hash": fhash,
            "chunks_count": len(chunks),
            "updated_at": int(time.time())
        }

    if docs:
        collection.upsert(documents=docs, metadatas=metadatas, ids=ids)
        
    print(f"[{layer_name}] Processed {len(files_to_process)} files ({total_chunks} chunks) in {time.time()-t0:.2f}s")
    return total_chunks

def run_indexing(incremental: bool = False):
    global collection
    os.chdir(WORKSPACE_ROOT)
    print("=" * 80)
    mode_str = "INCREMENTAL (FRESHNESS CHECK)" if incremental else "FULL RE-INDEX"
    print(f"PEGASUSX CODEBASE RAG INDEXER: {mode_str}")
    print(f"Workspace Root: {WORKSPACE_ROOT}")
    print(f"ChromaDB Storage: {MEMORY_DIR}")
    print("=" * 80)

    hash_cache = load_hash_cache() if incremental else {}
    if not incremental:
        print("[RESET] Recreating pegasusx_codebase collection...")
        try:
            client.delete_collection("pegasusx_codebase")
        except Exception:
            pass
        collection = client.get_or_create_collection(name="pegasusx_codebase", embedding_function=emb_fn)
        hash_cache = {}

    start_time = time.time()
    total_indexed_chunks = 0
    
    # 1. Backend Layer (Go)
    backend_files = (
        collect_files("apps/backend-go", [".go"]) +
        collect_files("apps/ai-worker", [".go"]) +
        collect_files("apps/handoff-service", [".go"])
    )
    total_indexed_chunks += index_layer_files("backend", backend_files, "go", hash_cache, incremental)
    
    # 2. Infra Layer (Spanner DDL & SQL)
    infra_files = (
        collect_files("apps/backend-go/schema", [".ddl", ".sql"]) +
        collect_files("infra", [".yaml", ".yml", ".tf", ".hcl", ".sql"])
    )
    total_indexed_chunks += index_layer_files("infra", infra_files, "sql", hash_cache, incremental)
    
    # 3. Client Web & Desktop (TypeScript / React)
    web_files = collect_files("apps", [".tsx", ".ts"], path_filter=lambda p: any(x in p for x in ('portal', 'desktop', 'terminal', 'bot-telegram')))
    total_indexed_chunks += index_layer_files("client-web", web_files, "typescript", hash_cache, incremental)
    
    # 4. Mobile Android (Kotlin)
    android_files = collect_files("apps", [".kt"], path_filter=lambda p: 'android' in p) + \
                    collect_files("packages", [".kt"], path_filter=lambda p: 'android' in p)
    total_indexed_chunks += index_layer_files("mobile-android", android_files, "kotlin", hash_cache, incremental)
    
    # 5. Mobile & Desktop Apple (Swift)
    apple_files = collect_files("apps", [".swift"], path_filter=lambda p: any(x in p for x in ('ios', 'macos', 'apple'))) + \
                  collect_files("packages", [".swift"], path_filter=lambda p: any(x in p for x in ('ios', 'macos', 'apple')))
    total_indexed_chunks += index_layer_files("mobile-ios", apple_files, "swift", hash_cache, incremental)
    
    # 6. Python Optimizers & Services
    python_files = (
        collect_files("apps/dispatch-optimizer-py", [".py"]) +
        collect_files("apps/ml-scorer-py", [".py"])
    )
    total_indexed_chunks += index_layer_files("python", python_files, "python", hash_cache, incremental)

    # 7. Shared Packages (TypeScript / TSX)
    shared_files = collect_files("packages", [".ts", ".tsx"], path_filter=lambda p: 'android' not in p and 'ios' not in p and 'macos' not in p)
    total_indexed_chunks += index_layer_files("shared-packages", shared_files, "typescript", hash_cache, incremental)
    
    save_hash_cache(hash_cache)
    duration = time.time() - start_time
    print("=" * 80)
    print(f"INDEXING PASS COMPLETE!")
    print(f"Active Chunks in ChromaDB: {collection.count()}")
    print(f"Total Elapsed Time: {duration:.2f}s")
    print("=" * 80)

if __name__ == "__main__":
    is_incremental = "--incremental" in sys.argv
    run_indexing(incremental=is_incremental)
