with open("apps/backend-go/payload/ship_units.go", "r") as f:
    content = f.read()

content = content.replace('\t"encoding/binary"\n', '')
content = content.replace('\t"hash/fnv"\n', '')

with open("apps/backend-go/payload/ship_units.go", "w") as f:
    f.write(content)
