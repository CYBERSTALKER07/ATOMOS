package com.pegasusx.driver.ui.screens.offload.components

import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CloudQueue
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.pegasusx.driver.R
import com.pegasusx.driver.ui.theme.LocalPegasusColors
import com.pegasusx.driver.ui.theme.StatusGreen
import com.pegasusx.driver.ui.theme.StatusRed

@Composable
fun FiscalFailedView(
    error: String?,
    statusNotice: String? = null,
    isCompleting: Boolean,
    onRetryFiscal: () -> Unit,
    onContinueNextStop: () -> Unit = {},
) {
    val lab = LocalPegasusColors.current
    Icon(
        imageVector = Icons.Default.CloudQueue,
        contentDescription = null,
        tint = StatusGreen,
        modifier = Modifier.size(80.dp)
    )
    Spacer(modifier = Modifier.height(24.dp))
    Text(
        text = stringResource(R.string.fiscal_retry_queued_title),
        fontSize = 11.sp,
        fontWeight = FontWeight.Black,
        fontFamily = FontFamily.Monospace,
        color = StatusGreen,
        letterSpacing = 1.5.sp
    )
    Spacer(modifier = Modifier.height(12.dp))
    Text(
        text = statusNotice?.takeIf { it.isNotBlank() }
            ?: stringResource(R.string.fiscal_retry_queued_message),
        fontSize = 14.sp,
        color = lab.fgTertiary,
        textAlign = TextAlign.Center
    )
    error?.let { err ->
        Spacer(modifier = Modifier.height(12.dp))
        Text(text = err, color = StatusRed, fontSize = 12.sp, textAlign = TextAlign.Center)
    }
    Spacer(modifier = Modifier.height(28.dp))
    Button(
        onClick = onContinueNextStop,
        enabled = !isCompleting,
        modifier = Modifier
            .fillMaxWidth()
            .height(56.dp),
        shape = MaterialTheme.shapes.medium,
        colors = ButtonDefaults.buttonColors(containerColor = StatusGreen)
    ) {
        Text(
            text = stringResource(R.string.fiscal_continue_next_stop),
            fontWeight = FontWeight.Bold,
            fontSize = 15.sp,
        )
    }
    Spacer(modifier = Modifier.height(12.dp))
    OutlinedButton(
        onClick = onRetryFiscal,
        enabled = !isCompleting,
        modifier = Modifier
            .fillMaxWidth()
            .height(52.dp),
        shape = MaterialTheme.shapes.medium,
    ) {
        if (isCompleting) {
            CircularProgressIndicator(
                color = StatusGreen,
                modifier = Modifier.size(20.dp),
                strokeWidth = 2.dp
            )
        } else {
            Text(
                text = stringResource(R.string.fiscal_retry_soliq_now),
                fontWeight = FontWeight.SemiBold,
                fontSize = 14.sp
            )
        }
    }
}
