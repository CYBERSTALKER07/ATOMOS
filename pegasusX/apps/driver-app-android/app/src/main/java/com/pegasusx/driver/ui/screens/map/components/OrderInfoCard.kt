package com.pegasusx.driver.ui.screens.map.components

import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Navigation
import androidx.compose.material.icons.filled.Schedule
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.pegasus.design.ui.PegasusSpacing
import com.pegasusx.driver.R
import com.pegasusx.driver.data.model.Order
import com.pegasusx.driver.data.model.OrderState
import com.pegasusx.driver.ui.screens.map.MapPhase
import com.pegasusx.driver.ui.screens.map.resolveMapPhase
import com.pegasusx.driver.ui.theme.Neutral40
import com.pegasusx.driver.ui.theme.StatusBlue
import com.pegasusx.driver.ui.theme.StatusGreen
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
    val phaseForCard = resolveMapPhase(activeOrder ?: order)

    // Design system tokens
    val blueColor = StatusBlue
    val greenColor = StatusGreen
    val greyColor = Neutral40

    Column(
        modifier = modifier
            .fillMaxWidth()
            .background(
                MaterialTheme.colorScheme.surfaceContainerHigh,
                RoundedCornerShape(topStart = 24.dp, topEnd = 24.dp)
            )
            .padding(start = 20.dp, end = 20.dp, top = 20.dp, bottom = 24.dp)
    ) {
        Text(
            text = order.retailerName,
            style = MaterialTheme.typography.titleLarge.copy(fontWeight = FontWeight.Bold),
            color = MaterialTheme.colorScheme.onSurface,
            maxLines = 1,
            overflow = TextOverflow.Ellipsis
        )
        Spacer(modifier = Modifier.height(4.dp))
        Text(
            text = order.deliveryAddress,
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            maxLines = 2,
            overflow = TextOverflow.Ellipsis
        )
        Spacer(modifier = Modifier.height(12.dp))

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
                    tint = blueColor,
                    modifier = Modifier.size(16.dp)
                )
                Text(
                    text = etaText,
                    style = MaterialTheme.typography.titleSmall.copy(
                        fontWeight = FontWeight.Bold,
                        fontFamily = FontFamily.Monospace
                    ),
                    color = blueColor
                )
            }
            Spacer(modifier = Modifier.height(12.dp))
        }

        Text(
            text = stringResource(R.string.mobile_driver_ui_name_size_itemif_s_else_formatamount, order.state.name, order.items.size, if (order.items.size != 1) "s" else "", formatAmount(order.totalAmount)),
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
        Spacer(modifier = Modifier.height(16.dp))

        // Actions grid - Thumb Zone
        Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
            if (order.latitude != null && order.longitude != null && phaseForCard != MapPhase.ARRIVED && phaseForCard != MapPhase.VERIFYING) {
                Button(
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
                    modifier = Modifier.fillMaxWidth().height(56.dp),
                    colors = ButtonDefaults.buttonColors(containerColor = blueColor)
                ) {
                    Icon(
                        imageVector = Icons.Default.Navigation,
                        contentDescription = null,
                        modifier = Modifier.size(20.dp)
                    )
                    Spacer(modifier = Modifier.width(8.dp))
                    Text("Navigate", style = MaterialTheme.typography.titleMedium)
                }
            }

            if (phaseForCard == MapPhase.ARRIVED || phaseForCard == MapPhase.VERIFYING) {
                Button(
                    onClick = onOpenScanner,
                    modifier = Modifier.fillMaxWidth().height(56.dp),
                    colors = ButtonDefaults.buttonColors(containerColor = greenColor)
                ) {
                    Text(
                        if (phaseForCard == MapPhase.VERIFYING) "Scan Proof of Delivery" else "Scan QR",
                        style = MaterialTheme.typography.titleMedium,
                    )
                }

                Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                    FilledTonalButton(
                        onClick = { onOpenCorrection(order.id, order.retailerName) },
                        modifier = Modifier.weight(1f).height(48.dp),
                    ) {
                        Text("Correction", style = MaterialTheme.typography.labelLarge)
                    }
                    FilledTonalButton(
                        onClick = onRequestRescue,
                        modifier = Modifier.weight(1f).height(48.dp),
                        colors = ButtonDefaults.filledTonalButtonColors(containerColor = MaterialTheme.colorScheme.errorContainer, contentColor = MaterialTheme.colorScheme.onErrorContainer)
                    ) {
                        Text("Rescue", style = MaterialTheme.typography.labelLarge)
                    }
                }
            } else {
                FilledTonalButton(
                    onClick = onRequestRescue,
                    modifier = Modifier.fillMaxWidth().height(48.dp),
                    colors = ButtonDefaults.filledTonalButtonColors(containerColor = greyColor, contentColor = Color.White)
                ) {
                    Text("Rescue", style = MaterialTheme.typography.labelLarge)
                }
            }
        }
    }
}

private fun formatAmount(amount: Long): String {
    val formatted = String.format(Locale.US, "%,d", amount).replace(',', ' ')
    return formatted
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
