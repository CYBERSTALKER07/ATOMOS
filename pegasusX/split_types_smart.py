import os
import re

input_file = 'packages/types/index.ts'
out_dir = 'packages/types/src'
os.makedirs(out_dir, exist_ok=True)

with open(input_file, 'r') as f:
    lines = f.readlines()

# 1. Split into chunks by headers
chunks = {}
current_header = "base"
current_lines = []

for line in lines:
    if line.startswith("// ──"):
        if current_lines:
            chunks[current_header] = current_lines
        current_lines = [line]
        name = line.strip().replace("// ──", "").strip("─- ").strip()
        name = re.sub(r'[^a-zA-Z0-9]+', '-', name).strip('-').lower()
        if "rfc" in name: current_header = "problem-detail"
        elif "role" in name: current_header = "primitives"
        elif "supplier" in name: current_header = "supplier"
        elif "compliance" in name: current_header = "compliance"
        elif "claims" in name: current_header = "claims"
        elif "envelope" in name: current_header = "envelope"
        elif "event-payloads" in name: current_header = "event-payloads"
        elif "discriminated" in name: current_header = "events"
        elif "warehouse" in name: current_header = "warehouse"
        elif "notification" in name: current_header = "notifications"
        elif "fleet" in name: current_header = "fleet"
        elif "auto-order" in name: current_header = "auto-order"
        elif "partner" in name: current_header = "partner"
        elif "market" in name: current_header = "market"
        elif "admin" in name: current_header = "admin"
        else: current_header = name
    else:
        current_lines.append(line)

if current_lines:
    chunks[current_header] = current_lines

# 2. Extract all exported symbols per chunk
chunk_exports = {}
export_regex = re.compile(r'^export\s+(type|interface|const|enum|function|class)\s+([a-zA-Z0-9_]+)')

for chunk_name, clines in chunks.items():
    exports = set()
    for line in clines:
        match = export_regex.match(line)
        if match:
            exports.add(match.group(2))
    chunk_exports[chunk_name] = exports

# 3. For each chunk, find what external symbols it uses
word_regex = re.compile(r'\b[a-zA-Z0-9_]+\b')

for chunk_name, clines in chunks.items():
    content = "".join(clines)
    used_words = set(word_regex.findall(content))
    
    # generate import statements
    imports_to_add = {}
    for other_chunk, other_exports in chunk_exports.items():
        if other_chunk == chunk_name:
            continue
        needed = used_words.intersection(other_exports)
        if needed:
            imports_to_add[other_chunk] = sorted(list(needed))
    
    # write to file
    with open(os.path.join(out_dir, f"{chunk_name}.ts"), 'w') as f:
        # write imports
        for other_chunk, needed in imports_to_add.items():
            f.write(f'import {{ {", ".join(needed)} }} from "./{other_chunk}";\n')
        if imports_to_add:
            f.write("\n")
        f.writelines(clines)

# 4. Generate index.ts
with open('packages/types/index.ts', 'w') as f:
    for chunk_name in chunks.keys():
        f.write(f'export * from "./src/{chunk_name}";\n')

print("Split completed with intelligent imports!")
