const fs = require('fs');

['apps/factory-app-ios/FactoryApp/Views/Dashboard/DashboardView.swift', 'apps/factory-app-ios/FactoryApp/Views/Staff/StaffView.swift'].forEach(f => {
  let code = fs.readFileSync(f, 'utf8');
  
  code = code.replace(
    /guard let eventType = event\.eventType else \{ return \}\n\s*switch eventType \{\n\s*case \.supplyRequestUpdate, \.transferUpdate, \.manifestUpdate:\n\s*Task \{ await load\(silent: true\) \}\n\s*case \.outboxFailed:\n\s*break\n\s*\}/g,
    \`if event.type.hasPrefix("TRANSFER_") || event.type.hasPrefix("MANIFEST_") || event.type.hasPrefix("WAREHOUSE_TRANSFER_") || event.type.hasPrefix("FACTORY_SUPPLY_") { Task { await load(silent: true) } }\`
  );

  code = code.replace(
    /guard let eventType = event\.eventType else \{ return \}\n\s*switch eventType \{\n\s*case \.supplyRequestUpdate, \.transferUpdate, \.manifestUpdate:\n\s*Task \{ await load\(\) \}\n\s*case \.outboxFailed:\n\s*break\n\s*\}/g,
    \`if event.type.hasPrefix("TRANSFER_") || event.type.hasPrefix("MANIFEST_") || event.type.hasPrefix("WAREHOUSE_TRANSFER_") || event.type.hasPrefix("FACTORY_SUPPLY_") { Task { await load() } }\`
  );

  fs.writeFileSync(f, code);
});
