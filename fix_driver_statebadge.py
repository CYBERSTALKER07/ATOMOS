with open("the-lab-monorepo/apps/driver-app-android/app/src/main/java/com/thelab/driver/ui/components/StateBadge.kt", "r") as f:
    text = f.read()

replacement = """        OrderState.QUARANTINE -> Destructive to "QUARANTINE"
        OrderState.DELIVERED_ON_CREDIT -> Warning to "ON CREDIT"
        else -> colorScheme.onSurfaceVariant to state.name.replace("_", " ")"""

text = text.replace('        OrderState.QUARANTINE -> Destructive to "QUARANTINE"\n        OrderState.DELIVERED_ON_CREDIT -> Warning to "ON CREDIT"', replacement)

with open("the-lab-monorepo/apps/driver-app-android/app/src/main/java/com/thelab/driver/ui/components/StateBadge.kt", "w") as f:
    f.write(text)
