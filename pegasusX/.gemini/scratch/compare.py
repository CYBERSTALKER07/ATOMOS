import re
from collections import defaultdict

roles = ['retailer', 'supplier', 'warehouse', 'factory', 'driver']

# Read the big txt files
data = defaultdict(lambda: defaultdict(set))
for role in roles:
    try:
        with open(f'.gemini/scratch/{role}.txt', 'r') as f:
            lines = f.read().splitlines()
        current_platform = None
        for line in lines:
            line = line.strip()
            if not line: continue
            m = re.match(r'=== .* (Android|iOS|Desktop) ===', line)
            if m:
                current_platform = m.group(1).lower()
            else:
                # normalize name (remove Screen.kt, View.swift, etc)
                name = re.sub(r'(Screen\.kt|View\.swift)$', '', line)
                if current_platform:
                    data[role][current_platform].add(name)
    except FileNotFoundError:
        pass

# Desktop pages
desktop_files = {
    'retailer': '.gemini/scratch/retailer_desktop.txt',
    'supplier': '.gemini/scratch/supplier_portal.txt',
    'warehouse': '.gemini/scratch/warehouse_portal.txt',
    'factory': '.gemini/scratch/factory_portal.txt'
}

for r, f in desktop_files.items():
    try:
        with open(f, 'r') as file:
            lines = file.read().splitlines()
            for line in lines:
                name = line.strip()
                if name:
                    # try to capitalize and match native names where possible or just keep it
                    # converting kebab-case to PascalCase
                    parts = name.replace('[id]', 'Detail').replace('[productId]', 'ProductDetail').split('-')
                    pascal = "".join(p.capitalize() for p in parts)
                    data[r]['desktop'].add(pascal)
    except FileNotFoundError:
        pass

# Now generate a report
with open('.gemini/scratch/report.md', 'w') as out:
    for role in roles:
        out.write(f"## {role.upper()}\n\n")
        platforms = list(data[role].keys())
        if not platforms:
            out.write("No data.\n\n")
            continue
        
        all_features = set()
        for p in platforms:
            all_features.update(data[role][p])
        
        # Build table
        header = "| Feature | " + " | ".join(platforms) + " |"
        out.write(header + "\n")
        separator = "|---|" + "|".join(["---" for _ in platforms]) + "|"
        out.write(separator + "\n")
        
        for f in sorted(list(all_features)):
            row = f"| {f} |"
            for p in platforms:
                if f in data[role][p]:
                    row += " ✅ |"
                else:
                    # Try to find a fuzzy match
                    found = False
                    for pf in data[role][p]:
                        if f.lower() in pf.lower() or pf.lower() in f.lower():
                            found = True
                            break
                    if found:
                        row += " ⚠️ (Fuzzy) |"
                    else:
                        row += " ❌ |"
            out.write(row + "\n")
        out.write("\n")
