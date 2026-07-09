package com.pegasusx.supplier.ui.screens.inventory

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.pegasus.design.showFullScreenLoading
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.ui.viewmodel.InventoryViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun InventoryScreen(
    viewModel: InventoryViewModel = hiltViewModel(),
) {
    val state by viewModel.state.collectAsStateWithLifecycle()
    var deltaInput by remember(state.adjustingSku) { mutableStateOf("") }

    Scaffold(topBar = { TopAppBar(title = { Text("Inventory") }) }) { padding ->
        Box(Modifier.padding(padding)) {
            when {
                showFullScreenLoading(state.loading, state.items.isNotEmpty()) -> PegasusLoadingState("Loading inventory…", "SKU list")
                state.error != null && state.items.isEmpty() -> PegasusStatePane(
                    kind = PegasusStateKind.Error,
                    headline = "Inventory unavailable",
                    body = state.error!!,
                    actionLabel = "Retry",
                    onAction = { viewModel.load() },
                )
                state.items.isEmpty() -> PegasusStatePane(
                    kind = PegasusStateKind.Empty,
                    headline = "No SKUs",
                    body = "Inventory items will appear here.",
                )
                else -> androidx.compose.foundation.lazy.grid.LazyVerticalGrid(
                    columns = androidx.compose.foundation.lazy.grid.GridCells.Adaptive(minSize = 340.dp),
                    contentPadding = PaddingValues(PegasusSpacing.lg),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                    horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                ) {
                    androidx.compose.foundation.lazy.grid.items(state.items, key = { it.sku }) { item ->
                        ListItem(
                            headlineContent = { Text(item.productName) },
                            supportingContent = { Text("SKU ${item.sku} · qty ${item.quantity}") },
                            modifier = Modifier.clickable { viewModel.showAdjust(item.sku) },
                        )
                    }
                }
            }
        }
    }

    val adjustingSku = state.adjustingSku
    if (adjustingSku != null) {
        AlertDialog(
            onDismissRequest = { viewModel.showAdjust(null) },
            title = { Text("Adjust quantity") },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                    Text("SKU: $adjustingSku")
                    OutlinedTextField(
                        value = deltaInput,
                        onValueChange = { deltaInput = it.filter { ch -> ch.isDigit() || ch == '-' } },
                        label = { Text("Quantity delta") },
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                        modifier = Modifier.fillMaxWidth(),
                    )
                    state.error?.let { Text(it, color = MaterialTheme.colorScheme.error) }
                }
            },
            confirmButton = {
                TextButton(
                    onClick = {
                        val delta = deltaInput.toLongOrNull() ?: return@TextButton
                        viewModel.adjustQuantity(adjustingSku, delta)
                    },
                    enabled = !state.adjustBusy,
                ) { Text("Apply") }
            },
            dismissButton = {
                TextButton(onClick = { viewModel.showAdjust(null) }) { Text("Cancel") }
            },
        )
    }
}
