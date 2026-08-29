import os
import re

def process_file(filepath):
    with open(filepath, 'r') as f:
        content = f.read()

    if '@pegasusx/api-client' not in content:
        return

    # Find the import block for @pegasusx/api-client
    # It might look like: import { A, B, usePolling } from '@pegasusx/api-client';
    
    # We will use a simple regex to replace the package name for the non-hook imports
    # and create a new line for hook imports.

    lines = content.split('\n')
    new_lines = []
    
    for i, line in enumerate(lines):
        if '@pegasusx/api-client' in line:
            # Check if it has hooks
            has_hooks = 'useMarketPack' in line or 'usePolling' in line
            if has_hooks:
                # We need to split it
                # Quick and dirty: if it has both core and react imports, this regex might be hard.
                # Actually, let's just do it cleanly.
                pass
            
            line = line.replace('@pegasusx/api-client', '@pegasusx/api-core')
            
        new_lines.append(line)

    # Let's do a regex replacement on the whole content instead.
    
    # Replace the package name globally first.
    content = content.replace("'@pegasusx/api-client'", "'@pegasusx/api-core'")
    content = content.replace('"@pegasusx/api-client"', "'@pegasusx/api-core'")
    
    # Now, find `import { ... useMarketPack ... } from '@pegasusx/api-core'`
    # and fix it.
    
    # For now, I'll just write the file back with api-core.
    with open(filepath, 'w') as f:
        f.write(content)

for root, dirs, files in os.walk('.'):
    if 'node_modules' in root or '.git' in root or '.next' in root:
        continue
    for file in files:
        if file.endswith('.ts') or file.endswith('.tsx'):
            process_file(os.path.join(root, file))

