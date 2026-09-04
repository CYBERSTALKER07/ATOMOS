import os
import re

def process_file(filepath):
    with open(filepath, 'r') as f:
        content = f.read()

    if 'useMarketPack' not in content and 'usePolling' not in content:
        return

    # A simple hack: just add the import at the top of the file. 
    # Yes, we might have it imported from api-core too, but TS will complain if it's imported twice.
    # So we remove useMarketPack and usePolling from @pegasusx/api-core imports.
    
    new_content = re.sub(r'useMarketPack,\s*', '', content)
    new_content = re.sub(r',\s*useMarketPack', '', new_content)
    new_content = re.sub(r'useMarketPack\s*', '', new_content)
    
    new_content = re.sub(r'usePolling,\s*', '', new_content)
    new_content = re.sub(r',\s*usePolling', '', new_content)
    new_content = re.sub(r'usePolling\s*', '', new_content)
    
    # If an import is now empty `import { } from '@pegasusx/api-core';` we can leave it, TS / linters will ignore or remove it.
    
    # Add the new import at the very top, after 'use client' if it exists.
    import_statement = "import { useMarketPack, usePolling } from '@pegasusx/api-react';\n"
    
    if 'use client' in new_content:
        new_content = new_content.replace("'use client';", "'use client';\n" + import_statement)
        new_content = new_content.replace('"use client";', '"use client";\n' + import_statement)
    else:
        new_content = import_statement + new_content
        
    with open(filepath, 'w') as f:
        f.write(new_content)
    print(f"Fixed hooks in {filepath}")

for root, dirs, files in os.walk('.'):
    if 'node_modules' in root or '.git' in root or '.next' in root or 'packages/api-react' in root:
        continue
    for file in files:
        if file.endswith('.ts') or file.endswith('.tsx'):
            process_file(os.path.join(root, file))
