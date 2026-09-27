import os
import re

doc_files = [
    '/Users/shakhzod/Desktop/V.O.I.D/pegasus/agents.md',
    '/Users/shakhzod/Desktop/V.O.I.D/pegasus/docs/ARCHITECTURE.md',
    '/Users/shakhzod/Desktop/V.O.I.D/pegasus/docs/BACKEND_SERVICES.md',
    '/Users/shakhzod/Desktop/V.O.I.D/pegasus/docs/FEATURES_AND_PORTALS.md',
    '/Users/shakhzod/Desktop/V.O.I.D/pegasus/docs/INFRASTRUCTURE.md',
    '/Users/shakhzod/Desktop/V.O.I.D/pegasusX/agents.md',
    '/Users/shakhzod/Desktop/V.O.I.D/pegasusX/docs/ARCHITECTURE.md',
    '/Users/shakhzod/Desktop/V.O.I.D/pegasusX/docs/BACKEND_SERVICES.md',
    '/Users/shakhzod/Desktop/V.O.I.D/pegasusX/docs/FEATURES_AND_ROLE_ROWS.md',
    '/Users/shakhzod/Desktop/V.O.I.D/pegasusX/docs/INFRASTRUCTURE.md',
    '/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/agents.md',
    '/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/docs/ARCHITECTURE.md',
    '/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/docs/BACKEND_AND_PLANNING.md',
    '/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/docs/FEATURES_AND_APPS.md',
    '/Users/shakhzod/Desktop/V.O.I.D/pegasus.x/docs/INFRASTRUCTURE.md',
]

print("=== CHECKING SECTION STRUCTURE & KEYWORDS ===")
for doc_path in doc_files:
    short = doc_path.replace('/Users/shakhzod/Desktop/V.O.I.D/', '')
    with open(doc_path, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # Check headers
    h1s = re.findall(r'^# (.+)$', content, re.MULTILINE)
    h2s = re.findall(r'^## (.+)$', content, re.MULTILINE)
    
    # Check for "what", "how", "why"
    has_what = bool(re.search(r'what it is', content, re.IGNORECASE))
    has_how = bool(re.search(r'how it works', content, re.IGNORECASE))
    has_why = bool(re.search(r'why it is (there|here|used|designed)', content, re.IGNORECASE) or re.search(r'why it exists', content, re.IGNORECASE))
    
    # Check for honesty / zero theatre
    has_zero_theatre = bool(re.search(r'zero[- ]theatre|no[- ]theatre|theatre', content, re.IGNORECASE))
    has_honesty = bool(re.search(r'honesty', content, re.IGNORECASE))
    
    # Check for suspicious TODO or dummy
    todos = len(re.findall(r'\b(TODO|FIXME|XXX|TBD)\b', content))
    
    print(f"\n--- {short} ({len(content)} bytes, {len(content.splitlines())} lines) ---")
    print(f"H1: {h1s}")
    print(f"H2 count: {len(h2s)} | Sample H2s: {h2s[:4]}")
    print(f"Coverage: What={has_what}, How={has_how}, Why={has_why}")
    print(f"Keywords: ZeroTheatre={has_zero_theatre}, Honesty={has_honesty}, TODOs={todos}")

