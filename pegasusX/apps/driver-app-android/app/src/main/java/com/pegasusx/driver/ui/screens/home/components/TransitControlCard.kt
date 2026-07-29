package com.pegasusx.driver.ui.screens.home.components

import android.content.Intent
import android.net.Uri
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material.icons.filled.Navigation
import androidx.compose.material.icons.filled.Schedule
import androidx.compose.material3.Button
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.pegasusx.driver.data.model.Order
import com.pegasusx.driver.data.model.OrderState
import com.pegasusx.driver.data.remote.TokenHolder
import com.pegasusx.driver.services.TelemetryService
import com.pegasusx.driver.ui.components.PegasusCard
import com.pegasusx.driver.ui.theme.LocalPegasusColors
import com.pegasusx.driver.ui.theme.MotionTokens
import com.pegasusx.driver.ui.theme.PegasusSpacing

@Composable
fun TransitControlCard(
    orders: List<Order>,
    onDepart: () -> Unit
) {
    val lab = LocalPegasusColors.current
    val context = LocalContext.current
    val loadedOrders = orders.filter { it.state == OrderState.LOADED }
    val inTransitOrders = orders.filter {
        it.state == OrderState.IN_TRANSIT || it.state == OrderState.ARRIVING
    }

    PegasusCard {
        Column(modifier = Modifier.padding(PegasusSpacing.s20)) {
            when {
                inTransitOrders.isNotEmpty() -> {
                    // Active transit
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(10.dp)
                    ) {
                        PulsingDot(color = lab.live)
                        Text(
                            text = "IN TRANSIT",
                            style = MaterialTheme.typography.labelMedium,
                            fontWeight = FontWeight.Black,
                            fontFamily = FontFamily.Monospace,
                            color = lab.live
                        )
                        Spacer(modifier = Modifier.weight(1f))
                        Text(
                            text = "${inTransitOrders.size} deliveries",
                            style = MaterialTheme.typography.bodySmall,
                            fontWeight = FontWeight.Medium,
                            color = lab.fgTertiary
                        )
                    }
                    Spacer(modifier = Modifier.height(8.dp))
                    Text(
                        text = "Telemetry active — drive safely",
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.Medium,
                        color = lab.fgTertiary
                    )
                }
                loadedOrders.isNotEmpty() -> {
                    // Ready to depart
                    Text(
                        text = "READY TO DEPART",
                        style = MaterialTheme.typography.labelMedium,
                        fontWeight = FontWeight.Black,
                        fontFamily = FontFamily.Monospace,
                        color = lab.fg
                    )
                    Spacer(modifier = Modifier.height(4.dp))
                    Text(
                        text = "${loadedOrders.size} orders loaded",
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.Medium,
                        color = lab.fgTertiary
                    )
                    Spacer(modifier = Modifier.height(14.dp))
                    Button(
                        onClick = {
                            val intent =
                                Intent(context, TelemetryService::class.java).apply {
                                    action = TelemetryService.ACTION_START
                                }
                            context.startForegroundService(intent)
                            onDepart()
                        },
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(PegasusSpacing.s48),
                    ) {
                        Row(
                            verticalAlignment = Alignment.CenterVertically,
                            horizontalArrangement = Arrangement.spacedBy(8.dp)
                        ) {
                            Icon(
                                imageVector = Icons.Default.LocalShipping,
                                contentDescription = null,
                                modifier = Modifier.size(18.dp)
                            )
                            Text(
                                text = "START TRANSIT",
                                style = MaterialTheme.typography.labelLarge,
                                fontWeight = FontWeight.Black,
                                fontFamily = FontFamily.Monospace,
                            )
                        }
                    }
                }
                else -> {
                    // No orders
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(10.dp)
                    ) {
                        Icon(
                            imageVector = Icons.Default.Schedule,
                            contentDescription = null,
                            tint = lab.fgTertiary,
                            modifier = Modifier.size(18.dp)
                        )
                        Text(
                            text = "No orders loaded yet",
                            style = MaterialTheme.typography.bodyMedium,
                            fontWeight = FontWeight.Medium,
                            color = lab.fgTertiary
                        )
                    }
                }
            }
        }
    }
}

