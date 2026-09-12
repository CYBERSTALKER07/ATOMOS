const fs = require('fs');

const files = process.argv.slice(2);

files.forEach(f => {
  let code = fs.readFileSync(f, 'utf8');
  let changed = false;

  code = code.replace(
    /guard let eventType = event\.eventType else \{ return \}\n\s*guard eventType == \.transferUpdate \|\| eventType == \.manifestUpdate else \{ return \}/g,
    \`guard event.type.hasPrefix("TRANSFER_") || event.type.hasPrefix("MANIFEST_") || event.type.hasPrefix("WAREHOUSE_TRANSFER_") else { return }\`
  );
  
  code = code.replace(
    /guard let eventType = event\.eventType else \{ return \}\n\s*guard eventType == \.manifestUpdate \|\| eventType == \.transferUpdate else \{ return \}/g,
    \`guard event.type.hasPrefix("TRANSFER_") || event.type.hasPrefix("MANIFEST_") || event.type.hasPrefix("WAREHOUSE_TRANSFER_") else { return }\`
  );
  
  code = code.replace(
    /guard let eventType = event\.eventType else \{ return \}\n\s*if eventType == \.manifestUpdate \{/g,
    \`if event.type.hasPrefix("MANIFEST_") {\`
  );
  
  code = code.replace(
    /guard let eventType = event\.eventType else \{ return \}\n\s*guard eventType == \.supplyRequestUpdate \|\| eventType == \.transferUpdate else \{ return \}/g,
    \`guard event.type.hasPrefix("TRANSFER_") || event.type.hasPrefix("WAREHOUSE_TRANSFER_") || event.type.hasPrefix("FACTORY_SUPPLY_") else { return }\`
  );
  
  code = code.replace(
    /guard let eventType = event\.eventType else \{ return \}\n\s*guard DashboardRollup\.shouldRefetch\(eventType\) else \{ return \}/g,
    \`guard DashboardRollup.shouldRefetch(event.type) else { return }\`
  );
  
  code = code.replace(
    /guard let eventType = event\.eventType else \{ return \}\n\s*Task \{ await load\(\) \}/g,
    \`Task { await load() }\`
  );

  fs.writeFileSync(f, code);
});
