package com.pegasusx.driver.ui.screens.map.components

import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Navigation
import androidx.compose.material.icons.filled.Schedule
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.pegasusx.driver.data.model.Order
import com.pegasusx.driver.ui.screens.map.MapPhase
import com.pegasusx.driver.ui.screens.map.resolveMapPhase
import java.text.SimpleDateFormat
import java.util.Locale
import java.util.TimeZone

@Composable
fun OrderInfoCard(
    order: Order,
    activeOrder: Order?,
    onOpenScanner: () -> Unit,
    onOpenCorrection: (orderId: String, retailerName: String) -> Unit,
    onRequestRescue: () -> Unit = {},
    modifier: Modifier = Modifier
) {
    val context = LocalContext.current
    Column(
        modifier = modifier
            .fillMaxWidth()
            .background(
                MaterialTheme.colorScheme.surfaceContainerHigh,
                RoundedCornerShape(16.dp)
            )
            .padding(16.dp)
    ) {
        Text(
            text = order.retailerName,
            style = MaterialTheme.typography.titleMedium,
            color = MaterialTheme.colorScheme.onSurface,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis
        )
        Spacer(modifier = Modifier.height(4.dp))
        Text(
            text = order.deliveryAddress,
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            maxLines = 2,
            overflow = TextOverflow.Ellipsis
        )
        Spacer(modifier = Modifier.height(8.dp))

        // ETA row
        val etaText = formatETA(order)
        if (etaText != null) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(6.dp)
            ) {
                Icon(
                    imageVector = Icons.Default.Schedule,
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.primary,
                    modifier = Modifier.size(14.dp)
                )
                Text(
                    text = etaText,
                    style = MaterialTheme.typography.labelMedium.copy(
                        fontWeight = FontWeight.Bold,
                        fontFamily = FontFamily.Monospace
                    ),
                    color = MaterialTheme.colorScheme.primary
                )
            }
            Spacer(modifier = Modifier.height(8.dp))
        }

        Text(
            text = "${order.state.name} — ${order.items.size} item${if (order.items.size != 1) "s" else ""} — ${formatAmount(order.totalAmount)}",
            style = MaterialTheme.typography.labelMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )

        if (order.latitude != null && order.longitude != null) {
            Spacer(modifier = Modifier.height(12.dp))
            FilledTonalButton(
                onClick = {
                    val uri = Uri.parse("google.navigation:q=${order.latitude},${order.longitude}&mode=d")
                    val intent = Intent(Intent.ACTION_VIEW, uri).apply {
                        setPackage("com.google.android.apps.maps")
                    }
                    if (intent.resolveActivity(context.packageManager) != null) {
                        context.startActivity(intent)
                    } else {
                        val webUri = Uri.parse("https://www.google.com/maps/dir/?api=1&destination=${order.latitude},${order.longitude}&travelmode=driving")
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
                Text("Navigate", style = MaterialTheme.typography.labelLarge)
            }
        }

        val phaseForCard = resolveMapPhase(activeOrder ?: order)
        if (phaseForCard == MapPhase.ARRIVED || phaseForCard == MapPhase.VERIFYING) {
            Spacer(modifier = Modifier.height(8.dp))
            FilledTonalButton(
                onClick = onOpenScanner,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(
                    if (phaseForCard == MapPhase.VERIFYING) "Scan Proof of Delivery" else "Scan QR",
                    style = MaterialTheme.typography.labelLarge,
                )
            }
            Spacer(modifier = Modifier.height(8.dp))
            FilledTonalButton(
                onClick = { onOpenCorrection(order.id, order.retailerName) },
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text("Delivery Correction", style = MaterialTheme.typography.labelLarge)
            }
        }

        Spacer(modifier = Modifier.height(8.dp))
        FilledTonalButton(
            onClick = onRequestRescue,
            modifier = Modifier.fillMaxWidth(),
        ) {
            Text("Rescue", style = MaterialTheme.typography.labelLarge)
        }
    }
}

private fun formatAmount(amount: Long): String {
    val formatted = String.format("%,d", amount).replace(',', ' ')
    return "$formatted"
}

private fun formatETA(order: Order): String? {
    val etaSec = order.etaDurationSec ?: return null
    val distM = order.etaDistanceM

    val parts = mutableListOf<String>()

    // Time part
    if (order.estimatedArrivalAt != null) {
        try {
            val sdf = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss", Locale.US)
            sdf.timeZone = TimeZone.getTimeZone("UTC")
            val date = sdf.parse(order.estimatedArrivalAt)
            if (date != null) {
                val localFmt = SimpleDateFormat("HH:mm", Locale.getDefault())
                parts.add("ETA ${localFmt.format(date)}")
            }
        } catch (_: Exception) {
            // Fall through to duration display
        }
    }

    // Duration part
    val mins = etaSec / 60
    if (mins >= 60) {
        parts.add("${mins / 60}h ${mins % 60}m")
    } else {
        parts.add("${mins}m")
    }

    // Distance part
    if (distM != null && distM > 0) {
        val km = distM / 1000.0
        parts.add(String.format(Locale.US, "%.1f km", km))
    }

    return parts.joinToString(" · ")
}
