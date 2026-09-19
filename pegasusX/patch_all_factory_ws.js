const fs = require('fs');
const files = [
  'apps/factory-portal/app/insights/page.tsx',
  'apps/factory-portal/app/payload-override/page.tsx',
  'apps/factory-portal/app/payload/page.tsx',
  'apps/factory-portal/app/transfers/page.tsx',
  'apps/factory-portal/app/manifest-exceptions/page.tsx',
  'apps/factory-portal/app/manifests/page.tsx',
  'apps/factory-portal/app/fleet/page.tsx'
];

files.forEach(f => {
  let code = fs.readFileSync(f, 'utf8');
  
  // For negative conditions (return early)
  code = code.replace(
    /if\s*\(\s*event\.type\s*!==\s*'FACTORY_(TRANSFER_UPDATE|SUPPLY_REQUEST_UPDATE|MANIFEST_UPDATE)'\s*&&\s*event\.type\s*!==\s*'FACTORY_(TRANSFER_UPDATE|SUPPLY_REQUEST_UPDATE|MANIFEST_UPDATE)'\s*\)\s*\{\s*return;\s*\}/g,
    `if (!event.type.startsWith('TRANSFER_') && !event.type.startsWith('MANIFEST_') && !event.type.startsWith('WAREHOUSE_TRANSFER_') && !event.type.startsWith('FACTORY_SUPPLY_')) { return; }`
  );
  
  // For positive conditions (void load)
  code = code.replace(
    /if\s*\(\s*event\?\.type\s*===\s*'FACTORY_(TRANSFER_UPDATE|MANIFEST_UPDATE)'\s*\|\|\s*event\?\.type\s*===\s*'FACTORY_(TRANSFER_UPDATE|MANIFEST_UPDATE)'\s*\)\s*\{/g,
    `if (event?.type.startsWith('TRANSFER_') || event?.type.startsWith('MANIFEST_') || event?.type.startsWith('WAREHOUSE_TRANSFER_')) {`
  );

  fs.writeFileSync(f, code);
});
