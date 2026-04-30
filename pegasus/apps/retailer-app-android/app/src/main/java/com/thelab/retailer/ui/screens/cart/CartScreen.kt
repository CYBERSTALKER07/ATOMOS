package com.pegasus.retailer.ui.screens.cart

import androidx.compose.foundation.background
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
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.shape.CircleShape
import com.pegasus.retailer.ui.theme.PillShape
import com.pegasus.retailer.ui.theme.SoftSquircleShape
import com.pegasus.retailer.ui.theme.SquircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Add
import androidx.compose.material.icons.outlined.Delete
import androidx.compose.material.icons.outlined.Remove
import androidx.compose.material.icons.outlined.ShoppingCart
import androidx.compose.material.icons.rounded.ArrowForward
import androidx.compose.material.icons.rounded.Eco
import androidx.compose.material.icons.rounded.GridView
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import com.pegasus.retailer.data.model.CartItem
import com.pegasus.retailer.ui.components.CheckoutSheet
import com.pegasus.retailer.ui.components.DefaultCheckoutPaymentOptions
import com.pegasus.retailer.ui.theme.StatusGreen
import com.pegasus.retailer.ui.theme.StatusRed

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CartScreen(
    viewModel: CartViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()
    val snackbarHostState = remember { SnackbarHostState() }
    var showSupplierClosedDialog by remember { mutableStateOf(false) }

    LaunchedEffect(uiState.removedItemMessage) {
        uiState.removedItemMessage?.let {
            snackbarHostState.showSnackbar(it)
            viewModel.clearRemovedItemMessage()
        }
    }

    LaunchedEffect(uiState.checkoutError) {
        uiState.checkoutError?.let {
            snackbarHostState.showSnackbar(it)
            viewModel.clearCheckoutError()
        }
    }

    Scaffold(
        snackbarHost = { SnackbarHost(snackbarHostState) },
        containerColor = MaterialTheme.colorScheme.background,
    ) { innerPadding ->
    Box(modifier = Modifier.fillMaxSize().padding(innerPadding)) {
        if (uiState.isEmpty) {
            EmptyCartView()
            return@Box
        }
        Column(modifier = Modifier.fillMaxSize()) {
            // ── Cart items ──
            LazyColumn(
                contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
                modifier = Modifier.weight(1f),
            ) {
                // Header row
                item {
                    Row(modifier = Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                        Text(
                            "${uiState.totalItems} items in your cart",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
                        )
                        Spacer(modifier = Modifier.weight(1f))
                        TextButton(onClick = viewModel::clearCart) {
                            Text("Clear All", style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.SemiBold), color = StatusRed)
                        }
                    }
                }

                itemsIndexed(uiState.items, key = { _, item -> item.id }) { _, item ->
                    CartItemCard(item = item, onUpdateQuantity = viewModel::updateQuantity, onRemove = viewModel::removeItem)
                }

                item { Spacer(modifier = Modifier.height(16.dp)) }
            }

            // ── Bottom bar ──
            CartBottomBar(
                subtotal = uiState.displaySubtotal,
                onCheckout = {
                    if (!uiState.supplierIsActive) {
                        showSupplierClosedDialog = true
                    } else {
                        viewModel.showCheckout()
                    }
                },
            )
        }

        if (uiState.showCheckout) {
            CheckoutSheet(
                phase = uiState.checkoutPhase,
                productName = uiState.firstProductName,
                itemCount = uiState.totalItems,
                subtotal = uiState.displaySubtotal,
                shipping = uiState.displayShipping,
                discount = uiState.displayDiscount,
                total = uiState.displayTotal,
                selectedPaymentGateway = uiState.selectedPaymentGateway,
                paymentLabel = uiState.selectedPaymentLabel,
                paymentOptions = DefaultCheckoutPaymentOptions,
                onBuy = viewModel::processPayment,
                onSelectPayment = viewModel::setSelectedPaymentGateway,
                onDismiss = viewModel::dismissCheckout,
            )
        }

        if (showSupplierClosedDialog) {
            AlertDialog(
                onDismissRequest = { showSupplierClosedDialog = false },
                title = { Text("Supplier is Currently Closed") },
                text = {
                    Text(
                        "This supplier is off-shift or outside business hours. " +
                        "Your order will be placed, but processing will not begin until they are back online."
                    )
                },
                confirmButton = {
                    Button(
                        onClick = {
                            showSupplierClosedDialog = false
                            viewModel.showCheckout()
                        }
                    ) {
                        Text("I Understand, Place Order")
                    }
                },
                dismissButton = {
                    TextButton(onClick = { showSupplierClosedDialog = false }) {
                        Text("Cancel")
                    }
                },
            )
        }
    }
    } // Scaffold
}

