const fs = require('fs');
const path = require('path');

function getFiles(dir, fileList = []) {
  const files = fs.readdirSync(dir);
  for (const file of files) {
    const filePath = path.join(dir, file);
    if (fs.statSync(filePath).isDirectory()) {
      getFiles(filePath, fileList);
    } else if (filePath.endsWith('page.tsx')) {
      fileList.push(filePath);
    }
  }
  return fileList;
}

const allPages = getFiles('/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/supplier-portal/app');

let count = 0;
for (const file of allPages) {
  let content = fs.readFileSync(file, 'utf8');
  if (content.includes("useSupplierSessionReconcile")) continue;

  // Add import if needed
  if (!content.includes('import { useSupplierSessionReconcile }')) {
    content = content.replace(/(import .*?;?\n)/, `$1import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";\n`);
  }

  // Look for a `load` function
  if (content.includes('const load = useCallback(') || content.includes('function load()')) {
    content = content.replace(/(\n  useEffect\()/g, `\n  useSupplierSessionReconcile(load);\n\n  useEffect(`);
    fs.writeFileSync(file, content);
    console.log(`Patched ${file} (used load)`);
    count++;
  } else if (content.includes('useEffect(() => {')) {
    // If there is a useEffect with `api.` call, we add refreshTick.
    if (content.includes('api.')) {
      if (!content.includes('const [refreshTick')) {
        content = content.replace(/(\n  const \[loading, setLoading\] = useState.*?;\n)/, `$1  const [refreshTick, setRefreshTick] = useState(0);\n  useSupplierSessionReconcile(() => setRefreshTick(t => t + 1));\n`);
      }
      
      // Update dependency array for the useEffect containing `api.`
      // This is a bit hacky, let's find the useEffect that sets cancelled = false;
      content = content.replace(/(\n  useEffect\(\(\) => \{\n\s*let cancelled = false;[\s\S]*?\}\, \[)(.*?)(\]\);)/g, (match, p1, p2, p3) => {
        if (p2.trim() === '') return `${p1}refreshTick${p3}`;
        if (!p2.includes('refreshTick')) return `${p1}${p2}, refreshTick${p3}`;
        return match;
      });

      fs.writeFileSync(file, content);
      console.log(`Patched ${file} (used refreshTick)`);
      count++;
    }
  }
}
console.log(`Patched ${count} files.`);
