sed -i '' '/if (showConfirmDialog) {/i\
    if (showStartTransitDialog) {\
        AlertDialog(\
            onDismissRequest = { showStartTransitDialog = false },\
            title = { Text("Start Transit", fontWeight = FontWeight.Bold) },\
            text = { Text("Are you sure you want to start transit? This will notify the other driver that you are on the way to this shared route.") },\
            confirmButton = {\
                Button(\
                    onClick = {\
                        showStartTransitDialog = false\
                        viewModel.startTransitForPartialOrder()\
                    }\
                ) {\
                    Text("Confirm")\
                }\
            },\
            dismissButton = {\
                TextButton(onClick = { showStartTransitDialog = false }) {\
                    Text("Cancel")\
                }\
            }\
        )\
    }\
' /Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/driver-app-android/app/src/main/java/com/pegasusx/driver/ui/screens/manifest/DeliveryCorrectionScreen.kt
