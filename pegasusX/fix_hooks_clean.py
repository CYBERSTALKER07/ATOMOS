import os
import re

files = [
    "./apps/warehouse-portal/app/page.tsx",
    "./apps/warehouse-portal/components/SessionPackChip.tsx",
    "./apps/warehouse-portal/lib/use-payment-catalog.ts",
    "./apps/warehouse-portal/lib/use-warehouse-fleet-live-map.ts",
    "./apps/warehouse-portal/lib/__tests__/freshness.test.ts",
    "./apps/retailer-app-desktop/components/SessionPackChip.tsx",
    "./apps/supplier-portal/app/(portal)/dispatch/use-dispatch-data.ts",
    "./apps/supplier-portal/app/(portal)/dashboard/use-dashboard-data.ts",
    "./apps/supplier-portal/app/(portal)/ops/map/page.tsx",
    "./apps/supplier-portal/components/SessionPackChip.tsx",
    "./apps/supplier-portal/lib/use-fleet-live-map.ts",
    "./apps/supplier-portal/lib/use-payment-catalog.ts",
    "./apps/factory-portal/app/page.tsx",
    "./apps/factory-portal/app/supply-requests/page.tsx",
    "./apps/factory-portal/app/payload-override/page.tsx",
    "./apps/factory-portal/app/manifest-exceptions/page.tsx",
    "./apps/factory-portal/components/SessionPackChip.tsx",
    "./apps/factory-portal/lib/use-factory-fleet-live-map.ts",
    "./apps/factory-portal/lib/__tests__/fleet-freshness.test.ts",
    "./packages/api-core/conditional-get.ts"
]

for filepath in files:
    if not os.path.exists(filepath):
        continue
    with open(filepath, 'r') as f:
        content = f.read()

    # Step 1: Change api-client to api-core globally
    content = content.replace("'@pegasusx/api-client'", "'@pegasusx/api-core'")
    content = content.replace('"@pegasusx/api-client"', "'@pegasusx/api-core'")

    # Step 2: Separate out the hook imports.
    lines = content.split('\n')
    new_lines = []
    has_useMarketPack = False
    has_usePolling = False
    
    for line in lines:
        if 'import' in line and '@pegasusx/api-core' in line:
            if 'useMarketPack' in line:
                has_useMarketPack = True
                line = re.sub(r'useMarketPack,\s*', '', line)
                line = re.sub(r',\s*useMarketPack', '', line)
                line = re.sub(r'useMarketPack\s*', '', line)
            if 'usePolling' in line:
                has_usePolling = True
                line = re.sub(r'usePolling,\s*', '', line)
                line = re.sub(r',\s*usePolling', '', line)
                line = re.sub(r'usePolling\s*', '', line)
            
            # If the import braces are now empty e.g. import { } from '@pegasusx/api-core';
            if re.search(r'import\s*{\s*}\s*from', line):
                continue
                
        new_lines.append(line)
        
    hooks_to_import = []
    if has_useMarketPack: hooks_to_import.append('useMarketPack')
    if has_usePolling: hooks_to_import.append('usePolling')
    
    final_content = '\n'.join(new_lines)
    
    if hooks_to_import:
        import_stmt = f"import {{ {', '.join(hooks_to_import)} }} from '@pegasusx/api-react';\n"
        if "'use client';" in final_content:
            final_content = final_content.replace("'use client';", "'use client';\n" + import_stmt)
        elif '"use client";' in final_content:
            final_content = final_content.replace('"use client";', '"use client";\n' + import_stmt)
        else:
            final_content = import_stmt + final_content
            
    with open(filepath, 'w') as f:
        f.write(final_content)
    print(f"Fixed correctly {filepath}")

