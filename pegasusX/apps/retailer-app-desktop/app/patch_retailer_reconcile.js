const fs = require('fs');

const files = [
  "app/(dashboard)/insights/page.tsx",
  "app/(dashboard)/settings/page.tsx",
  "app/(dashboard)/dashboard/page.tsx",
  "app/(dashboard)/procurement/page.tsx",
  "app/(dashboard)/tracking/page.tsx",
  "app/(dashboard)/orders/page.tsx",
  "app/(dashboard)/dock/page.tsx"
];

const basePath = "/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/retailer-app-desktop";

for (const file of files) {
  const filePath = `${basePath}/${file}`;
  if (!fs.existsSync(filePath)) continue;
  let content = fs.readFileSync(filePath, 'utf8');
  
  if (content.includes("useRetailerSessionReconcile")) continue;

  const importStatement = `import { useRetailerSessionReconcile } from "../../../lib/use-retailer-session-reconcile";\n`;
  content = content.replace(/(import .*?;?\n)/, `$1${importStatement}`);

  // Find the mutate statements from useLiveData
  const match = content.match(/mutate: (\w+)/g);
  if (match) {
    const mutates = match.map(m => m.split(': ')[1]);
    const hookCall = `  useRetailerSessionReconcile(() => {\n${mutates.map(m => `    void ${m}();`).join('\n')}\n  });\n`;
    
    // Inject hookCall right after the first `const [activeTab, setActiveTab]` or `const router` or something similar inside the component.
    // Or, just before the `return` statement.
    content = content.replace(/(\n  return \()/g, `\n${hookCall}$1`);
    
    fs.writeFileSync(filePath, content);
    console.log(`Patched ${file} with mutates: ${mutates.join(', ')}`);
  }
}
