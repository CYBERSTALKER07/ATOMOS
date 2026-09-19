import os
import glob
import chromadb
from chromadb.utils import embedding_functions

# Initialize ChromaDB in the local .agents/memory directory
MEMORY_DIR = os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(__file__))), ".agents", "memory")
os.makedirs(MEMORY_DIR, exist_ok=True)
client = chromadb.PersistentClient(path=MEMORY_DIR)

# We use sentence-transformers as a fast, free local embedding model
emb_fn = embedding_functions.SentenceTransformerEmbeddingFunction(model_name="all-MiniLM-L6-v2")
collection = client.get_or_create_collection(name="pegasusx_codebase", embedding_function=emb_fn)

# Tree-Sitter setup (Lazy loaded to avoid crash if not installed yet)
def get_parser(language):
    import tree_sitter
    parser = tree_sitter.Parser()
    if language == 'go':
        import tree_sitter_go
        parser.set_language(tree_sitter.Language(tree_sitter_go.language(), "go"))
    elif language == 'typescript' or language == 'tsx':
        import tree_sitter_typescript
        if language == 'tsx':
            parser.set_language(tree_sitter.Language(tree_sitter_typescript.language_tsx(), "tsx"))
        else:
            parser.set_language(tree_sitter.Language(tree_sitter_typescript.language_typescript(), "typescript"))
    return parser

def chunk_code_ast(filepath, content, language):
    """Chunks code semantically using AST (extracts functions/classes)."""
    try:
        parser = get_parser(language)
        tree = parser.parse(bytes(content, "utf8"))
    except Exception as e:
        # Fallback to naive chunking if parser fails or missing language
        return [content[i:i+1000] for i in range(0, len(content), 1000)]
    
    chunks = []
    # Simplified AST traversal: look for function/method declarations
    def traverse(node):
        if node.type in ['function_declaration', 'method_declaration', 'lexical_declaration', 'class_declaration']:
            start_byte = node.start_byte
            end_byte = node.end_byte
            chunk_text = content.encode("utf8")[start_byte:end_byte].decode("utf8")
            chunks.append(chunk_text)
        for child in node.children:
            traverse(child)
            
    traverse(tree.root_node)
    
    # If no functions found, just return the whole file
    if not chunks:
        chunks = [content]
    return chunks

def index_layer(layer_name, glob_pattern, language_hint):
    print(f"Indexing {layer_name}...")
    files = glob.glob(glob_pattern, recursive=True)
    
    docs = []
    metadatas = []
    ids = []
    
    for idx, filepath in enumerate(files):
        if not os.path.isfile(filepath):
            continue
            
        with open(filepath, 'r', encoding='utf-8', errors='ignore') as f:
            content = f.read()
            
        chunks = chunk_code_ast(filepath, content, language_hint)
        
        for chunk_idx, chunk in enumerate(chunks):
            chunk_id = f"{filepath}_{chunk_idx}"
            docs.append(chunk)
            metadatas.append({
                "layer": layer_name,
                "file": filepath,
                "language": language_hint
            })
            ids.append(chunk_id)
            
            # Batch upsert to avoid huge memory spikes
            if len(docs) >= 100:
                collection.upsert(documents=docs, metadatas=metadatas, ids=ids)
                docs, metadatas, ids = [], [], []
                
    if docs:
        collection.upsert(documents=docs, metadatas=metadatas, ids=ids)

if __name__ == "__main__":
    WORKSPACE = os.path.dirname(os.path.dirname(os.path.dirname(__file__)))
    os.chdir(WORKSPACE)
    
    # 1. Backend Layer (Go)
    index_layer("backend", "apps/backend-go/**/*.go", "go")
    
    # 2. Infra Layer (SQL/DDL)
    index_layer("infra", "apps/backend-go/schema/**/*.ddl", "sql")
    
    # 3. Client Web (Next.js/React)
    index_layer("client-web", "apps/*-portal/**/*.tsx", "tsx")
    index_layer("client-web-ts", "apps/*-portal/**/*.ts", "typescript")
    
    # 4. Mobile Apps & Shared Packages
    index_layer("mobile-android", "packages/mobile-android-design/**/*.kt", "kotlin")
    index_layer("mobile-ios", "packages/mobile-ios-design/**/*.swift", "swift")
    index_layer("shared-packages", "packages/**/*.ts", "typescript")
    index_layer("shared-packages-tsx", "packages/**/*.tsx", "tsx")
    
    print(f"Finished indexing PegasusX! Total chunks in DB: {collection.count()}")
