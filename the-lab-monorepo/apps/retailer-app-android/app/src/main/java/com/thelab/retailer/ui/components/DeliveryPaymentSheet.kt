package com.thelab.retailer.ui.components

import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material.icons.rounded.Close
import androidx.compose.material.icons.rounded.CreditCard
import androidx.compose.material.icons.rounded.LocalAtm
import androidx.compose.material.icons.rounded.Payments
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.window.Dialog
import androidx.compose.material3.Surface
import com.thelab.retailer.data.api.RetailerWSMessage
import com.thelab.retailer.ui.theme.StatusGreen
import com.thelab.retailer.ui.theme.StatusGreenSoft
import com.thelab.retailer.ui.theme.StatusOrange
import com.thelab.retailer.ui.theme.StatusOrangeSoft
import com.thelab.retailer.ui.theme.StatusRed
import com.thelab.retailer.ui.theme.StatusRedSoft

enum class PaymentPhase { CHOOSE, PROCESSING, CASH_PENDING, SUCCESS, FAILED }

private data class CardGatewayOption(
    val gateway: String,
    val label: String,
    val description: String,
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DeliveryPaymentSheet(
    event: RetailerWSMessage,
    phase: PaymentPhase,
    errorMessage: String?,
    isCompact: Boolean = true,
    onSelectCash: () -> Unit,
    onSelectCard: (gateway: String) -> Unit,
    onRetry: () -> Unit,
    onDismiss: () -> Unit,
) {
    if (isCompact) {
        val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
        ModalBottomSheet(
            onDismissRequest = {
                if (phase == PaymentPhase.CHOOSE || phase == PaymentPhase.FAILED) onDismiss()
            },
            sheetState = sheetState,
            shape = RoundedCornerShape(topStart = 32.dp, topEnd = 32.dp),
            containerColor = MaterialTheme.colorScheme.surface,
            tonalElevation = 0.dp,
            dragHandle = { Spacer(Modifier.height(32.dp)) }
        ) {
            DeliveryPaymentSheetContent(event, phase, errorMessage, onSelectCash, onSelectCard, onRetry, onDismiss)
        }
    } else {
        Dialog(onDismissRequest = {
            if (phase == PaymentPhase.CHOOSE || phase == PaymentPhase.FAILED) onDismiss()
        }) {
            Surface(
                shape = RoundedCornerShape(32.dp),
                color = MaterialTheme.colorScheme.surface,
                tonalElevation = 0.dp
            ) {
                Column(Modifier.padding(vertical = 32.dp)) {
                    DeliveryPaymentSheetContent(event, phase, errorMessage, onSelectCash, onSelectCard, onRetry, onDismiss)
                }
            }
        }
    }
}

@Composable
fun DeliveryPaymentSheetContent(
    event: RetailerWSMessage,
    phase: PaymentPhase,
    errorMessage: String?,
    onSelectCash: () -> Unit,
    onSelectCard: (gateway: String) -> Unit,
    onRetry: () -> Unit,
    onDismiss: () -> Unit,
) {
    AnimatedContent(
        targetState = phase,
        transitionSpec = { fadeIn() togetherWith fadeOut() },
        label = "payment_phase",
    ) { currentPhase ->
        when (currentPhase) {
            PaymentPhase.CHOOSE -> ChooseContent(event, onSelectCash, onSelectCard)
            PaymentPhase.PROCESSING -> ProcessingContent()
            PaymentPhase.CASH_PENDING -> CashPendingContent(event)
            PaymentPhase.SUCCESS -> SuccessContent(event, onDismiss)
            PaymentPhase.FAILED -> FailedContent(errorMessage, onRetry, onDismiss)
        }
    }
}

@Composable
private fun ChooseContent(
    event: RetailerWSMessage,
    onSelectCash: () -> Unit,
    onSelectCard: (String) -> Unit,
) {
    val cardGatewayOptions = resolveCardGatewayOptions(event)

    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 20.dp)
            .padding(bottom = 32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(
            modifier = Modifier
                .size(72.dp)
                .clip(CircleShape)
                .background(StatusOrangeSoft),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                Icons.Rounded.Payments,
                contentDescription = null,
                modifier = Modifier.size(32.dp),
                tint = StatusOrange,
            )
        }
        Spacer(Modifier.height(20.dp))

        Text(
            "Payment Required",
            style = MaterialTheme.typography.headlineSmall.copy(fontWeight = FontWeight.Bold),
        )
        Spacer(Modifier.height(8.dp))

        Text(
            "Amount Due",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f),
        )
        Spacer(Modifier.height(4.dp))

        // Show crossed-out original amount if items were rejected during offload
        val wasAmended = event.originalAmount > 0 && event.originalAmount != event.amount
        if (wasAmended) {
            Text(
                "${event.originalAmount}",
                style = MaterialTheme.typography.titleMedium.copy(
                    textDecoration = TextDecoration.LineThrough,
                ),
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.4f),
            )
            Spacer(Modifier.height(2.dp))
        }

        Text(
            "${event.amount}",
            style = MaterialTheme.typography.displaySmall.copy(fontWeight = FontWeight.Bold),
            color = if (wasAmended) StatusOrange else MaterialTheme.colorScheme.onSurface,
        )
        Spacer(Modifier.height(4.dp))
        Text(
            "Order #${event.orderId.takeLast(6)}",
            style = MaterialTheme.typography.bodySmall.copy(fontSize = 12.sp),
            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.4f),
        )
        Spacer(Modifier.height(24.dp))

        Text(
            "Choose Payment Method",
            style = MaterialTheme.typography.titleSmall.copy(fontWeight = FontWeight.SemiBold),
            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f),
        )
        Spacer(Modifier.height(12.dp))

        PaymentOptionRow(
            icon = Icons.Rounded.LocalAtm,
            label = "Cash on Delivery",
            description = "Pay the driver in cash",
            onClick = onSelectCash,
        )
        cardGatewayOptions.forEach { option ->
            Spacer(Modifier.height(8.dp))
            PaymentOptionRow(
                icon = Icons.Rounded.CreditCard,
                label = option.label,
                description = option.description,
                onClick = { onSelectCard(option.gateway) },
            )
        }
    }
}

