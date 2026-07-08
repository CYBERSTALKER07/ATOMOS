sed -i '' '/var showConfirmDialog by remember { mutableStateOf(false) }/a\
    var showStartTransitDialog by remember { mutableStateOf(false) }
' /Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/driver-app-android/app/src/main/java/com/pegasusx/driver/ui/screens/manifest/DeliveryCorrectionScreen.kt
sed -i '' 's/showConfirmDialog = true; \/\* We will use showConfirmDialog but we need a different one for Start Transit \*\//showStartTransitDialog = true/' /Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/driver-app-android/app/src/main/java/com/pegasusx/driver/ui/screens/manifest/DeliveryCorrectionScreen.kt
