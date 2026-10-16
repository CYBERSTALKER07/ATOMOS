package com.pegasusx.enterprise.theme

import android.os.Build
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Shapes
import androidx.compose.material3.Typography
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.dynamicDarkColorScheme
import androidx.compose.material3.dynamicLightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

// MARK: - Swiss CAD Material Design 3 Color Tokens
object CADColors {
    // Primary Monochromatic Foundation (WCAG AAA)
    val PitchBlack = Color(0xFF000000)
    val PanelDark = Color(0xFF0C0C0E)
    val CardDark = Color(0xFF131317)
    val CardHover = Color(0xFF1B1B21)
    val InputDark = Color(0xFF050507)

    // Material Design 3 Expressive Accent Hues
    val ElectricCyan = Color(0xFFA9EBF9)
    val VoltLime = Color(0xFFFBFF63)
    val EmeraldGreen = Color(0xFF8DDC96)
    val LaserOrange = Color(0xFFFE5934)
    val SkyBlue = Color(0xFF64B5F6)
    val PurpleRadar = Color(0xFFC084FC)

    // Architectural Strokes & Overlays
    val BorderStroke = Color(0x2EFFFFFF)
    val BorderStrong = Color(0x59FFFFFF)
    val BorderCyan = Color(0x99A9EBF9)
    val BorderLime = Color(0x99FBFF63)

    // Contrast-Compliant Text
    val TextWhite = Color(0xFFF8F9FA)
    val TextMuted = Color(0x9EFFFFFF)
    val TextDim = Color(0x61FFFFFF)

    // Surface Container Hierarchy (Material Design 3)
    val SurfaceDim = Color(0xFF08080A)
    val SurfaceContainer = Color(0xFF141418)
    val SurfaceContainerHigh = Color(0xFF1E1E24)
    val SurfaceContainerHighest = Color(0xFF282830)

    fun statusColor(status: String): Color = when (status.lowercase()) {
        "new" -> VoltLime
        "in_review" -> ElectricCyan
        "qualified" -> EmeraldGreen
        "contacted" -> SkyBlue
        "closed" -> Color(0xFF888888)
        else -> TextMuted
    }

    fun statusBg(status: String): Color = statusColor(status).copy(alpha = 0.14f)
    fun statusBorder(status: String): Color = statusColor(status).copy(alpha = 0.45f)
}

// Material Design 3 Shapes (4dp Grid Compliant)
val CADShapes = Shapes(
    extraSmall = RoundedCornerShape(4.dp),
    small = RoundedCornerShape(8.dp),
    medium = RoundedCornerShape(12.dp),
    large = RoundedCornerShape(16.dp),
    extraLarge = RoundedCornerShape(24.dp)
)

