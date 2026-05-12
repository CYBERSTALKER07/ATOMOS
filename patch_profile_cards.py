import re

target = "pegasus/apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/profile/ProfileScreen.kt"
with open(target, "r") as f:
    text = f.read()

# Make sure we add a button to navigate if not there.

old_params = "onNavigateToFamily: () -> Unit = {},"
new_params = "onNavigateToFamily: () -> Unit = {},\n    onNavigateToSavedCards: () -> Unit = {},"

if "onNavigateToSavedCards" not in text:
    text = text.replace(old_params, new_params)

pattern = r"(PegasusButton\(\s+text = \"Family Members / Staff\",\s+onClick = onNavigateToFamily,\s+modifier = Modifier.fillMaxWidth\(\)\s+\))"
replacement = """PegasusButton(
                    text = "Saved Cards / Payment",
                    onClick = onNavigateToSavedCards,
                    modifier = Modifier.fillMaxWidth()
                )
                Spacer(modifier = Modifier.height(16.dp))
                \1"""

text = re.sub(pattern, replacement, text)

with open(target, "w") as f:
    f.write(text)

print("Profile patched")

