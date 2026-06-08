import re

with open("/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/payload/service.go", "r") as f:
    content = f.read()

# Fix NewService to not use NewInMemoryRepository
content = content.replace("c.Repo = NewInMemoryRepository()", "panic(\"repo is required\")")

# Remove Spanner logic from ensureDemoDataLocked
content = re.sub(
r'''func \(s \*Service\) ensureDemoDataLocked\(\) \{
\s*if !s\.spannerLoaded \{
\s*if r, ok := s\.repo\.\(\*SpannerRepository\); ok \{
\s*if err := r\.hydrateWhileLocked\(context\.Background\(\), s\.supplierID, s\); err == nil && len\(s\.manifests\) > 0 \{
\s*s\.spannerLoaded = true
\s*return
\s*\}
\s*\}
\s*\}''',
r'''func (s *Service) ensureDemoDataLocked() {''', content, flags=re.MULTILINE)

# Remove the r.SeedDemoManifests block
content = re.sub(
r'''\s*if r, ok := s\.repo\.\(\*SpannerRepository\); ok && !s\.spannerLoaded \{
\s*snap := s\.buildPersistenceSnapshotLocked\(\)
\s*if err := r\.SeedDemoManifests\(context\.Background\(\), s\.supplierID, snap\); err == nil \{
\s*s\.spannerLoaded = true
\s*\}
\s*\}''',
"", content)

with open("/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/backend-go/payload/service.go", "w") as f:
    f.write(content)

