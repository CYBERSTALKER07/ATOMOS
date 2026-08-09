package com.pegasusx.driver.ui.screens.offload.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.HourglassTop
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.driver.ui.theme.LocalPegasusColors
import com.pegasusx.driver.ui.theme.formattedAmount
import com.pegasusx.driver.R

@Composable
fun FiscalizingView(amount: Long) {
    val lab = LocalPegasusColors.current
    Icon(
        imageVector = Icons.Default.HourglassTop,
        contentDescription = null,
        tint = lab.fgTertiary,
        modifier = Modifier.size(80.dp)
    )
    Spacer(modifier = Modifier.height(24.dp))
    Text(
        text = "FISCALIZING",
        fontSize = 11.sp,
        fontWeight = FontWeight.Black,
        fontFamily = FontFamily.Monospace,
        color = lab.fgTertiary,
        letterSpacing = 1.5.sp
    )
    Spacer(modifier = Modifier.height(12.dp))
    Text(
        text = amount.formattedAmount(),
        fontSize = 34.sp,
        fontWeight = FontWeight.Bold,
        color = lab.fg
    )
    Spacer(modifier = Modifier.height(16.dp))
    CircularProgressIndicator(
        color = lab.fg,
        modifier = Modifier.size(28.dp),
        strokeWidth = 2.dp
    )
    Spacer(modifier = Modifier.height(16.dp))
    Text(
        text = stringResource(R.string.mobile_driver_ui_cash_captured_waiting_for_fiscal_receipt),
        fontSize = 14.sp,
        color = lab.fgTertiary,
        textAlign = TextAlign.Center
    )
}
