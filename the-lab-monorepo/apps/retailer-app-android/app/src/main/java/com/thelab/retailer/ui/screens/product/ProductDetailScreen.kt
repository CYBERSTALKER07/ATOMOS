package com.thelab.retailer.ui.screens.product

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
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.ArrowBack
import androidx.compose.material.icons.outlined.AutoAwesome
import androidx.compose.material.icons.outlined.ShoppingCart
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Surface
import androidx.compose.material3.Switch
import androidx.compose.material3.SwitchDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import coil.compose.AsyncImage
import com.thelab.retailer.data.model.Product
import com.thelab.retailer.data.model.Variant
import com.thelab.retailer.ui.screens.autoorder.EnableTarget
import com.thelab.retailer.ui.theme.SoftSquircleShape
import kotlinx.coroutines.launch

@Composable
fun ProductDetailScreen(
    productId: String,
    onBack: () -> Unit,
    onAddToCart: ((Product, Variant) -> Unit)? = null,
    viewModel: ProductDetailViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()
    val snackbarHostState = remember { SnackbarHostState() }
    val scope = rememberCoroutineScope()

    LaunchedEffect(productId) {
        viewModel.load(productId)
    }

    // History/Fresh dialog
    val pendingTarget = uiState.pendingEnableTarget
    if (pendingTarget != null) {
        val entityLabel = when (pendingTarget) {
            is EnableTarget.Product -> "this product"
            is EnableTarget.Variant -> "this variant / SKU"
            else -> "this item"
        }
        AlertDialog(
            onDismissRequest = viewModel::dismissEnableDialog,
            title = { Text("Use Previous Analytics?") },
            text = { Text("Use existing order history for $entityLabel, or start fresh? Starting fresh requires at least 2 orders.") },
            confirmButton = {
                TextButton(onClick = { viewModel.confirmEnable(useHistory = true) }) {
                    Text("Use History")
                }
            },
            dismissButton = {
                Row {
                    TextButton(onClick = viewModel::dismissEnableDialog) { Text("Cancel") }
                    TextButton(onClick = { viewModel.confirmEnable(useHistory = false) }) { Text("Start Fresh") }
                }
            },
        )
    }

    Scaffold(
        snackbarHost = { SnackbarHost(snackbarHostState) },
        bottomBar = {
            val product = uiState.product
            if (product != null && !uiState.isLoading && onAddToCart != null) {
                Surface(
                    tonalElevation = 3.dp,
                    shadowElevation = 8.dp,
                ) {
                    Button(
                        onClick = {
                            val variant = product.defaultVariant ?: Variant(
                                id = product.id,
                                size = product.name,
                                pack = "Single",
                                packCount = 1,
                                weightPerUnit = "",
                                price = (product.price ?: 0).toDouble(),
                            )
                            onAddToCart(product, variant)
                            scope.launch { snackbarHostState.showSnackbar("${product.name} added to cart") }
                        },
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(16.dp)
                            .height(52.dp),
                        shape = RoundedCornerShape(14.dp),
                        colors = ButtonDefaults.buttonColors(
                            containerColor = MaterialTheme.colorScheme.primary,
                        ),
                    ) {
                        Icon(
                            Icons.Outlined.ShoppingCart,
                            contentDescription = null,
                            modifier = Modifier.size(20.dp),
                        )
                        Spacer(modifier = Modifier.width(8.dp))
                        Text(
                            "Add to Cart \u2014 ${product.displayPrice}",
                            style = MaterialTheme.typography.titleSmall.copy(fontWeight = FontWeight.Bold),
                        )
                    }
                }
            }
        },
    ) { scaffoldPadding ->
    Column(modifier = Modifier.fillMaxSize().padding(scaffoldPadding)) {
        // ── Top bar ──
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 8.dp, vertical = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = onBack) {
                Icon(Icons.AutoMirrored.Rounded.ArrowBack, contentDescription = "Back")
            }
            Text(
                text = uiState.product?.name ?: "Product",
                style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.Bold),
                modifier = Modifier.weight(1f),
                maxLines = 1,
            )
        }

        if (uiState.isLoading) {
            Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                CircularProgressIndicator(color = MaterialTheme.colorScheme.primary)
            }
        } else if (uiState.product == null) {
            Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                Text("Product not found.", color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f))
            }
        } else {
            val product = uiState.product!!

        val settings = uiState.settings
        val productAutoOrderEnabled = settings?.productOverrides
            ?.firstOrNull { it.productId == product.id }
            ?.enabled == true

        LazyColumn(
            contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
            verticalArrangement = Arrangement.spacedBy(14.dp),
        ) {
            // ── Hero image ──
            if (product.imageUrl != null) {
                item {
                    AsyncImage(
                        model = product.imageUrl,
                        contentDescription = product.name,
                        contentScale = ContentScale.Crop,
                        modifier = Modifier
                            .fillMaxWidth()
                            .height(220.dp)
                            .clip(SoftSquircleShape),
                    )
                }
            }

            // ── Product info card ──
            item {
                Surface(
                    modifier = Modifier
                        .fillMaxWidth()
                        .shadow(4.dp, SoftSquircleShape, ambientColor = Color.Black.copy(alpha = 0.06f)),
                    shape = SoftSquircleShape,
                    color = MaterialTheme.colorScheme.surface,
                ) {
                    Column(modifier = Modifier.padding(16.dp)) {
                        Text(product.name, style = MaterialTheme.typography.titleLarge.copy(fontWeight = FontWeight.Bold))
                        if (product.description.isNotBlank()) {
                            Spacer(modifier = Modifier.height(8.dp))
                            Text(
                                product.description,
                                style = MaterialTheme.typography.bodyMedium,
                                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.65f),
                            )
                        }
                        if (product.nutrition.isNotBlank()) {
                            Spacer(modifier = Modifier.height(8.dp))
                            Text(
                                "Nutrition: ${product.nutrition}",
                                style = MaterialTheme.typography.bodySmall.copy(fontSize = 11.sp),
                                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.4f),
                            )
                        }
                    }
                }
            }

            // ── Product-level auto-order toggle ──
            item {
                Surface(
                    modifier = Modifier
                        .fillMaxWidth()
                        .shadow(2.dp, SoftSquircleShape, ambientColor = Color.Black.copy(alpha = 0.04f)),
                    shape = SoftSquircleShape,
                    color = MaterialTheme.colorScheme.surface,
                ) {
                    Row(
                        modifier = Modifier.padding(horizontal = 16.dp, vertical = 14.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Box(
                            modifier = Modifier
                                .size(36.dp)
                                .clip(RoundedCornerShape(10.dp))
                                .background(
                                    if (productAutoOrderEnabled) MaterialTheme.colorScheme.primary.copy(alpha = 0.12f)
                                    else MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.3f),
                                ),
                            contentAlignment = Alignment.Center,
                        ) {
                            Icon(
                                Icons.Outlined.AutoAwesome,
                                contentDescription = null,
                                modifier = Modifier.size(18.dp),
                                tint = if (productAutoOrderEnabled) MaterialTheme.colorScheme.primary
                                       else MaterialTheme.colorScheme.onSurface.copy(alpha = 0.4f),
                            )
                        }
                        Spacer(modifier = Modifier.width(12.dp))
                        Column(modifier = Modifier.weight(1f)) {
                            Text(
                                "Auto-Order this product",
                                style = MaterialTheme.typography.bodyMedium.copy(fontWeight = FontWeight.SemiBold),
                            )
                            Text(
                                "Applies to all variants of ${product.name}",
                                style = MaterialTheme.typography.bodySmall.copy(fontSize = 11.sp),
                                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.4f),
                            )
                        }
                        Switch(
                            checked = productAutoOrderEnabled,
                            onCheckedChange = { viewModel.onToggleProduct(product.id, it) },
                            colors = SwitchDefaults.colors(
                                checkedTrackColor = MaterialTheme.colorScheme.primary,
                                checkedThumbColor = MaterialTheme.colorScheme.onPrimary,
                            ),
                        )
                    }
                }
            }

            // ── Variants section ──
            if (product.variants.isNotEmpty()) {
                item {
                    Text(
                        "Variants",
                        style = MaterialTheme.typography.labelMedium.copy(fontWeight = FontWeight.Bold),
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
                        modifier = Modifier.padding(start = 4.dp, top = 4.dp),
                    )
                }
                items(product.variants, key = { it.id }) { variant ->
                    val variantAutoOrderEnabled = settings?.variantOverrides
                        ?.firstOrNull { it.skuId == variant.id }
                        ?.enabled == true
                    VariantRow(
                        variant = variant,
                        autoOrderEnabled = variantAutoOrderEnabled,
                        onToggle = { viewModel.onToggleVariant(variant.id, it) },
                        onAddToCart = if (onAddToCart != null) {
                            {
                                onAddToCart(product, variant)
                                scope.launch { snackbarHostState.showSnackbar("${variant.size} added to cart") }
                            }
                        } else null,
                    )
                }
            }

            item { Spacer(modifier = Modifier.height(32.dp)) }
        }
        } // else
    }
    } // Scaffold
}

