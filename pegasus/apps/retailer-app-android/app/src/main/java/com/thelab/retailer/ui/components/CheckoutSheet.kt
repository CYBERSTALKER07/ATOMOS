package com.pegasus.retailer.ui.components

import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.togetherWith
import com.pegasus.retailer.ui.theme.MotionTokens
import androidx.compose.foundation.background
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
import androidx.compose.foundation.shape.RoundedCornerShape
import com.pegasus.retailer.ui.theme.PillShape
import com.pegasus.retailer.ui.theme.SoftSquircleShape
import com.pegasus.retailer.ui.theme.SquircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material.icons.rounded.Eco
import androidx.compose.material.icons.rounded.KeyboardArrowDown
import androidx.compose.material.icons.rounded.Payment
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

enum class CheckoutPhase { REVIEW, PROCESSING, COMPLETE }

data class CheckoutPaymentOption(
    val gateway: String,
    val label: String,
)

val DefaultCheckoutPaymentOptions = listOf(
    
    
    CheckoutPaymentOption(gateway = "GLOBAL_PAY", label = "GlobalPay"),
    CheckoutPaymentOption(gateway = "CASH", label = "Cash on Delivery"),
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CheckoutSheet(
    phase: CheckoutPhase,
    productName: String,
    itemCount: Int,
    subtotal: String,
    shipping: String,
    discount: String,
    total: String,
    selectedPaymentGateway: String,
    paymentLabel: String,
    paymentOptions: List<CheckoutPaymentOption>,
    onBuy: () -> Unit,
    onSelectPayment: (String) -> Unit,
    onDismiss: () -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
        shape = RoundedCornerShape(topStart = 32.dp, topEnd = 32.dp),
        containerColor = MaterialTheme.colorScheme.surface,
        tonalElevation = 0.dp,
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 20.dp)
                .padding(bottom = 32.dp),
        ) {
            // Title
            Text(
                "Order details",
                style = MaterialTheme.typography.headlineSmall.copy(fontWeight = FontWeight.Bold),
            )

            Spacer(modifier = Modifier.height(4.dp))

            // M3 wavy linear progress (just a simple LinearProgressIndicator stand-in)
            if (phase == CheckoutPhase.REVIEW) {
                androidx.compose.material3.LinearProgressIndicator(
                    modifier = Modifier.fillMaxWidth().height(4.dp).clip(PillShape),
                )
            }

            Spacer(modifier = Modifier.height(20.dp))

            // Animated content between phases
            AnimatedContent(
                targetState = phase,
                transitionSpec = {
                    fadeIn(tween(MotionTokens.DurationMedium2, easing = MotionTokens.EasingEmphasizedDecelerate)) togetherWith
                        fadeOut(tween(MotionTokens.DurationShort2, easing = MotionTokens.EasingEmphasizedAccelerate))
                },
                label = "checkout_phase",
            ) { currentPhase ->
                when (currentPhase) {
                    CheckoutPhase.REVIEW -> ReviewContent(
                        productName = productName,
                        itemCount = itemCount,
                        subtotal = subtotal,
                        shipping = shipping,
                        discount = discount,
                        total = total,
                        selectedPaymentGateway = selectedPaymentGateway,
                        paymentLabel = paymentLabel,
                        paymentOptions = paymentOptions,
                        onBuy = onBuy,
                        onSelectPayment = onSelectPayment,
                    )
                    CheckoutPhase.PROCESSING -> ProcessingContent()
                    CheckoutPhase.COMPLETE -> CompleteContent()
                }
            }
        }
    }
}

