import re

target = "pegasus/apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/profile/ProfileScreen.kt"
with open(target, "r") as f:
    text = f.read()

# Make sure we add a button to navigate if not there.

old_params = "onNavigateBack: () -> Unit,"
new_params = "onNavigateBack: () -> Unit,\n    onNavigateToFamily: () -> Unit = {},"

if "onNavigateToFamily" not in text:
    text = text.replace(old_params, new_params)

pattern = r"(Spacer\(modifier = Modifier.height\(24.dp\)\)\s+PegasusButton\(\s+text = \"Log Out\")"
replacement = """PegasusButton(
                    text = "Family Members / Staff",
                    onClick = onNavigateToFamily,
                    modifier = Modifier.fillMaxWidth()
                )
                Spacer(modifier = Modifier.height(16.dp))
                \1"""

text = re.sub(pattern, replacement, text)

with open(target, "w") as f:
    f.write(text)

