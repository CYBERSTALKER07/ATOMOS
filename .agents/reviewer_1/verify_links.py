import os
import re
import sys

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

def clean_url(url):
    # Strip trailing punctuation often caught in markdown parsing
    url = url.rstrip(').,;:`\'"')
    return url

total_links_all = 0
broken_links_all = []
valid_links_all = []

file_stats = {}

file_url_pattern = re.compile(r'file:///([^\s\)\>`\'"]+)')

for doc_path in doc_files:
    if not os.path.exists(doc_path):
        print(f"ERROR: Doc file does not exist: {doc_path}")
        continue
    
    with open(doc_path, 'r', encoding='utf-8') as f:
        lines = f.readlines()
        
    doc_total = 0
    doc_valid = 0
    doc_broken = []
    
    for line_idx, line in enumerate(lines, start=1):
        matches = file_url_pattern.findall(line)
        for m in matches:
            raw_target = '/' + clean_url(m)
            doc_total += 1
            total_links_all += 1
            
            # Check for anchor or line number
            target_path = raw_target
            line_spec = None
            if '#L' in target_path:
                target_path, line_spec = target_path.split('#L', 1)
            elif '#l' in target_path:
                target_path, line_spec = target_path.split('#l', 1)
            elif ':' in os.path.basename(target_path):
                # could be path:line
                parts = target_path.rsplit(':', 1)
                if parts[1].isdigit():
                    target_path = parts[0]
                    line_spec = parts[1]
                    
            # Check if file exists on disk
            exists = os.path.exists(target_path)
            line_valid = True
            line_err = ""
            
            if exists:
                if os.path.isfile(target_path) and line_spec:
                    try:
                        with open(target_path, 'r', encoding='utf-8', errors='ignore') as tf:
                            target_line_count = len(tf.readlines())
                        
                        # parse line_spec like 42 or 42-L60 or 42-60
                        start_line = int(re.split(r'[-L]', line_spec)[0])
                        if start_line > target_line_count + 10:  # slight leeway
                            line_valid = False
                            line_err = f"Target file has {target_line_count} lines, link points to L{start_line}"
                    except Exception as e:
                        pass
                doc_valid += 1
                valid_links_all.append((doc_path, line_idx, raw_target, target_path, line_spec, line_valid, line_err))
            else:
                doc_broken.append((line_idx, raw_target, target_path))
                broken_links_all.append((doc_path, line_idx, raw_target, target_path))
                
    file_stats[doc_path] = {
        'total': doc_total,
        'valid': doc_valid,
        'broken': doc_broken
    }

print("="*80)
print("EMPIRICAL LINK RESOLUTION AUDIT REPORT")
print("="*80)
for doc_path, s in file_stats.items():
    short_name = doc_path.replace('/Users/shakhzod/Desktop/V.O.I.D/', '')
    print(f"{short_name:45s} | Total: {s['total']:3d} | Valid: {s['valid']:3d} | Broken: {len(s['broken']):2d}")
    for b in s['broken']:
        print(f"   [BROKEN] Line {b[0]}: {b[1]} -> {b[2]}")

print("="*80)
print(f"GRAND TOTAL LINKS SCANNED: {total_links_all}")
print(f"TOTAL VALID LINKS        : {len(valid_links_all)}")
print(f"TOTAL BROKEN LINKS       : {len(broken_links_all)}")
resolution_pct = (len(valid_links_all) / total_links_all * 100) if total_links_all > 0 else 0.0
print(f"RESOLUTION PERCENTAGE    : {resolution_pct:.2f}%")
print("="*80)

# Check line number warnings
line_warnings = [v for v in valid_links_all if not v[5]]
print(f"Line number warnings (file exists, but specified line exceeds file length): {len(line_warnings)}")
for lw in line_warnings:
    print(f"  In {lw[0].replace('/Users/shakhzod/Desktop/V.O.I.D/', '')}:{lw[1]} -> {lw[2]} ({lw[6]})")
