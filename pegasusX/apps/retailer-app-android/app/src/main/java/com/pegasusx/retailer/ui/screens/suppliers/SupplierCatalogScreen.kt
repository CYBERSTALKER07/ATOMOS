package com.pegasusx.retailer.ui.screens.suppliers

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.items

import androidx.compose.foundation.lazy.grid.GridItemSpan

import androidx.compose.foundation.lazy.grid.LazyVerticalGrid

import androidx.compose.foundation.lazy.grid.GridCells

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
import androidx.compose.foundation.background
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.ui.draw.clip
import com.pegasusx.retailer.ui.theme.StatusGreen
import com.pegasusx.retailer.ui.theme.StatusRed
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.ArrowBack
import androidx.compose.material.icons.rounded.Inventory2
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import com.pegasusx.retailer.ui.components.PegasusEmptyState
import com.pegasusx.retailer.ui.components.ProductCard
import com.pegasusx.retailer.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SupplierCatalogScreen(
    supplierId: String,
    supplierName: String,
    supplierCategory: String,
    supplierIsActive: Boolean = true,
    onBack: () -> Unit,
    onProductCash: (String) -> Unit,
    viewModel: SupplierCatalogViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()
    val groupedProducts = uiState.products.groupBy { it.categoryName ?: supplierCategory.ifBlank { "Other" } }
        .toSortedMap(String.CASE_INSENSITIVE_ORDER)

    LaunchedEffect(supplierId) {
        viewModel.load(supplierId)
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column {
                        Text(supplierName, style = MaterialTheme.typography.titleMedium.copy(fontWeight = FontWeight.SemiBold))
                        if (supplierCategory.isNotBlank()) {
                            Text(
                                supplierCategory,
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f),
                            )
                        }
                        Row(
                            verticalAlignment = Alignment.CenterVertically,
                            horizontalArrangement = Arrangement.spacedBy(4.dp),
                            modifier = Modifier.padding(top = 2.dp),
                        ) {
                            Box(
                                modifier = Modifier
                                    .size(7.dp)
                                    .background(
                                        if (supplierIsActive) StatusGreen else StatusRed,
                                        CircleShape,
                                    )
                            )
                            Text(
                                text = if (supplierIsActive) "OPEN" else "CLOSED",
                                style = MaterialTheme.typography.labelSmall,
                                color = if (supplierIsActive) StatusGreen else StatusRed,
                            )
                        }
                    }
                },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Rounded.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(containerColor = MaterialTheme.colorScheme.surface),
            )
        },
    ) { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding),
        ) {
            val syncMessage = when {
                uiState.loadIssue != null -> uiState.error ?: uiState.syncMessage.orEmpty()
                uiState.isLoading && uiState.products.isNotEmpty() -> "Syncing supplier catalog..."
                else -> null
            }

            if (syncMessage != null) {
                val loadIssue = uiState.loadIssue
                val containerColor = when (loadIssue) {
                    SupplierCatalogLoadIssue.RESTRICTED -> MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.5f)
                    SupplierCatalogLoadIssue.OFFLINE -> MaterialTheme.colorScheme.tertiaryContainer.copy(alpha = 0.5f)
                    SupplierCatalogLoadIssue.ERROR -> MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.35f)
                    null -> MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.4f)
                }
                val contentColor = when (loadIssue) {
                    SupplierCatalogLoadIssue.RESTRICTED -> MaterialTheme.colorScheme.onErrorContainer
                    SupplierCatalogLoadIssue.OFFLINE -> MaterialTheme.colorScheme.onTertiaryContainer
                    SupplierCatalogLoadIssue.ERROR -> MaterialTheme.colorScheme.onErrorContainer
                    null -> MaterialTheme.colorScheme.onPrimaryContainer
                }

                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 16.dp, vertical = 8.dp)
                        .clip(RoundedCornerShape(12.dp))
                        .background(containerColor)
                        .padding(horizontal = 12.dp, vertical = 10.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        text = syncMessage,
                        modifier = Modifier.weight(1f),
                        style = MaterialTheme.typography.labelMedium,
                        color = contentColor,
                    )
                    if (loadIssue != null) {
                        TextButton(onClick = { viewModel.load(supplierId) }) {
                            Text("Retry", color = contentColor)
                        }
                    }
                }
            }

            Box(modifier = Modifier.fillMaxSize()) {
                when {
                    uiState.isLoading && uiState.products.isEmpty() -> {
                        LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        
                            modifier = Modifier.fillMaxSize(),
                            contentPadding = PaddingValues(16.dp),
                            verticalArrangement = Arrangement.spacedBy(16.dp),
        horizontalArrangement = Arrangement.spacedBy(16.dp)
    ) {
                            item {
                                Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                                    Box(
                                        modifier = Modifier
                                            .fillMaxWidth(0.44f)
                                            .height(18.dp)
                                            .clip(androidx.compose.foundation.shape.RoundedCornerShape(6.dp))
                                            .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.4f)),
                                    )
                                    Box(
                                        modifier = Modifier
                                            .fillMaxWidth(0.28f)
                                            .height(12.dp)
                                            .clip(androidx.compose.foundation.shape.RoundedCornerShape(6.dp))
                                            .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.3f)),
                                    )
                                }
                            }
                            items(5) {
                                SupplierCatalogSkeletonCard()
                            }
                        }
                    }

                    uiState.products.isEmpty() -> {
                        PegasusEmptyState(
                            icon = Icons.Rounded.Inventory2,
                            title = when (uiState.loadIssue) {
                                SupplierCatalogLoadIssue.RESTRICTED -> "Supplier Catalog Access Restricted"
                                SupplierCatalogLoadIssue.OFFLINE -> "Supplier Catalog Offline"
                                SupplierCatalogLoadIssue.ERROR -> "Supplier Catalog Unavailable"
                                null -> "No Products Yet"
                            },
                            message = uiState.error ?: "This supplier has no active products in the catalog.",
                            modifier = Modifier.fillMaxSize(),
                            actionLabel = if (uiState.error != null) "Retry" else "Refresh Catalog",
                            onAction = { viewModel.load(supplierId) },
                        )
                    }

                    else -> {
                        LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        
                            modifier = Modifier.fillMaxSize(),
                            contentPadding = PaddingValues(16.dp),
                            verticalArrangement = Arrangement.spacedBy(16.dp),
        horizontalArrangement = Arrangement.spacedBy(16.dp)
    ) {
                            groupedProducts.forEach { (categoryName, products) ->
                                item(key = "header-$categoryName") {
                                    Column(verticalArrangement = Arrangement.spacedBy(4.dp)) {
                                        Text(
                                            categoryName,
                                            style = MaterialTheme.typography.titleSmall.copy(fontWeight = FontWeight.Bold),
                                        )
                                        Text(
                                            stringResource(R.string.mobile_retailer_ui_size_products, products.size),
                                            style = MaterialTheme.typography.bodySmall,
                                            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f),
                                        )
                                    }
                                }
                                items(products, key = { it.id }) { product ->
                                    ProductCard(
                                        product = product,
                                        onClick = { onProductCash(product.id) },
                                        modifier = Modifier.fillMaxWidth(),
                                    )
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun SupplierCatalogSkeletonCard() {
    Surface(
        modifier = Modifier.fillMaxWidth(),
        color = MaterialTheme.colorScheme.surface,
        shape = com.pegasusx.retailer.ui.theme.SoftSquircleShape,
    ) {
        Row(
            modifier = Modifier.padding(16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Box(
                modifier = Modifier
                    .size(64.dp)
                    .clip(androidx.compose.foundation.shape.RoundedCornerShape(18.dp))
                    .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.35f)),
            )
            Spacer(modifier = Modifier.size(12.dp))
            Column(modifier = Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                Box(
                    modifier = Modifier
                        .fillMaxWidth(0.6f)
                        .height(14.dp)
                        .clip(androidx.compose.foundation.shape.RoundedCornerShape(6.dp))
                        .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.45f)),
                )
                Box(
                    modifier = Modifier
                        .fillMaxWidth(0.4f)
                        .height(12.dp)
                        .clip(androidx.compose.foundation.shape.RoundedCornerShape(6.dp))
                        .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.3f)),
                )
            }
            Spacer(modifier = Modifier.size(12.dp))
            Box(
                modifier = Modifier
                    .size(width = 34.dp, height = 22.dp)
                    .clip(androidx.compose.foundation.shape.RoundedCornerShape(11.dp))
                    .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.35f)),
            )
        }
    }
}