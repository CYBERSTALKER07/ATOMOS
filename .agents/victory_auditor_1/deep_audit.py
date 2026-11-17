import re
import os
import urllib.parse

files_to_check = [
    'pegasus/agents.md',
    'pegasusX/agents.md',
    'pegasus.x/agents.md',
    'pegasus/docs/ARCHITECTURE.md',
    'pegasus/docs/BACKEND_SERVICES.md',
    'pegasus/docs/FEATURES_AND_PORTALS.md',
    'pegasus/docs/INFRASTRUCTURE.md',
    'pegasusX/docs/ARCHITECTURE.md',
    'pegasusX/docs/BACKEND_SERVICES.md',
    'pegasusX/docs/FEATURES_AND_ROLE_ROWS.md',
    'pegasusX/docs/INFRASTRUCTURE.md',
    'pegasus.x/docs/ARCHITECTURE.md',
    'pegasus.x/docs/BACKEND_AND_PLANNING.md',
    'pegasus.x/docs/FEATURES_AND_APPS.md',
    'pegasus.x/docs/INFRASTRUCTURE.md'
]

base_dir = '/Users/shakhzod/Desktop/V.O.I.D'

all_links = []
broken_links = []
line_range_issues = []
unique_targets = set()

for rel in files_to_check:
    full = os.path.join(base_dir, rel)
    with open(full, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # We find all matches of file://...
    # Links in markdown can be enclosed in backticks or parens:
    # `file:///path/to/file:10-20` or [title](file:///path/to/file)
    raw_matches = re.findall(r'file://([^\s`\)\"\'\]\>]+(?:\s*,\s*\d+(?:-\d+)?)*)', content)
    for raw in raw_matches:
        raw_full = 'file://' + raw
        # Separate the file path from line numbers / anchors
        match = re.match(r'^(.*?)(?::(\d.*)|#L(\d.*))?$', raw)
        if not match:
            broken_links.append((rel, raw_full, 'Cannot parse URL'))
            continue
        
        path_part = match.group(1)
        line_part = match.group(2) or match.group(3)
        
        clean_path = urllib.parse.unquote(path_part)
        
        if not os.path.exists(clean_path):
            broken_links.append((rel, raw_full, clean_path, 'Target does not exist'))
        else:
            unique_targets.add(clean_path)
            all_links.append((rel, raw_full, clean_path, line_part))
            if os.path.isfile(clean_path) and line_part:
                with open(clean_path, 'r', encoding='utf-8', errors='ignore') as tf:
                    total_lines = len(tf.readlines())
                # Parse all numbers in line_part
                nums = [int(n) for n in re.findall(r'\d+', line_part)]
                for n in nums:
                    if n > total_lines + 5: # tolerance of 5 lines
                        line_range_issues.append((rel, raw_full, clean_path, n, total_lines))

print(f'Total links evaluated: {len(all_links)}')
print(f'Unique files/directories targeted: {len(unique_targets)}')
print(f'Broken links (missing path): {len(broken_links)}')
print(f'Line numbers out of bounds: {len(line_range_issues)}')

if broken_links:
    for b in broken_links:
        print('BROKEN:', b)

if line_range_issues:
    for l in line_range_issues:
        print('OUT OF BOUNDS:', l)
