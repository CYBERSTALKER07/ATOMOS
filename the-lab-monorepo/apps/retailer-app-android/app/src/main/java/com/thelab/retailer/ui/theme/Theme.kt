package com.thelab.retailer.ui.theme

import android.os.Build
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.dynamicDarkColorScheme
import androidx.compose.material3.dynamicLightColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.SideEffect
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalView
import androidx.core.view.WindowCompat

/**
 * The Lab — M3 Retailer Theme
 *
 * Neutral B&W brand palette as baseline.
 * Dynamic color (Android 12+) activates Material You personalization.
 * M3 surface container hierarchy, shape scale, and motion tokens.
 */

private val LabLightColorScheme = lightColorScheme(
    primary = Black,
    onPrimary = White,
    primaryContainer = Neutral94,
    onPrimaryContainer = Neutral10,

    secondary = Neutral40,
    onSecondary = White,
    secondaryContainer = Neutral92,
    onSecondaryContainer = Neutral10,

    tertiary = Neutral30,
    onTertiary = White,
    tertiaryContainer = Neutral90,
    onTertiaryContainer = Neutral10,

    error = StatusRed,
    onError = White,
    errorContainer = StatusRedSoft,
    onErrorContainer = StatusRed,

    background = Neutral95,
    onBackground = Neutral10,

    surface = White,
    onSurface = Neutral10,
    surfaceVariant = Neutral94,
    onSurfaceVariant = Neutral40,

    outline = Neutral70,
    outlineVariant = Neutral87,

    surfaceContainerLowest = White,
    surfaceContainerLow = Neutral98,
    surfaceContainer = Neutral96,
    surfaceContainerHigh = Neutral94,
    surfaceContainerHighest = Neutral92,
)

private val LabDarkColorScheme = darkColorScheme(
    primary = White,
    onPrimary = Black,
    primaryContainer = Neutral17,
    onPrimaryContainer = Neutral90,

    secondary = Neutral60,
    onSecondary = Neutral10,
    secondaryContainer = Neutral22,
    onSecondaryContainer = Neutral90,

    tertiary = Neutral50,
    onTertiary = Neutral10,
    tertiaryContainer = Neutral24,
    onTertiaryContainer = Neutral90,

    error = StatusRed,
    onError = White,
    errorContainer = StatusRedSoft,
    onErrorContainer = StatusRed,

    background = Neutral6,
    onBackground = Neutral90,

    surface = Neutral10,
    onSurface = Neutral90,
    surfaceVariant = Neutral17,
    onSurfaceVariant = Neutral60,

    outline = Neutral40,
    outlineVariant = Neutral22,

    surfaceContainerLowest = Neutral4,
    surfaceContainerLow = Neutral10,
    surfaceContainer = Neutral12,
    surfaceContainerHigh = Neutral17,
    surfaceContainerHighest = Neutral22,
)

@Composable
fun LabRetailerTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    dynamicColor: Boolean = true,
    content: @Composable () -> Unit,
) {
    val context = LocalContext.current
    val colorScheme = when {
        dynamicColor && Build.VERSION.SDK_INT >= Build.VERSION_CODES.S -> {
            if (darkTheme) dynamicDarkColorScheme(context)
            else dynamicLightColorScheme(context)
        }
        darkTheme -> LabDarkColorScheme
        else -> LabLightColorScheme
    }

    val view = LocalView.current
    if (!view.isInEditMode) {
        SideEffect {
            val window = (view.context as android.app.Activity).window
            WindowCompat.getInsetsController(window, view).apply {
                isAppearanceLightStatusBars = !darkTheme
                isAppearanceLightNavigationBars = !darkTheme
            }
            window.statusBarColor = Color.Transparent.toArgb()
            window.navigationBarColor = Color.Transparent.toArgb()
        }
    }

    MaterialTheme(
        colorScheme = colorScheme,
        typography = LabTypography,
        shapes = LabShapes,
        content = content,
    )
}
