const fs = require('fs');
let code = fs.readFileSync('apps/factory-portal/app/loading-bay/page.tsx', 'utf8');

code = code.replace(
  /if \(event\.type !== 'FACTORY_TRANSFER_UPDATE' && event\.type !== 'FACTORY_MANIFEST_UPDATE'\) {\n\s*return;\n\s*}/s,
  `if (!event.type.startsWith('TRANSFER_') && !event.type.startsWith('MANIFEST_') && !event.type.startsWith('WAREHOUSE_TRANSFER_')) {
          return;
        }`
);

fs.writeFileSync('apps/factory-portal/app/loading-bay/page.tsx', code);
