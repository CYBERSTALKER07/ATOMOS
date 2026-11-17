import chromadb
import os
from chromadb.utils import embedding_functions

client = chromadb.PersistentClient(path=os.path.join(os.path.dirname(os.path.dirname(os.getcwd())), ".agents", "memory"))
emb_fn = embedding_functions.SentenceTransformerEmbeddingFunction(model_name="all-MiniLM-L6-v2")
coll = client.get_collection("pegasusx_codebase", embedding_function=emb_fn)

res = coll.query(query_texts=["FCM stale token deletion"], n_results=1)
print("Top Result File:", res['metadatas'][0][0]['file'])
print("Content Preview:", res['documents'][0][0][:200].replace('\n', ' '))
