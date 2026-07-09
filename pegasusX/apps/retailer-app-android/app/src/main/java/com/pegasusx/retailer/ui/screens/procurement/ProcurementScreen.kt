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

@Composable
private fun ProcurementHeader(uiState: ProcurementUiState) {
    Surface(
        modifier = Modifier.fillMaxWidth(),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.35f),
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Icon(Icons.Outlined.AutoAwesome, contentDescription = null, tint = MaterialTheme.colorScheme.primary)
                Spacer(modifier = Modifier.width(8.dp))
                Column {
                    Text(
                        "AI Procurement",
                        style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.Bold),
                    )
                    Text(
                        "Smart suggestions based on demand analysis",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f),
                    )
                }
            }
            Spacer(modifier = Modifier.height(16.dp))
            Row(modifier = Modifier.fillMaxWidth()) {
                StatBlock(label = "Suggestions", value = "${uiState.forecasts.size}", modifier = Modifier.weight(1f))
                StatBlock(
                    label = "Selected",
                    value = "${uiState.selectedCount}",
                    modifier = Modifier.weight(1f),
                    highlighted = true,
                )
            }
        }
    }
}

@Composable
private fun StatBlock(
    label: String,
    value: String,
    modifier: Modifier = Modifier,
    highlighted: Boolean = false,
) {
    Column(modifier = modifier, horizontalAlignment = Alignment.CenterHorizontally) {
        Text(
            value,
            style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.Bold),
            color = if (highlighted) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurface,
        )
        Text(
            label,
            style = MaterialTheme.typography.labelSmall,
            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
        )
    }
}

@Composable
private fun SuggestionCard(
    forecast: DemandForecast,
    isSelected: Boolean,
    quantity: Int,
    onToggle: () -> Unit,
    onDecrement: () -> Unit,
    onIncrement: () -> Unit,
) {
    val borderModifier = if (isSelected) {
        Modifier.border(2.dp, MaterialTheme.colorScheme.primary.copy(alpha = 0.4f), SoftSquircleShape)
    } else {
        Modifier
    }

    Surface(
        modifier = Modifier
            .fillMaxWidth()
            .then(borderModifier),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Column(modifier = Modifier.padding(16.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Checkbox(checked = isSelected, onCheckedChange = { onToggle() })
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        forecast.productName,
                        style = MaterialTheme.typography.bodyMedium.copy(fontWeight = FontWeight.SemiBold),
                        maxLines = 1,
                    )
                    Text(
                        "Confidence: ${forecast.confidencePercent}",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
                    )
                }
                if (isSelected) {
                    QuantityStepper(quantity = quantity, onDecrement = onDecrement, onIncrement = onIncrement)
                } else {
                    Text(
                        "${forecast.predictedQuantity} units",
                        style = MaterialTheme.typography.labelMedium.copy(fontWeight = FontWeight.Bold),
                        color = MaterialTheme.colorScheme.primary,
                    )
                }
            }
            Spacer(modifier = Modifier.height(8.dp))
            Text(
                forecast.reasoning,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.55f),
                maxLines = 2,
            )
        }
    }
}

@Composable
private fun SelectedSummary(selectedCount: Int, selectedUnits: Int) {
    Surface(
        modifier = Modifier.fillMaxWidth(),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.35f),
    ) {
        Row(
            modifier = Modifier.padding(16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                "$selectedCount items selected",
                modifier = Modifier.weight(1f),
                style = MaterialTheme.typography.bodyMedium,
            )
            Text(
                "$selectedUnits units",
                style = MaterialTheme.typography.bodyMedium.copy(fontWeight = FontWeight.Bold),
                color = MaterialTheme.colorScheme.primary,
            )
        }
    }
}

@Composable
private fun ProcurementActionBar(
    isSubmitting: Boolean,
    onCreateOrder: () -> Unit,
    onAddToCart: () -> Unit,
) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .background(MaterialTheme.colorScheme.surface)
            .padding(16.dp),
    ) {
        Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            Button(
                onClick = onCreateOrder,
                enabled = !isSubmitting,
                modifier = Modifier.weight(1f),
            ) {
                if (isSubmitting) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(18.dp),
                        strokeWidth = 2.dp,
                    )
                } else {
                    Text("Create Order")
                }
            }
            OutlinedButton(
                onClick = onAddToCart,
                enabled = !isSubmitting,
                modifier = Modifier.weight(1f),
            ) {
                Icon(Icons.Outlined.ShoppingCart, contentDescription = null, modifier = Modifier.size(18.dp))
                Spacer(modifier = Modifier.width(6.dp))
                Text("Add to Cart")
            }
        }
    }
}

@Composable
private fun QuantityStepper(
    quantity: Int,
    onDecrement: () -> Unit,
    onIncrement: () -> Unit,
) {
    Row(
        modifier = Modifier
            .clip(RoundedCornerShape(999.dp))
            .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.5f))
            .padding(horizontal = 4.dp, vertical = 2.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        IconButton(onClick = onDecrement, modifier = Modifier.size(28.dp)) {
            Icon(Icons.Outlined.Remove, contentDescription = "Decrease", modifier = Modifier.size(16.dp))
        }
        Text(
            "$quantity",
            style = MaterialTheme.typography.labelLarge.copy(fontWeight = FontWeight.Bold, fontSize = 13.sp),
            modifier = Modifier.width(28.dp),
        )
        IconButton(onClick = onIncrement, modifier = Modifier.size(28.dp)) {
            Icon(Icons.Outlined.Add, contentDescription = "Increase", modifier = Modifier.size(16.dp))
        }
    }
}

private fun addSelectedToCart(
    selections: List<Pair<Product, Int>>,
    cartViewModel: CartViewModel,
) {
    selections.forEach { (product, qty) ->
        val variant: Variant = product.defaultVariant ?: return@forEach
        cartViewModel.addToCart(product, variant)
        val itemId = "${product.id}_${variant.id}"
        cartViewModel.updateQuantity(itemId, qty)
    }
}
