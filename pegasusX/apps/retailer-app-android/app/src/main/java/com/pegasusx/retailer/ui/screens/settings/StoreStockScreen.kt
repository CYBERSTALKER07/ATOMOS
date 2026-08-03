package com.pegasusx.retailer.ui.screens.settings

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.ViewModel
import com.pegasusx.retailer.data.api.PegasusApi
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.launch
import javax.inject.Inject

data class StockRowUi(
    val sku: String,
    val bin: String,
    val onHand: Long,
    val available: Long,
)

@HiltViewModel
class StoreStockViewModel @Inject constructor(val api: PegasusApi) : ViewModel()

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun StoreStockScreen(
    onNavigateBack: () -> Unit,
    viewModel: StoreStockViewModel = hiltViewModel(),
) {
    val scope = rememberCoroutineScope()
    var rows by remember { mutableStateOf<List<StockRowUi>>(emptyList()) }
    var locationId by remember { mutableStateOf<String?>(null) }
    var banner by remember { mutableStateOf<String?>(null) }
    var orderId by remember { mutableStateOf("") }
    var sku by remember { mutableStateOf("") }
    var qty by remember { mutableStateOf("1") }
    var busy by remember { mutableStateOf(false) }

    fun reload() {
        scope.launch {
            try {
                if (locationId == null) {
                    val locs = viewModel.api.getLocations().asJsonObject
                    locationId = locs.get("active_location_id")?.asString
                        ?: locs.getAsJsonArray("items")?.firstOrNull()?.asJsonObject?.get("location_id")?.asString
                }
                val el = viewModel.api.getStoreStock(locationId).asJsonObject
                rows = el.getAsJsonArray("items")?.map { item ->
                    val o = item.asJsonObject
                    StockRowUi(
                        sku = o.get("sku")?.asString.orEmpty(),
                        bin = o.get("stock_bin")?.asString.orEmpty(),
                        onHand = o.get("on_hand")?.asLong ?: 0L,
                        available = o.get("available")?.asLong ?: 0L,
                    )
                }.orEmpty()
            } catch (e: Exception) {
                banner = e.message
            }
        }
    }

    LaunchedEffect(Unit) { reload() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Store stock") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        LazyColumn(
            Modifier.fillMaxSize().padding(padding).padding(horizontal = 16.dp),
            contentPadding = PaddingValues(vertical = 12.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            item {
                Text(
                    "Receive Pegasus deliveries into backroom, putaway to floor, adjust, count.",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            banner?.let { item { Text(it, color = MaterialTheme.colorScheme.primary) } }
            item {
                Card {
                    Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text("Receive order", style = MaterialTheme.typography.titleMedium)
                        OutlinedTextField(
                            value = orderId,
                            onValueChange = { orderId = it },
                            label = { Text("Order ID") },
                            modifier = Modifier.fillMaxWidth(),
                        )
                        Button(enabled = !busy && orderId.isNotBlank(), onClick = {
                            scope.launch {
                                busy = true
                                try {
                                    viewModel.api.receiveStoreStock(
                                        body = mapOf(
                                            "order_id" to orderId,
                                            "location_id" to (locationId ?: ""),
                                            "confirm" to true,
                                            "stock_bin" to "BACKROOM",
                                        ),
                                        idempotencyKey = "recv-${System.currentTimeMillis()}",
                                    )
                                    banner = "Received into BACKROOM"
                                    orderId = ""
                                    reload()
                                } catch (e: Exception) {
                                    banner = e.message
                                } finally {
                                    busy = false
                                }
                            }
                        }) { Text("Receive") }
                    }
                }
            }
            item {
                Card {
                    Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text("Putaway / adjust", style = MaterialTheme.typography.titleMedium)
                        OutlinedTextField(value = sku, onValueChange = { sku = it }, label = { Text("SKU") }, modifier = Modifier.fillMaxWidth())
                        OutlinedTextField(value = qty, onValueChange = { qty = it }, label = { Text("Qty") }, modifier = Modifier.fillMaxWidth())
                        Button(enabled = !busy && sku.isNotBlank(), onClick = {
                            scope.launch {
                                busy = true
                                try {
                                    viewModel.api.transferStoreStock(
                                        body = mapOf(
                                            "location_id" to (locationId ?: ""),
                                            "sku" to sku,
                                            "qty" to (qty.toLongOrNull() ?: 1L),
                                            "from_bin" to "BACKROOM",
                                            "to_bin" to "FLOOR",
                                        ),
                                        idempotencyKey = "xfer-${System.currentTimeMillis()}",
                                    )
                                    banner = "Putaway BACKROOM→FLOOR"
                                    reload()
                                } catch (e: Exception) {
                                    banner = e.message
                                } finally {
                                    busy = false
                                }
                            }
                        }) { Text("Putaway to floor") }
                        Button(enabled = !busy && sku.isNotBlank(), onClick = {
                            scope.launch {
                                busy = true
                                try {
                                    viewModel.api.adjustStoreStock(
                                        body = mapOf(
                                            "location_id" to (locationId ?: ""),
                                            "sku" to sku,
                                            "qty_delta" to (qty.toLongOrNull() ?: 0L),
                                            "stock_bin" to "BACKROOM",
                                            "note" to "mobile_adjust",
                                        ),
                                        idempotencyKey = "adj-${System.currentTimeMillis()}",
                                    )
                                    banner = "Adjusted"
                                    reload()
                                } catch (e: Exception) {
                                    banner = e.message
                                } finally {
                                    busy = false
                                }
                            }
                        }) { Text("Adjust BACKROOM by qty") }
                    }
                }
            }
            items(rows, key = { "${it.sku}-${it.bin}" }) { row ->
                Card {
                    Column(Modifier.padding(14.dp)) {
                        Text(row.sku, style = MaterialTheme.typography.titleSmall)
                        Text("${row.bin}: on hand ${row.onHand} · available ${row.available}", style = MaterialTheme.typography.bodySmall)
                    }
                }
            }
        }
    }
}
