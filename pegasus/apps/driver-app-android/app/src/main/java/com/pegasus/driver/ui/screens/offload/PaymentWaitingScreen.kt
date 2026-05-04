package com.pegasus.driver.ui.screens.offload

import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.CreditCard
import androidx.compose.material.icons.filled.HourglassTop
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import com.pegasus.driver.ui.theme.LocalPegasusColors
import com.pegasus.driver.ui.theme.MotionTokens
import com.pegasus.driver.ui.theme.StatusGreen
import com.pegasus.driver.ui.theme.StatusRed
import com.pegasus.driver.ui.theme.formattedAmount

@Composable
fun PaymentWaitingScreen(
    onComplete: () -> Unit,
    viewModel: PaymentWaitingViewModel = hiltViewModel()
) {
    val state by viewModel.state.collectAsState()
    val lab = LocalPegasusColors.current

    if (state.completed) {
        onComplete()
        return
    }

    val pulseTransition = rememberInfiniteTransition(label = "pulse")
    val pulseAlpha by pulseTransition.animateFloat(
        initialValue = 0.4f,
        targetValue = 1f,
        animationSpec = infiniteRepeatable(tween(MotionTokens.DurationExtraLong4), RepeatMode.Reverse),
        label = "pulse_alpha"
    )

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(lab.bg)
            .padding(32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        Icon(
            imageVector = if (state.paymentSettled) Icons.Default.CheckCircle else Icons.Default.HourglassTop,
            contentDescription = null,
            tint = if (state.paymentSettled) StatusGreen else lab.fgTertiary,
            modifier = Modifier
                .size(80.dp)
                .alpha(if (state.paymentSettled) 1f else pulseAlpha)
        )

        Spacer(modifier = Modifier.height(24.dp))

        Text(
            text = if (state.paymentSettled) "PAYMENT RECEIVED" else "AWAITING PAYMENT",
            fontSize = 11.sp,
            fontWeight = FontWeight.Black,
            fontFamily = FontFamily.Monospace,
            color = if (state.paymentSettled) StatusGreen else lab.fgTertiary,
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
            text = "Payme",
            fontSize = 13.sp,
            color = lab.fgTertiary
        )

        if (!state.paymentSettled) {
            Spacer(modifier = Modifier.height(32.dp))
            Text(
                text = "Waiting for retailer to complete payment...",
                fontSize = 13.sp,
                color = lab.fgTertiary,
                textAlign = TextAlign.Center
            )
        }

        state.error?.let { error ->
            Spacer(modifier = Modifier.height(12.dp))
            Text(text = error, color = StatusRed, fontSize = 12.sp, textAlign = TextAlign.Center)
        }

        Spacer(modifier = Modifier.height(40.dp))

        Button(
            onClick = { viewModel.completeOrder() },
            enabled = state.paymentSettled && !state.isCompleting,
            modifier = Modifier
                .fillMaxWidth()
                .height(52.dp),
            shape = MaterialTheme.shapes.medium,
            colors = ButtonDefaults.buttonColors(
                containerColor = StatusGreen,
                disabledContainerColor = MaterialTheme.colorScheme.surfaceContainerLow
            )
        ) {
            if (state.isCompleting) {
                CircularProgressIndicator(
                    color = MaterialTheme.colorScheme.onPrimary,
                    modifier = Modifier.size(20.dp),
                    strokeWidth = 2.dp
                )
            } else {
                Text(
                    text = "Complete Delivery",
                    fontWeight = FontWeight.Bold,
                    fontSize = 15.sp
                )
            }
        }
    }
}
