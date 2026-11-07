package com.pegasusx.driver.ui.screens.profile.components

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
import androidx.compose.material.icons.filled.Schedule
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.driver.data.model.DriverHistoryRow
import com.pegasusx.driver.ui.components.PegasusCard
import com.pegasusx.driver.ui.components.StaggeredAppear
import com.pegasusx.driver.ui.components.StatusPill
import com.pegasusx.driver.ui.theme.LocalPegasusColors
import com.pegasusx.driver.ui.theme.PegasusSpacing
import com.pegasusx.driver.ui.theme.formattedAmount
import com.pegasusx.driver.R

@Composable
fun HistorySection(historyRows: List<DriverHistoryRow>) {
    val lab = LocalPegasusColors.current

    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = PegasusSpacing.s8),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = stringResource(R.string.mobile_driver_ui_ride_history),
                fontSize = 17.sp,
                fontWeight = FontWeight.Bold,
                color = lab.fg,
            )
            Text(
                text = stringResource(R.string.mobile_driver_ui_size_rides, historyRows.size),
                fontSize = 12.sp,
                fontWeight = FontWeight.Medium,
                fontFamily = FontFamily.Monospace,
                color = lab.fgTertiary,
            )
        }

        if (historyRows.isEmpty()) {
            PegasusCard {
                Column(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(vertical = 30.dp),
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.spacedBy(10.dp)
                ) {
                    Icon(
                        imageVector = Icons.Default.Schedule,
                        contentDescription = null,
                        tint = lab.fgTertiary,
                        modifier = Modifier.size(24.dp)
                    )
                    Text(
                        text = stringResource(R.string.mobile_driver_ui_no_completed_rides_yet),
                        fontSize = 14.sp,
                        fontWeight = FontWeight.Medium,
                        color = lab.fgSecondary
                    )
                }
            }
        } else {
            historyRows.forEachIndexed { index, row ->
                StaggeredAppear(index = index) {
                    HistoryRow(row)
                }
            }
        }
    }
}

@Composable
fun HistoryRow(row: DriverHistoryRow) {
    val lab = LocalPegasusColors.current

    PegasusCard {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(PegasusSpacing.s16),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(14.dp)
        ) {
            Box(
                modifier = Modifier
                    .size(36.dp)
                    .clip(CircleShape)
                    .background(lab.success.copy(alpha = 0.15f)),
                contentAlignment = Alignment.Center
            ) {
                Icon(
                    imageVector = Icons.Default.CheckCircle,
                    contentDescription = null,
                    tint = lab.success,
                    modifier = Modifier.size(13.dp)
                )
            }

            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = row.orderId,
                    fontSize = 13.sp,
                    fontWeight = FontWeight.Bold,
                    fontFamily = FontFamily.Monospace,
                    color = lab.fg
                )
                Text(
                    text = "${row.status.ifBlank { "COMPLETED" }} · ${row.totalMinor.formattedAmount()}",
                    fontSize = 11.sp,
                    fontWeight = FontWeight.Medium,
                    color = lab.fgSecondary
                )
            }

            StatusPill(label = row.status.ifBlank { "DELIVERED" }, color = lab.success)
        }
    }
}
