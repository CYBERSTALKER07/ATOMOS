import os

STATUS_BLOCK = """> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.

"""

def update_file(filepath):
    with open(filepath, 'r') as f:
        content = f.read()
    
    if "**Current Project State:** GCP Migration (Phase 2)" in content:
        return # Already updated
        
    # Inject after the first line (usually the title)
    lines = content.split('\n')
    if len(lines) > 0 and lines[0].startswith('#'):
        lines.insert(1, '\n' + STATUS_BLOCK)
    else:
        lines.insert(0, STATUS_BLOCK)
        
    with open(filepath, 'w') as f:
        f.write('\n'.join(lines))

for root, dirs, files in os.walk('.'):
    if 'node_modules' in root or '.git' in root or 'vendor' in root:
        continue
    for file in files:
        if file.endswith('.md'):
            update_file(os.path.join(root, file))
            
print("Updated all md files")
