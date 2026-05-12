import re

target = "pegasus/apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/navigation/RetailerNavigation.kt"
with open(target, "r") as f:
    text = f.read()

# Add to RetailerScreen
if "object AutoOrder : RetailerScreen" not in text:
    text = text.replace('object SavedCards : RetailerScreen("saved_cards")', 'object SavedCards : RetailerScreen("saved_cards")\n    object AutoOrder : RetailerScreen("auto_order")')

if "import com.pegasus.retailer.ui.screens.profile.AutoOrderScreen" not in text:
    text = text.replace('import com.pegasus.retailer.ui.screens.profile.SavedCardsScreen', 'import com.pegasus.retailer.ui.screens.profile.SavedCardsScreen\nimport com.pegasus.retailer.ui.screens.profile.AutoOrderScreen')

old_profile_nav = """        composable(RetailerScreen.Profile.route) {
            ProfileScreen(
                onNavigateBack = { navController.popBackStack() },
                onNavigateToFamily = { navController.navigate(RetailerScreen.FamilyMembers.route) },
                onNavigateToSavedCards = { navController.navigate(RetailerScreen.SavedCards.route) }
            )
        }"""
new_profile_nav = """        composable(RetailerScreen.Profile.route) {
            ProfileScreen(
                onNavigateBack = { navController.popBackStack() },
                onNavigateToFamily = { navController.navigate(RetailerScreen.FamilyMembers.route) },
                onNavigateToSavedCards = { navController.navigate(RetailerScreen.SavedCards.route) },
                onNavigateToAutoOrder = { navController.navigate(RetailerScreen.AutoOrder.route) }
            )
        }
        composable(RetailerScreen.AutoOrder.route) {
            AutoOrderScreen(
                onNavigateBack = { navController.popBackStack() }
            )
        }"""
text = text.replace(old_profile_nav, new_profile_nav)

with open(target, "w") as f:
    f.write(text)

print("Nav updated for auto order")