private fun resolveCardGatewayOptions(event: RetailerWSMessage): List<CardGatewayOption> {
    val gateways = event.availableCardGateways
        .mapNotNull(::normalizeCardGateway)
        .distinct()
        .ifEmpty { listOf("GLOBAL_PAY", "UZCARD", "CASH") }

    return gateways.mapNotNull { gateway ->
        when (gateway) {
            
            
            "GLOBAL_PAY" -> CardGatewayOption(gateway, "GlobalPay", "Pay via GlobalPay checkout")
            else -> null
        }
    }
}

private fun normalizeCardGateway(gateway: String): String? {
    return when (gateway.trim().uppercase()) {
        "GLOBAL_PAY", "UZCARD", "CASH" -> gateway.trim().uppercase()
        else -> null
    }
}

@Composable
private fun PaymentOptionRow(
    icon: ImageVector,
    label: String,
    description: String,
    onClick: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(12.dp))
            .border(
                width = 1.dp,
                color = MaterialTheme.colorScheme.outlineVariant,
                shape = RoundedCornerShape(12.dp),
            )
            .clickable(onClick = onClick)
            .padding(16.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Box(
            modifier = Modifier
                .size(40.dp)
                .clip(CircleShape)
                .background(MaterialTheme.colorScheme.surfaceVariant),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                icon,
                contentDescription = null,
                modifier = Modifier.size(20.dp),
                tint = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
        Spacer(Modifier.width(12.dp))
        Column(modifier = Modifier.weight(1f)) {
            Text(
                label,
                style = MaterialTheme.typography.titleSmall.copy(fontWeight = FontWeight.SemiBold),
            )
            Text(
                description,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f),
            )
        }
    }
}

@Composable
private fun ProcessingContent() {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 20.dp)
            .padding(vertical = 48.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        CircularProgressIndicator(
            modifier = Modifier.size(48.dp),
            strokeWidth = 4.dp,
            strokeCap = StrokeCap.Round,
        )
        Text(
            "Processing...",
            style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.Bold),
        )
        Text(
            "Connecting to payment service",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f),
        )
    }
}

