package com.pegasusx.retailer.ui.screens.catalog

import androidx.compose.foundation.ExperimentalFoundationApi
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
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Search
import androidx.compose.material.icons.rounded.Inventory2
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue

import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import com.pegasusx.retailer.data.model.ProductCategory
import com.pegasusx.retailer.ui.components.ProductCard
import com.pegasusx.retailer.ui.components.RetailerLoadingState
import com.pegasusx.retailer.ui.components.RetailerSectionHeader
import com.pegasusx.retailer.ui.components.RetailerStateKind
import com.pegasusx.retailer.ui.components.RetailerStatePane
import com.pegasusx.retailer.ui.theme.PegasusSpacing
import com.pegasusx.retailer.ui.theme.PillShape

@OptIn(ExperimentalFoundationApi::class, ExperimentalMaterial3Api::class)
@Composable
fun CatalogScreen(
    onProductCash: (productId: String) -> Unit = {},
    onCategoryCash: (categoryId: String, categoryName: String) -> Unit = { _, _ -> },
    onNavigateToSuppliers: () -> Unit = {},
    viewModel: CatalogViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()
    val isSearching = uiState.searchQuery.length >= 2
    val buyFilters = listOf("Categories", "All products", "Suppliers")

    Column(modifier = Modifier.fillMaxSize()) {
        OutlinedTextField(
            value = uiState.searchQuery,
            onValueChange = viewModel::onSearchChanged,
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 12.dp),
            placeholder = { Text("Search products, suppliers, or categories") },
            leadingIcon = {
                Icon(
                    imageVector = Icons.Outlined.Search,
                    contentDescription = "Search catalog",
                )
            },
            singleLine = true,
            shape = PillShape,
            colors = OutlinedTextFieldDefaults.colors(
                unfocusedContainerColor = MaterialTheme.colorScheme.surfaceContainerLow,
                focusedContainerColor = MaterialTheme.colorScheme.surfaceContainerLow,
                unfocusedBorderColor = MaterialTheme.colorScheme.outlineVariant,
                focusedBorderColor = MaterialTheme.colorScheme.primary,
            ),
        )

        if (!isSearching) {
            LazyRow(
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                contentPadding = PaddingValues(horizontal = 16.dp, vertical = 4.dp),
                modifier = Modifier.fillMaxWidth(),
            ) {
                items(buyFilters.size) { index ->
                    val mode = when (index) {
                        1 -> CatalogBrowseMode.ALL_PRODUCTS
                        else -> CatalogBrowseMode.CATEGORIES
                    }
                    val selected = when (index) {
                        0 -> uiState.browseMode == CatalogBrowseMode.CATEGORIES
                        1 -> uiState.browseMode == CatalogBrowseMode.ALL_PRODUCTS
                        else -> false
                    }
                    FilterChip(
                        selected = selected,
                        onClick = {
                            when (index) {
                                2 -> onNavigateToSuppliers()
                                else -> viewModel.onBrowseModeSelected(mode)
                            }
                        },
                        label = {
                            Text(
                                text = buyFilters[index],
                                style = MaterialTheme.typography.labelLarge,
                            )
                        },
                    )
                }
            }
        }

        when {
            isSearching && uiState.filteredProducts.isNotEmpty() -> {
                LazyVerticalGrid(
                    columns = GridCells.Adaptive(minSize = 160.dp),
                    contentPadding = PaddingValues(start = 16.dp, end = 16.dp, top = 8.dp, bottom = 32.dp),
                    horizontalArrangement = Arrangement.spacedBy(16.dp),
                    verticalArrangement = Arrangement.spacedBy(16.dp),
                    modifier = Modifier.fillMaxSize(),
                ) {
                    item(span = { GridItemSpan(maxLineSpan) }) {
                        RetailerSectionHeader(
                            title = "Buy results",
                            subtitle = "${uiState.filteredProducts.size} products",
                        )
                    }
                    items(uiState.filteredProducts, key = { it.id }) { product ->
                        ProductCard(
                            product = product,
                            onClick = { onProductCash(product.id) },
                        )
                    }
                }
            }

            isSearching -> {
                RetailerStatePane(
                    kind = RetailerStateKind.NoResults,
                    headline = "No products found",
                    body = "Try a different name, category, or supplier keyword.",
                    actionLabel = "Clear search",
                    onAction = { viewModel.onSearchChanged("") },
                )
            }

            uiState.browseMode == CatalogBrowseMode.ALL_PRODUCTS -> {
                if (uiState.isLoadingProducts && uiState.products.isEmpty()) {
                    RetailerLoadingState(
                        title = "Loading catalog",
                        body = "Fetching all products from connected suppliers…",
                    )
                } else if (uiState.products.isEmpty()) {
                    RetailerStatePane(
                        kind = RetailerStateKind.Empty,
                        headline = "No products",
                        body = "Connected suppliers have not published catalog SKUs yet.",
                        actionLabel = "Refresh",
                        onAction = viewModel::refresh,
                    )
                } else {
                    LazyVerticalGrid(
                        columns = GridCells.Adaptive(minSize = 160.dp),
                        contentPadding = PaddingValues(start = 16.dp, end = 16.dp, top = 8.dp, bottom = 32.dp),
                        horizontalArrangement = Arrangement.spacedBy(16.dp),
                        verticalArrangement = Arrangement.spacedBy(16.dp),
                        modifier = Modifier.fillMaxSize(),
                    ) {
                        if (uiState.supplierFilters.isNotEmpty()) {
                            item(span = { GridItemSpan(maxLineSpan) }) {
                                LazyRow(
                                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                                    contentPadding = PaddingValues(bottom = 4.dp),
                                ) {
                                    item {
                                        FilterChip(
                                            selected = uiState.selectedSupplierId.isNullOrBlank(),
                                            onClick = { viewModel.onSupplierFilterSelected(null) },
                                            label = { Text("All suppliers") },
                                        )
                                    }
                                    items(uiState.supplierFilters, key = { it.id }) { supplier ->
                                        FilterChip(
                                            selected = uiState.selectedSupplierId == supplier.id,
                                            onClick = { viewModel.onSupplierFilterSelected(supplier.id) },
                                            label = { Text(supplier.name, maxLines = 1, overflow = TextOverflow.Ellipsis) },
                                        )
                                    }
                                }
                            }
                        }
                        item(span = { GridItemSpan(maxLineSpan) }) {
                            RetailerSectionHeader(
                                title = "All products",
                                subtitle = "${uiState.displayedProducts.size} SKUs from connected suppliers",
                            )
                        }
                        items(uiState.displayedProducts, key = { it.id }) { product ->
                            ProductCard(
                                product = product,
                                onClick = { onProductCash(product.id) },
                            )
                        }
                    }
                }
            }

            else -> {
                LazyVerticalGrid(
                    columns = GridCells.Adaptive(minSize = 160.dp),
                    contentPadding = PaddingValues(start = 16.dp, end = 16.dp, top = 8.dp, bottom = 32.dp),
                    horizontalArrangement = Arrangement.spacedBy(16.dp),
                    verticalArrangement = Arrangement.spacedBy(16.dp),
                    modifier = Modifier.fillMaxSize(),
                ) {
                    item(span = { GridItemSpan(maxLineSpan) }) {
                        RetailerSectionHeader(
                            title = "Buy workspace",
                            subtitle = "Pick a category to browse supplier catalogs",
                        )
                    }
                    items(uiState.categories, key = { it.id }) { category ->
                        CategoryCard(
                            category = category,
                            onClick = { onCategoryCash(category.id, category.name) },
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun CategoryCard(
    category: ProductCategory,
    onClick: () -> Unit,
) {
    Surface(
        modifier = Modifier.fillMaxWidth(),
        onClick = onClick,
        shape = MaterialTheme.shapes.large,
        color = MaterialTheme.colorScheme.surfaceContainerLow,
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .height(152.dp)
                .padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.SpaceBetween,
        ) {
            Box(
                modifier = Modifier
                    .size(48.dp)
                    .background(
                        color = MaterialTheme.colorScheme.secondaryContainer,
                        shape = CircleShape,
                    ),
                contentAlignment = Alignment.Center,
            ) {
                Icon(
                    imageVector = Icons.Rounded.Inventory2,
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.onSecondaryContainer,
                )
            }

            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(
                    text = category.name,
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.SemiBold,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
                Row(
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    category.productCount?.let {
                        CategoryMetaPill(label = "$it products")
                    }
                    category.supplierCount?.let {
                        CategoryMetaPill(label = "$it suppliers")
                    }
                }
            }
        }
    }
}

@Composable
private fun CategoryMetaPill(label: String) {
    Surface(
        shape = PillShape,
        color = MaterialTheme.colorScheme.surfaceContainerHighest,
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.labelMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(horizontal = 10.dp, vertical = 6.dp),
        )
    }
}