@Composable
private fun CartItemCard(
    item: CartItem,
    onUpdateQuantity: (String, Int) -> Unit,
    onRemove: (String) -> Unit,
) {
    Surface(
        modifier = Modifier.fillMaxWidth()
            .shadow(4.dp, SoftSquircleShape, ambientColor = Color.Black.copy(alpha = 0.06f), spotColor = Color.Black.copy(alpha = 0.06f)),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Row(modifier = Modifier.padding(12.dp), verticalAlignment = Alignment.CenterVertically) {
            // Image placeholder
            Box(
                modifier = Modifier.size(72.dp).clip(SquircleShape)
                    .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.3f)),
                contentAlignment = Alignment.Center,
            ) {
                Icon(Icons.Rounded.Eco, contentDescription = null, modifier = Modifier.size(26.dp), tint = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.2f))
            }

            Spacer(modifier = Modifier.width(12.dp))

            // Info
            Column(modifier = Modifier.weight(1f)) {
                Text(item.product.name, style = MaterialTheme.typography.titleSmall.copy(fontWeight = FontWeight.SemiBold), maxLines = 2, overflow = TextOverflow.Ellipsis)
                Spacer(modifier = Modifier.height(4.dp))
                Row(horizontalArrangement = Arrangement.spacedBy(4.dp)) {
                    TagPill(item.variant.size)
                    TagPill(item.variant.pack)
                }
                Spacer(modifier = Modifier.height(2.dp))
                Text(
                    "%,.0f each".format(item.variant.price),
                    style = MaterialTheme.typography.bodySmall.copy(fontSize = 11.sp),
                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.4f),
                )
            }

            Spacer(modifier = Modifier.width(8.dp))

            // Price + stepper column
            Column(horizontalAlignment = Alignment.End) {
                Text(
                    "%,.0f".format(item.totalPrice),
                    style = MaterialTheme.typography.titleSmall.copy(fontWeight = FontWeight.Bold),
                )
                Spacer(modifier = Modifier.height(6.dp))
                QuantityStepper(
                    quantity = item.quantity,
                    onDecrement = { onUpdateQuantity(item.id, item.quantity - 1) },
                    onIncrement = { onUpdateQuantity(item.id, item.quantity + 1) },
                )
            }
        }
    }
}

@Composable
private fun QuantityStepper(quantity: Int, onDecrement: () -> Unit, onIncrement: () -> Unit) {
    Row(verticalAlignment = Alignment.CenterVertically) {
        IconButton(onClick = onDecrement, modifier = Modifier.size(28.dp)) {
            Icon(
                if (quantity <= 1) Icons.Outlined.Delete else Icons.Outlined.Remove,
                contentDescription = null,
                modifier = Modifier.size(14.dp),
                tint = if (quantity <= 1) StatusRed else MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f),
            )
        }
        Text(
            "$quantity",
            style = MaterialTheme.typography.labelMedium.copy(fontWeight = FontWeight.Bold),
            modifier = Modifier.width(24.dp),
            textAlign = androidx.compose.ui.text.style.TextAlign.Center,
        )
        IconButton(onClick = onIncrement, modifier = Modifier.size(28.dp)) {
            Icon(Icons.Outlined.Add, contentDescription = null, modifier = Modifier.size(14.dp), tint = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f))
        }
    }
}

