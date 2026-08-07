package com.pegasusx.supplier.ui.screens.catalog

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Button
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExposedDropdownMenuBox
import androidx.compose.material3.ExposedDropdownMenuDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.pegasusx.supplier.data.model.CatalogProduct
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import java.text.NumberFormat
import java.util.Locale

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CatalogList(
    products: List<CatalogProduct>,
    draftVU: MutableMap<String, String>,
    draftBarcode: MutableMap<String, String>,
    draftUnitsPerCase: MutableMap<String, String>,
    draftSaleUnit: MutableMap<String, String>,
    savingId: String?,
    imageSavingId: String?,
    error: String?,
    onSaveUnitVolume: (CatalogProduct) -> Unit,
    onOpenProduct: (String) -> Unit,
    onChangeImage: (String) -> Unit,
) {
    val fmt = remember { NumberFormat.getIntegerInstance(Locale.US) }

    LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        contentPadding = PaddingValues(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
    ) {
        if (error != null) {
            item {
                Text(
                    error,
                    color = MaterialTheme.colorScheme.error,
                    modifier = Modifier.padding(bottom = PegasusSpacing.sm),
                )
            }
        }
        items(products, key = { it.productId }) { product ->
            val vuValue = draftVU[product.productId] ?: product.unitVolumeVu.toString()
            val barcodeValue = draftBarcode[product.productId] ?: product.barcode.orEmpty()
            val saleUnit = draftSaleUnit[product.productId] ?: product.saleUnit
            val unitsPerCaseValue = draftUnitsPerCase[product.productId]
                ?: product.unitsPerCase?.toString().orEmpty()
            
            val vuDirty = vuValue != product.unitVolumeVu.toString()
            val barcodeDirty = barcodeValue != product.barcode.orEmpty()
            val saleUnitDirty = saleUnit != product.saleUnit
            val unitsPerCaseDirty = unitsPerCaseValue != product.unitsPerCase?.toString().orEmpty()
            val dirty = vuDirty || barcodeDirty || saleUnitDirty || unitsPerCaseDirty

            ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                Column(
                    modifier = Modifier.padding(PegasusSpacing.lg),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                ) {
                    Text(product.name, style = MaterialTheme.typography.titleMedium)
                    TextButton(onClick = { onOpenProduct(product.productId) }) {
                        Text("View details")
                    }
                    Text(stringResource(R.string.mobile_supplier_ui_format_currency_unit, fmt.format(product.priceMinor), product.currency, product.unit),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    Text(
                        "Sale: ${saleUnit.lowercase()}${if (saleUnit == "CASE" && unitsPerCaseValue.isNotBlank()) " ($unitsPerCaseValue/case)" else ""}",
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    if (!product.imageUrl.isNullOrBlank()) {
                        Text(
                            "Image attached",
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.tertiary,
                        )
                    }
                    TextButton(
                        onClick = { onChangeImage(product.productId) },
                        enabled = imageSavingId != product.productId,
                    ) {
                        Text(
                            when {
                                imageSavingId == product.productId -> "Uploading…"
                                product.imageUrl.isNullOrBlank() -> "Add image"
                                else -> "Change image"
                            },
                        )
                    }
                    CatalogBarcodeField(
                        value = barcodeValue,
                        onValueChange = { draftBarcode[product.productId] = it },
                        enabled = savingId != product.productId,
                    )
                    var rowSaleUnitExpanded by remember(product.productId) { mutableStateOf(false) }
                    ExposedDropdownMenuBox(
                        expanded = rowSaleUnitExpanded,
                        onExpandedChange = { rowSaleUnitExpanded = it },
                    ) {
                        OutlinedTextField(
                            value = if (saleUnit == "CASE") "Case" else "Unit",
                            onValueChange = {},
                            readOnly = true,
                            label = { Text("Sale unit") },
                            trailingIcon = { ExposedDropdownMenuDefaults.TrailingIcon(expanded = rowSaleUnitExpanded) },
                            modifier = Modifier
                                .menuAnchor()
                                .fillMaxWidth(),
                        )
                        ExposedDropdownMenu(
                            expanded = rowSaleUnitExpanded,
                            onDismissRequest = { rowSaleUnitExpanded = false },
                        ) {
                            listOf("UNIT" to "Unit", "CASE" to "Case").forEach { (value, label) ->
                                DropdownMenuItem(
                                    text = { Text(label) },
                                    onClick = {
                                        draftSaleUnit[product.productId] = value
                                        rowSaleUnitExpanded = false
                                    },
                                )
                            }
                        }
                    }
                    if (saleUnit == "CASE") {
                        OutlinedTextField(
                            value = unitsPerCaseValue,
                            onValueChange = { draftUnitsPerCase[product.productId] = it },
                            label = { Text("Units per case") },
                            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                            singleLine = true,
                            modifier = Modifier.fillMaxWidth(),
                        )
                    }
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                    ) {
                        OutlinedTextField(
                            value = vuValue,
                            onValueChange = { draftVU[product.productId] = it },
                            label = { Text("Unit VU") },
                            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                            singleLine = true,
                            modifier = Modifier.widthIn(min = 120.dp, max = 160.dp),
                        )
                        Button(
                            onClick = { onSaveUnitVolume(product) },
                            enabled = dirty && savingId != product.productId,
                        ) {
                            Text(if (savingId == product.productId) "…" else "Save")
                        }
                    }
                }
            }
        }
    }
}
