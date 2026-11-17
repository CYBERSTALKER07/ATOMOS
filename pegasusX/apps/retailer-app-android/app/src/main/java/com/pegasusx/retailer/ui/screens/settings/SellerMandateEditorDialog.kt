package com.pegasusx.retailer.ui.screens.settings

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.pegasusx.retailer.R
import com.pegasusx.retailer.util.OrderLineContractHelper

@Composable
fun SellerMandateEditorDialog(
    form: SellerMandateFormState,
    isLoading: Boolean,
    isSaving: Boolean,
    error: String?,
    onFormChange: ((SellerMandateFormState) -> SellerMandateFormState) -> Unit,
    onSave: () -> Unit,
    onDismiss: () -> Unit,
) {
    AlertDialog(
        onDismissRequest = { if (!isSaving) onDismiss() },
        title = {
            Column {
                Text(
                    text = stringResource(R.string.retailer_mandate_title),
                    style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.Bold),
                )
                Text(
                    text = if (form.memberRole.isNotBlank()) "${form.memberName} (${form.memberRole})" else form.memberName,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        },
        text = {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .verticalScroll(rememberScrollState()),
                verticalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                if (isLoading) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.Center,
                    ) {
                        CircularProgressIndicator()
                    }
                } else {
                    MandateToggleRow(
                        label = "Can receive goods",
                        checked = form.canReceiveGoods,
                        enabled = !isSaving,
                        onCheckedChange = { checked -> onFormChange { it.copy(canReceiveGoods = checked) } },
                    )
                    MandateToggleRow(
                        label = "Can pay cash",
                        checked = form.canPayCash,
                        enabled = !isSaving,
                        onCheckedChange = { checked -> onFormChange { it.copy(canPayCash = checked) } },
                    )
                    MandateToggleRow(
                        label = "Can use credit",
                        checked = form.canUseCredit,
                        enabled = !isSaving,
                        onCheckedChange = { checked -> onFormChange { it.copy(canUseCredit = checked) } },
                    )

                    OutlinedTextField(
                        value = form.dailyCreditCapSom,
                        onValueChange = { v -> onFormChange { it.copy(dailyCreditCapSom = v.filter { ch -> ch.isDigit() }) } },
                        label = { Text(stringResource(R.string.retailer_mandate_daily_credit)) },
                        singleLine = true,
                        enabled = !isSaving,
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                        modifier = Modifier.fillMaxWidth(),
                    )
                    OutlinedTextField(
                        value = form.dailyCashCapSom,
                        onValueChange = { v -> onFormChange { it.copy(dailyCashCapSom = v.filter { ch -> ch.isDigit() }) } },
                        label = { Text(stringResource(R.string.retailer_mandate_daily_cash)) },
                        singleLine = true,
                        enabled = !isSaving,
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                        modifier = Modifier.fillMaxWidth(),
                    )
                    OutlinedTextField(
                        value = form.maxSingleOrderSom,
                        onValueChange = { v -> onFormChange { it.copy(maxSingleOrderSom = v.filter { ch -> ch.isDigit() }) } },
                        label = { Text(stringResource(R.string.retailer_mandate_max_order)) },
                        singleLine = true,
                        enabled = !isSaving,
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                        modifier = Modifier.fillMaxWidth(),
                    )

                    Text(
                        text = "Enter 0 for unlimited cap. Today used: Credit ${
                            OrderLineContractHelper.formatWholeSom(form.dailyUsage.spentCreditMinor)
                        } · Cash ${
                            OrderLineContractHelper.formatWholeSom(form.dailyUsage.spentCashMinor)
                        } (${form.dailyUsage.orderCount} orders)",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }

                if (!error.isNullOrBlank()) {
                    Text(
                        text = error,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.error,
                    )
                }
            }
        },
        confirmButton = {
            Button(onClick = onSave, enabled = !isLoading && !isSaving) {
                Text(if (isSaving) "…" else stringResource(R.string.retailer_mandate_save))
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss, enabled = !isSaving) {
                Text("Cancel")
            }
        },
    )
}

@Composable
private fun MandateToggleRow(
    label: String,
    checked: Boolean,
    enabled: Boolean,
    onCheckedChange: (Boolean) -> Unit,
) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(text = label, style = MaterialTheme.typography.bodyMedium)
        Switch(checked = checked, onCheckedChange = onCheckedChange, enabled = enabled)
    }
}
