package com.pegasusx.driver.ui.screens.profile.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.background
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.DirectionsCar
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.driver.data.model.Order
import com.pegasusx.driver.data.model.OrderState
import com.pegasusx.driver.data.remote.TokenHolder
import com.pegasusx.driver.ui.components.PegasusCard
import com.pegasusx.driver.ui.components.StatusPill
import com.pegasusx.driver.ui.theme.LocalPegasusColors
import com.pegasusx.driver.ui.theme.PegasusSpacing

@Composable
fun DriverCard(orders: List<Order>, hasActiveRoute: Boolean) {
    val lab = LocalPegasusColors.current
    val driverName = TokenHolder.driverName ?: "Driver"
    val driverId = TokenHolder.userId ?: "—"
    val truckId = TokenHolder.vehicleType ?: "—"
    val plate = TokenHolder.licensePlate ?: "—"
    val completedCount = orders.count { it.state == OrderState.COMPLETED }

    PegasusCard {
        Column(modifier = Modifier.padding(PegasusSpacing.s20)) {
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(14.dp)
            ) {
                // Avatar
                Box(
                    modifier = Modifier
                        .size(52.dp)
                        .clip(CircleShape)
                        .background(lab.fg),
                    contentAlignment = Alignment.Center
                ) {
                    Text(
                        text = driverName.take(1).uppercase(),
                        fontSize = 22.sp,
                        fontWeight = FontWeight.Bold,
                        color = lab.buttonFg
                    )
                }

                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = driverName,
                        fontSize = 17.sp,
                        fontWeight = FontWeight.Bold,
                        color = lab.fg
                    )
                    Text(
                        text = driverId,
                        fontSize = 12.sp,
                        fontWeight = FontWeight.SemiBold,
                        fontFamily = FontFamily.Monospace,
                        color = lab.fgSecondary
                    )
                }

                StatusPill(
                    label = if (hasActiveRoute) "ON DUTY" else "IDLE",
                    color = if (hasActiveRoute) lab.success else lab.fgSecondary
                )
            }

            Spacer(modifier = Modifier.height(16.dp))

            // Info tiles
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                InfoTile(
                    label = stringResource(R.string.mobile_driver_ui_truck),
                    value = truckId,
                    icon = Icons.Default.LocalShipping,
                    modifier = Modifier.weight(1f)
                )
                InfoTile(
                    label = stringResource(R.string.factory_portal_fleet_text_plate),
                    value = plate,
                    icon = Icons.Default.DirectionsCar,
                    modifier = Modifier.weight(1f)
                )
                InfoTile(
                    label = stringResource(R.string.portal_page_orders_filter_completed),
                    value = "$completedCount",
                    icon = Icons.Default.CheckCircle,
                    modifier = Modifier.weight(1f)
                )
            }
        }
    }
}

@Composable
fun InfoTile(
    label: String,
    value: String,
    icon: ImageVector,
    modifier: Modifier = Modifier
) {
    val lab = LocalPegasusColors.current
    val isDark = isSystemInDarkTheme()
    Column(
        modifier = modifier
            .clip(RoundedCornerShape(12.dp))
            .background(lab.fg.copy(alpha = 0.03f))
            .padding(vertical = 12.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(6.dp)
    ) {
        Icon(
            imageVector = icon,
            contentDescription = null,
            tint = lab.fgSecondary,
            modifier = Modifier.size(14.dp)
        )
        Text(
            text = value,
            fontSize = 14.sp,
            fontWeight = FontWeight.Bold,
            fontFamily = FontFamily.Monospace,
            color = lab.fg,
            textAlign = TextAlign.Center
        )
        Text(
            text = label,
            fontSize = 10.sp,
            fontWeight = FontWeight.Medium,
            color = lab.fgTertiary
        )
    }
}