@Composable
fun ReturningToWarehouseCard(
    returnLines: List<com.pegasusx.driver.data.model.ReturnGoodsLine>,
    totalUnits: Long,
    onNavigate: () -> Unit,
    onArrived: () -> Unit
) {
    val lab = LocalPegasusColors.current
    val context = LocalContext.current
    // Dynamic warehouse coords from backend (fallback to Tashkent depot)
    val depotLat = TokenHolder.warehouseLat.takeIf { it != 0.0 } ?: 41.2995
    val depotLng = TokenHolder.warehouseLng.takeIf { it != 0.0 } ?: 69.2401
    val warehouseLabel = TokenHolder.warehouseName ?: "Warehouse"

    PegasusCard {
        Column(modifier = Modifier.padding(PegasusSpacing.s20)) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(10.dp)
            ) {
                PulsingDot(color = lab.warning)
                Text(
                    text = "RETURNING TO WAREHOUSE",
                    style = MaterialTheme.typography.labelMedium,
                    fontWeight = FontWeight.Black,
                    fontFamily = FontFamily.Monospace,
                    color = lab.warning
                )
            }
            Spacer(modifier = Modifier.height(8.dp))
            Text(
                text = "All deliveries completed. Return to warehouse to finish shift.",
                style = MaterialTheme.typography.bodyMedium,
                fontWeight = FontWeight.Medium,
                color = lab.fgTertiary
            )
            if (totalUnits > 0) {
                Spacer(modifier = Modifier.height(8.dp))
                Text(
                    text = "$totalUnits item(s) to return on truck",
                    style = MaterialTheme.typography.labelLarge,
                    fontWeight = FontWeight.Bold,
                    color = lab.warning
                )
                returnLines.take(4).forEach { line ->
                    Text(
                        text = "• ${line.productName} × ${line.quantity} (${line.reason})",
                        style = MaterialTheme.typography.bodySmall,
                        color = lab.fgTertiary
                    )
                }
            }
            Spacer(modifier = Modifier.height(16.dp))

            // Navigate to warehouse
            FilledTonalButton(
                onClick = {
                    val uri = Uri.parse("google.navigation:q=$depotLat,$depotLng&mode=d")
                    val intent = Intent(Intent.ACTION_VIEW, uri).apply {
                        setPackage("com.google.android.apps.maps")
                    }
                    if (intent.resolveActivity(context.packageManager) != null) {
                        context.startActivity(intent)
                    } else {
                        val webUri = Uri.parse("https://www.google.com/maps/dir/?api=1&destination=$depotLat,$depotLng&travelmode=driving")
                        context.startActivity(Intent(Intent.ACTION_VIEW, webUri))
                    }
                },
                modifier = Modifier.fillMaxWidth()
            ) {
                Icon(
                    imageVector = Icons.Default.Navigation,
                    contentDescription = null,
                    modifier = Modifier.size(16.dp)
                )
                Spacer(modifier = Modifier.width(8.dp))
                Text("Navigate to $warehouseLabel", style = MaterialTheme.typography.labelLarge)
            }

            Spacer(modifier = Modifier.height(8.dp))

            // Arrived at warehouse
            Button(
                onClick = onArrived,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(PegasusSpacing.s48),
            ) {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    Icon(
                        imageVector = Icons.Default.Home,
                        contentDescription = null,
                        modifier = Modifier.size(18.dp)
                    )
                    Text(
                        text = "ARRIVED AT WAREHOUSE",
                        style = MaterialTheme.typography.labelLarge,
                        fontWeight = FontWeight.Black,
                        fontFamily = FontFamily.Monospace,
                    )
                }
            }
        }
    }
}

@Composable
fun PulsingDot(color: androidx.compose.ui.graphics.Color) {
    val transition = rememberInfiniteTransition(label = "pulse")
    val alpha by transition.animateFloat(
        initialValue = 1f,
        targetValue = 0.3f,
        animationSpec = infiniteRepeatable(
            animation = tween(MotionTokens.DurationExtraLong4),
            repeatMode = RepeatMode.Reverse
        ),
        label = "pulse_alpha"
    )
    Box(
        modifier = Modifier
            .size(8.dp)
            .alpha(alpha)
            .clip(CircleShape)
            .background(color)
    )
}
