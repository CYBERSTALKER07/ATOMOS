package com.pegasus.payload.ui.theme

import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.dynamicDarkColorScheme
import androidx.compose.material3.dynamicLightColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import android.os.Build

// ── Monochrome Design Tokens ──
private val Neutral0 = Color(0xFF000000)
private val Neutral4 = Color(0xFF0C0C0E)
private val Neutral6 = Color(0xFF121214)
private val Neutral10 = Color(0xFF1C1C1E)
private val Neutral17 = Color(0xFF2C2C2E)
private val Neutral22 = Color(0xFF38383A)
private val Neutral24 = Color(0xFF3A3A3C)
private val Neutral30 = Color(0xFF48484A)
private val Neutral40 = Color(0xFF636366)
private val Neutral50 = Color(0xFF8E8E93)
private val Neutral60 = Color(0xFFAEAEB2)
private val Neutral70 = Color(0xFFC7C7CC)
private val Neutral87 = Color(0xFFE5E5EA)
private val Neutral90 = Color(0xFFE5E5EA)
private val Neutral92 = Color(0xFFEBEBF0)
private val Neutral94 = Color(0xFFF2F2F7)
private val Neutral95 = Color(0xFFF2F2F7)
private val Neutral96 = Color(0xFFF5F5FA)
private val Neutral98 = Color(0xFFFAFAFF)
private val Neutral100 = Color(0xFFFFFFFF)

private val StatusRed = Color(0xFFFF3B30)
private val StatusRedSoft = Color(0x1AFF3B30)

private val DarkScheme = darkColorScheme(
    primary = Neutral100,
    onPrimary = Neutral0,
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
    background = Neutral6,
    onBackground = Neutral90,
    surface = Neutral10,
    onSurface = Neutral90,
    surfaceVariant = Neutral17,
    onSurfaceVariant = Neutral60,
    error = StatusRed,
    onError = Neutral100,
    errorContainer = StatusRedSoft,
    onErrorContainer = StatusRed,
    outline = Neutral40,
    surfaceContainerLowest = Neutral4,
    surfaceContainerLow = Neutral6,
    surfaceContainer = Neutral10,
    surfaceContainerHigh = Neutral17,
    surfaceContainerHighest = Neutral22,
)

private val LightScheme = lightColorScheme(
    primary = Neutral0,
    onPrimary = Neutral100,
    primaryContainer = Neutral94,
    onPrimaryContainer = Neutral10,
    secondary = Neutral40,
    onSecondary = Neutral100,
    secondaryContainer = Neutral92,
    onSecondaryContainer = Neutral10,
    tertiary = Neutral30,
    onTertiary = Neutral100,
    tertiaryContainer = Neutral90,
    onTertiaryContainer = Neutral10,
    background = Neutral95,
    onBackground = Neutral10,
    surface = Neutral100,
    onSurface = Neutral10,
    surfaceVariant = Neutral94,
    onSurfaceVariant = Neutral40,
    error = StatusRed,
    onError = Neutral100,
    errorContainer = StatusRedSoft,
    onErrorContainer = StatusRed,
    outline = Neutral70,
    outlineVariant = Neutral87,
    surfaceContainerLowest = Neutral100,
    surfaceContainerLow = Neutral98,
    surfaceContainer = Neutral96,
    surfaceContainerHigh = Neutral94,
    surfaceContainerHighest = Neutral92,
)

@Composable
fun LabPayloadTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    dynamicColor: Boolean = false,
    content: @Composable () -> Unit
) {
    val context = LocalContext.current
    val scheme = when {
        dynamicColor && Build.VERSION.SDK_INT >= Build.VERSION_CODES.S ->
            if (darkTheme) dynamicDarkColorScheme(context) else dynamicLightColorScheme(context)
        darkTheme -> DarkScheme
        else -> LightScheme
    }
    MaterialTheme(colorScheme = scheme, content = content)
}
