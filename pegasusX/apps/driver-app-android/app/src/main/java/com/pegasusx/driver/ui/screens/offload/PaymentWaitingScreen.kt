package com.pegasusx.driver.ui.screens.offload

import androidx.compose.ui.res.stringResource

import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.CreditCard
import androidx.compose.material.icons.filled.HourglassTop
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import com.pegasusx.driver.ui.theme.LocalPegasusColors
import com.pegasusx.driver.ui.theme.MotionTokens
import com.pegasusx.driver.ui.theme.StatusGreen
import com.pegasusx.driver.ui.theme.StatusRed
import com.pegasusx.driver.ui.theme.formattedAmount
import com.pegasusx.driver.R

@Composable
fun PaymentWaitingScreen(
    onComplete: () -> Unit,
    onCashCollectionRequired: (orderId: String, amount: Long) -> Unit = { _, _ -> },
    viewModel: PaymentWaitingViewModel = hiltViewModel()
) {
    val state by viewModel.state.collectAsState()
    val lab = LocalPegasusColors.current

    LaunchedEffect(Unit) {
        viewModel.cashCollectionRequired.collect {
            onCashCollectionRequired(state.orderId, state.amount)
        }
    }

    if (state.completed) {
        onComplete()
        return
    }

    val pulseTransition = rememberInfiniteTransition(label = stringResource(R.string.mobile_driver_ui_pulse))
    val pulseAlpha by pulseTransition.animateFloat(
        initialValue = 0.4f,
        targetValue = 1f,
        animationSpec = infiniteRepeatable(tween(MotionTokens.DurationExtraLong4), RepeatMode.Reverse),
        label = "pulse_alpha"
    )

    val title = when {
        state.fiscalFailed -> "FISCAL FAILED"
        state.fiscalizing -> "FISCALIZING"
        state.paymentSettled -> "PAYMENT RECEIVED"
        else -> "AWAITING PAYMENT"
    }
    val subtitle = when {
        state.fiscalFailed -> "Fiscal receipt failed. Retry or call supervisor for force-complete."
        state.fiscalizing -> "Payment captured. Waiting for fiscal receipt…"
        state.paymentSettled -> "Finalizing delivery…"
        else -> "Waiting for retailer to complete payment..."
    }
    val icon = when {
        state.fiscalFailed -> Icons.Default.Warning
        state.fiscalizing -> Icons.Default.HourglassTop
        state.paymentSettled -> Icons.Default.CheckCircle
        else -> Icons.Default.HourglassTop
    }
    val iconTint = when {
        state.fiscalFailed -> StatusRed
        state.paymentSettled && !state.fiscalizing -> StatusGreen
        else -> lab.fgTertiary
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(lab.bg)
            .padding(32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        Icon(
            imageVector = icon,
            contentDescription = null,
            tint = iconTint,
            modifier = Modifier
                .size(80.dp)
                .alpha(if (state.paymentSettled && !state.fiscalizing && !state.fiscalFailed) 1f else pulseAlpha)
        )

        Spacer(modifier = Modifier.height(24.dp))

        Text(
            text = title,
            fontSize = 11.sp,
            fontWeight = FontWeight.Black,
            fontFamily = FontFamily.Monospace,
            color = if (state.fiscalFailed) StatusRed else if (state.paymentSettled && !state.fiscalizing) StatusGreen else lab.fgTertiary,
            letterSpacing = 1.5.sp
        )

        Spacer(modifier = Modifier.height(12.dp))

        Text(
            text = state.amount.formattedAmount(),
            fontSize = 34.sp,
            fontWeight = FontWeight.Bold,
            color = lab.fg
        )

        Spacer(modifier = Modifier.height(8.dp))

        Icon(
            imageVector = Icons.Default.CreditCard,
            contentDescription = null,
            tint = lab.fgTertiary,
            modifier = Modifier.size(20.dp)
        )

        Spacer(modifier = Modifier.height(4.dp))

        Text(
            text = stringResource(R.string.mobile_driver_ui_card_terminal),
            fontSize = 13.sp,
            color = lab.fgTertiary
        )

        Spacer(modifier = Modifier.height(32.dp))
        Text(
            text = subtitle,
            fontSize = 13.sp,
            color = lab.fgTertiary,
            textAlign = TextAlign.Center
        )

        if (state.fiscalizing || state.isCompleting) {
            Spacer(modifier = Modifier.height(24.dp))
            CircularProgressIndicator(
                color = lab.fg,
                modifier = Modifier.size(28.dp),
                strokeWidth = 2.dp
            )
        }

        state.error?.let { error ->
            Spacer(modifier = Modifier.height(12.dp))
            Text(text = error, color = StatusRed, fontSize = 12.sp, textAlign = TextAlign.Center)
        }

        if (state.fiscalFailed) {
            Spacer(modifier = Modifier.height(24.dp))
            Button(
                onClick = { viewModel.retryFiscal() },
                enabled = !state.isCompleting,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(52.dp),
                shape = MaterialTheme.shapes.medium,
                colors = ButtonDefaults.buttonColors(containerColor = StatusGreen),
            ) {
                Text(text = stringResource(R.string.mobile_driver_ui_retry_fiscal), fontWeight = FontWeight.Bold, fontSize = 15.sp)
            }
        } else if (state.error != null && state.paymentSettled && !state.fiscalizing) {
            Spacer(modifier = Modifier.height(24.dp))
            Button(
                onClick = { viewModel.completeOrder() },
                enabled = !state.isCompleting,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(52.dp),
                shape = MaterialTheme.shapes.medium,
                colors = ButtonDefaults.buttonColors(containerColor = StatusGreen),
            ) {
                Text(text = stringResource(R.string.mobile_driver_ui_retry_capture), fontWeight = FontWeight.Bold, fontSize = 15.sp)
            }
        }
    }
}
