import os
import re

def scan_files(root_dir, banned_patterns):
    total_go_files = 0
    violations = []
    # Pattern to match import "..." or import ( ... )
    import_single = re.compile(r'^\s*import\s+"([^"]+)"')
    import_block_start = re.compile(r'^\s*import\s*\(')
    import_block_end = re.compile(r'^\s*\)')
    import_line = re.compile(r'^\s*(?:[a-zA-Z0-9_]+\s+)?"([^"]+)"')

    for dirpath, dirnames, filenames in os.walk(root_dir):
        # skip git, node_modules, vendor
        dirnames[:] = [d for d in dirnames if d not in ('.git', 'node_modules', 'vendor', '__pycache__')]
        for f in filenames:
            if f.endswith('.go'):
                total_go_files += 1
                filepath = os.path.join(dirpath, f)
                with open(filepath, 'r', encoding='utf-8', errors='ignore') as fp:
                    in_block = False
                    for line_no, line in enumerate(fp, 1):
                        single_match = import_single.match(line)
                        if single_match:
                            imp = single_match.group(1)
                            for pat in banned_patterns:
                                if pat.lower() in imp.lower():
                                    violations.append((filepath, line_no, imp, pat))
                            continue
                        if import_block_start.match(line):
                            in_block = True
                            continue
                        if in_block:
                            if import_block_end.match(line):
                                in_block = False
                                continue
                            line_match = import_line.match(line)
                            if line_match:
                                imp = line_match.group(1)
                                for pat in banned_patterns:
                                    if pat.lower() in imp.lower():
                                        violations.append((filepath, line_no, imp, pat))

    return total_go_files, violations

if __name__ == '__main__':
    px_files, px_viol = scan_files('pegasus.x', ['spanner', 'kafka'])
    print(f"pegasus.x: Total Go files: {px_files}, Violations: {len(px_viol)}")
    for v in px_viol:
        print(f"  VIOLATION: {v[0]}:{v[1]} imports '{v[2]}' (matches banned '{v[3]}')")

    pxc_files, pxc_viol = scan_files('pegasusX/apps/backend-go', ['jackc/pgx', 'lib/pq', 'jmoiron/sqlx'])
    print(f"pegasusX/apps/backend-go: Total Go files: {pxc_files}, Violations: {len(pxc_viol)}")
    for v in pxc_viol:
        print(f"  VIOLATION: {v[0]}:{v[1]} imports '{v[2]}' (matches banned '{v[3]}')")
