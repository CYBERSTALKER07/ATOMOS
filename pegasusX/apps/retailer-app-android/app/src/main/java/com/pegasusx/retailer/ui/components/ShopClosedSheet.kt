package com.pegasusx.retailer.ui.components

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.slideOutVertically
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Warning
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.pegasusx.retailer.data.api.ShopClosedAlert
import com.pegasusx.retailer.ui.theme.MotionTokens
import com.pegasusx.retailer.ui.theme.SoftSquircleShape

private fun shopClosedOptionLabel(option: String): String = when (option) {
    "OPEN_NOW" -> "I am open now"
    "5_MIN" -> "I will be back in 5 mins"
    "CALL_ME" -> "Call me"
    "CLOSED_TODAY" -> "Closed for the day"
    else -> option
}

@Composable
fun ShopClosedSheet(
    alert: ShopClosedAlert,
    isSubmitting: Boolean,
    errorMessage: String?,
    onRespond: (String) -> Unit,
) {
    AnimatedVisibility(
        visible = true,
        enter = fadeIn(tween(MotionTokens.DurationShort4)) +
            slideInVertically(
                animationSpec = tween(MotionTokens.DurationMedium2, easing = MotionTokens.EasingEmphasizedDecelerate),
            ) { it / 3 },
        exit = fadeOut(tween(MotionTokens.DurationShort2)) +
            slideOutVertically(
                animationSpec = tween(MotionTokens.DurationShort4, easing = MotionTokens.EasingEmphasizedAccelerate),
            ) { it / 3 },
    ) {
        Box(
            modifier = Modifier
                .fillMaxSize()
                .background(MaterialTheme.colorScheme.scrim.copy(alpha = 0.45f)),
            contentAlignment = Alignment.BottomCenter,
        ) {
            Surface(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 12.dp, vertical = 16.dp),
                shape = SoftSquircleShape,
                color = MaterialTheme.colorScheme.surface,
                tonalElevation = 6.dp,
            ) {
                Column(
                    modifier = Modifier.padding(20.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                ) {
                    Icon(
                        Icons.Rounded.Warning,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.tertiary,
                    )
                    Text(
                        text = "Driver ${alert.driverName} reported your shop is closed.",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                    )
                    Text(
                        text = "Confirm your status so we can manage order #${alert.orderId.takeLast(6)}.",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    if (!errorMessage.isNullOrBlank()) {
                        Text(
                            text = errorMessage,
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.error,
                        )
                    }
                    alert.options.forEach { option ->
                        Button(
                            onClick = { onRespond(option) },
                            enabled = !isSubmitting,
                            modifier = Modifier.fillMaxWidth(),
                        ) {
                            if (isSubmitting) {
                                CircularProgressIndicator(
                                    modifier = Modifier.padding(end = 8.dp),
                                    strokeWidth = 2.dp,
                                )
                            }
                            Text(shopClosedOptionLabel(option))
                        }
                    }
                }
            }
        }
    }
}
