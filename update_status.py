import os
import glob

status_block = """
> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.
"""

files_to_update = glob.glob("pegasusX/apps/*/README.md")
files_to_update.extend(glob.glob("pegasusX/context/*.md"))
files_to_update.append("pegasusX/README.md")
files_to_update.append("pegasus/README.md")

for filepath in files_to_update:
    if not os.path.exists(filepath):
        continue
    with open(filepath, 'r') as f:
        content = f.read()
    
    if "Current Project State:" in content and "[!NOTE]" in content:
        print(f"Skipping {filepath}, already has status.")
        continue
        
    lines = content.split('\n')
    
    if not lines:
        continue
        
    # Find the first H1 header
    insert_idx = 0
    for i, line in enumerate(lines):
        if line.startswith('# '):
            insert_idx = i + 1
            break
            
    if insert_idx > 0:
        lines.insert(insert_idx, status_block)
        with open(filepath, 'w') as f:
            f.write('\n'.join(lines))
        print(f"Updated {filepath}")
    else:
        print(f"No H1 found in {filepath}")