@Composable
private fun CashPendingContent(event: RetailerWSMessage) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 20.dp)
            .padding(bottom = 32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(
            modifier = Modifier
                .size(72.dp)
                .clip(CircleShape)
                .background(StatusOrangeSoft),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                Icons.Rounded.LocalAtm,
                contentDescription = null,
                modifier = Modifier.size(36.dp),
                tint = StatusOrange,
            )
        }
        Spacer(Modifier.height(20.dp))
        Text(
            "Cash Collection Pending",
            style = MaterialTheme.typography.headlineSmall.copy(fontWeight = FontWeight.Bold),
        )
        Spacer(Modifier.height(8.dp))
        Text(
            "${event.amount}",
            style = MaterialTheme.typography.titleLarge.copy(fontWeight = FontWeight.SemiBold),
            color = StatusOrange,
        )
        Spacer(Modifier.height(12.dp))
        Text(
            "Please have the cash ready.\nThe driver will collect it shortly.",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f),
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(24.dp))

        Row(
            modifier = Modifier
                .clip(CircleShape)
                .background(MaterialTheme.colorScheme.surfaceVariant)
                .padding(horizontal = 16.dp, vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            CircularProgressIndicator(
                modifier = Modifier.size(14.dp),
                strokeWidth = 2.dp,
                strokeCap = StrokeCap.Round,
            )
            Text(
                "Waiting for driver confirmation",
                style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.Medium),
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f),
            )
        }
    }
}

@Composable
private fun SuccessContent(event: RetailerWSMessage, onDismiss: () -> Unit) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 20.dp)
            .padding(bottom = 32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(
            modifier = Modifier
                .size(72.dp)
                .clip(CircleShape)
                .background(StatusGreenSoft),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                Icons.Rounded.Check,
                contentDescription = null,
                modifier = Modifier.size(36.dp),
                tint = StatusGreen,
            )
        }
        Spacer(Modifier.height(20.dp))
        Text(
            "Payment Complete",
            style = MaterialTheme.typography.headlineSmall.copy(fontWeight = FontWeight.Bold),
        )
        Spacer(Modifier.height(8.dp))
        Text(
            "${event.amount}",
            style = MaterialTheme.typography.titleLarge.copy(fontWeight = FontWeight.SemiBold),
            color = StatusGreen,
        )
        Spacer(Modifier.height(32.dp))
        Button(
            onClick = onDismiss,
            modifier = Modifier
                .fillMaxWidth()
                .height(52.dp),
            colors = ButtonDefaults.buttonColors(containerColor = StatusGreen),
        ) {
            Text("Done", fontWeight = FontWeight.Bold)
        }
    }
}

@Composable
private fun FailedContent(errorMessage: String?, onRetry: () -> Unit, onDismiss: () -> Unit) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 20.dp)
            .padding(bottom = 32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(
            modifier = Modifier
                .size(72.dp)
                .clip(CircleShape)
                .background(StatusRedSoft),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                Icons.Rounded.Close,
                contentDescription = null,
                modifier = Modifier.size(36.dp),
                tint = StatusRed,
            )
        }
        Spacer(Modifier.height(20.dp))
        Text(
            "Payment Failed",
            style = MaterialTheme.typography.headlineSmall.copy(fontWeight = FontWeight.Bold),
        )
        if (errorMessage != null) {
            Spacer(Modifier.height(8.dp))
            Text(
                errorMessage,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f),
                textAlign = TextAlign.Center,
            )
        }
        Spacer(Modifier.height(32.dp))
        Button(
            onClick = onRetry,
            modifier = Modifier
                .fillMaxWidth()
                .height(52.dp),
            colors = ButtonDefaults.buttonColors(
                containerColor = MaterialTheme.colorScheme.onSurface,
                contentColor = MaterialTheme.colorScheme.surface,
            ),
        ) {
            Text("Retry", fontWeight = FontWeight.Bold)
        }
        Spacer(Modifier.height(12.dp))
        OutlinedButton(
            onClick = onDismiss,
            modifier = Modifier.fillMaxWidth().height(48.dp),
        ) {
            Text("Cancel")
        }
    }
}
