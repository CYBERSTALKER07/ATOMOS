import re
with open("packages/api-client/index.ts") as f:
    text = f.read()
methods = re.findall(r"(?:async\s+)?(\w+)\s*\([^)]*\)\s*:\s*Promise<", text)
with open("api_client_methods.txt","w") as f:
    for m in sorted(set(methods)):
        f.write(m+"\n")
print("count", len(set(methods)))
