package com.pegasusx.retailer.ui.screens.procurement

import androidx.compose.foundation.lazy.grid.itemsIndexed
import androidx.compose.foundation.lazy.itemsIndexed

import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.items

import androidx.compose.foundation.lazy.grid.GridItemSpan

import androidx.compose.foundation.lazy.grid.LazyVerticalGrid

import androidx.compose.foundation.lazy.grid.GridCells

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width

import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Add
import androidx.compose.material.icons.outlined.AutoAwesome
import androidx.compose.material.icons.outlined.Lightbulb
import androidx.compose.material.icons.outlined.Remove
import androidx.compose.material.icons.outlined.ShoppingCart
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Checkbox
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import com.pegasusx.retailer.data.model.DemandForecast
import com.pegasusx.retailer.data.model.Product
import com.pegasusx.retailer.data.model.Variant
import com.pegasusx.retailer.ui.screens.cart.CartViewModel
import com.pegasusx.retailer.ui.theme.SoftSquircleShape
import com.pegasusx.retailer.ui.screens.procurement.components.*

@Composable
fun ProcurementScreen(
    viewModel: ProcurementViewModel = hiltViewModel(),
    cartViewModel: CartViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()

    if (uiState.showSuccess) {
        AlertDialog(
            onDismissRequest = viewModel::dismissSuccess,
            title = { Text("Order Created") },
            text = { Text("Your procurement order has been submitted successfully.") },
            confirmButton = {
                TextButton(onClick = viewModel::dismissSuccess) { Text("OK") }
            },
        )
    }

    uiState.submitError?.let { message ->
        AlertDialog(
            onDismissRequest = viewModel::clearSubmitError,
            title = { Text("Order Failed") },
            text = { Text(message) },
            confirmButton = {
                TextButton(onClick = { viewModel.clearSubmitError(); viewModel.createOrder() }) {
                    Text("Retry")
                }
            },
            dismissButton = {
                TextButton(onClick = viewModel::clearSubmitError) { Text("Cancel") }
            },
        )
    }

    Column(modifier = Modifier.fillMaxSize()) {
        LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        
            modifier = Modifier.weight(1f),
            contentPadding = PaddingValues(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        horizontalArrangement = Arrangement.spacedBy(16.dp)
    ) {
            item { ProcurementHeader(uiState) }
            item {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Icon(
                            Icons.Outlined.Lightbulb,
                            contentDescription = null,
                            tint = MaterialTheme.colorScheme.primary,
                            modifier = Modifier.size(18.dp),
                        )
                        Spacer(modifier = Modifier.width(8.dp))
                        Text(
                            "Suggestions",
                            style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.SemiBold),
                        )
                    }
                    TextButton(onClick = viewModel::toggleSelectAll) {
                        Text(
                            if (uiState.selectedIds.size == uiState.forecasts.size && uiState.forecasts.isNotEmpty()) {
                                "Deselect All"
                            } else {
                                "Select All"
                            },
                        )
                    }
                }
            }

            if (uiState.isLoading) {
                item {
                    Box(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(vertical = 32.dp),
                        contentAlignment = Alignment.Center,
                    ) {
                        CircularProgressIndicator()
                    }
                }
            } else if (uiState.forecasts.isEmpty()) {
                item {
                    Text(
                        uiState.syncMessage ?: "No procurement suggestions yet.",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f),
                    )
                }
            } else {
                itemsIndexed(uiState.forecasts, key = { _, forecast -> forecast.id }) { _, forecast ->
                    val isSelected = uiState.selectedIds.contains(forecast.id)
                    val quantity = uiState.quantities[forecast.id] ?: forecast.predictedQuantity
                    SuggestionCard(
                        forecast = forecast,
                        isSelected = isSelected,
                        quantity = quantity,
                        onToggle = { viewModel.toggleSelection(forecast) },
                        onDecrement = { viewModel.updateQuantity(forecast.id, quantity - 1) },
                        onIncrement = { viewModel.updateQuantity(forecast.id, quantity + 1) },
                    )
                }
            }

            if (uiState.selectedIds.isNotEmpty()) {
                item {
                    SelectedSummary(
                        selectedCount = uiState.selectedCount,
                        selectedUnits = uiState.selectedUnits,
                    )
                }
            }
        }

        if (uiState.selectedIds.isNotEmpty()) {
            ProcurementActionBar(
                isSubmitting = uiState.isSubmitting,
                onCreateOrder = viewModel::createOrder,
                onAddToCart = {
                    addSelectedToCart(
                        selections = viewModel.selectedProducts(),
                        cartViewModel = cartViewModel,
                    )
                    viewModel.clearSelections()
                },
            )
        }
    }
}

private fun addSelectedToCart(
    selections: List<Pair<com.pegasusx.retailer.data.model.Product, Int>>,
    cartViewModel: com.pegasusx.retailer.ui.screens.cart.CartViewModel,
) {
    selections.forEach { (product, qty) ->
        val variant = product.defaultVariant ?: return@forEach
        cartViewModel.addToCart(product, variant)
        val itemId = "${product.id}_${variant.id}"
        cartViewModel.updateQuantity(itemId, qty)
    }
}
