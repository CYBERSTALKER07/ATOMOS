package com.pegasusx.supplier.ui.screens.orders

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.SupplierOrder
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierOpsListCard
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.components.formatMinorAmount
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun OrdersScreen(ops: SupplierOperationsRepository) {
    var orders by remember { mutableStateOf<List<SupplierOrder>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = ops.getOrders(limit = 500)
                orders = if (resp.isSuccessful) resp.body()?.orders.orEmpty() else emptyList()
                if (!resp.isSuccessful) error = "Failed (${resp.code()})"
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Orders") },
                actions = {
                    IconButton(onClick = { load() }) {
                        Icon(Icons.Default.Refresh, contentDescription = "Refresh")
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> SupplierLoadingState(
                title = "Loading orders…",
                body = "Supplier order queue",
                modifier = Modifier.padding(padding),
            )
            error != null -> SupplierStatePane(
                kind = SupplierStateKind.Error,
                headline = "Orders unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            orders.isEmpty() -> SupplierStatePane(
                kind = SupplierStateKind.Empty,
                headline = "No orders",
                body = "Pending supplier orders will appear here.",
                modifier = Modifier.padding(padding),
            )
            else -> LazyColumn(
                modifier = Modifier
                    .padding(padding)
                    .fillMaxSize(),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                items(orders, key = { it.orderId }) { order ->
                    val amount = formatMinorAmount(order.totalMinor, order.currency)
                    SupplierOpsListCard(
                        headline = order.orderId.take(12),
                        supporting = buildString {
                            append(amount)
                            append(" · Retailer ")
                            append(order.retailerId.take(8))
                            order.updatedAt.takeIf { it.isNotBlank() }?.let { append(" · $it") }
                        },
                        status = order.status.ifBlank { order.decision },
                    )
                }
            }
        }
    }
}
