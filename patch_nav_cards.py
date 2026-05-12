import re

target = "pegasus/apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/navigation/RetailerNavigation.kt"
with open(target, "r") as f:
    text = f.read()

# Add to RetailerScreen
if "object SavedCards : RetailerScreen" not in text:
    text = text.replace('object FamilyMembers : RetailerScreen("family_members")', 'object FamilyMembers : RetailerScreen("family_members")\n    object SavedCards : RetailerScreen("saved_cards")')

if "import com.pegasus.retailer.ui.screens.profile.SavedCardsScreen" not in text:
    text = text.replace('import com.pegasus.retailer.ui.screens.profile.FamilyMembersScreen', 'import com.pegasus.retailer.ui.screens.profile.FamilyMembersScreen\nimport com.pegasus.retailer.ui.screens.profile.SavedCardsScreen')


old_profile_nav = """        composable(RetailerScreen.Profile.route) {
            ProfileScreen(
                onNavigateBack = { navController.popBackStack() },
                onNavigateToFamily = { navController.navigate(RetailerScreen.FamilyMembers.route) }
            )
        }"""
new_profile_nav = """        composable(RetailerScreen.Profile.route) {
            ProfileScreen(
                onNavigateBack = { navController.popBackStack() },
                onNavigateToFamily = { navController.navigate(RetailerScreen.FamilyMembers.route) },
                onNavigateToSavedCards = { navController.navigate(RetailerScreen.SavedCards.route) }
            )
        }
        composable(RetailerScreen.SavedCards.route) {
            SavedCardsScreen(
                onNavigateBack = { navController.popBackStack() }
            )
        }"""
text = text.replace(old_profile_nav, new_profile_nav)

with open(target, "w") as f:
    f.write(text)

print("Nav updated for cards")
