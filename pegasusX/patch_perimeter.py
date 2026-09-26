import re

with open('apps/backend-go/warehouse/perimeter.go', 'r') as f:
    content = f.read()

old_block = """	pipe.Del(ctx, key)
	if len(cells) > 0 {
		var args []interface{}
		for _, cell := range cells {
			args = append(args, cell)
		}
		pipe.SAdd(ctx, key, args...)
	}"""

new_block = """	pipe.Del(ctx, key)
	var args []interface{}
	if len(cells) > 0 {
		for _, cell := range cells {
			args = append(args, cell)
		}
	} else {
		// Cache an empty result using a sentinel value to prevent cache stampedes
		// on suppliers with zero coverage.
		args = append(args, "__empty__")
	}
	pipe.SAdd(ctx, key, args...)"""

if old_block in content:
    content = content.replace(old_block, new_block)
    with open('apps/backend-go/warehouse/perimeter.go', 'w') as f:
        f.write(content)
    print("Patched PublishSupplierPerimeter successfully.")
else:
    print("Could not find the target block to replace.")
