const fs = require('fs');
const files = [
  'apps/factory-portal/app/manifest-exceptions/page.tsx',
  'apps/factory-portal/app/manifests/page.tsx'
];

files.forEach(f => {
  let code = fs.readFileSync(f, 'utf8');
  
  code = code.replace(
    /if\s*\(\s*event\.type\s*===\s*'FACTORY_(TRANSFER_UPDATE|MANIFEST_UPDATE)'\s*\|\|\s*event\.type\s*===\s*'FACTORY_(TRANSFER_UPDATE|MANIFEST_UPDATE)'\s*\)\s*\{/g,
    `if (event.type.startsWith('TRANSFER_') || event.type.startsWith('MANIFEST_') || event.type.startsWith('WAREHOUSE_TRANSFER_')) {`
  );

  fs.writeFileSync(f, code);
});
