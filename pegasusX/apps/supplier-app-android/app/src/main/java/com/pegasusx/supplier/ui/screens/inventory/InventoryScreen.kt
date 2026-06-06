package com.pegasusx.supplier.ui.screens.inventory

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.remote.SupplierApi
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun InventoryScreen(api: SupplierApi) {
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var items by remember { mutableStateOf(emptyList<com.pegasusx.supplier.data.model.InventoryItem>()) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(Unit) {
        scope.launch {
            try {
                val resp = api.getInventory()
                items = if (resp.isSuccessful) resp.body()?.items.orEmpty() else emptyList()
                if (!resp.isSuccessful) error = "Failed (${resp.code()})"
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    Scaffold(topBar = { TopAppBar(title = { Text("Inventory") }) }) { padding ->
        Box(Modifier.padding(padding)) {
            when {
                loading -> SupplierLoadingState("Loading inventory…", "SKU list")
                error != null -> SupplierStatePane(
                    kind = SupplierStateKind.Error,
                    headline = "Inventory unavailable",
                    body = error!!,
                )
                items.isEmpty() -> SupplierStatePane(
                    kind = SupplierStateKind.Empty,
                    headline = "No SKUs",
                    body = "Inventory items will appear here.",
                )
                else -> LazyColumn(
                    contentPadding = PaddingValues(PegasusSpacing.lg),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                ) {
                    items(items, key = { it.sku }) { item ->
                        ListItem(
                            headlineContent = { Text(item.productName) },
                            supportingContent = { Text("SKU ${item.sku} · qty ${item.quantity}") },
                        )
                    }
                }
            }
        }
    }
}
