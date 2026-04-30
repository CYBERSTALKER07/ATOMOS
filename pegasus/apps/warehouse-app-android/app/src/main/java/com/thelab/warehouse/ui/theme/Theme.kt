package com.pegasus.warehouse.ui.theme

import android.os.Build
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.platform.LocalContext

private val LightColorScheme = lightColorScheme(
    primary = WarehousePrimary,
    onPrimary = WarehouseOnPrimary,
    primaryContainer = WarehousePrimaryContainer,
    onPrimaryContainer = WarehouseOnPrimaryContainer,
    secondary = WarehouseSecondary,
    onSecondary = WarehouseOnSecondary,
    secondaryContainer = WarehouseSecondaryContainer,
    onSecondaryContainer = WarehouseOnSecondaryContainer,
    tertiary = WarehouseTertiary,
    onTertiary = WarehouseOnTertiary,
    tertiaryContainer = WarehouseTertiaryContainer,
    onTertiaryContainer = WarehouseOnTertiaryContainer,
    error = WarehouseError,
    onError = WarehouseOnError,
    errorContainer = WarehouseErrorContainer,
    onErrorContainer = WarehouseOnErrorContainer,
    surface = WarehouseSurfaceLight,
    onSurface = WarehouseOnSurfaceLight,
    surfaceVariant = WarehouseSurfaceVariantLight,
    onSurfaceVariant = WarehouseOnSurfaceVariantLight,
    outline = WarehouseOutlineLight,
    surfaceContainer = WarehouseSurfaceContainerLight,
    surfaceContainerHigh = WarehouseSurfaceContainerHighLight,
)

private val DarkColorScheme = darkColorScheme(
    primary = WarehousePrimaryContainer,
    onPrimary = WarehouseOnPrimaryContainer,
    primaryContainer = WarehousePrimary,
    onPrimaryContainer = WarehouseOnPrimary,
    secondary = WarehouseSecondaryContainer,
    onSecondary = WarehouseOnSecondaryContainer,
    secondaryContainer = WarehouseSecondary,
    onSecondaryContainer = WarehouseOnSecondary,
    tertiary = WarehouseTertiaryContainer,
    onTertiary = WarehouseOnTertiaryContainer,
    tertiaryContainer = WarehouseTertiary,
    onTertiaryContainer = WarehouseOnTertiary,
    error = WarehouseErrorContainer,
    onError = WarehouseOnErrorContainer,
    errorContainer = WarehouseError,
    onErrorContainer = WarehouseOnError,
    surface = WarehouseSurfaceDark,
    onSurface = WarehouseOnSurfaceDark,
    surfaceVariant = WarehouseSurfaceVariantDark,
    onSurfaceVariant = WarehouseOnSurfaceVariantDark,
    outline = WarehouseOutlineDark,
    surfaceContainer = WarehouseSurfaceContainerDark,
    surfaceContainerHigh = WarehouseSurfaceContainerHighDark,
)

@Composable
fun LabWarehouseTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    dynamicColor: Boolean = true,
    content: @Composable () -> Unit,
) {
    val colorScheme = when {
        dynamicColor && Build.VERSION.SDK_INT >= Build.VERSION_CODES.S -> {
            val context = LocalContext.current
            if (darkTheme) dynamicDarkColorScheme(context) else dynamicLightColorScheme(context)
        }
        darkTheme -> DarkColorScheme
        else -> LightColorScheme
    }

    MaterialTheme(
        colorScheme = colorScheme,
        typography = LabTypography,
        content = content,
    )
}
