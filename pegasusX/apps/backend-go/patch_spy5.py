import re

with open("apps/backend-go/factory/service_test.go", "r") as f:
    content = f.read()

content = re.sub(r'svc := newFactoryTestService\(([^)]+)\)', r'svc := newFactoryTestService(\1)\n\trepo.svc = svc', content)

# But wait, line 66 has `svc := newFactoryTestService(&factoryRepoSpy{}, &factoryCacheBackendSpy{})` which doesn't use `repo` variable. We must change that too!
content = content.replace("svc := newFactoryTestService(&factoryRepoSpy{}, &factoryCacheBackendSpy{})", "repo := &factoryRepoSpy{}\n\tsvc := newFactoryTestService(repo, &factoryCacheBackendSpy{})\n\trepo.svc = svc")

with open("apps/backend-go/factory/service_test.go", "w") as f:
    f.write(content)

with open("apps/backend-go/payload/service_test.go", "r") as f:
    content = f.read()

content = re.sub(r'svc := newPayloadTestService\(([^)]+)\)', r'svc := newPayloadTestService(\1)\n\trepo.svc = svc', content)

content = content.replace("svc := newPayloadTestService(&payloadRepoSpy{}, &payloadCacheBackendSpy{})", "repo := &payloadRepoSpy{}\n\tsvc := newPayloadTestService(repo, &payloadCacheBackendSpy{})\n\trepo.svc = svc")


with open("apps/backend-go/payload/service_test.go", "w") as f:
    f.write(content)

