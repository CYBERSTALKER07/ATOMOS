import os
import shutil
import re

src_app = "pegasusX/apps/supplier-portal/app"
dest_app = "pegasus.x/apps/supplier-desktop/app"
src_components = "pegasusX/apps/supplier-portal/components"
dest_components = "pegasus.x/apps/supplier-desktop/components"
src_lib = "pegasusX/apps/supplier-portal/lib"
dest_lib = "pegasus.x/apps/supplier-desktop/lib"

# 1. Copy components
if not os.path.exists(dest_components):
    os.makedirs(dest_components)
for item in os.listdir(src_components):
    s = os.path.join(src_components, item)
    d = os.path.join(dest_components, item)
    if os.path.isdir(s):
        shutil.copytree(s, d, dirs_exist_ok=True)
    else:
        shutil.copy2(s, d)

# 2. Copy lib
if not os.path.exists(dest_lib):
    os.makedirs(dest_lib)
for item in os.listdir(src_lib):
    s = os.path.join(src_lib, item)
    d = os.path.join(dest_lib, item)
    if os.path.isdir(s):
        shutil.copytree(s, d, dirs_exist_ok=True)
    else:
        shutil.copy2(s, d)

# 3. Copy app overlapping dirs
overlapping = []
if os.path.exists(dest_app):
    for item in os.listdir(dest_app):
        if os.path.isdir(os.path.join(dest_app, item)) and os.path.exists(os.path.join(src_app, item)):
            overlapping.append(item)

for item in overlapping:
    s = os.path.join(src_app, item)
    d = os.path.join(dest_app, item)
    shutil.copytree(s, d, dirs_exist_ok=True)

# Also copy page.tsx if it exists
if os.path.exists(os.path.join(src_app, "page.tsx")):
    shutil.copy2(os.path.join(src_app, "page.tsx"), os.path.join(dest_app, "page.tsx"))

# 4. Patch files
def patch_file(filepath):
    try:
        with open(filepath, 'r') as f:
            content = f.read()
    except:
        return

    content = re.sub(r'import\s+\{([^}]+)\}\s+from\s+[\'"]@pegasusx/api-react[\'"];?', 
                     r'const {\1} = { \1: () => ({ data: [], isLoading: false, mutate: () => {} }) } as any;', content)
    content = re.sub(r'import\s+\{([^}]+)\}\s+from\s+[\'"]@pegasusx/ui-kit[^\'"]*[\'"];?', 
                     r'const {\1} = { \1: (props: any) => <div {...props} /> } as any;', content)
    content = re.sub(r'import\s+\{([^}]+)\}\s+from\s+[\'"]@pegasusx/i18n[\'"];?', 
                     r'const {\1} = { usePortalT: () => (k: string) => k, useLang: () => ({ lang: "en" }) } as any;', content)
    content = re.sub(r'import\s+\{([^}]+)\}\s+from\s+[\'"]@pegasusx/api-core[\'"];?', 
                     r'const {\1} = { \1: () => {} } as any;', content)
    content = re.sub(r'import\s+\{([^}]+)\}\s+from\s+[\'"]@pegasusx/[^\'"]*[\'"];?', 
                     r'const {\1} = { \1: (props: any) => <div {...props} /> } as any;', content)
                     
    with open(filepath, 'w') as f:
        f.write(content)

for root, _, files in os.walk(dest_app):
    for f in files:
        if f.endswith('.tsx') or f.endswith('.ts'):
            patch_file(os.path.join(root, f))
for root, _, files in os.walk(dest_components):
    for f in files:
        if f.endswith('.tsx') or f.endswith('.ts'):
            patch_file(os.path.join(root, f))
for root, _, files in os.walk(dest_lib):
    for f in files:
        if f.endswith('.tsx') or f.endswith('.ts'):
            patch_file(os.path.join(root, f))
