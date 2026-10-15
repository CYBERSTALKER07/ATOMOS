package com.pegasusx.enterprise.theme

import androidx.compose.foundation.shape.CutCornerShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Shapes
import androidx.compose.material3.Typography
import androidx.compose.material3.darkColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
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

// Material Design 3 Micro-Chamfered CAD Shapes
val CADShapes = Shapes(
    extraSmall = RoundedCornerShape(2.dp),
    small = RoundedCornerShape(4.dp),
    medium = RoundedCornerShape(4.dp),
    large = RoundedCornerShape(6.dp),
    extraLarge = RoundedCornerShape(8.dp)
)

// Material Design 3 Strict Typography Hierarchy
val CADTypography = Typography(
    displayLarge = TextStyle(
        fontFamily = FontFamily.SansSerif,
        fontWeight = FontWeight.Light,
        fontSize = 34.sp,
        letterSpacing = (-0.5).sp,
        color = CADColors.TextWhite
    ),
    headlineLarge = TextStyle(
        fontFamily = FontFamily.SansSerif,
        fontWeight = FontWeight.Light,
        fontSize = 26.sp,
        color = CADColors.TextWhite
    ),
    headlineMedium = TextStyle(
        fontFamily = FontFamily.SansSerif,
        fontWeight = FontWeight.Normal,
        fontSize = 20.sp,
        color = CADColors.TextWhite
    ),
    headlineSmall = TextStyle(
        fontFamily = FontFamily.SansSerif,
        fontWeight = FontWeight.Medium,
        fontSize = 17.sp,
        color = CADColors.TextWhite
    ),
    titleLarge = TextStyle(
        fontFamily = FontFamily.SansSerif,
        fontWeight = FontWeight.SemiBold,
        fontSize = 16.sp,
        color = CADColors.TextWhite
    ),
    titleMedium = TextStyle(
        fontFamily = FontFamily.SansSerif,
        fontWeight = FontWeight.Medium,
        fontSize = 14.sp,
        color = CADColors.TextWhite
    ),
    titleSmall = TextStyle(
        fontFamily = FontFamily.Monospace,
        fontWeight = FontWeight.Bold,
        fontSize = 12.sp,
        letterSpacing = 0.5.sp,
        color = CADColors.ElectricCyan
    ),
    bodyLarge = TextStyle(
        fontFamily = FontFamily.SansSerif,
        fontWeight = FontWeight.Normal,
        fontSize = 15.sp,
        lineHeight = 22.sp,
        color = CADColors.TextWhite
    ),
    bodyMedium = TextStyle(
        fontFamily = FontFamily.SansSerif,
        fontWeight = FontWeight.Normal,
        fontSize = 13.sp,
        lineHeight = 19.sp,
        color = CADColors.TextWhite
    ),
    bodySmall = TextStyle(
        fontFamily = FontFamily.SansSerif,
        fontWeight = FontWeight.Normal,
        fontSize = 11.sp,
        color = CADColors.TextMuted
    ),
    labelLarge = TextStyle(
        fontFamily = FontFamily.Monospace,
        fontWeight = FontWeight.Bold,
        fontSize = 12.sp,
        letterSpacing = 1.sp,
        color = CADColors.TextWhite
    ),
    labelMedium = TextStyle(
        fontFamily = FontFamily.Monospace,
        fontWeight = FontWeight.SemiBold,
        fontSize = 11.sp,
        letterSpacing = 0.8.sp,
        color = CADColors.TextMuted
    ),
    labelSmall = TextStyle(
        fontFamily = FontFamily.Monospace,
        fontWeight = FontWeight.Bold,
        fontSize = 10.sp,
        letterSpacing = 1.sp,
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
fun SwissCADTheme(content: @Composable () -> Unit) {
    MaterialTheme(
        colorScheme = CADColorScheme,
        shapes = CADShapes,
        typography = CADTypography,
        content = content
    )
}
