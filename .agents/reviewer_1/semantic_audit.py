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

samples = []

for doc_path in doc_files:
    with open(doc_path, 'r', encoding='utf-8') as f:
        lines = f.readlines()
    for idx, line in enumerate(lines, 1):
        matches = file_url_pattern.findall(line)
        for m in matches:
            m = m.rstrip(').,;:`\'"')
            raw_full = '/' + m
            matched = re.match(r'^(.*?)(?::(\d+(?:-\d+)?)|#[Ll](\d+(?:-[Ll]?\d+)?))?$', raw_full)
            if matched:
                path = matched.group(1)
                line_spec = matched.group(2) or matched.group(3)
                samples.append({
                    'doc': doc_path.replace('/Users/shakhzod/Desktop/V.O.I.D/', ''),
                    'doc_line': idx,
                    'context': line.strip(),
                    'target_file': path,
                    'line_spec': line_spec
                })

print(f"Total extracted link references: {len(samples)}")

# Take 2 samples from each of the 15 doc files (30 samples total)
selected_samples = []
for doc_path in doc_files:
    short_doc = doc_path.replace('/Users/shakhzod/Desktop/V.O.I.D/', '')
    doc_samples = [s for s in samples if s['doc'] == short_doc]
    if len(doc_samples) >= 2:
        selected_samples.append(doc_samples[0])
        selected_samples.append(doc_samples[len(doc_samples)//2])
    elif doc_samples:
        selected_samples.append(doc_samples[0])

print(f"Conducting deep semantic audit on {len(selected_samples)} representative citations across all 15 files:\n")

for i, s in enumerate(selected_samples, 1):
    print(f"--- [Sample {i:02d}] {s['doc']}:{s['doc_line']} ---")
    print(f"Doc Claim/Context: {s['context'][:120]}...")
    target = s['target_file']
    print(f"Target Path      : {target} (lines {s['line_spec']})")
    
    if os.path.exists(target):
        if os.path.isfile(target):
            with open(target, 'r', encoding='utf-8', errors='ignore') as tf:
                all_t_lines = tf.readlines()
            if s['line_spec']:
                parts = re.split(r'[-L]', s['line_spec'])
                start = max(1, int(parts[0]))
                end = int(parts[-1]) if len(parts) > 1 and parts[-1] else min(start + 4, len(all_t_lines))
                # clamp
                start_idx = max(0, start - 1)
                end_idx = min(len(all_t_lines), end)
                snippet = "".join(all_t_lines[start_idx:min(start_idx+5, end_idx)]).strip()
                print(f"Target Source ({start}-{min(start+4, end)} of {len(all_t_lines)}):")
                print(f"  {snippet[:200].replace(chr(10), ' // ')}")
            else:
                print(f"Target File Exists: {len(all_t_lines)} lines")
        elif os.path.isdir(target):
            print(f"Target is Directory: {len(os.listdir(target))} entries")
    else:
        print("ERROR: Target does not exist!")
    print()
