import re

with open("apps/backend-go/payload/ship_units.go", "r") as f:
    content = f.read()

pattern = re.compile(r'func ssccSerial\(manifestID, orderID string, seq int64\) uint64 \{\n\th := fnv\.New64a\(\)\n\t_, _ = h\.Write\(\[\]byte\(manifestID \+ "\|" \+ orderID\)\)\n\tvar buf \[8\]byte\n\tbinary\.BigEndian\.PutUint64\(buf\[:\], uint64\(seq\)\)\n\t_, _ = h\.Write\(buf\[:\]\)\n\treturn h\.Sum64\(\)\n\}')

replacement = r"""var ssccGlobalCounter uint64 = uint64(time.Now().UnixNano())

func ssccSerial(manifestID, orderID string, seq int64) uint64 {
	return atomic.AddUint64(&ssccGlobalCounter, 1)
}"""

content = pattern.sub(replacement, content)
content = content.replace('import (\n\t"context"', 'import (\n\t"context"\n\t"sync/atomic"')

with open("apps/backend-go/payload/ship_units.go", "w") as f:
    f.write(content)
