import re

with open("service.go", "r") as f:
    code = f.read()

# 1. Remove Repository and inMemoryRepository definitions (they are now in repository.go)
repo_pattern = r'// Repository is the mutation seam.*?\n// Service stores additive in-memory payload operations state\.'
replacement = r'// Service stores additive in-memory payload operations state.'
code = re.sub(repo_pattern, replacement, code, flags=re.DOTALL)

with open("service.go", "w") as f:
    f.write(code)
print("Updated payload service.go")
