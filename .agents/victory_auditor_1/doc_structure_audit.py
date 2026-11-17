import os
import re

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

for rel in files_to_check:
    full = os.path.join(base_dir, rel)
    with open(full, 'r', encoding='utf-8') as f:
        text = f.read()
    
    lines = text.splitlines()
    words = len(text.split())
    bytes_count = len(text.encode('utf-8'))
    
    h1 = [l for l in lines if l.startswith('# ')]
    h2 = [l for l in lines if l.startswith('## ')]
    h3 = [l for l in lines if l.startswith('### ')]
    links = re.findall(r'file://[^\s`\)\"\'\]\>]+', text)
    
    print(f"=== {rel} ===")
    print(f"  Title: {h1[0] if h1 else 'None'}")
    print(f"  Lines: {len(lines)} | Words: {words} | Bytes: {bytes_count} | Links: {len(links)}")
    print(f"  H2 sections ({len(h2)}): {', '.join([h.replace('## ', '') for h in h2[:6]])}...")
    print()