@Composable
private fun VariantRow(
    variant: Variant,
    autoOrderEnabled: Boolean,
    onToggle: (Boolean) -> Unit,
    onAddToCart: (() -> Unit)? = null,
) {
    Surface(
        modifier = Modifier
            .fillMaxWidth()
            .shadow(2.dp, SoftSquircleShape, ambientColor = Color.Black.copy(alpha = 0.04f)),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Column(modifier = Modifier.padding(horizontal = 16.dp, vertical = 12.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        "${variant.size} — ${variant.pack}",
                        style = MaterialTheme.typography.bodyMedium.copy(fontWeight = FontWeight.Medium),
                        maxLines = 1,
                    )
                    Text(
                        "Pack: ${variant.packCount}  ·  ${variant.weightPerUnit}  ·  ${"%,.0f".format(variant.price)}",
                        style = MaterialTheme.typography.bodySmall.copy(fontSize = 11.sp),
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.45f),
                    )
                }
                Switch(
                    checked = autoOrderEnabled,
                    onCheckedChange = onToggle,
                    colors = SwitchDefaults.colors(
                        checkedTrackColor = MaterialTheme.colorScheme.primary,
                        checkedThumbColor = MaterialTheme.colorScheme.onPrimary,
                    ),
                )
            }
            if (onAddToCart != null) {
                Spacer(modifier = Modifier.height(8.dp))
                Button(
                    onClick = onAddToCart,
                    modifier = Modifier.fillMaxWidth().height(40.dp),
                    shape = RoundedCornerShape(10.dp),
                    colors = ButtonDefaults.buttonColors(
                        containerColor = MaterialTheme.colorScheme.primary,
                    ),
                ) {
                    Icon(
                        Icons.Outlined.ShoppingCart,
                        contentDescription = null,
                        modifier = Modifier.size(16.dp),
                    )
                    Spacer(modifier = Modifier.width(6.dp))
                    Text("Add to Cart", style = MaterialTheme.typography.labelMedium.copy(fontWeight = FontWeight.SemiBold))
                }
            }
        }
    }
}
