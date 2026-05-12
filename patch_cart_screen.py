import re

target = "pegasus/apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/cart/CartScreen.kt"
with open(target, "r") as f:
    text = f.read()

text = text.replace("paymentOptions = DefaultCheckoutPaymentOptions,", "paymentOptions = uiState.paymentOptions.ifEmpty { DefaultCheckoutPaymentOptions },")

with open(target, "w") as f:
    f.write(text)

print("CartScreen patched to use dynamic paymentOptions.")
