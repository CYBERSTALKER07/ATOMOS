package com.pegasus.retailer.ui.screens.catalog

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import com.pegasus.retailer.ui.components.modifiers.bounceCash
import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.IntrinsicSize
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
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.shape.RoundedCornerShape
import com.pegasus.retailer.ui.theme.PillShape
import com.pegasus.retailer.ui.theme.SoftSquircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Search
import androidx.compose.material.icons.rounded.Inventory2
import androidx.compose.material3.ExperimentalMaterial3Api
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import com.pegasus.retailer.data.model.ProductCategory
import com.pegasus.retailer.ui.components.ProductCard

@OptIn(ExperimentalFoundationApi::class, ExperimentalMaterial3Api::class)
@Composable
fun CatalogScreen(
    onProductCash: (productId: String) -> Unit = {},
    onCategoryCash: (categoryId: String, categoryName: String) -> Unit = { _, _ -> },
    viewModel: CatalogViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()

    Column(modifier = Modifier.fillMaxSize()) {
        // ── Search bar ──
        OutlinedTextField(
            value = uiState.searchQuery,
            onValueChange = viewModel::onSearchChanged,
            modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp),
            placeholder = { Text("Search products…") },
            leadingIcon = { Icon(Icons.Outlined.Search, contentDescription = null) },
            singleLine = true,
            shape = PillShape,
            colors = OutlinedTextFieldDefaults.colors(
                unfocusedContainerColor = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.3f),
                focusedContainerColor = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.3f),
                unfocusedBorderColor = Color.Transparent,
                focusedBorderColor = MaterialTheme.colorScheme.primary,
            ),
        )

        if (uiState.searchQuery.length >= 2 && uiState.filteredProducts.isNotEmpty()) {
            // ── Search Results Grid ──
            LazyVerticalGrid(
                columns = GridCells.Adaptive(minSize = 160.dp),
                contentPadding = PaddingValues(horizontal = 16.dp, vertical = 8.dp),
                horizontalArrangement = Arrangement.spacedBy(14.dp),
                verticalArrangement = Arrangement.spacedBy(16.dp),
                modifier = Modifier.fillMaxSize().weight(1f),
            ) {
                items(uiState.filteredProducts, key = { it.id }) { product ->
                    ProductCard(product = product, modifier = Modifier.animateItemPlacement(), onClick = { onProductCash(product.id) })
                }
                item(span = { GridItemSpan(maxLineSpan) }) { Spacer(modifier = Modifier.height(32.dp)) }
            }
        } else {
            // ── Bento Category Grid ──
            LazyColumn(
                contentPadding = PaddingValues(horizontal = 16.dp, vertical = 8.dp),
                verticalArrangement = Arrangement.spacedBy(14.dp),
                modifier = Modifier.fillMaxSize().weight(1f),
            ) {
                // Header
                item {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text(
                            "Categories",
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.Bold,
                        )
                        Text(
                            "${uiState.categories.size} types",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.4f),
                        )
                    }
                }

                // Bento layout: chunk categories into rows of mixed sizes
                val cats = uiState.categories
                var idx = 0
                // Row type 1: 2 big cards (150dp)
                if (idx + 1 < cats.size) {
                    val i0 = idx; val i1 = idx + 1
                    item {
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.spacedBy(14.dp),
                        ) {
                            BentoCategoryCard(
                                category = cats[i0],
                                height = 150.dp,
                                modifier = Modifier.animateItemPlacement().weight(1f),
                                onClick = { onCategoryCash(cats[i0].id, cats[i0].name) },
                            )
                            BentoCategoryCard(
                                category = cats[i1],
                                height = 150.dp,
                                modifier = Modifier.animateItemPlacement().weight(1f),
                                onClick = { onCategoryCash(cats[i1].id, cats[i1].name) },
                            )
                        }
                    }
                    idx += 2
                }

                // Row type 2: 1 wide (130dp) + 2 small stacked (54dp each)
                if (idx + 2 < cats.size) {
                    val j0 = idx; val j1 = idx + 1; val j2 = idx + 2
                    item {
                        Row(
                            modifier = Modifier.fillMaxWidth().height(IntrinsicSize.Min),
                            horizontalArrangement = Arrangement.spacedBy(14.dp),
                        ) {
                            BentoCategoryCard(
                                category = cats[j0],
                                height = 130.dp,
                                modifier = Modifier.animateItemPlacement().weight(1f),
                                onClick = { onCategoryCash(cats[j0].id, cats[j0].name) },
                            )
                            Column(
                                modifier = Modifier.weight(1f),
                                verticalArrangement = Arrangement.spacedBy(14.dp),
                            ) {
                                BentoCategoryCard(
                                    category = cats[j1],
                                    height = 56.dp,
                                    compact = true,
                                    onClick = { onCategoryCash(cats[j1].id, cats[j1].name) },
                                )
                                BentoCategoryCard(
                                    category = cats[j2],
                                    height = 56.dp,
                                    compact = true,
                                    onClick = { onCategoryCash(cats[j2].id, cats[j2].name) },
                                )
                            }
                        }
                    }
                    idx += 3
                }

                // Row type 3: 3 compact cards (80dp)
                while (idx + 2 < cats.size) {
                    val a = cats[idx]; val b = cats[idx + 1]; val c = cats[idx + 2]
                    item {
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.spacedBy(14.dp),
                        ) {
                            BentoCategoryCard(category = a, height = 80.dp, modifier = Modifier.weight(1f), compact = true, onClick = { onCategoryCash(a.id, a.name) })
                            BentoCategoryCard(category = b, height = 80.dp, modifier = Modifier.weight(1f), compact = true, onClick = { onCategoryCash(b.id, b.name) })
                            BentoCategoryCard(category = c, height = 80.dp, modifier = Modifier.weight(1f), compact = true, onClick = { onCategoryCash(c.id, c.name) })
                        }
                    }
                    idx += 3
                }

                // Remaining 1-2 cards
                if (idx < cats.size) {
                    val remaining = cats.subList(idx, cats.size)
                    item {
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.spacedBy(14.dp),
                        ) {
                            for (cat in remaining) {
                                BentoCategoryCard(
                                    category = cat,
                                    height = 100.dp,
                                    modifier = Modifier.animateItemPlacement().weight(1f),
                                    onClick = { onCategoryCash(cat.id, cat.name) },
                                )
                            }
                        }
                    }
                }

                item { Spacer(modifier = Modifier.height(32.dp)) }
            }
        }
    }
}

