package com.pegasusx.driver.ui.screens.home.components

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.driver.data.model.OrderState
import com.pegasusx.driver.ui.components.PegasusCard
import com.pegasusx.driver.ui.screens.home.MoneyHealthCounts
import com.pegasusx.driver.ui.screens.home.RemainingStop
import com.pegasusx.driver.ui.theme.LocalPegasusColors
import com.pegasusx.driver.ui.theme.PegasusSpacing
import com.pegasusx.driver.ui.theme.pressable

@Composable
fun RemainingStopsStepper(
    stops: List<RemainingStop>,
    onSelect: (String) -> Unit = {},
    modifier: Modifier = Modifier,
) {
    val lab = LocalPegasusColors.current
    PegasusCard(modifier = modifier.fillMaxWidth()) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.s16),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Text(
                text = "REMAINING STOPS",
                fontSize = 11.sp,
                fontWeight = FontWeight.Black,
                fontFamily = FontFamily.Monospace,
                color = lab.fgTertiary,
            )
            if (stops.isEmpty()) {
                Text("No remaining stops", fontSize = 13.sp, color = lab.fgTertiary)
            } else {
                stops.forEachIndexed { index, stop ->
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .pressable { onSelect(stop.id) },
                        verticalAlignment = Alignment.Top,
                    ) {
                        Column(horizontalAlignment = Alignment.CenterHorizontally) {
                            Box(
                                modifier = Modifier
                                    .size(10.dp)
                                    .clip(CircleShape)
                                    .background(
                                        when (stop.state) {
                                            OrderState.FISCAL_FAILED -> lab.destructive
                                            OrderState.ARRIVED_SHOP_CLOSED -> lab.warning
                                            else -> lab.fgSecondary
                                        },
                                    ),
                            )
                            if (index < stops.lastIndex) {
                                Box(
                                    modifier = Modifier
                                        .width(2.dp)
                                        .height(22.dp)
                                        .background(lab.fgTertiary.copy(alpha = 0.35f)),
                                )
                            }
                        }
                        Column(modifier = Modifier.padding(start = 10.dp)) {
                            Text(
                                text = stop.title.ifBlank { stop.id },
                                fontSize = 14.sp,
                                fontWeight = FontWeight.SemiBold,
                                color = lab.fg,
                            )
                            Text(
                                text = stop.state.name.replace('_', ' '),
                                fontSize = 11.sp,
                                fontWeight = FontWeight.Bold,
                                fontFamily = FontFamily.Monospace,
                                color = if (stop.firstClass) lab.destructive else lab.fgSecondary,
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
fun FieldMoneyStrip(counts: MoneyHealthCounts, modifier: Modifier = Modifier) {
    val lab = LocalPegasusColors.current
    Row(modifier = modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        listOf(
            "CASH" to counts.pendingCash,
            "FISCAL" to counts.openFiscal,
            "CREDIT" to counts.creditLeave,
        ).forEach { (label, count) ->
            PegasusCard(modifier = Modifier.weight(1f)) {
                Column(Modifier.padding(10.dp)) {
                    Text(label, fontSize = 10.sp, fontWeight = FontWeight.Black, fontFamily = FontFamily.Monospace, color = lab.fgTertiary)
                    Text(
                        if (count == 0) "empty" else count.toString(),
                        fontSize = 13.sp,
                        fontWeight = FontWeight.SemiBold,
                        color = if (count == 0) lab.fgTertiary else lab.fg,
                    )
                }
            }
        }
    }
}

@Composable
fun ManifestBulletMeter(
    state: String?,
    usedVU: Double?,
    maxVU: Double,
    modifier: Modifier = Modifier,
) {
    val lab = LocalPegasusColors.current
    PegasusCard(modifier = modifier.fillMaxWidth()) {
        Column(Modifier.padding(PegasusSpacing.s16), verticalArrangement = Arrangement.spacedBy(6.dp)) {
            Text("MANIFEST", fontSize = 11.sp, fontWeight = FontWeight.Black, fontFamily = FontFamily.Monospace, color = lab.fgTertiary)
            Text(
                if (state.isNullOrBlank()) "unavailable" else state,
                fontSize = 13.sp,
                fontWeight = FontWeight.Bold,
                fontFamily = FontFamily.Monospace,
                color = lab.fg,
            )
            if (usedVU != null && maxVU > 0) {
                LinearProgressIndicator(
                    progress = (usedVU / maxVU).toFloat().coerceIn(0f, 1f),
                    modifier = Modifier.fillMaxWidth(),
                )
                Text("%.0f / %.0f VU".format(usedVU, maxVU), fontSize = 11.sp, fontFamily = FontFamily.Monospace, color = lab.fgSecondary)
            } else {
                Text("VU unavailable", fontSize = 11.sp, color = lab.fgTertiary)
            }
        }
    }
}
