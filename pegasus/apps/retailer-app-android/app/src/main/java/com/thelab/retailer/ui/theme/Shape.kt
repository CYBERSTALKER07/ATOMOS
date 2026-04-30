package com.thelab.retailer.ui.theme

import androidx.compose.foundation.shape.GenericShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Shapes
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.RoundRect
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.unit.Density
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.LayoutDirection
import kotlin.math.PI
import kotlin.math.cos
import kotlin.math.min
import kotlin.math.sin

/**
 * M3 Expressive Shape System — The Lab Industries
 *
 * Custom shape morphs matching Material Design 3 Expressive spec.
 * Reference: material.io/blog/material-3-expressive-update
 */

// ── Squircle (Superellipse) ── The signature M3 container shape
val SquircleShape: Shape = GenericShape { size, _ ->
    squirclePath(size)
}

fun squircleShape(cornerFraction: Float = 0.2f): Shape = GenericShape { size, _ ->
    squirclePath(size, cornerFraction)
}

private fun Path.squirclePath(size: Size, cornerFraction: Float = 0.2f) {
    val w = size.width
    val h = size.height
    val r = min(w, h) * cornerFraction

    moveTo(r, 0f)
    lineTo(w - r, 0f)
    cubicTo(w, 0f, w, 0f, w, r)
    lineTo(w, h - r)
    cubicTo(w, h, w, h, w - r, h)
    lineTo(r, h)
    cubicTo(0f, h, 0f, h, 0f, h - r)
    lineTo(0f, r)
    cubicTo(0f, 0f, 0f, 0f, r, 0f)
    close()
}

// ── Clover (4-petal) ── For small badges, avatars
val CloverShape: Shape = GenericShape { size, _ ->
    cloverPath(size)
}

private fun Path.cloverPath(size: Size) {
    val cx = size.width / 2f
    val cy = size.height / 2f
    val r = min(cx, cy) * 0.55f
    val dist = min(cx, cy) * 0.35f

    val centers = listOf(
        cx to cy - dist,      // top
        cx + dist to cy,      // right
        cx to cy + dist,      // bottom
        cx - dist to cy,      // left
    )

    // Build with arcs through the petal centers
    moveTo(cx, 0f)
    for (i in centers.indices) {
        val (px, py) = centers[i]
        val (nx, ny) = centers[(i + 1) % centers.size]
        val midX = (px + nx) / 2f
        val midY = (py + ny) / 2f
        // Outer curve through petal
        cubicTo(px + (px - cx) * 0.5f, py + (py - cy) * 0.5f,
            nx + (nx - cx) * 0.5f, ny + (ny - cy) * 0.5f,
            if (i == 0) size.width else if (i == 1) cx else if (i == 2) 0f else cx,
            if (i == 0) cy else if (i == 1) size.height else if (i == 2) cy else 0f)
    }
    close()
}

// ── Scallop ── Wavy/petal-edged circle for badges, overlays
fun scallopShape(petalCount: Int = 8): Shape = GenericShape { size, _ ->
    scallopPath(size, petalCount)
}

private fun Path.scallopPath(size: Size, petalCount: Int) {
    val cx = size.width / 2f
    val cy = size.height / 2f
    val outerR = min(cx, cy)
    val innerR = outerR * 0.82f
    val step = (2.0 * PI / petalCount).toFloat()

    moveTo(cx + outerR, cy)
    for (i in 0 until petalCount) {
        val startAngle = i * step
        val midAngle = startAngle + step / 2f
        val endAngle = startAngle + step

        val midX = cx + innerR * cos(midAngle)
        val midY = cy + innerR * sin(midAngle)
        val endX = cx + outerR * cos(endAngle)
        val endY = cy + outerR * sin(endAngle)

        quadraticTo(midX, midY, endX, endY)
    }
    close()
}

val ScallopShape: Shape = scallopShape(8)

// ── Star (N-point) ── For highlights, ratings, badges
fun starShape(points: Int = 5, innerRadiusFraction: Float = 0.45f): Shape = GenericShape { size, _ ->
    starPath(size, points, innerRadiusFraction)
}

private fun Path.starPath(size: Size, points: Int, innerFraction: Float) {
    val cx = size.width / 2f
    val cy = size.height / 2f
    val outerR = min(cx, cy)
    val innerR = outerR * innerFraction
    val step = (PI / points).toFloat()
    val startOffset = (-PI / 2f).toFloat()

    moveTo(
        cx + outerR * cos(startOffset),
        cy + outerR * sin(startOffset),
    )
    for (i in 1 until points * 2) {
        val angle = startOffset + i * step
        val r = if (i % 2 == 0) outerR else innerR
        lineTo(cx + r * cos(angle), cy + r * sin(angle))
    }
    close()
}

val StarShape: Shape = starShape(5)

