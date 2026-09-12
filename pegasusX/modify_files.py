import os

# Desktop
desktop_path = "/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/retailer-app-desktop/app/(dashboard)/settings/page.tsx"
with open(desktop_path, 'r') as f:
    d_lines = f.readlines()

new_d_lines = []
skip = False
for i, line in enumerate(d_lines):
    if line.strip() == "/* ── Toggle Switch ── */":
        skip = True
    if line.strip() == "/* ── Main Page ── */":
        skip = False
    if line.startswith("function ProfileField({"):
        skip = True
        
    if not skip:
        new_d_lines.append(line)

last_import = 0
for i, line in enumerate(new_d_lines):
    if line.startswith("import "):
        last_import = i

new_d_lines.insert(last_import + 1, 'import { Toggle, OverrideRow, OverrideSection } from "../../../components/settings/OverrideComponents";\n')
new_d_lines.insert(last_import + 2, 'import { ProfileField, ProfileTimeField } from "../../../components/settings/ProfileFields";\n')

with open(desktop_path, 'w') as f:
    f.writelines(new_d_lines)

# Android
android_path = "/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/retailer-app-android/app/src/main/java/com/pegasusx/retailer/ui/screens/profile/ProfileScreen.kt"
with open(android_path, 'r') as f:
    a_lines = f.readlines()

new_a_lines = []
skip = False
for i, line in enumerate(a_lines):
    if line.startswith("@Composable"):
        if i + 1 < len(a_lines) and "fun ProfileHeaderCard" in a_lines[i+1]:
            skip = True
    if not skip:
        new_a_lines.append(line)

last_import = 0
for i, line in enumerate(new_a_lines):
    if line.startswith("import "):
        last_import = i

imports = [
    "import com.pegasusx.retailer.ui.screens.profile.components.ProfileHeaderCard\n",
    "import com.pegasusx.retailer.ui.screens.profile.components.StatsRow\n",
    "import com.pegasusx.retailer.ui.screens.profile.components.EmpathyEngineCard\n",
    "import com.pegasusx.retailer.ui.screens.profile.components.SettingsSection\n"
]
for imp in reversed(imports):
    new_a_lines.insert(last_import + 1, imp)

with open(android_path, 'w') as f:
    f.writelines(new_a_lines)

# iOS
ios_path = "/Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/retailer-app-ios/retailerapp/retailerapp/Screens/ProfileView.swift"
with open(ios_path, 'r') as f:
    i_lines = f.readlines()

new_i_lines = []
skip = False
for i, line in enumerate(i_lines):
    # Replace the body calls
    if "userCard.slideIn(delay: 0)" in line:
        line = line.replace("userCard.slideIn(delay: 0)", "UserCard(displayName: displayName, displayCompany: displayCompany, userEmail: user.email, profilePhone: profilePhone).slideIn(delay: 0)")
    elif "statsRow.slideIn(delay: 0.05)" in line:
        line = line.replace("statsRow.slideIn(delay: 0.05)", "StatsRowView(orderCount: orderCount, totalSpent: totalSpent, totalSpentCurrency: totalSpentCurrency).slideIn(delay: 0.05)")
    elif "orderHistoryLink.slideIn(delay: 0.07)" in line:
        line = line.replace("orderHistoryLink.slideIn(delay: 0.07)", "OrderHistoryLink(orderCount: orderCount).slideIn(delay: 0.07)")
    elif "empathyEngineSection.slideIn(delay: 0.09)" in line:
        line = line.replace("empathyEngineSection.slideIn(delay: 0.09)", "EmpathyEngineSection(globalAutoOrder: $globalAutoOrder, showHistoryAlert: $showHistoryAlert, toggleAction: { enabled, useHistory in await toggleGlobalAutoOrder(enabled: enabled, useHistory: useHistory) }).slideIn(delay: 0.09)")
    elif "preferencesSection.slideIn(delay: 0.15)" in line:
        line = line.replace("preferencesSection.slideIn(delay: 0.15)", "PreferencesSection(aiAutoOrder: $aiAutoOrder, notificationsEnabled: $notificationsEnabled).slideIn(delay: 0.15)")
    elif 'settingsSection("Company"' in line:
        line = line.replace('settingsSection("Company"', 'SettingsSectionView(title: "Company"')
    elif 'settingsSection("Support"' in line:
        line = line.replace('settingsSection("Support"', 'SettingsSectionView(title: "Support"')
        
    if "private var userCard: some View {" in line:
        skip = True
    if skip and "private func loadProfile() async {" in line:
        skip = False

    if not skip:
        new_i_lines.append(line)

with open(ios_path, 'w') as f:
    f.writelines(new_i_lines)

print("Done")
