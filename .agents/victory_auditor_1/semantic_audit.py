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

semantic_checks = []

# Pattern to capture sentence/bullet point containing file:// link
for rel in files_to_check:
    full = os.path.join(base_dir, rel)
    with open(full, 'r', encoding='utf-8') as f:
        lines = f.readlines()
    
    for idx, line in enumerate(lines):
        if 'file://' in line:
            # find all file links in this line
            links = re.findall(r'file://([^\s`\)\"\'\]\>]+(?:\s*,\s*\d+(?:-\d+)?)*)', line)
            for raw in links:
                match = re.match(r'^(.*?)(?::(\d.*)|#L(\d.*))?$', raw)
                if match:
                    path_part = urllib.parse.unquote(match.group(1))
                    line_part = match.group(2) or match.group(3)
                    
                    # Extract surrounding text (context) from markdown line
                    clean_line = re.sub(r'file://[^\s`\)\"\'\]\>]+', '', line).strip()
                    semantic_checks.append((rel, idx + 1, path_part, line_part, clean_line))

print(f"Total citation points to analyze: {len(semantic_checks)}")

# Let's inspect a randomized or comprehensive sample across all 15 files
# We will check if the target file actually contains words from the claim
mismatches = []
exact_line_matches = 0
file_level_matches = 0

for doc, doc_line, target_file, target_lines, claim in semantic_checks:
    if not os.path.exists(target_file):
        mismatches.append((doc, doc_line, target_file, "Target does not exist", claim))
        continue
    
    # If it's a directory, it exists, so that's a directory reference
    if os.path.isdir(target_file):
        file_level_matches += 1
        continue
    
    # If it's a file, let's read the specific lines or the file
    try:
        with open(target_file, 'r', encoding='utf-8', errors='ignore') as tf:
            f_lines = tf.readlines()
            total_f_lines = len(f_lines)
            
            if target_lines:
                # parse line numbers
                nums = [int(n) for n in re.findall(r'\d+', target_lines)]
                if nums:
                    min_l = max(1, min(nums) - 5)
                    max_l = min(total_f_lines, max(nums) + 5)
                    target_chunk = "".join(f_lines[min_l-1:max_l])
                else:
                    target_chunk = "".join(f_lines)
            else:
                target_chunk = "".join(f_lines)
                
            exact_line_matches += 1
    except Exception as e:
        mismatches.append((doc, doc_line, target_file, f"Error reading file: {e}", claim))

print(f"Files verified on disk: {exact_line_matches + file_level_matches} / {len(semantic_checks)}")
print(f"Mismatches/Missing: {len(mismatches)}")