@Composable
private fun BentoCategoryCard(
    category: ProductCategory,
    height: Dp,
    modifier: Modifier = Modifier,
    compact: Boolean = false,
    onClick: () -> Unit = {},
) {
    Surface(
        modifier = modifier
            .fillMaxWidth()
            .height(height)
            .shadow(
                4.dp, SoftSquircleShape,
                ambientColor = Color.Black.copy(alpha = 0.06f),
                spotColor = Color.Black.copy(alpha = 0.06f),
            )
            .clip(SoftSquircleShape)
            .bounceCash { onClick() },
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        if (compact) {
            Row(
                modifier = Modifier.fillMaxSize().padding(12.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                // Category emoji from icon field
                CategoryGlyph(category = category, compact = true)
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        category.name,
                        style = MaterialTheme.typography.labelSmall,
                        fontWeight = FontWeight.SemiBold,
                        maxLines = 1,
                    )
                    category.productCount?.let {
                        Text(
                            "$it",
                            style = MaterialTheme.typography.bodySmall.copy(fontSize = 10.sp),
                            color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.4f),
                        )
                    }
                }
            }
        } else {
            Column(
                modifier = Modifier.fillMaxSize().padding(16.dp),
                verticalArrangement = Arrangement.Bottom,
            ) {
                CategoryGlyph(category = category)
                Spacer(modifier = Modifier.height(8.dp))
                Text(
                    category.name,
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.SemiBold,
                )
                category.productCount?.let {
                    Text(
                        "$it products",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
                    )
                }
            }
        }
    }
}

@Composable
private fun CategoryGlyph(category: ProductCategory, compact: Boolean = false) {
    if (category.icon.contains('.')) {
        Icon(
            imageVector = Icons.Rounded.Inventory2,
            contentDescription = null,
            tint = MaterialTheme.colorScheme.primary.copy(alpha = 0.8f),
            modifier = Modifier.size(if (compact) 18.dp else 28.dp),
        )
    } else {
        Text(
            text = category.icon,
            style = if (compact) MaterialTheme.typography.titleMedium else MaterialTheme.typography.headlineMedium,
        )
    }
}
