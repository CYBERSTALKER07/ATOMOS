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

md_link_pattern = re.compile(r'\[([^\]]+)\]\(([^)]+)\)')

non_file_links = []
relative_links = []

for doc_path in doc_files:
    dir_path = os.path.dirname(doc_path)
    with open(doc_path, 'r', encoding='utf-8') as f:
        lines = f.readlines()
    for line_idx, line in enumerate(lines, start=1):
        for text, url in md_link_pattern.findall(line):
            if url.startswith('file:///'):
                continue
            elif url.startswith('http://') or url.startswith('https://'):
                non_file_links.append((doc_path, line_idx, text, url, 'http'))
            elif url.startswith('#'):
                # anchor link
                non_file_links.append((doc_path, line_idx, text, url, 'anchor'))
            else:
                # relative link
                target = os.path.normpath(os.path.join(dir_path, url.split('#')[0]))
                exists = os.path.exists(target)
                relative_links.append((doc_path, line_idx, text, url, target, exists))

print(f"Total non-file:/// markdown links: {len(non_file_links) + len(relative_links)}")
print(f"Anchor links: {len([x for x in non_file_links if x[4] == 'anchor'])}")
print(f"HTTP links: {len([x for x in non_file_links if x[4] == 'http'])}")
print(f"Relative links: {len(relative_links)}")

broken_rel = [r for r in relative_links if not r[5]]
print(f"Broken relative links: {len(broken_rel)}")
for br in broken_rel:
    print(f"  Broken rel link in {br[0]}:{br[1]} -> text: {br[2]}, url: {br[3]}, target: {br[4]}")