// Material Design 3 Strict Typography Hierarchy (Roboto / System Font, Type Scale Compliant)
val CADTypography = Typography(
    displayLarge = TextStyle(
        fontFamily = FontFamily.SansSerif,
        fontWeight = FontWeight.Light,
        fontSize = 57.sp,
        lineHeight = 64.sp,
        letterSpacing = (-0.25).sp,
        color = CADColors.TextWhite
    ),
    displayMedium = TextStyle(
        fontFamily = FontFamily.SansSerif,
        fontWeight = FontWeight.Light,
        fontSize = 45.sp,
        lineHeight = 52.sp,
        color = CADColors.TextWhite
    ),
    displaySmall = TextStyle(
        fontFamily = FontFamily.SansSerif,
        fontWeight = FontWeight.Normal,
        fontSize = 36.sp,
        lineHeight = 44.sp,
        color = CADColors.TextWhite
    ),
    headlineLarge = TextStyle(
        fontFamily = FontFamily.SansSerif,
        fontWeight = FontWeight.Light,
        fontSize = 32.sp,
        lineHeight = 40.sp,
        color = CADColors.TextWhite
    ),
    headlineMedium = TextStyle(
        fontFamily = FontFamily.SansSerif,
        fontWeight = FontWeight.Normal,
        fontSize = 28.sp,
        lineHeight = 36.sp,
        color = CADColors.TextWhite
    ),
    headlineSmall = TextStyle(
        fontFamily = FontFamily.SansSerif,
        fontWeight = FontWeight.Medium,
        fontSize = 24.sp,
        lineHeight = 32.sp,
        color = CADColors.TextWhite
    ),
    titleLarge = TextStyle(
        fontFamily = FontFamily.SansSerif,
        fontWeight = FontWeight.SemiBold,
        fontSize = 22.sp,
        lineHeight = 28.sp,
        color = CADColors.TextWhite
    ),
    titleMedium = TextStyle(
        fontFamily = FontFamily.SansSerif,
        fontWeight = FontWeight.Medium,
        fontSize = 16.sp,
        lineHeight = 24.sp,
        color = CADColors.TextWhite
    ),
    titleSmall = TextStyle(
        fontFamily = FontFamily.Monospace,
        fontWeight = FontWeight.Bold,
        fontSize = 14.sp,
        lineHeight = 20.sp,
        letterSpacing = 0.5.sp,
        color = CADColors.ElectricCyan
    ),
    bodyLarge = TextStyle(
        fontFamily = FontFamily.SansSerif,
        fontWeight = FontWeight.Normal,
        fontSize = 16.sp,
        lineHeight = 24.sp,
        color = CADColors.TextWhite
    ),
    bodyMedium = TextStyle(
        fontFamily = FontFamily.SansSerif,
        fontWeight = FontWeight.Normal,
        fontSize = 14.sp,
        lineHeight = 20.sp,
        color = CADColors.TextWhite
    ),
    bodySmall = TextStyle(
        fontFamily = FontFamily.SansSerif,
        fontWeight = FontWeight.Normal,
        fontSize = 12.sp,
        lineHeight = 16.sp,
        color = CADColors.TextMuted
    ),
    labelLarge = TextStyle(
        fontFamily = FontFamily.Monospace,
        fontWeight = FontWeight.Bold,
        fontSize = 14.sp,
        lineHeight = 20.sp,
        letterSpacing = 0.5.sp,
        color = CADColors.TextWhite
    ),
    labelMedium = TextStyle(
        fontFamily = FontFamily.Monospace,
        fontWeight = FontWeight.SemiBold,
        fontSize = 12.sp,
        lineHeight = 16.sp,
        letterSpacing = 0.5.sp,
        color = CADColors.TextMuted
    ),
    labelSmall = TextStyle(
        fontFamily = FontFamily.Monospace,
        fontWeight = FontWeight.Bold,
        fontSize = 11.sp,
        lineHeight = 16.sp,
        letterSpacing = 0.5.sp,
        color = CADColors.TextDim
    )
)

// Material 3 Dark Color Scheme mapping
val CADColorScheme = darkColorScheme(
    primary = CADColors.ElectricCyan,
    onPrimary = Color(0xFF00363F),
    primaryContainer = Color(0xFF004F5C),
    onPrimaryContainer = CADColors.ElectricCyan,

    secondary = CADColors.VoltLime,
    onSecondary = Color(0xFF323600),
    secondaryContainer = Color(0xFF494E00),
    onSecondaryContainer = CADColors.VoltLime,

    tertiary = CADColors.EmeraldGreen,
    onTertiary = Color(0xFF00391A),
    tertiaryContainer = Color(0xFF005328),
    onTertiaryContainer = CADColors.EmeraldGreen,

    error = CADColors.LaserOrange,
    onError = Color(0xFF601400),
    errorContainer = Color(0xFF8C1D00),
    onErrorContainer = Color(0xFFFFDAD4),

    background = CADColors.PitchBlack,
    onBackground = CADColors.TextWhite,

    surface = CADColors.PitchBlack,
    onSurface = CADColors.TextWhite,
    surfaceVariant = CADColors.PanelDark,
    onSurfaceVariant = CADColors.TextMuted,

    surfaceContainerLowest = CADColors.SurfaceDim,
    surfaceContainerLow = CADColors.PanelDark,
    surfaceContainer = CADColors.CardDark,
    surfaceContainerHigh = CADColors.CardHover,
    surfaceContainerHighest = CADColors.SurfaceContainerHighest,

    outline = CADColors.BorderStroke,
    outlineVariant = Color(0x1FFFFFFF),
    scrim = Color.Black
)

@Composable
fun SwissCADTheme(
    dynamicColor: Boolean = false,
    content: @Composable () -> Unit
) {
    val colorScheme = when {
        dynamicColor && Build.VERSION.SDK_INT >= Build.VERSION_CODES.S -> {
            val context = LocalContext.current
            if (isSystemInDarkTheme()) dynamicDarkColorScheme(context) else dynamicLightColorScheme(context)
        }
        else -> CADColorScheme
    }

    MaterialTheme(
        colorScheme = colorScheme,
        shapes = CADShapes,
        typography = CADTypography,
        content = content
    )
}
