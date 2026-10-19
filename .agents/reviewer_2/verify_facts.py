import os
import re
import glob

def check_pegasus():
    print("=== PEGASUS FACTS VERIFICATION ===")
    root = "/Users/shakhzod/Desktop/V.O.I.D/pegasus"
    
    # apps
    apps = [d for d in os.listdir(f"{root}/apps") if os.path.isdir(f"{root}/apps/{d}")]
    print(f"Apps count ({len(apps)}): {sorted(apps)}")
    
    # services
    services = [d for d in os.listdir(f"{root}/services") if os.path.isdir(f"{root}/services/{d}")]
    print(f"Services count ({len(services)}): {sorted(services)}")

    # packages
    packages = [d for d in os.listdir(f"{root}/packages") if os.path.isdir(f"{root}/packages/{d}")]
    print(f"Packages count ({len(packages)}): {sorted(packages)}")

    # Spanner tables
    schema_files = glob.glob(f"{root}/**/*.sql", recursive=True) + glob.glob(f"{root}/**/schema*.ddl", recursive=True)
    spanner_tables = set()
    for sf in schema_files:
        if "schema" in sf or "migration" in sf or "spanner" in sf:
            try:
                with open(sf, 'r', errors='ignore') as f:
                    content = f.read()
                    matches = re.findall(r'CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z0-9_]+)', content, re.I)
                    spanner_tables.update(matches)
            except Exception:
                pass
    print(f"Unique CREATE TABLE found in schemas: {len(spanner_tables)}")

def check_pegasusx():
    print("\n=== PEGASUSX FACTS VERIFICATION ===")
    root = "/Users/shakhzod/Desktop/V.O.I.D/pegasusX"
    
    # apps
    apps = [d for d in os.listdir(f"{root}/apps") if os.path.isdir(f"{root}/apps/{d}")]
    print(f"Apps count ({len(apps)}): {sorted(apps)}")

    # migrations
    mig_dir = f"{root}/database/migrations"
    if os.path.exists(mig_dir):
        migs = [f for f in os.listdir(mig_dir) if f.endswith('.sql')]
        print(f"Migrations count in {mig_dir}: {len(migs)}")
    
    # Spanner tables
    schema_files = glob.glob(f"{root}/database/migrations/*.sql") + glob.glob(f"{root}/**/schema.sql", recursive=True)
    tables = set()
    for sf in schema_files:
        try:
            with open(sf, 'r', errors='ignore') as f:
                matches = re.findall(r'CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z0-9_]+)', f.read(), re.I)
                tables.update(matches)
        except Exception:
            pass
    print(f"Unique CREATE TABLE in PegasusX migrations/schemas: {len(tables)}")

    # Route modules / endpoints in backend-go
    bg_root = f"{root}/apps/backend-go"
    endpoints = []
    if os.path.exists(bg_root):
        for r, d, files in os.walk(bg_root):
            for file in files:
                if file.endswith('.go'):
                    fp = os.path.join(r, file)
                    with open(fp, 'r', errors='ignore') as f:
                        for line in f:
                            if re.search(r'\.(Get|Post|Put|Delete|Patch|Handle|HandleFunc)\s*\(', line):
                                endpoints.append(line.strip())
    print(f"HTTP handler registrations found in backend-go: {len(endpoints)}")

def check_pegasus_dot_x():
    print("\n=== PEGASUS.X FACTS VERIFICATION ===")
    root = "/Users/shakhzod/Desktop/V.O.I.D/pegasus.x"

    # apps
    apps = [d for d in os.listdir(f"{root}/apps") if os.path.isdir(f"{root}/apps/{d}")]
    print(f"Apps count ({len(apps)}): {sorted(apps)}")

    # migrations
    mig_dir = f"{root}/database/migrations"
    if os.path.exists(mig_dir):
        migs = [f for f in os.listdir(mig_dir) if f.endswith('.sql')]
        print(f"Migrations count in {mig_dir}: {len(migs)}")

    # Go version
    go_mod = f"{root}/go.work"
    if os.path.exists(go_mod):
        with open(go_mod, 'r') as f:
            print("go.work first 5 lines:", [next(f) for _ in range(5)])

    # Living loop steps in smokecheck
    smoke = f"{root}/backend/cmd/smokecheck/main.go"
    if os.path.exists(smoke):
        with open(smoke, 'r', errors='ignore') as f:
            smoke_content = f.read()
        steps = re.findall(r'Step\s+(\d+)|STEP\s+(\d+)|step(\d+)', smoke_content, re.I)
        print(f"Smokecheck file size: {len(smoke_content)} chars, matches for step: {len(steps)}")
        # Look for the max step number
        step_nums = []
        for m in re.finditer(r'//\s*[-=]+\s*Step\s+(\d+)|func\s+step(\d+)|Step\s+(\d+)\b', smoke_content, re.I):
            for g in m.groups():
                if g and g.isdigit():
                    step_nums.append(int(g))
        if step_nums:
            print(f"Max step number found in smokecheck: {max(step_nums)}, distinct steps: {len(set(step_nums))}")

if __name__ == "__main__":
    check_pegasus()
    check_pegasusx()
    check_pegasus_dot_x()
