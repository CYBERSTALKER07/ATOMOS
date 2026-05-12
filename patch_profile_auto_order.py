import re

target = "pegasus/apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/profile/ProfileScreen.kt"
with open(target, "r") as f:
    text = f.read()

# Add to parameters
old_params = "onNavigateToSavedCards: () -> Unit = {},"
new_params = "onNavigateToSavedCards: () -> Unit = {},\n    onNavigateToAutoOrder: () -> Unit = {},"

if "onNavigateToAutoOrder" not in text:
    text = text.replace(old_params, new_params)

pattern = r"(PegasusButton\(\s+text = \"Saved Cards / Payment\",\s+onClick = onNavigateToSavedCards,\s+modifier = Modifier.fillMaxWidth\(\)\s+\))"
replacement = """PegasusButton(
                    text = "AI Auto-Ordering Settings",
                    onClick = onNavigateToAutoOrder,
                    modifier = Modifier.fillMaxWidth()
                )
                Spacer(modifier = Modifier.height(16.dp))
                \1"""

text = re.sub(pattern, replacement, text)

with open(target, "w") as f:
    f.write(text)

print("Profile patched for Auto Order")
