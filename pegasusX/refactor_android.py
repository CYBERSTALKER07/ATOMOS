import os
import re

base_path = '/Users/shakhzod/ATOMOS/pegasusX/apps/supplier-app-android/app/src/main/java/com/pegasusx/supplier/ui/screens/orgfleet'
components_path = os.path.join(base_path, 'components')

if not os.path.exists(components_path):
    os.makedirs(components_path)

with open(os.path.join(base_path, 'OrgFleetScreen.kt'), 'r') as f:
    content = f.read()

# Common imports for the extracted files
common_imports = """package com.pegasusx.supplier.ui.screens.orgfleet.components

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.*
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing

"""

# Extract Rosters
def extract_composable(name):
    pattern = r"@Composable\s+private fun " + name + r"\(.*?\)\s*\{.*?^\}"
    match = re.search(pattern, content, re.MULTILINE | re.DOTALL)
    if match:
        return match.group(0).replace("private fun", "fun")
    else:
        # try without private
        pattern = r"@Composable\s+fun " + name + r"\(.*?\)\s*\{.*?^\}"
        match = re.search(pattern, content, re.MULTILINE | re.DOTALL)
        if match:
            return match.group(0)
    return ""

def extract_optin_composable(name):
    pattern = r"@OptIn.*?@Composable\s+private fun " + name + r"\(.*?\)\s*\{.*?^\}"
    match = re.search(pattern, content, re.MULTILINE | re.DOTALL)
    if match:
        return match.group(0).replace("private fun", "fun")
    return ""

rosters_code = common_imports + extract_composable("DriverRoster") + "\n\n" + \
               extract_composable("VehicleRoster") + "\n\n" + \
               extract_composable("OrgRoster")
               
with open(os.path.join(components_path, 'OrgFleetRosters.kt'), 'w') as f:
    f.write(rosters_code)

# Extract Utils
utils_pattern = r"private fun nodeLabel\(.*?\): String\s*\{.*?^\}"
match = re.search(utils_pattern, content, re.MULTILINE | re.DOTALL)
utils_code = """package com.pegasusx.supplier.ui.screens.orgfleet.components\n\nimport com.pegasusx.supplier.data.model.SupplierTopologyResponse\n\n"""
if match:
    utils_code += match.group(0).replace("private fun", "fun")
with open(os.path.join(components_path, 'OrgFleetUtils.kt'), 'w') as f:
    f.write(utils_code)

# Extract Dialogs
dialogs_code = common_imports + extract_optin_composable("CreateDriverDialog") + "\n\n" + \
               extract_composable("CreateVehicleDialog") + "\n\n" + \
               extract_composable("CreateOrgMemberDialog") + "\n\n" + \
               extract_composable("EditOrgMemberDialog")
with open(os.path.join(components_path, 'OrgFleetDialogs.kt'), 'w') as f:
    f.write(dialogs_code)

# Extract Pickers
pickers_code = common_imports + extract_optin_composable("NodeTypePicker") + "\n\n" + \
               extract_optin_composable("NodePicker") + "\n\n" + \
               extract_optin_composable("VehiclePicker") + "\n\n" + \
               extract_optin_composable("RolePicker")
with open(os.path.join(components_path, 'OrgFleetPickers.kt'), 'w') as f:
    f.write(pickers_code)

# Rewrite OrgFleetScreen.kt to keep only the main screen and import the components
main_screen_pattern = r"^package.*?@Composable\s+fun OrgFleetScreen\(.*?\)\s*\{.*?^\}"
match = re.search(main_screen_pattern, content, re.MULTILINE | re.DOTALL)
if match:
    new_content = match.group(0)
    # Add import for components
    import_statement = "import com.pegasusx.supplier.ui.screens.orgfleet.components.*\n"
    new_content = new_content.replace("import com.pegasus.design.PegasusStatePane\n", 
                                      "import com.pegasus.design.PegasusStatePane\n" + import_statement)
    with open(os.path.join(base_path, 'OrgFleetScreen.kt'), 'w') as f:
        f.write(new_content)
        
print("Android Components created and OrgFleetScreen refactored!")
