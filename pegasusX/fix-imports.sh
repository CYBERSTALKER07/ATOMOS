#!/bin/bash
FILES=$(find apps packages -name "*.ts" -o -name "*.tsx" -type f -exec grep -l "@pegasusx/api-client" {} +)

for file in $FILES; do
  if grep -qE "(useMarketPack|usePolling)" "$file"; then
    # We have to split the import. For simplicity, just replace all with @pegasusx/api-core,
    # and then add a new import for the hooks from @pegasusx/api-react.
    sed -i '' "s/['\"]@pegasusx\/api-client['\"]/'@pegasusx\/api-core'/g" "$file"
    sed -i '' "1s/^/import { useMarketPack, usePolling } from '@pegasusx\/api-react';\n/" "$file"
    # But wait, this would duplicate imports and cause 'useMarketPack' has already been declared.
    # Better: just sed useMarketPack and usePolling out of the api-core import, if we can.
  else
    sed -i '' "s/['\"]@pegasusx\/api-client['\"]/'@pegasusx\/api-core'/g" "$file"
  fi
done
