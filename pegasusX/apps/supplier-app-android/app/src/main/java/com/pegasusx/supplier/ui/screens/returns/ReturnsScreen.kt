package com.pegasusx.supplier.ui.screens.returns

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.SupplierOrder
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ReturnsScreen(ops: SupplierOperationsRepository, onBack: () -> Unit) {
    var orders by remember { mutableStateOf<List<SupplierOrder>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = ops.getOrders(filter = "RETURNS", limit = 100, offset = 0)
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
                title = { Text("Returns") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> SupplierLoadingState("Loading returns…", "Cancelled and rejected orders")
            error != null -> SupplierStatePane(
                kind = SupplierStateKind.Error,
                headline = "Returns unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            orders.isEmpty() -> SupplierStatePane(
                kind = SupplierStateKind.Empty,
                headline = "No returns",
                body = "No cancelled or rejected orders in the current window.",
                modifier = Modifier.padding(padding),
            )
            else -> LazyColumn(
                modifier = Modifier.padding(padding).fillMaxSize(),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                items(orders, key = { it.orderId }) { order ->
                    ElevatedCard(Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(PegasusSpacing.lg)) {
                            Text(order.orderId, style = MaterialTheme.typography.labelLarge)
                            Text(order.status, style = MaterialTheme.typography.bodyMedium)
                            order.decision?.let { Text(it, style = MaterialTheme.typography.bodySmall) }
                            order.note?.let { Text(it, style = MaterialTheme.typography.bodySmall) }
                        }
                    }
                }
            }
        }
    }
}
