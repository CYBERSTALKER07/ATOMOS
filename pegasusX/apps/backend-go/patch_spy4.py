import re

with open("apps/backend-go/factory/service_test.go", "r") as f:
    content = f.read()

# After returning from setupTestFactoryService, inject svc
content = re.sub(r'svc := setupTestFactoryService\(repo\)', r'svc := setupTestFactoryService(repo)\n\trepo.svc = svc', content)

with open("apps/backend-go/factory/service_test.go", "w") as f:
    f.write(content)

with open("apps/backend-go/payload/service_test.go", "r") as f:
    content = f.read()

content = re.sub(r'svc := setupTestPayloadService\(repo\)', r'svc := setupTestPayloadService(repo)\n\trepo.svc = svc', content)

with open("apps/backend-go/payload/service_test.go", "w") as f:
    f.write(content)