// ── Diamond (rotated square) ── For small indicators, chips
val DiamondShape: Shape = GenericShape { size, _ ->
    val cx = size.width / 2f
    val cy = size.height / 2f
    moveTo(cx, 0f)
    lineTo(size.width, cy)
    lineTo(cx, size.height)
    lineTo(0f, cy)
    close()
}

// ── Hexagon ── For profile avatars, category icons
val HexagonShape: Shape = GenericShape { size, _ ->
    hexagonPath(size)
}

private fun Path.hexagonPath(size: Size) {
    val cx = size.width / 2f
    val cy = size.height / 2f
    val r = min(cx, cy)
    val startOffset = (-PI / 2f).toFloat()

    for (i in 0 until 6) {
        val angle = startOffset + i * (PI.toFloat() / 3f)
        val x = cx + r * cos(angle)
        val y = cy + r * sin(angle)
        if (i == 0) moveTo(x, y) else lineTo(x, y)
    }
    close()
}

// ── Soft Squircle (iOS-like) ── Higher corner smoothing for containers
val SoftSquircleShape: Shape = squircleShape(cornerFraction = 0.28f)

// ── Pentagon ── For action badges
val PentagonShape: Shape = GenericShape { size, _ ->
    val cx = size.width / 2f
    val cy = size.height / 2f
    val r = min(cx, cy)
    val startOffset = (-PI / 2f).toFloat()

    for (i in 0 until 5) {
        val angle = startOffset + i * (2f * PI.toFloat() / 5f)
        val x = cx + r * cos(angle)
        val y = cy + r * sin(angle)
        if (i == 0) moveTo(x, y) else lineTo(x, y)
    }
    close()
}

// ── Cookie (rounded clover) ── For profile pictures, organic feel
val CookieShape: Shape = GenericShape { size, _ ->
    val cx = size.width / 2f
    val cy = size.height / 2f
    val r = min(cx, cy)

    // 4-lobe rounded shape
    val lobeR = r * 0.62f
    val dist = r * 0.38f

    reset()
    // Top-right lobe
    addOval(androidx.compose.ui.geometry.Rect(
        cx + dist - lobeR, cy - dist - lobeR, cx + dist + lobeR, cy - dist + lobeR))
    // Bottom-right lobe
    addOval(androidx.compose.ui.geometry.Rect(
        cx + dist - lobeR, cy + dist - lobeR, cx + dist + lobeR, cy + dist + lobeR))
    // Bottom-left lobe
    addOval(androidx.compose.ui.geometry.Rect(
        cx - dist - lobeR, cy + dist - lobeR, cx - dist + lobeR, cy + dist + lobeR))
    // Top-left lobe
    addOval(androidx.compose.ui.geometry.Rect(
        cx - dist - lobeR, cy - dist - lobeR, cx - dist + lobeR, cy - dist + lobeR))
}

// ── Heart ── For favorites, likes
val HeartShape: Shape = GenericShape { size, _ ->
    val w = size.width
    val h = size.height

    moveTo(w / 2f, h * 0.35f)
    cubicTo(w * 0.15f, -h * 0.1f, -w * 0.1f, h * 0.4f, w / 2f, h)
    moveTo(w / 2f, h * 0.35f)
    cubicTo(w * 0.85f, -h * 0.1f, w * 1.1f, h * 0.4f, w / 2f, h)
    close()
}

// ── Pill ── Standard M3 capsule for buttons, chips
val PillShape: Shape = GenericShape { size, _ ->
    addRoundRect(RoundRect(
        left = 0f, top = 0f, right = size.width, bottom = size.height,
        cornerRadius = CornerRadius(size.height / 2f, size.height / 2f),
    ))
}

// ── Blob (organic asymmetric) ── For decorative backgrounds
val BlobShape: Shape = GenericShape { size, _ ->
    val w = size.width
    val h = size.height
    moveTo(w * 0.5f, 0f)
    cubicTo(w * 0.8f, h * 0.05f, w, h * 0.25f, w * 0.95f, h * 0.5f)
    cubicTo(w, h * 0.75f, w * 0.75f, h, w * 0.5f, h * 0.95f)
    cubicTo(w * 0.2f, h, 0f, h * 0.7f, w * 0.05f, h * 0.45f)
    cubicTo(0f, h * 0.2f, w * 0.2f, 0f, w * 0.5f, 0f)
    close()
}

// ── M3 Shape Scale (MDC 1.14 spec) ──
val LabShapes = Shapes(
    extraSmall = RoundedCornerShape(4.dp),   // Corner.ExtraSmall
    small = RoundedCornerShape(8.dp),        // Corner.Small
    medium = RoundedCornerShape(12.dp),      // Corner.Medium
    large = RoundedCornerShape(16.dp),       // Corner.Large
    extraLarge = RoundedCornerShape(28.dp),  // Corner.ExtraLarge
)
