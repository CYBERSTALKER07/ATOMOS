import re

with open("service.go", "r") as f:
    code = f.read()

# Fix s.factoryNode to s.factoryID
code = code.replace('s.factoryNode', 's.factoryID')

# Fix unused r and snap
code = re.sub(
    r'if r, ok := s\.repo\.\(\*SpannerRepository\); ok && !s\.spannerLoaded \{\n\t\tsnap := s\.buildPersistenceSnapshotLocked\(\)\n\t\t// SeedDemoManifests removed\n\t\tif true \{\n\t\t\ts\.spannerLoaded = true\n\t\t\}\n\t\}',
    r'if _, ok := s.repo.(*SpannerRepository); ok && !s.spannerLoaded {\n\t\ts.spannerLoaded = true\n\t}',
    code
)

with open("service.go", "w") as f:
    f.write(code)
print("Updated service.go unused vars")
