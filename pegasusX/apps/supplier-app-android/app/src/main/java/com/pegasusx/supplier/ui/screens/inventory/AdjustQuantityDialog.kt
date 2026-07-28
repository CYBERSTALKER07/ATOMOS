package com.pegasusx.supplier.ui.screens.inventory

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import com.pegasusx.supplier.ui.theme.PegasusSpacing

@Composable
fun AdjustQuantityDialog(
    adjustingSku: String,
    deltaInput: String,
    onDeltaInputChanged: (String) -> Unit,
    error: String?,
    adjustBusy: Boolean,
    onDismiss: () -> Unit,
    onApply: () -> Unit,
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Adjust quantity") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                Text("SKU: $adjustingSku")
                OutlinedTextField(
                    value = deltaInput,
                    onValueChange = onDeltaInputChanged,
                    label = { Text("Quantity delta") },
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                    modifier = Modifier.fillMaxWidth(),
                )
                error?.let { Text(it, color = MaterialTheme.colorScheme.error) }
            }
        },
        confirmButton = {
            TextButton(
                onClick = onApply,
                enabled = !adjustBusy,
            ) { Text("Apply") }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) { Text("Cancel") }
        },
    )
}
