import glob
import os
import re

files = [
    "./apps/factory-app-android/app/src/main/java/com/pegasusx/factory/service/AutoUpdater.kt",
    "./apps/driver-app-android/app/src/main/java/com/pegasusx/driver/service/AutoUpdater.kt",
    "./apps/warehouse-app-android/app/src/main/java/com/pegasusx/warehouse/service/AutoUpdater.kt",
    "./apps/retailer-app-android/app/src/main/java/com/pegasusx/retailer/updater/AutoUpdater.kt",
    "./apps/retailer-app-android/app/src/main/java/com/pegasusx/retailer/service/AutoUpdater.kt",
    "./apps/supplier-app-android/app/src/main/java/com/pegasusx/supplier/service/AutoUpdater.kt",
    "./apps/payload-app-android/app/src/main/java/com/pegasus/payload/service/AutoUpdater.kt"
]

for filepath in files:
    if not os.path.exists(filepath):
        print(f"Not found: {filepath}")
        continue
        
    with open(filepath, 'r') as f:
        content = f.read()

    # 1. Add CoroutineScope and launch imports if not there
    if "import kotlinx.coroutines.CoroutineScope" not in content:
        content = content.replace("import kotlinx.coroutines.Dispatchers", "import kotlinx.coroutines.CoroutineScope\nimport kotlinx.coroutines.Dispatchers\nimport kotlinx.coroutines.launch")

    # 2. Wrap verifyAndInstall in CoroutineScope
    old_verify = 'verifyAndInstall(id, prefs.getString("expected_hash", "") ?: "")'
    new_verify = 'CoroutineScope(Dispatchers.IO).launch {\n                verifyAndInstall(id, prefs.getString("expected_hash", "") ?: "")\n            }'
    content = content.replace(old_verify, new_verify)

    # 3. Fix openStoreListing thread
    old_store = 'if (!EnterpriseUpdateConfig.enableCdnOta) {\n            openStoreListing()\n            return@withContext\n        }'
    new_store = 'if (!EnterpriseUpdateConfig.enableCdnOta) {\n            withContext(Dispatchers.Main) {\n                openStoreListing()\n            }\n            return@withContext\n        }'
    content = content.replace(old_store, new_store)

    # 4. Fix verifyHash security bypass
    old_hash = 'if (expectedHash.isBlank()) return true'
    new_hash = 'if (expectedHash.isBlank()) return false'
    content = content.replace(old_hash, new_hash)

    with open(filepath, 'w') as f:
        f.write(content)
        
    print(f"Patched {filepath}")

