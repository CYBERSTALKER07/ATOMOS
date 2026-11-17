package com.pegasusx.driver.ui.screens.profile.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.DirectionsCar
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.driver.data.model.DriverEarningsResponse
import com.pegasusx.driver.ui.components.PegasusCard
import com.pegasusx.driver.ui.theme.LocalPegasusColors
import com.pegasusx.driver.ui.theme.PegasusSpacing
import com.pegasusx.driver.ui.theme.formattedAmount
import com.pegasusx.driver.R

@Composable
fun StatsSection(earnings: DriverEarningsResponse?) {
    val lab = LocalPegasusColors.current
    val totalValue = earnings?.totalVolume ?: 0L
    val deliveries = earnings?.totalDeliveries ?: 0L

    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Text(
            text = stringResource(R.string.portal_nav_earnings),
            fontSize = 17.sp,
            fontWeight = FontWeight.Bold,
            color = lab.fg,
            modifier = Modifier.padding(horizontal = PegasusSpacing.s8)
        )

        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            StatCard(
                title = stringResource(R.string.warehouse_portal_supply_requests_id_text_total_volume),
                value = if (totalValue > 0) totalValue.formattedAmount() else "—",
                icon = Icons.Default.DirectionsCar,
                modifier = Modifier.weight(1f)
            )
            StatCard(
                title = stringResource(R.string.mobile_driver_ui_deliveries),
                value = if (deliveries > 0) deliveries.toString() else "—",
                icon = Icons.Default.LocalShipping,
                modifier = Modifier.weight(1f)
            )
        }
    }
}

@Composable
fun StatCard(
    title: String,
    value: String,
    icon: ImageVector,
    modifier: Modifier = Modifier
) {
    val lab = LocalPegasusColors.current
    PegasusCard(modifier = modifier) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(PegasusSpacing.s16),
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                tint = lab.fgSecondary,
                modifier = Modifier.size(14.dp)
            )
            Text(
                text = value,
                fontSize = 15.sp,
                fontWeight = FontWeight.Bold,
                fontFamily = FontFamily.Monospace,
                color = lab.fg,
                maxLines = 1
            )
            Text(
                text = title,
                fontSize = 11.sp,
                fontWeight = FontWeight.Medium,
                color = lab.fgTertiary
            )
        }
    }
}
