package com.pegasusx.warehouse.ui.screens.supply

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.CreateWarehouseSupplyRequestItem
import com.pegasusx.warehouse.data.model.DemandForecastProduct
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import java.time.LocalDate
import java.time.ZoneOffset
import java.time.format.DateTimeFormatter
import com.pegasusx.warehouse.R

data class SupplyRequestFormResult(
    val factoryId: String,
    val priority: String,
    val notes: String,
    val useDemandForecast: Boolean,
    val requestedDeliveryDate: String?,
    val items: List<CreateWarehouseSupplyRequestItem>,
)

@Composable
fun CreateSupplyRequestDialog(
    api: WarehouseApi,
    onDismiss: () -> Unit,
    onCreate: (SupplyRequestFormResult) -> Unit,
) {
    var factoryId by remember { mutableStateOf("") }
    var factoryLocked by remember { mutableStateOf(false) }
    var priority by remember { mutableStateOf("NORMAL") }
    var notes by remember { mutableStateOf("") }
    var useForecast by remember { mutableStateOf(true) }
    var deliveryDate by remember { mutableStateOf("") }
    var forecast by remember { mutableStateOf<List<DemandForecastProduct>>(emptyList()) }
    var forecastLoading by remember { mutableStateOf(false) }
    var manualItems by remember { mutableStateOf(listOf(ManualSupplyLine())) }
    val scope = rememberCoroutineScope()

    fun loadForecast() {
        scope.launch {
            forecastLoading = true
            try {
                val resp = api.getDemandForecast(days = 7)
                if (resp.isSuccessful) forecast = resp.body()?.products.orEmpty()
            } finally {
                forecastLoading = false
            }
        }
    }

    LaunchedEffect(Unit) {
        runCatching { api.getOpsSupplyFactory() }.getOrNull()?.body()?.factoryId
            ?.takeIf { it.isNotBlank() }
            ?.let {
                factoryId = it
                factoryLocked = true
            }
    }

    LaunchedEffect(useForecast) {
        if (useForecast) loadForecast()
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("New Supply Request") },
        text = {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .heightIn(max = 520.dp)
                    .verticalScroll(rememberScrollState()),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                OutlinedTextField(
                    value = factoryId,
                    onValueChange = { if (!factoryLocked) factoryId = it },
                    readOnly = factoryLocked,
                    label = { Text("Factory ID") },
                    supportingText = if (factoryLocked) {
                        { Text("Nearest factory from the engine.") }
                    } else {
                        null
                    },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                )

                OutlinedTextField(
                    value = deliveryDate,
                    onValueChange = { deliveryDate = it },
                    label = { Text("Delivery date (YYYY-MM-DD)") },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                    placeholder = { Text("Optional") },
                )

                Row(verticalAlignment = Alignment.CenterVertically) {
                    Switch(checked = useForecast, onCheckedChange = { useForecast = it })
                    Spacer(Modifier.width(PegasusSpacing.sm))
                    Text("Use AI demand forecast", style = MaterialTheme.typography.bodyMedium)
                }

                Text("Priority", style = MaterialTheme.typography.labelMedium)
                Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                    listOf("NORMAL", "URGENT", "CRITICAL").forEach { option ->
                        FilterChip(
                            selected = priority == option,
                            onClick = { priority = option },
                            label = { Text(option) },
                        )
                    }
                }

                if (useForecast) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text("Demand forecast (7-day)", style = MaterialTheme.typography.labelMedium)
                        TextButton(onClick = { loadForecast() }) { Text("Refresh") }
                    }
                    if (forecastLoading) {
                        LinearProgressIndicator(modifier = Modifier.fillMaxWidth())
                    } else if (forecast.isEmpty()) {
                        Text("No forecast data available", style = MaterialTheme.typography.bodySmall)
                    } else {
                        LazyColumn(
                            modifier = Modifier.heightIn(max = 160.dp),
                            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
                        ) {
                            itemsIndexed(forecast, key = { _, p -> p.productId }) { _, product ->
                                Text(
                                    stringResource(R.string.mobile_warehouse_ui_take_stock_currentstock_rec_recommendedqty, product.productName.ifBlank { product.productId.take(8) }, product.currentStock, product.recommendedQty),
                                    style = MaterialTheme.typography.bodySmall,
                                )
                            }
                        }
                    }
                } else {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text("Manual items", style = MaterialTheme.typography.labelMedium)
                        IconButton(onClick = { manualItems = manualItems + ManualSupplyLine() }) {
                            Icon(Icons.Default.Add, contentDescription = stringResource(R.string.mobile_warehouse_ui_add_item))
                        }
                    }
                    manualItems.forEachIndexed { index, line ->
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            OutlinedTextField(
                                value = line.productId,
                                onValueChange = { value ->
                                    manualItems = manualItems.toMutableList().also { it[index] = line.copy(productId = value) }
                                },
                                label = { Text("Product ID") },
                                singleLine = true,
                                modifier = Modifier.weight(1f),
                            )
                            OutlinedTextField(
                                value = line.quantity,
                                onValueChange = { value ->
                                    manualItems = manualItems.toMutableList().also { it[index] = line.copy(quantity = value) }
                                },
                                label = { Text("Qty") },
                                singleLine = true,
                                modifier = Modifier.width(88.dp),
                            )
                            if (manualItems.size > 1) {
                                IconButton(onClick = {
                                    manualItems = manualItems.filterIndexed { i, _ -> i != index }
                                }) {
                                    Icon(Icons.Default.Delete, contentDescription = stringResource(R.string.supplier_portal_demand_payday_calendar_text_remove))
                                }
                            }
                        }
                    }
                }

                OutlinedTextField(
                    value = notes,
                    onValueChange = { notes = it },
                    label = { Text("Notes") },
                    modifier = Modifier.fillMaxWidth(),
                    minLines = 2,
                )
            }
        },
        confirmButton = {
            val canSubmit = factoryId.isNotBlank() && (
                useForecast || manualItems.any { it.productId.isNotBlank() && (it.quantity.toIntOrNull() ?: 0) > 0 }
                )
            Button(
                onClick = {
                    val deliveryISO = deliveryDate.trim().takeIf { it.isNotBlank() }?.let { raw ->
                        runCatching {
                            LocalDate.parse(raw, DateTimeFormatter.ISO_LOCAL_DATE)
                                .atStartOfDay(ZoneOffset.UTC)
                                .toInstant()
                                .toString()
                        }.getOrNull()
                    }
                    val items = if (useForecast) {
                        emptyList()
                    } else {
                        manualItems.mapNotNull { line ->
                            val qty = line.quantity.toIntOrNull() ?: return@mapNotNull null
                            val pid = line.productId.trim()
                            if (pid.isBlank() || qty <= 0) return@mapNotNull null
                            CreateWarehouseSupplyRequestItem(
                                productId = pid,
                                requestedQuantity = qty,
                                recommendedQty = qty,
                                unitVolumeVu = 0.0,
                            )
                        }
                    }
                    onCreate(
                        SupplyRequestFormResult(
                            factoryId = factoryId.trim(),
                            priority = priority,
                            notes = notes.trim(),
                            useDemandForecast = useForecast,
                            requestedDeliveryDate = deliveryISO,
                            items = items,
                        ),
                    )
                },
                enabled = canSubmit,
            ) { Text("Submit") }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}

private data class ManualSupplyLine(
    val productId: String = "",
    val quantity: String = "",
)
