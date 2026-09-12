const fs = require('fs');
let code = fs.readFileSync('apps/factory-portal/lib/auth.ts', 'utf8');

code = code.replace(
  /function isFactoryEventType.*?\n.*?}/s,
  `function isFactoryEventType(value: string): boolean {
  return value.startsWith('MANIFEST_') || 
         value.startsWith('WAREHOUSE_TRANSFER_') || 
         value.startsWith('SUPPLY_TRANSFER_') || 
         value.startsWith('FACTORY_') || 
         value.startsWith('TRANSFER_');
}`
);

code = code.replace(
  /export interface FactoryLiveEvent \{\n  type: FactoryLiveEventType;\n  \[key: string\]: unknown;\n\}/s,
  `export interface FactoryLiveEvent {
  type: string;
  [key: string]: unknown;
}`
);

fs.writeFileSync('apps/factory-portal/lib/auth.ts', code);
