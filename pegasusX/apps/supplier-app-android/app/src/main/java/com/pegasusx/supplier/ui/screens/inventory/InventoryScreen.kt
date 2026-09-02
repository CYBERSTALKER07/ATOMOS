package com.pegasusx.supplier.ui.screens.inventory

import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.pegasus.design.network.showFullScreenLoading
import com.pegasus.design.ui.PegasusLoadingState
import com.pegasus.design.ui.PegasusStateKind
import com.pegasus.design.ui.PegasusStatePane
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
                else -> InventoryList(
                    items = state.items,
                    onShowAdjust = { viewModel.showAdjust(it) }
                )
            }
        }
    }

    val adjustingSku = state.adjustingSku
    if (adjustingSku != null) {
        AdjustQuantityDialog(
            adjustingSku = adjustingSku,
            deltaInput = deltaInput,
            onDeltaInputChanged = { deltaInput = it.filter { ch -> ch.isDigit() || ch == '-' } },
            error = state.error,
            adjustBusy = state.adjustBusy,
            onDismiss = { viewModel.showAdjust(null) },
            onApply = {
                val delta = deltaInput.toLongOrNull() ?: return@AdjustQuantityDialog
                viewModel.adjustQuantity(adjustingSku, delta)
            }
        )
    }
}
