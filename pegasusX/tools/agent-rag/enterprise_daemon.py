import os
import hashlib
import json
import logging
from watchfiles import watch, Change
import chromadb
from chromadb.utils import embedding_functions

logging.basicConfig(level=logging.INFO, format='%(asctime)s - [RAG Daemon] %(message)s')

WORKSPACE = os.path.dirname(os.path.dirname(os.path.dirname(__file__)))
MEMORY_DIR = os.path.join(WORKSPACE, ".agents", "memory")
HASH_DB_PATH = os.path.join(MEMORY_DIR, "file_hashes.json")

os.makedirs(MEMORY_DIR, exist_ok=True)
client = chromadb.PersistentClient(path=MEMORY_DIR)
emb_fn = embedding_functions.SentenceTransformerEmbeddingFunction(model_name="all-MiniLM-L6-v2")
collection = client.get_or_create_collection(name="pegasusx_codebase", embedding_function=emb_fn)

# Load state
if os.path.exists(HASH_DB_PATH):
    with open(HASH_DB_PATH, 'r') as f:
        file_hashes = json.load(f)
else:
    file_hashes = {}

def save_state():
    with open(HASH_DB_PATH, 'w') as f:
        json.dump(file_hashes, f)

def get_file_hash(filepath):
    hasher = hashlib.sha256()
    try:
        with open(filepath, 'rb') as f:
            buf = f.read()
            hasher.update(buf)
        return hasher.hexdigest()
    except Exception:
        return None

def get_language(filepath):
    ext = os.path.splitext(filepath)[1].lower()
    if ext == '.go': return 'go'
    if ext in ['.ts', '.js']: return 'typescript'
    if ext in ['.tsx', '.jsx']: return 'tsx'
    if ext == '.kt': return 'kotlin'
    if ext == '.swift': return 'swift'
    if ext == '.ddl': return 'sql'
    return None

def get_layer(filepath):
    if 'backend-go' in filepath: return 'backend'
    if 'schema' in filepath: return 'infra'
    if 'portal' in filepath: return 'client-web'
    if 'mobile-android' in filepath: return 'mobile-android'
    if 'mobile-ios' in filepath: return 'mobile-ios'
    if 'packages' in filepath: return 'shared-packages'
    return 'unknown'

def chunk_code_ast(filepath, content, language):
    import tree_sitter
    try:
        parser = tree_sitter.Parser()
        if language == 'go':
            import tree_sitter_go
            parser.set_language(tree_sitter.Language(tree_sitter_go.language(), "go"))
        elif language == 'typescript':
            import tree_sitter_typescript
            parser.set_language(tree_sitter.Language(tree_sitter_typescript.language_typescript(), "typescript"))
        elif language == 'tsx':
            import tree_sitter_typescript
            parser.set_language(tree_sitter.Language(tree_sitter_typescript.language_tsx(), "tsx"))
        else:
            return [content[i:i+1000] for i in range(0, len(content), 1000)]
            
        tree = parser.parse(bytes(content, "utf8"))
        chunks = []
        def traverse(node):
            if node.type in ['function_declaration', 'method_declaration', 'class_declaration', 'interface_declaration', 'type_alias_declaration']:
                chunks.append(content.encode("utf8")[node.start_byte:node.end_byte].decode("utf8"))
            for child in node.children:
                traverse(child)
        traverse(tree.root_node)
        return chunks if chunks else [content]
    except Exception as e:
        return [content[i:i+1000] for i in range(0, len(content), 1000)]

def remove_file_from_index(filepath):
    # Find all chunks belonging to this file and delete them
    try:
        collection.delete(where={"file": filepath})
        if filepath in file_hashes:
            del file_hashes[filepath]
            save_state()
        logging.info(f"🗑️  Removed {filepath} from RAG index.")
    except Exception as e:
        logging.error(f"Failed to remove {filepath}: {e}")

def index_file(filepath):
    lang = get_language(filepath)
    if not lang:
        return

    current_hash = get_file_hash(filepath)
    if not current_hash:
        return

    # Skip if unchanged
    if file_hashes.get(filepath) == current_hash:
        return

    # Delete old chunks first
    remove_file_from_index(filepath)

    try:
        with open(filepath, 'r', encoding='utf-8', errors='ignore') as f:
            content = f.read()
    except Exception:
        return

    chunks = chunk_code_ast(filepath, content, lang)
    docs, metadatas, ids = [], [], []
    layer = get_layer(filepath)
    
    for idx, chunk in enumerate(chunks):
        docs.append(chunk)
        metadatas.append({"file": filepath, "layer": layer, "language": lang})
        ids.append(f"{filepath}_{idx}")

    if docs:
        collection.upsert(documents=docs, metadatas=metadatas, ids=ids)
        file_hashes[filepath] = current_hash
        save_state()
        logging.info(f"✅ Indexed {filepath} ({len(chunks)} chunks).")

def should_watch(path):
    if '.git' in path or 'node_modules' in path or '.agents' in path or 'venv' in path:
        return False
    return bool(get_language(path))

if __name__ == "__main__":
    logging.info(f"👁️  Starting Enterprise RAG Daemon on {WORKSPACE}...")
    for changes in watch(WORKSPACE, watch_filter=lambda change, path: should_watch(path)):
        for change, path in changes:
            if change == Change.deleted:
                remove_file_from_index(path)
            else:
                index_file(path)
