import re

target = "pegasus/apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/cart/CartViewModel.kt"
with open(target, "r") as f:
    text = f.read()

# Make it inject PegasusApi so we can load cards.
# Wait, it already has PegasusApi
