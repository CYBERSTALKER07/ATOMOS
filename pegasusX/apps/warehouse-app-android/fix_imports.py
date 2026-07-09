import os
import re

app_dir = '/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/warehouse-app-android/app/src/main/java/com/pegasusx/warehouse/ui/screens/'

for root, _, files in os.walk(app_dir):
    for file in files:
        if file.endswith('.kt'):
            filepath = os.path.join(root, file)
            with open(filepath, 'r') as f:
                content = f.read()

            changed = False
            
            if '340.dp' in content and 'import androidx.compose.ui.unit.dp' not in content:
                content = content.replace('import androidx.compose.ui.Modifier', 'import androidx.compose.ui.Modifier\nimport androidx.compose.ui.unit.dp')
                changed = True
            
            if 'PaddingValues(' in content and 'import androidx.compose.foundation.layout.PaddingValues' not in content:
                content = content.replace('import androidx.compose.foundation.layout.Arrangement', 'import androidx.compose.foundation.layout.Arrangement\nimport androidx.compose.foundation.layout.PaddingValues')
                changed = True

            if file == 'MoreHubScreen.kt':
                # Re-add LazyColumn and standard items for MoreHubScreen because it might be using them.
                if 'import androidx.compose.foundation.lazy.LazyColumn' not in content:
                    content = content.replace('import androidx.compose.foundation.layout.fillMaxSize', 'import androidx.compose.foundation.layout.fillMaxSize\nimport androidx.compose.foundation.lazy.LazyColumn\nimport androidx.compose.foundation.lazy.items')
                    changed = True
            
            if changed:
                with open(filepath, 'w') as f:
                    f.write(content)
                    print(f"Fixed imports in {filepath}")

