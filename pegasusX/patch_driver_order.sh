sed -i '' '/val routeId: String? = null,/a\
    @SerialName("is_partial") val isPartial: Boolean = false,\
    @SerialName("split_group_id") val splitGroupId: String? = null,
' /Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/driver-app-android/app/src/main/java/com/pegasusx/driver/data/model/DriverModels.kt
sed -i '' '/val itemsJson: String/a\
    , val isPartial: Boolean = false, val splitGroupId: String? = null
' /Users/shakhzod/Desktop/V.O.I.D/pegasusX/apps/driver-app-android/app/src/main/java/com/pegasusx/driver/data/model/DriverModels.kt
