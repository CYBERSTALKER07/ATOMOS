package com.pegasusx.driver.ui.screens.offload.components

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.driver.ui.theme.LocalPegasusColors
import com.pegasusx.driver.ui.theme.StatusGreen
import com.pegasusx.driver.ui.theme.StatusRed
import com.pegasusx.driver.ui.theme.formattedAmount

@Composable
fun OffloadSummaryCard(
    originalTotal: Double,
    adjustedTotal: Double,
    hasRejections: Boolean
) {
    val lab = LocalPegasusColors.current
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(lab.card)
            .padding(horizontal = 16.dp, vertical = 12.dp),
        horizontalArrangement = Arrangement.SpaceBetween
    ) {
        Column {
            Text("Original", fontSize = 10.sp, color = lab.fgTertiary, fontFamily = FontFamily.Monospace)
            Text(originalTotal.formattedAmount(), fontSize = 14.sp, fontWeight = FontWeight.Bold, color = lab.fg)
        }
        Column(horizontalAlignment = Alignment.End) {
            Text("Adjusted", fontSize = 10.sp, color = lab.fgTertiary, fontFamily = FontFamily.Monospace)
            Text(
                adjustedTotal.formattedAmount(),
                fontSize = 14.sp,
                fontWeight = FontWeight.Bold,
                color = if (hasRejections) StatusRed else StatusGreen
            )
        }
    }
}
