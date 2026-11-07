package com.pegasusx.driver.ui.screens.offload.components

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.driver.ui.theme.LocalPegasusColors
import com.pegasusx.driver.ui.theme.StatusGreen
import com.pegasusx.driver.ui.theme.StatusRed
import com.pegasusx.driver.R

@Composable
fun FiscalFailedView(
    error: String?,
    isCompleting: Boolean,
    onRetryFiscal: () -> Unit
) {
    val lab = LocalPegasusColors.current
    Icon(
        imageVector = Icons.Default.Warning,
        contentDescription = null,
        tint = StatusRed,
        modifier = Modifier.size(80.dp)
    )
    Spacer(modifier = Modifier.height(24.dp))
    Text(
        text = stringResource(R.string.mobile_driver_ui_fiscal_failed),
        fontSize = 11.sp,
        fontWeight = FontWeight.Black,
        fontFamily = FontFamily.Monospace,
        color = StatusRed,
        letterSpacing = 1.5.sp
    )
    Spacer(modifier = Modifier.height(12.dp))
    Text(
        text = stringResource(R.string.mobile_driver_ui_retry_fiscal_receipt_or_call_supervisor_for_force_complete),
        fontSize = 14.sp,
        color = lab.fgTertiary,
        textAlign = TextAlign.Center
    )
    error?.let { err ->
        Spacer(modifier = Modifier.height(12.dp))
        Text(text = err, color = StatusRed, fontSize = 12.sp, textAlign = TextAlign.Center)
    }
    Spacer(modifier = Modifier.height(32.dp))
    Button(
        onClick = onRetryFiscal,
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
            Text(text = stringResource(R.string.mobile_driver_ui_retry_fiscal), fontWeight = FontWeight.Bold, fontSize = 15.sp)
        }
    }
}
