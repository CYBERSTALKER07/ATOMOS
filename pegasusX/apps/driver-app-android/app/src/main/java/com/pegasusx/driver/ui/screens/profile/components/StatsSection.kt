package com.pegasusx.driver.ui.screens.profile.components

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.DirectionsCar
import androidx.compose.material.icons.filled.LocationOn
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.driver.data.model.Order
import com.pegasusx.driver.ui.components.PegasusCard
import com.pegasusx.driver.ui.theme.LocalPegasusColors
import com.pegasusx.driver.ui.theme.PegasusSpacing
import com.pegasusx.driver.ui.theme.formattedAmount

@Composable
fun StatsSection(completedOrders: List<Order>) {
    val lab = LocalPegasusColors.current
    val totalValue = completedOrders.sumOf { it.totalAmount }

    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Text(
            text = "Session Stats",
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
                title = "Total Value",
                value = if (totalValue > 0) totalValue.formattedAmount() else "—",
                icon = Icons.Default.DirectionsCar,
                modifier = Modifier.weight(1f)
            )
            StatCard(
                title = "Avg Distance",
                value = "—",
                icon = Icons.Default.LocationOn,
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
