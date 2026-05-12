import re

target = "pegasus/apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/navigation/RetailerNavigation.kt"
with open(target, "r") as f:
    text = f.read()

# Add to RetailerScreen
if "object FamilyMembers : RetailerScreen" not in text:
    text = text.replace('object Profile : RetailerScreen("profile")', 'object Profile : RetailerScreen("profile")\n    object FamilyMembers : RetailerScreen("family_members")')

if "import com.pegasus.retailer.ui.screens.profile.FamilyMembersScreen" not in text:
    text = text.replace('import com.pegasus.retailer.ui.screens.profile.ProfileScreen', 'import com.pegasus.retailer.ui.screens.profile.ProfileScreen\nimport com.pegasus.retailer.ui.screens.profile.FamilyMembersScreen')

old_profile_nav = """        composable(RetailerScreen.Profile.route) {
            ProfileScreen(
                onNavigateBack = { navController.popBackStack() }
            )
        }"""
new_profile_nav = """        composable(RetailerScreen.Profile.route) {
            ProfileScreen(
                onNavigateBack = { navController.popBackStack() },
                onNavigateToFamily = { navController.navigate(RetailerScreen.FamilyMembers.route) }
            )
        }
        composable(RetailerScreen.FamilyMembers.route) {
            FamilyMembersScreen(
                onNavigateBack = { navController.popBackStack() }
            )
        }"""
text = text.replace(old_profile_nav, new_profile_nav)

with open(target, "w") as f:
    f.write(text)

print("Nav updated")
