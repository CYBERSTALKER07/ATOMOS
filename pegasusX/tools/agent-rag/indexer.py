import os
import sys
import time
import re

os.environ["HF_HUB_DISABLE_SYMLINKS_WARNING"] = "1"
os.environ["TOKENIZERS_PARALLELISM"] = "false"
os.environ["TRANSFORMERS_VERBOSITY"] = "error"
os.environ["TQDM_DISABLE"] = "1"

import chromadb
from chromadb.utils import embedding_functions

# Initialize ChromaDB in the local .agents/memory directory
MEMORY_DIR = os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))), ".agents", "memory")
os.makedirs(MEMORY_DIR, exist_ok=True)
client = chromadb.PersistentClient(path=MEMORY_DIR)

# Fast local sentence-transformers embedding function
emb_fn = embedding_functions.SentenceTransformerEmbeddingFunction(model_name="all-MiniLM-L6-v2")
collection = client.get_or_create_collection(name="pegasusx_codebase", embedding_function=emb_fn)

IGNORE_DIRS = {
    'node_modules', '.next', 'dist', 'build', '.git', '.venv', 'venv',
    'target', '.turbo', '.gradle', 'Pods', 'DerivedData', '__pycache__'
}

def collect_files(base_dir, extensions, path_filter=None):
    """Fast file collection using os.walk with directory pruning."""
    collected = []
    if not os.path.exists(base_dir):
        return collected
        
    for root, dirs, files in os.walk(base_dir):
        # Prune ignored directories in-place so os.walk never enters them
        dirs[:] = [d for d in dirs if d not in IGNORE_DIRS and not d.startswith('.')]
        for f in files:
            if any(f.endswith(ext) for ext in extensions):
                full_path = os.path.join(root, f)
                if path_filter is None or path_filter(full_path):
                    collected.append(full_path)
    return sorted(collected)

def chunk_code(filepath, content, language):
    """Semantically chunks code based on language, with safe sliding-window fallback."""
    if not content.strip():
        return []
        
    # SQL / DDL: split by statements (CREATE TABLE, CREATE INDEX, ALTER TABLE)
    if language == 'sql':
        raw_chunks = re.split(r'(?i)(?=(?:CREATE\s+TABLE|CREATE\s+(?:OR\s+REPLACE\s+)?INDEX|ALTER\s+TABLE))', content)
        chunks = [c.strip() for c in raw_chunks if len(c.strip()) > 30]
        return chunks if chunks else [content[:2000]]
        
    # Kotlin / Swift: split by function / class / struct / interface boundaries
    if language in ('kotlin', 'swift'):
        raw_chunks = re.split(r'(?m)(?=^(?:public\s+|private\s+|internal\s+)?(?:class|struct|enum|fun|func|interface|extension)\s+)', content)
        chunks = [c.strip() for c in raw_chunks if len(c.strip()) > 40]
        return chunks if chunks else [content[:2000]]

    # Go and TypeScript / TSX: use Tree-sitter if available, else regex function boundaries
    try:
        import tree_sitter
        if language == 'go':
            import tree_sitter_go
            parser = tree_sitter.Parser()
            parser.set_language(tree_sitter.Language(tree_sitter_go.language(), "go"))
        elif language in ('typescript', 'tsx'):
            import tree_sitter_typescript
            parser = tree_sitter.Parser()
            if language == 'tsx':
                parser.set_language(tree_sitter.Language(tree_sitter_typescript.language_tsx(), "tsx"))
            else:
                parser.set_language(tree_sitter.Language(tree_sitter_typescript.language_typescript(), "typescript"))
        else:
            parser = None

        if parser:
            tree = parser.parse(bytes(content, "utf8"))
            chunks = []
            def traverse(node):
                if node.type in [
                    'function_declaration', 'method_declaration', 'type_declaration',
                    'lexical_declaration', 'class_declaration', 'interface_declaration'
                ]:
                    start_byte = node.start_byte
                    end_byte = node.end_byte
                    chunk_text = content.encode("utf8")[start_byte:end_byte].decode("utf8", errors="ignore").strip()
                    if len(chunk_text) > 40:
                        chunks.append(chunk_text)
                for child in node.children:
                    traverse(child)
            traverse(tree.root_node)
            if chunks:
                return chunks
    except Exception:
        pass

    # Regex fallback for Go / TS
    if language == 'go':
        raw_chunks = re.split(r'(?m)(?=^func\s+|^type\s+)', content)
        chunks = [c.strip() for c in raw_chunks if len(c.strip()) > 40]
        if chunks:
            return chunks
            
    # Universal Sliding Window fallback (1200 chars with 200 char overlap)
    window_size = 1200
    overlap = 200
    step = window_size - overlap
    chunks = []
    for i in range(0, len(content), step):
        chunk = content[i:i + window_size].strip()
        if len(chunk) > 30:
            chunks.append(chunk)
    return chunks if chunks else [content]

