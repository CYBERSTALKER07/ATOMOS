import re

target = "pegasus/apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/cart/CartViewModel.kt"
with open(target, "r") as f:
    text = f.read()

text = text.replace("api.getSavedCards()", "api.getCards()")

with open(target, "w") as f:
    f.write(text)

print("CartViewModel method patched.")
