package com.pegasusx.driver.ui.screens.home.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.pegasusx.driver.data.model.Order
import com.pegasusx.driver.ui.components.DriverSectionTitle
import com.pegasusx.driver.ui.components.DriverStateKind
import com.pegasusx.driver.ui.components.DriverStatePane
import com.pegasusx.driver.ui.components.PegasusCard
import com.pegasusx.driver.ui.theme.LocalPegasusColors
import com.pegasusx.driver.ui.theme.PegasusSpacing
import com.pegasusx.driver.ui.theme.formattedAmount
import com.pegasusx.driver.R

@Composable
fun RecentActivitySection(completedOrders: List<Order>) {
    val lab = LocalPegasusColors.current
    Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
        DriverSectionTitle(
            title = "RECENT",
            modifier = Modifier.padding(horizontal = PegasusSpacing.s4),
        )

        if (completedOrders.isEmpty()) {
            DriverStatePane(
                kind = DriverStateKind.Delivery,
                headline = "No deliveries yet",
                body = "Completed stops will appear here after you finish your route.",
                compact = true,
                usePegasusCard = true,
            )
        } else {
            completedOrders.take(3).forEach { order ->
                PegasusCard {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(PegasusSpacing.s12),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(12.dp)
                    ) {
                        Box(
                            modifier = Modifier
                                .size(40.dp)
                                .clip(CircleShape)
                                .background(lab.success.copy(alpha = 0.12f)),
                            contentAlignment = Alignment.Center
                        ) {
                            Icon(
                                imageVector = Icons.Default.CheckCircle,
                                contentDescription = null,
                                tint = lab.success,
                                modifier = Modifier.size(16.dp)
                            )
                        }

                        Column(modifier = Modifier.weight(1f)) {
                            Text(
                                text = order.id,
                                style = MaterialTheme.typography.titleSmall,
                                fontWeight = FontWeight.Bold,
                                fontFamily = FontFamily.Monospace,
                                color = lab.fg
                            )
                            Text(
                                text = order.totalAmount.formattedAmount(),
                                style = MaterialTheme.typography.bodySmall,
                                fontWeight = FontWeight.Medium,
                                color = lab.fgSecondary
                            )
                            if (order.deliveryFeeMinor > 0) {
                                Text(
                                    text = stringResource(R.string.mobile_driver_ui_formattedamount_delivery_fee, order.deliveryFeeMinor.formattedAmount()),
                                    style = MaterialTheme.typography.labelSmall,
                                    color = lab.warning,
                                )
                            }
                        }

                        Text(
                            text = order.retailerName,
                            style = MaterialTheme.typography.labelMedium,
                            fontWeight = FontWeight.Bold,
                            color = lab.fgTertiary
                        )
                    }
                }
            }
        }
    }
}
