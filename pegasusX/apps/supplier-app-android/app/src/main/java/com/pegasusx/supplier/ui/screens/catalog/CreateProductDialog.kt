package com.pegasusx.supplier.ui.screens.catalog

import androidx.compose.ui.res.stringResource

import android.net.Uri
import androidx.activity.compose.ManagedActivityResultLauncher
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import com.pegasusx.supplier.data.model.CatalogCategory
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CreateProductDialog(
    categories: List<CatalogCategory>,
    currency: String,
    creating: Boolean,
    createName: String,
    onNameChange: (String) -> Unit,
    createPrice: String,
    onPriceChange: (String) -> Unit,
    createVu: String,
    onVuChange: (String) -> Unit,
    createBarcode: String,
    onBarcodeChange: (String) -> Unit,
    createSaleUnit: String,
    onSaleUnitChange: (String) -> Unit,
    createUnitsPerCase: String,
    onUnitsPerCaseChange: (String) -> Unit,
    createCategoryId: String,
    onCategoryIdChange: (String) -> Unit,
    createImageLabel: String?,
    createImagePicker: ManagedActivityResultLauncher<String, Uri?>,
    createError: String?,
    categoryMenuExpanded: Boolean,
    onCategoryMenuExpandedChange: (Boolean) -> Unit,
    saleUnitMenuExpanded: Boolean,
    onSaleUnitMenuExpandedChange: (Boolean) -> Unit,
    onCreateProduct: () -> Unit,
    onDismiss: () -> Unit,
) {
    AlertDialog(
        onDismissRequest = { if (!creating) onDismiss() },
        title = { Text("Add product") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                OutlinedTextField(
                    value = createName,
                    onValueChange = onNameChange,
                    label = { Text("Name") },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                )
                ExposedDropdownMenuBox(
                    expanded = categoryMenuExpanded,
                    onExpandedChange = onCategoryMenuExpandedChange,
                ) {
                    OutlinedTextField(
                        value = categories.find { it.categoryId == createCategoryId }?.name ?: "Select category",
                        onValueChange = {},
                        readOnly = true,
                        label = { Text("Category") },
                        trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = categoryMenuExpanded) },
                        modifier = Modifier
                            .menuAnchor()
                            .fillMaxWidth(),
                    )
                    ExposedDropdownMenu(
                        expanded = categoryMenuExpanded,
                        onDismissRequest = { onCategoryMenuExpandedChange(false) },
                    ) {
                        categories.forEach { category ->
                            DropdownMenuItem(
                                text = { Text(category.name) },
                                onClick = {
                                    onCategoryIdChange(category.categoryId)
                                    onCategoryMenuExpandedChange(false)
                                },
                            )
                        }
                    }
                }
                OutlinedTextField(
                    value = createPrice,
                    onValueChange = onPriceChange,
                    label = { Text(stringResource(R.string.mobile_supplier_ui_price_currency_minor, currency)) },
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                )
                OutlinedTextField(
                    value = createVu,
                    onValueChange = onVuChange,
                    label = { Text("Unit VU") },
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                )
                ExposedDropdownMenuBox(
                    expanded = saleUnitMenuExpanded,
                    onExpandedChange = onSaleUnitMenuExpandedChange,
                ) {
                    OutlinedTextField(
                        value = if (createSaleUnit == "CASE") "Case" else "Unit",
                        onValueChange = {},
                        readOnly = true,
                        label = { Text("Sale unit") },
                        trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = saleUnitMenuExpanded) },
                        modifier = Modifier
                            .menuAnchor()
                            .fillMaxWidth(),
                    )
                    ExposedDropdownMenu(
                        expanded = saleUnitMenuExpanded,
                        onDismissRequest = { onSaleUnitMenuExpandedChange(false) },
                    ) {
                        listOf("UNIT" to "Unit", "CASE" to "Case").forEach { (value, label) ->
                            DropdownMenuItem(
                                text = { Text(label) },
                                onClick = {
                                    onSaleUnitChange(value)
                                    onSaleUnitMenuExpandedChange(false)
                                },
                            )
                        }
                    }
                }
                if (createSaleUnit == "CASE") {
                    OutlinedTextField(
                        value = createUnitsPerCase,
                        onValueChange = onUnitsPerCaseChange,
                        label = { Text("Units per case") },
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
                CatalogBarcodeField(
                    value = createBarcode,
                    onValueChange = onBarcodeChange,
                    enabled = !creating,
                )
                Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                    TextButton(onClick = { createImagePicker.launch("image/*") }) {
                        Text(if (createImageLabel != null) "Image selected" else "Add image")
                    }
                }
                createError?.let {
                    Text(it, color = MaterialTheme.colorScheme.error)
                }
            }
        },
        confirmButton = {
            TextButton(
                onClick = onCreateProduct,
                enabled = !creating && categories.isNotEmpty(),
            ) {
                Text(if (creating) "Creating…" else "Create")
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss, enabled = !creating) {
                Text("Cancel")
            }
        },
    )
}
