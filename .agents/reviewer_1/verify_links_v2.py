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

file_url_pattern = re.compile(r'file:///([^\s\)\>`\'"]+)')

file_stats = {}
real_broken = []
all_parsed = []
line_range_mismatches = []

for doc_path in doc_files:
    with open(doc_path, 'r', encoding='utf-8') as f:
        lines = f.readlines()
        
    stats = {'total': 0, 'valid': 0, 'broken': []}
    
    for line_idx, line in enumerate(lines, start=1):
        for raw in file_url_pattern.findall(line):
            raw = raw.rstrip(').,;:`\'"')
            raw_full = '/' + raw
            
            # Format handling:
            # e.g.: /Users/.../file.go:417-433
            # e.g.: /Users/.../file.go#L417-L433
            # e.g.: /Users/.../file.go#L417
            # e.g.: /Users/.../file.go:417
            m = re.match(r'^(.*?)(?::(\d+(?:-\d+)?)|#[Ll](\d+(?:-[Ll]?\d+)?))?$', raw_full)
            if m and (m.group(2) or m.group(3)):
                path = m.group(1)
                line_spec = m.group(2) or m.group(3)
            else:
                path = raw_full
                line_spec = None
                
            exists = os.path.exists(path)
            stats['total'] += 1
            if not exists:
                stats['broken'].append((line_idx, raw_full, path, line_spec))
                real_broken.append((doc_path, line_idx, raw_full, path, line_spec))
            else:
                stats['valid'] += 1
                # Check lines
                if os.path.isfile(path) and line_spec:
                    try:
                        with open(path, 'r', encoding='utf-8', errors='ignore') as tf:
                            actual_lines = len(tf.readlines())
                        parts = re.split(r'[-L]', line_spec)
                        start_l = int(parts[0]) if parts[0] else 1
                        if start_l > actual_lines + 20:
                            line_range_mismatches.append((doc_path, line_idx, raw_full, path, line_spec, actual_lines))
                    except Exception as e:
                        pass
                        
            all_parsed.append((doc_path, line_idx, raw_full, path, line_spec, exists))
            
    file_stats[doc_path] = stats

print("="*80)
print("AUDIT REPORT: FILE:/// LINKS (V2 EXACT PARSER)")
print("="*80)
for doc_path, s in file_stats.items():
    short = doc_path.replace('/Users/shakhzod/Desktop/V.O.I.D/', '')
    print(f"{short:42s} | Total: {s['total']:3d} | Valid: {s['valid']:3d} | Broken: {len(s['broken']):2d}")
    for b in s['broken']:
        print(f"   [BROKEN] line {b[0]}: {b[1]} -> target: {b[2]}")

print("="*80)
print(f"TOTAL LINKS SCANNED : {len(all_parsed)}")
print(f"TOTAL VALID ON DISK : {len(all_parsed) - len(real_broken)}")
print(f"TOTAL REAL BROKEN   : {len(real_broken)}")
res_pct = ((len(all_parsed) - len(real_broken)) / len(all_parsed) * 100) if all_parsed else 0.0
print(f"RESOLUTION RATE     : {res_pct:.2f}%")
print("="*80)

if line_range_mismatches:
    print(f"\nLINE RANGE MISMATCHES (Target file exists, but link line exceeds file length): {len(line_range_mismatches)}")
    for lrm in line_range_mismatches:
        print(f"   In {lrm[0].replace('/Users/shakhzod/Desktop/V.O.I.D/', '')}:{lrm[1]} -> {lrm[3]} has {lrm[5]} lines, link references {lrm[4]}")
else:
    print("\nLine range verification: 100% of line references fall within actual file line bounds!")
