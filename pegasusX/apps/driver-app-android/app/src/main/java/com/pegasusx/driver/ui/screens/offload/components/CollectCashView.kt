package com.pegasusx.driver.ui.screens.offload.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Payments
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.driver.ui.components.DriverGpsBanner
import com.pegasusx.driver.ui.theme.LocalPegasusColors
import com.pegasusx.driver.ui.theme.StatusGreen
import com.pegasusx.driver.ui.theme.StatusRed
import com.pegasusx.driver.ui.theme.formattedAmount

@Composable
fun CollectCashView(
    amount: Long,
    amountReceivedMinor: Long,
    amountReceivedInput: String,
    onAmountReceivedChanged: (String) -> Unit,
    shortfallMinor: Long,
    overageMinor: Long,
    error: String?,
    isCompleting: Boolean,
    cashReceived: Boolean,
    onRecordSplitPayment: () -> Unit,
    onCollectCash: () -> Unit,
    onAcknowledgeCashReceived: () -> Unit
) {
    val lab = LocalPegasusColors.current
    Icon(
        imageVector = Icons.Default.Payments,
        contentDescription = null,
        tint = StatusGreen,
        modifier = Modifier.size(80.dp)
    )
    Spacer(modifier = Modifier.height(24.dp))
    Text(
        text = stringResource(R.string.mobile_driver_ui_collect_cash),
        fontSize = 11.sp,
        fontWeight = FontWeight.Black,
        fontFamily = FontFamily.Monospace,
        color = lab.fgTertiary,
        letterSpacing = 1.5.sp
    )
    Spacer(modifier = Modifier.height(12.dp))
    Text(
        text = stringResource(R.string.mobile_driver_ui_expected_formattedamount, amount.formattedAmount()),
        fontSize = 18.sp,
        fontWeight = FontWeight.SemiBold,
        color = lab.fgTertiary
    )
    Spacer(modifier = Modifier.height(8.dp))
    Text(
        text = amountReceivedMinor.formattedAmount(),
        fontSize = 38.sp,
        fontWeight = FontWeight.Bold,
        color = lab.fg
    )
    Spacer(modifier = Modifier.height(12.dp))
    OutlinedTextField(
        value = amountReceivedInput,
        onValueChange = onAmountReceivedChanged,
        label = { Text("Amount received (tiyin)") },
        singleLine = true,
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
        modifier = Modifier.fillMaxWidth(),
        enabled = !isCompleting,
    )
    if (shortfallMinor > 0) {
        Spacer(modifier = Modifier.height(8.dp))
        Text(
            text = stringResource(R.string.mobile_driver_ui_shortfall_formattedamount_fiscal_uses_received_amount, shortfallMinor.formattedAmount()),
            fontSize = 13.sp,
            color = StatusRed,
            textAlign = TextAlign.Center,
        )
    } else if (overageMinor > 0) {
        Spacer(modifier = Modifier.height(8.dp))
        Text(
            text = stringResource(R.string.mobile_driver_ui_overage_formattedamount_recorded, overageMinor.formattedAmount()),
            fontSize = 13.sp,
            color = lab.fgTertiary,
            textAlign = TextAlign.Center,
        )
    }
    Spacer(modifier = Modifier.height(16.dp))
    Text(
        text = stringResource(R.string.mobile_driver_ui_enter_cash_actually_taken_fiscal_receipt_uses_this_amount_not_or),
        fontSize = 14.sp,
        color = lab.fgTertiary,
        textAlign = TextAlign.Center
    )
    error?.let { err ->
        Spacer(modifier = Modifier.height(12.dp))
        if (err.contains("GPS", ignoreCase = true)) {
            DriverGpsBanner(
                message = err,
                modifier = Modifier.fillMaxWidth(),
            )
        } else {
            Text(text = err, color = StatusRed, fontSize = 12.sp, textAlign = TextAlign.Center)
        }
    }
    Spacer(modifier = Modifier.height(48.dp))
    OutlinedButton(
        onClick = onRecordSplitPayment,
        enabled = !isCompleting,
        modifier = Modifier
            .fillMaxWidth()
            .height(48.dp),
        shape = MaterialTheme.shapes.medium,
    ) {
        Text(
            text = stringResource(R.string.mobile_driver_ui_split_payment_pay_now_pay_later),
            fontWeight = FontWeight.Medium,
            fontSize = 14.sp
        )
    }
    Spacer(modifier = Modifier.height(12.dp))
    Button(
        onClick = {
            if (cashReceived) {
                onCollectCash()
            } else {
                onAcknowledgeCashReceived()
            }
        },
        enabled = !isCompleting,
        modifier = Modifier
            .fillMaxWidth()
            .height(56.dp),
        shape = MaterialTheme.shapes.medium,
        colors = ButtonDefaults.buttonColors(containerColor = StatusGreen)
    ) {
        if (isCompleting) {
            CircularProgressIndicator(
                color = MaterialTheme.colorScheme.onPrimary,
                modifier = Modifier.size(20.dp),
                strokeWidth = 2.dp
            )
        } else {
            Text(
                text = if (cashReceived) "Confirm cash capture" else "Cash Received",
                fontWeight = FontWeight.Bold,
                fontSize = 15.sp
            )
        }
    }
}
