package com.pegasusx.retailer.ui.screens.settings

import androidx.compose.ui.res.stringResource

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
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.rememberModalBottomSheetState
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
import com.pegasusx.retailer.data.local.TokenManager
import com.pegasusx.retailer.data.model.Order
import com.pegasusx.retailer.data.model.OrderLineItem
import com.pegasusx.retailer.data.model.OrderStatus
import com.pegasusx.retailer.data.model.TrackingOrder
import com.pegasusx.retailer.ui.components.FileClaimHost
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.launch
import javax.inject.Inject
import com.pegasusx.retailer.R
import com.pegasusx.retailer.data.json.*

data class StockRowUi(
    val sku: String,
    val bin: String,
    val onHand: Long,
    val available: Long,
)

@HiltViewModel
class StoreStockViewModel @Inject constructor(
    val api: PegasusApi,
    val tokenManager: TokenManager,
) : ViewModel() {
    fun retailerId(): String = tokenManager.getUserId().orEmpty()
}

private fun TrackingOrder.toClaimOrder(): Order {
    val status = runCatching { OrderStatus.valueOf(state) }.getOrDefault(OrderStatus.COMPLETED)
    return Order(
        id = orderId,
        retailerId = "",
        supplierId = supplierId,
        supplierName = supplierName,
        status = status,
        items = items.map { item ->
            OrderLineItem(
                id = item.productId,
                productId = item.productId,
                productName = item.productName,
                quantity = item.quantity.toInt().coerceAtLeast(0),
                unitPrice = item.unitPrice.toDouble(),
                totalPrice = item.lineTotal.toDouble(),
            )
        },
        totalAmount = totalAmount,
        createdAt = createdAt,
        orderSource = orderSource.ifBlank { "MANUAL" },
        qrCode = deliveryToken.ifBlank { null },
    )
}

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

    var showOrderPicker by remember { mutableStateOf(false) }
    var preferredSku by remember { mutableStateOf<String?>(null) }
    var claimableOrders by remember { mutableStateOf<List<Order>>(emptyList()) }
    var pickerLoading by remember { mutableStateOf(false) }
    var pickerQuery by remember { mutableStateOf("") }
    var pickerError by remember { mutableStateOf<String?>(null) }
    var claimOrder by remember { mutableStateOf<Order?>(null) }
    var claimPreferredSku by remember { mutableStateOf<String?>(null) }

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

    fun openRequestReturn(skuHint: String? = null) {
        preferredSku = skuHint
        pickerQuery = ""
        pickerError = null
        showOrderPicker = true
        scope.launch {
            pickerLoading = true
            try {
                val rid = viewModel.retailerId()
                val orders = if (rid.isNotBlank()) {
                    viewModel.api.getOrders(rid)
                } else {
                    emptyList()
                }
                claimableOrders = orders.filter {
                    it.status == OrderStatus.COMPLETED || it.status == OrderStatus.DELIVERED_ON_CREDIT
                }
            } catch (e: Exception) {
                pickerError = e.message
                claimableOrders = emptyList()
            } finally {
                pickerLoading = false
            }
        }
    }

    fun pickOrder(order: Order) {
        scope.launch {
            pickerLoading = true
            pickerError = null
            try {
                var next = order
                if (next.items.isEmpty()) {
                    val tracking = runCatching { viewModel.api.getTrackingOrders() }.getOrNull()
                    val hit = listOfNotNull(tracking?.orders, tracking?.recentReceipts)
                        .flatten()
                        .firstOrNull { it.orderId == order.id }
                    if (hit != null) next = hit.toClaimOrder()
                }
                claimPreferredSku = preferredSku
                claimOrder = next
                showOrderPicker = false
            } catch (e: Exception) {
                pickerError = e.message
            } finally {
                pickerLoading = false
            }
        }
    }

    LaunchedEffect(Unit) { reload() }

    claimOrder?.let { order ->
        FileClaimHost(
            order = order,
            preferredSku = claimPreferredSku,
            onDismiss = {
                claimOrder = null
                claimPreferredSku = null
            },
        )
    }

    if (showOrderPicker) {
        val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
        val filtered = remember(claimableOrders, pickerQuery) {
            val q = pickerQuery.trim().lowercase()
            if (q.isEmpty()) claimableOrders
            else claimableOrders.filter { it.id.lowercase().contains(q) }
        }
        ModalBottomSheet(
            onDismissRequest = { showOrderPicker = false },
            sheetState = sheetState,
        ) {
            Column(
                Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 20.dp)
                    .padding(bottom = 28.dp),
                verticalArrangement = Arrangement.spacedBy(10.dp),
            ) {
                Text("Request return / chargeback", style = MaterialTheme.typography.titleLarge)
                Text(
                    "Pick a completed delivery, then file the same claim as order detail. Window is within 48h (server enforces).",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                preferredSku?.let {
                    Text(
                        stringResource(R.string.mobile_retailer_ui_preferred_sku_from_stock_it, it),
                        style = MaterialTheme.typography.labelMedium,
                    )
                }
                OutlinedTextField(
                    value = pickerQuery,
                    onValueChange = { pickerQuery = it },
                    label = { Text("Search by order id") },
                    modifier = Modifier.fillMaxWidth(),
                )
                pickerError?.let {
                    Text(it, color = MaterialTheme.colorScheme.error)
                }
                if (pickerLoading) {
                    CircularProgressIndicator()
                } else if (filtered.isEmpty()) {
                    Text(
                        "No COMPLETED / DELIVERED_ON_CREDIT orders found.",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                } else {
                    filtered.forEach { order ->
                        OutlinedButton(
                            onClick = { pickOrder(order) },
                            modifier = Modifier.fillMaxWidth(),
                        ) {
                            Column(Modifier.fillMaxWidth()) {
                                Text(stringResource(R.string.mobile_retailer_ui_takelast_replace, order.id.takeLast(8), order.status.name.replace('_', ' ')))
                                Text(order.id, style = MaterialTheme.typography.labelSmall)
                            }
                        }
                    }
                }
            }
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Store stock") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
                actions = {
                    TextButton(onClick = { openRequestReturn() }) {
                        Text("Return")
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
            item {
                Button(
                    onClick = { openRequestReturn() },
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text("Request return / chargeback")
                }
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
                    Column(
                        Modifier.padding(14.dp),
                        verticalArrangement = Arrangement.spacedBy(6.dp),
                    ) {
                        Text(row.sku, style = MaterialTheme.typography.titleSmall)
                        Text(
                            stringResource(R.string.mobile_retailer_ui_bin_on_hand_onhand_available_available, row.bin, row.onHand, row.available),
                            style = MaterialTheme.typography.bodySmall,
                        )
                        TextButton(onClick = { openRequestReturn(row.sku) }) {
                            Text("Request return")
                        }
                    }
                }
            }
        }
    }
}