@Composable
private fun ReviewContent(
    productName: String,
    itemCount: Int,
    subtotal: String,
    shipping: String,
    discount: String,
    total: String,
    selectedPaymentGateway: String,
    paymentLabel: String,
    paymentOptions: List<CheckoutPaymentOption>,
    onBuy: () -> Unit,
    onSelectPayment: (String) -> Unit,
) {
    var paymentMenuExpanded by remember { mutableStateOf(false) }

    // Product card
    Surface(
        modifier = Modifier.fillMaxWidth(),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surfaceContainerLow,
    ) {
        Row(
            modifier = Modifier.padding(16.dp),
            verticalAlignment = Alignment.Top,
        ) {
            // Product image placeholder
            Box(
                modifier = Modifier
                    .size(100.dp)
                    .clip(SquircleShape)
                    .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.4f)),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    Icons.Rounded.Eco,
                    contentDescription = null,
                    modifier = Modifier.size(36.dp),
                    tint = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.2f),
                )
            }

            Spacer(modifier = Modifier.width(16.dp))

            Column(modifier = Modifier.weight(1f)) {
                Text(
                    if (itemCount > 1) "$itemCount items" else productName,
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f),
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )

                Spacer(modifier = Modifier.height(12.dp))

                // Breakdown rows
                BreakdownRow("Subtotal", subtotal)
                Spacer(modifier = Modifier.height(4.dp))
                BreakdownRow("Shipping", shipping)
                Spacer(modifier = Modifier.height(4.dp))
                BreakdownRow("Discount", discount)

                Spacer(modifier = Modifier.height(12.dp))

                // Total
                Row(verticalAlignment = Alignment.Bottom) {
                    Text(
                        "Total",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f),
                        modifier = Modifier.padding(bottom = 4.dp),
                    )
                    Spacer(modifier = Modifier.width(8.dp))
                    Text(
                        total,
                        style = MaterialTheme.typography.headlineMedium.copy(fontWeight = FontWeight.Bold),
                    )
                }
            }
        }
    }

    Spacer(modifier = Modifier.height(20.dp))

    Text(
        text = "Payment Method",
        style = MaterialTheme.typography.labelLarge,
        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.65f),
    )

    Spacer(modifier = Modifier.height(8.dp))

    // Buy button row
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.Center,
    ) {
        // Buy button
        Surface(
            onClick = onBuy,
            shape = RoundedCornerShape(topStart = 100.dp, bottomStart = 100.dp, topEnd = 0.dp, bottomEnd = 0.dp),
            color = MaterialTheme.colorScheme.primary,
        ) {
            Row(
                modifier = Modifier.padding(horizontal = 32.dp, vertical = 14.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Icon(
                    Icons.Rounded.Payment,
                    contentDescription = null,
                    modifier = Modifier.size(18.dp),
                    tint = MaterialTheme.colorScheme.onPrimary,
                )
                Spacer(modifier = Modifier.width(8.dp))
                Text(
                    "Buy",
                    style = MaterialTheme.typography.labelLarge.copy(fontWeight = FontWeight.Bold),
                    color = MaterialTheme.colorScheme.onPrimary,
                )
            }
        }

        // Dropdown segment
        Box {
            Surface(
                onClick = { paymentMenuExpanded = true },
                shape = RoundedCornerShape(topStart = 0.dp, bottomStart = 0.dp, topEnd = 100.dp, bottomEnd = 100.dp),
                color = MaterialTheme.colorScheme.primary.copy(alpha = 0.85f),
            ) {
                Row(
                    modifier = Modifier.padding(start = 16.dp, end = 14.dp, top = 14.dp, bottom = 14.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        text = paymentLabel,
                        style = MaterialTheme.typography.labelLarge.copy(fontWeight = FontWeight.SemiBold),
                        color = MaterialTheme.colorScheme.onPrimary,
                    )
                    Spacer(modifier = Modifier.width(8.dp))
                    Icon(
                        Icons.Rounded.KeyboardArrowDown,
                        contentDescription = "Payment options",
                        modifier = Modifier.size(20.dp),
                        tint = MaterialTheme.colorScheme.onPrimary,
                    )
                }
            }

            DropdownMenu(
                expanded = paymentMenuExpanded,
                onDismissRequest = { paymentMenuExpanded = false },
            ) {
                paymentOptions.forEach { option ->
                    DropdownMenuItem(
                        text = { Text(option.label) },
                        onClick = {
                            paymentMenuExpanded = false
                            onSelectPayment(option.gateway)
                        },
                        trailingIcon = {
                            if (option.gateway == selectedPaymentGateway) {
                                Icon(
                                    imageVector = Icons.Rounded.Check,
                                    contentDescription = null,
                                    tint = MaterialTheme.colorScheme.primary,
                                )
                            }
                        },
                    )
                }
            }
        }
    }
}

@Composable
private fun ProcessingContent() {
    Surface(
        modifier = Modifier.fillMaxWidth(),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surfaceContainerLow,
    ) {
        Column(
            modifier = Modifier.padding(vertical = 48.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            CircularProgressIndicator(
                modifier = Modifier.size(48.dp),
                strokeWidth = 5.dp,
                strokeCap = StrokeCap.Round,
            )
            Spacer(modifier = Modifier.height(16.dp))
            Text(
                "Processing payment...",
                style = MaterialTheme.typography.bodyLarge,
            )
        }
    }
}

@Composable
private fun CompleteContent() {
    Surface(
        modifier = Modifier.fillMaxWidth(),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surfaceContainerLow,
    ) {
        Column(
            modifier = Modifier.padding(vertical = 48.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Icon(
                Icons.Rounded.Check,
                contentDescription = null,
                modifier = Modifier.size(48.dp),
                tint = MaterialTheme.colorScheme.primary,
            )
            Spacer(modifier = Modifier.height(16.dp))
            Text(
                "Payment complete",
                style = MaterialTheme.typography.bodyLarge,
            )
        }
    }
}

@Composable
private fun BreakdownRow(label: String, value: String) {
    Row(modifier = Modifier.fillMaxWidth()) {
        Text(
            label,
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
        )
        Spacer(modifier = Modifier.weight(1f))
        Text(
            value,
            style = MaterialTheme.typography.bodyMedium.copy(fontWeight = FontWeight.SemiBold),
        )
    }
}
