import re

with open("service.go", "r") as f:
    code = f.read()

# 1. Remove Repository and inMemoryRepository definitions (they are now in repository.go)
repo_pattern = r'// Repository is the mutation seam.*?\n// Service stores additive in-memory data for factory operational surfaces\.'
replacement = r'// Service stores additive in-memory data for factory operational surfaces.'
code = re.sub(repo_pattern, replacement, code, flags=re.DOTALL)

# 2. Fix ensureDemoDataLocked
code = re.sub(
    r'if err := r\.hydrateWhileLocked\(context\.Background\(\), s\); err != nil \{',
    r'if err := r.Hydrate(context.Background(), s.factoryNode, s); err != nil {',
    code
)

# 3. Fix SeedDemoManifests
code = re.sub(
    r'if err := r\.SeedDemoManifests\(context\.Background\(\), snap\); err == nil \{',
    r'// SeedDemoManifests removed\n\t\tif true {',
    code
)

with open("service.go", "w") as f:
    f.write(code)
print("Updated service.go")
