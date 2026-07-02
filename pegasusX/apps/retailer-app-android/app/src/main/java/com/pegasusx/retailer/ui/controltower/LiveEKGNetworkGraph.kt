package com.pegasusx.retailer.ui.controltower

import androidx.compose.animation.core.*
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.drawscope.DrawScope
import androidx.compose.ui.graphics.drawscope.Stroke
import kotlin.math.cos
import kotlin.math.sin

data class NetworkNode(
    val id: String,
    val type: String, // "warehouse", "retailer", "driver"
    val label: String,
    var x: Float = 0f,
    var y: Float = 0f
)

data class NetworkLink(
    val sourceId: String,
    val targetId: String
)

@Composable
fun LiveEKGNetworkGraph(
    nodes: List<NetworkNode>,
    links: List<NetworkLink>,
    modifier: Modifier = Modifier
) {
    // Pulse animation
    val infiniteTransition = rememberInfiniteTransition()
    val pulseProgress by infiniteTransition.animateFloat(
        initialValue = 0f,
        targetValue = 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(2000, easing = LinearEasing),
            repeatMode = RepeatMode.Restart
        )
    )

    Canvas(
        modifier = modifier
            .fillMaxSize()
            .background(Color.Black)
    ) {
        val width = size.width
        val height = size.height

        // Calculate positions (simple radial layout for demo)
        val radius = minOf(width, height) * 0.35f
        val center = Offset(width / 2, height / 2)
        
        nodes.forEachIndexed { index, node ->
            val angle = (2 * Math.PI * index / nodes.size).toFloat()
            node.x = center.x + radius * cos(angle)
            node.y = center.y + radius * sin(angle)
            
            // Bring warehouse to center for better visual
            if (node.type == "warehouse") {
                node.x = center.x
                node.y = center.y
            }
        }

        // Draw Links
        links.forEach { link ->
            val source = nodes.find { it.id == link.sourceId }
            val target = nodes.find { it.id == link.targetId }
            
            if (source != null && target != null) {
                val dx = target.x - source.x
                val dy = target.y - source.y
                val dist = kotlin.math.sqrt(dx * dx + dy * dy)
                
                // Curve calculation
                val controlPoint = Offset(
                    source.x + dx / 2 - dy * 0.2f,
                    source.y + dy / 2 + dx * 0.2f
                )

                val path = Path().apply {
                    moveTo(source.x, source.y)
                    quadraticBezierTo(controlPoint.x, controlPoint.y, target.x, target.y)
                }

                drawPath(
                    path = path,
                    color = Color(0xFF4B5563),
                    style = Stroke(width = 3f)
                )

                // Draw Pulse
                val pt = getQuadraticBezierPoint(
                    pulseProgress, 
                    Offset(source.x, source.y), 
                    controlPoint, 
                    Offset(target.x, target.y)
                )
                
                drawCircle(
                    color = Color(0xFF10B981),
                    radius = 8f,
                    center = pt,
                    alpha = 0.8f
                )
            }
        }

        // Draw Nodes
        nodes.forEach { node ->
            val color = when (node.type) {
                "warehouse" -> Color(0xFF10B981) // Emerald
                "retailer" -> Color(0xFF3B82F6)  // Blue
                else -> Color(0xFFF59E0B)        // Amber
            }

            when (node.type) {
                "warehouse" -> drawTriangle(Offset(node.x, node.y), 30f, color)
                "retailer" -> drawRect(
                    color = color,
                    topLeft = Offset(node.x - 15f, node.y - 15f),
                    size = androidx.compose.ui.geometry.Size(30f, 30f)
                )
                else -> drawCircle(
                    color = color,
                    radius = 15f,
                    center = Offset(node.x, node.y)
                )
            }
        }
    }
}

fun DrawScope.drawTriangle(center: Offset, size: Float, color: Color) {
    val path = Path().apply {
        moveTo(center.x, center.y - size)
        lineTo(center.x - size * 0.866f, center.y + size * 0.5f)
        lineTo(center.x + size * 0.866f, center.y + size * 0.5f)
        close()
    }
    drawPath(path = path, color = color)
}

fun getQuadraticBezierPoint(t: Float, p0: Offset, p1: Offset, p2: Offset): Offset {
    val x = (1 - t) * (1 - t) * p0.x + 2 * (1 - t) * t * p1.x + t * t * p2.x
    val y = (1 - t) * (1 - t) * p0.y + 2 * (1 - t) * t * p1.y + t * t * p2.y
    return Offset(x, y)
}
