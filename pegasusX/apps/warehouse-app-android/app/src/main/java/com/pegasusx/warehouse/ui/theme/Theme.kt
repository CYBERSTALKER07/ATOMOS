package com.pegasusx.warehouse.ui.theme

import androidx.compose.runtime.Composable
import com.pegasus.design.PegasusMonochromeTheme

@Composable
fun PegasusWarehouseTheme(
    darkTheme: Boolean = androidx.compose.foundation.isSystemInDarkTheme(),
    dynamicColor: Boolean = false,
    content: @Composable () -> Unit,
) {
    PegasusMonochromeTheme(
        darkTheme = darkTheme,
        dynamicColor = dynamicColor,
        typography = PegasusTypography,
        content = content,
    )
}
