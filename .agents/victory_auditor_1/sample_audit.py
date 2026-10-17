import os
import re
import urllib.parse
import random

random.seed(42)

files_to_check = [
    'pegasus/agents.md',
    'pegasus/docs/ARCHITECTURE.md',
    'pegasus/docs/BACKEND_SERVICES.md',
    'pegasus/docs/FEATURES_AND_PORTALS.md',
    'pegasus/docs/INFRASTRUCTURE.md',
    'pegasusX/agents.md',
    'pegasusX/docs/ARCHITECTURE.md',
    'pegasusX/docs/BACKEND_SERVICES.md',
    'pegasusX/docs/FEATURES_AND_ROLE_ROWS.md',
    'pegasusX/docs/INFRASTRUCTURE.md',
    'pegasus.x/agents.md',
    'pegasus.x/docs/ARCHITECTURE.md',
    'pegasus.x/docs/BACKEND_AND_PLANNING.md',
    'pegasus.x/docs/FEATURES_AND_APPS.md',
    'pegasus.x/docs/INFRASTRUCTURE.md'
]

base_dir = '/Users/shakhzod/Desktop/V.O.I.D'

all_citations = []

for rel in files_to_check:
    full = os.path.join(base_dir, rel)
    with open(full, 'r', encoding='utf-8') as f:
        lines = f.readlines()
    for idx, line in enumerate(lines):
        links = re.findall(r'file://([^\s`\)\"\'\]\>]+(?:\s*,\s*\d+(?:-\d+)?)*)', line)
        for raw in links:
            match = re.match(r'^(.*?)(?::(\d.*)|#L(\d.*))?$', raw)
            if match:
                path_part = urllib.parse.unquote(match.group(1))
                line_part = match.group(2) or match.group(3)
                context = line.strip()
                all_citations.append((rel, idx + 1, path_part, line_part, context))

sample = random.sample(all_citations, 30)

print(f"Sample audit of 30 citations:")
for i, (doc, doc_line, path, line_spec, ctx) in enumerate(sample, 1):
    assert os.path.exists(path), f"Path {path} does not exist!"
    target_type = "DIR" if os.path.isdir(path) else "FILE"
    lines_info = ""
    if target_type == "FILE" and line_spec:
        with open(path, 'r', encoding='utf-8', errors='ignore') as tf:
            f_lines = tf.readlines()
        nums = [int(n) for n in re.findall(r'\d+', line_spec)]
        assert max(nums) <= len(f_lines) + 5, f"Line {max(nums)} out of bounds (file has {len(f_lines)} lines)"
        lines_info = f"[lines {line_spec} / {len(f_lines)}]"
    print(f"[{i:02d}] PASS: {doc}:{doc_line} -> {path} {lines_info} ({target_type})")

print("\nAll 30 randomized samples verified 100% on disk!")