@Composable
private fun TagPill(text: String) {
    Text(
        text,
        style = MaterialTheme.typography.labelSmall.copy(fontSize = 10.sp, fontWeight = FontWeight.Medium),
        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.4f),
        modifier = Modifier.background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.3f), PillShape).padding(horizontal = 6.dp, vertical = 2.dp),
    )
}

@Composable
private fun CartBottomBar(subtotal: String, onCheckout: () -> Unit) {
    Column {
        HorizontalDivider(thickness = 0.5.dp, color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.3f))

        Column(modifier = Modifier.fillMaxWidth().background(MaterialTheme.colorScheme.surface).padding(16.dp)) {
            // Subtotal row
            Row(modifier = Modifier.fillMaxWidth()) {
                Text("Subtotal", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f))
                Spacer(modifier = Modifier.weight(1f))
                Text(subtotal, style = MaterialTheme.typography.bodySmall.copy(fontWeight = FontWeight.Medium))
            }
            Spacer(modifier = Modifier.height(6.dp))
            // Delivery row
            Row(modifier = Modifier.fillMaxWidth()) {
                Text("Delivery", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f))
                Spacer(modifier = Modifier.weight(1f))
                Text("Free", style = MaterialTheme.typography.bodySmall.copy(fontWeight = FontWeight.Medium), color = StatusGreen)
            }

            Spacer(modifier = Modifier.height(10.dp))
            HorizontalDivider(thickness = 0.5.dp, color = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.2f))
            Spacer(modifier = Modifier.height(10.dp))

            // Total + Checkout button
            Row(modifier = Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                Column {
                    Text("Total", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f))
                    Text(subtotal, style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.Bold))
                }
                Spacer(modifier = Modifier.weight(1f))
                Surface(
                    onClick = onCheckout,
                    shape = PillShape,
                    color = MaterialTheme.colorScheme.primary,
                ) {
                    Row(modifier = Modifier.padding(horizontal = 24.dp, vertical = 12.dp), verticalAlignment = Alignment.CenterVertically) {
                        Text("Checkout", style = MaterialTheme.typography.labelMedium.copy(fontWeight = FontWeight.Bold), color = MaterialTheme.colorScheme.onPrimary)
                        Spacer(modifier = Modifier.width(4.dp))
                        Icon(Icons.Rounded.ArrowForward, contentDescription = null, modifier = Modifier.size(14.dp), tint = MaterialTheme.colorScheme.onPrimary)
                    }
                }
            }
        }
    }
}

@Composable
private fun EmptyCartView() {
    Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        Column(horizontalAlignment = Alignment.CenterHorizontally, modifier = Modifier.padding(32.dp)) {
            // Double ring empty state
            Box(contentAlignment = Alignment.Center) {
                Box(modifier = Modifier.size(130.dp).clip(CircleShape).background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.15f)))
                Box(modifier = Modifier.size(100.dp).clip(CircleShape).background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.3f)), contentAlignment = Alignment.Center) {
                    Icon(Icons.Outlined.ShoppingCart, contentDescription = null, modifier = Modifier.size(40.dp), tint = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.3f))
                }
            }
            Spacer(modifier = Modifier.height(20.dp))
            Text("Your cart is empty", style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.Bold))
            Spacer(modifier = Modifier.height(8.dp))
            Text(
                "Browse the catalog and add\nproducts to get started",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
                textAlign = androidx.compose.ui.text.style.TextAlign.Center,
            )
            Spacer(modifier = Modifier.height(20.dp))
            Surface(
                onClick = { /* user taps Catalog tab instead */ },
                shape = PillShape,
                color = MaterialTheme.colorScheme.primary,
            ) {
                Row(modifier = Modifier.padding(horizontal = 24.dp, vertical = 12.dp), verticalAlignment = Alignment.CenterVertically) {
                    Icon(Icons.Rounded.GridView, contentDescription = null, modifier = Modifier.size(14.dp), tint = MaterialTheme.colorScheme.onPrimary)
                    Spacer(modifier = Modifier.width(6.dp))
                    Text("Browse Catalog", style = MaterialTheme.typography.labelMedium.copy(fontWeight = FontWeight.SemiBold), color = MaterialTheme.colorScheme.onPrimary)
                }
            }
        }
    }
}