def index_layer_files(layer_name, files, language_hint, workspace_root):
    print(f"[{layer_name}] Found {len(files)} files to index...")
    t0 = time.time()
    
    docs = []
    metadatas = []
    ids = []
    total_chunks = 0
    batch_size = 200
    
    for idx, filepath in enumerate(files):
        try:
            with open(filepath, 'r', encoding='utf-8', errors='ignore') as f:
                content = f.read()
        except Exception:
            continue
            
        chunks = chunk_code(filepath, content, language_hint)
        rel_path = os.path.relpath(filepath, workspace_root)
        
        for chunk_idx, chunk in enumerate(chunks):
            chunk_id = f"{rel_path}_{chunk_idx}"
            docs.append(chunk)
            metadatas.append({
                "layer": layer_name,
                "file": rel_path,
                "language": language_hint
            })
            ids.append(chunk_id)
            total_chunks += 1
            
            if len(docs) >= batch_size:
                collection.upsert(documents=docs, metadatas=metadatas, ids=ids)
                docs, metadatas, ids = [], [], []

    if docs:
        collection.upsert(documents=docs, metadatas=metadatas, ids=ids)
        
    print(f"[{layer_name}] Completed {len(files)} files ({total_chunks} chunks) in {time.time()-t0:.2f}s")
    return total_chunks

def main():
    workspace = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
    os.chdir(workspace)
    print("=" * 80)
    print(f"STARTING COMPREHENSIVE PEGASUSX CODEBASE INDEXING")
    print(f"Workspace Root: {workspace}")
    print(f"ChromaDB Storage: {MEMORY_DIR}")
    print("=" * 80)
    
    start_time = time.time()
    total_indexed_chunks = 0
    
    # 1. Backend Layer (Go)
    backend_files = collect_files("apps/backend-go", [".go"])
    total_indexed_chunks += index_layer_files("backend", backend_files, "go", workspace)
    
    # 2. Infra Layer (Spanner DDL & SQL)
    infra_files = collect_files("apps/backend-go/schema", [".ddl", ".sql"])
    total_indexed_chunks += index_layer_files("infra", infra_files, "sql", workspace)
    
    # 3. Client Web & Desktop (Next.js / React / TypeScript)
    web_files = collect_files("apps", [".tsx", ".ts"], path_filter=lambda p: any(x in p for x in ('portal', 'desktop', 'terminal')))
    total_indexed_chunks += index_layer_files("client-web", web_files, "typescript", workspace)
    
    # 4. Mobile Android (Kotlin)
    android_files = collect_files("apps", [".kt"], path_filter=lambda p: 'android' in p) + \
                    collect_files("packages", [".kt"], path_filter=lambda p: 'android' in p)
    total_indexed_chunks += index_layer_files("mobile-android", android_files, "kotlin", workspace)
    
    # 5. Mobile iOS (Swift)
    ios_files = collect_files("apps", [".swift"], path_filter=lambda p: 'ios' in p) + \
                collect_files("packages", [".swift"], path_filter=lambda p: 'ios' in p)
    total_indexed_chunks += index_layer_files("mobile-ios", ios_files, "swift", workspace)
    
    # 6. Shared Packages (TypeScript / TSX)
    shared_files = collect_files("packages", [".ts", ".tsx"], path_filter=lambda p: 'android' not in p and 'ios' not in p)
    total_indexed_chunks += index_layer_files("shared-packages", shared_files, "typescript", workspace)
    
    duration = time.time() - start_time
    print("=" * 80)
    print(f"INDEXING PASS COMPLETE!")
    print(f"Total Chunks in ChromaDB: {collection.count()}")
    print(f"Total Elapsed Time: {duration:.2f}s")
    print("=" * 80)

if __name__ == "__main__":
    main()
