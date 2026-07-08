sed -i '' '/retailerName = order.retailerName.ifBlank { retailerName },/a\
                    isPartial = order.isPartial,\
                    splitGroupId = order.splitGroupId,
' /Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/driver-app-android/app/src/main/java/com/pegasusx/driver/ui/screens/manifest/CorrectionViewModel.kt
