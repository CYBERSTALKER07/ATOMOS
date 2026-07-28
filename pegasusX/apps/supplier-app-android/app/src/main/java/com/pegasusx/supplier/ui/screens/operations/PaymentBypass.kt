package com.pegasusx.supplier.ui.screens.operations

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardCapitalization
import com.pegasusx.supplier.ui.components.SupplierSectionTitle
import com.pegasusx.supplier.ui.theme.PegasusSpacing

@Composable
fun PaymentBypass(
    orderId: String,
    bypassReason: String,
    bypassToken: String?,
    bypassing: Boolean,
    onOrderIdChange: (String) -> Unit,
    onBypassReasonChange: (String) -> Unit,
    onShowConfirmChange: (Boolean) -> Unit,
) {
    Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md)) {
        HorizontalDivider()
        SupplierSectionTitle("Payment bypass")
        OutlinedTextField(
            value = orderId,
            onValueChange = onOrderIdChange,
            label = { Text("Order ID (AWAITING_PAYMENT)") },
            modifier = Modifier.fillMaxWidth(),
            singleLine = true,
            keyboardOptions = KeyboardOptions(capitalization = KeyboardCapitalization.None),
        )
        OutlinedTextField(
            value = bypassReason,
            onValueChange = onBypassReasonChange,
            label = { Text("Reason (optional)") },
            modifier = Modifier.fillMaxWidth(),
            singleLine = true,
        )
        OutlinedButton(
            onClick = { onShowConfirmChange(true) },
            enabled = !bypassing && orderId.isNotBlank(),
            modifier = Modifier.fillMaxWidth(),
        ) {
            Text(if (bypassing) "Issuing…" else "Issue bypass token")
        }
        bypassToken?.let { token ->
            Text("Driver token: $token", style = MaterialTheme.typography.bodyMedium)
        }
    }
}
