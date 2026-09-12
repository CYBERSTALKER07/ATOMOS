package com.pegasusx.retailer.ui.screens.suppliers

import androidx.compose.ui.res.stringResource

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
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.itemsIndexed
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import com.pegasusx.retailer.ui.theme.PillShape
import com.pegasusx.retailer.ui.theme.SoftSquircleShape
import com.pegasusx.retailer.ui.theme.SquircleShape
import androidx.compose.foundation.combinedClickable
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.Add
import androidx.compose.material.icons.rounded.Business
import androidx.compose.material.icons.rounded.Sync
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
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
import com.pegasusx.retailer.data.model.Supplier
import com.pegasusx.retailer.ui.components.PegasusEmptyState
import com.pegasusx.retailer.ui.theme.StatusGreen
import com.pegasusx.retailer.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MySuppliersScreen(
    onSupplierCash: (Supplier) -> Unit,
    viewModel: MySuppliersViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()
    var showConnectSheet by remember { mutableStateOf(false) }
    var supplierToRemove by remember { mutableStateOf<Supplier?>(null) }

    if (showConnectSheet) {
        ConnectSupplierSheet(
            viewModel = viewModel,
            onDismiss = {
                showConnectSheet = false
                viewModel.searchSuppliers("")
            },
        )
    }

    supplierToRemove?.let { supplier ->
        AlertDialog(
            onDismissRequest = { supplierToRemove = null },
            title = { Text("Remove vendor?") },
            text = { Text(stringResource(R.string.mobile_retailer_ui_name_will_be_removed_from_your_connected_suppliers, supplier.name)) },
            confirmButton = {
                TextButton(
                    onClick = {
                        viewModel.removeSupplier(supplier.id)
                        supplierToRemove = null
                    },
                ) { Text("Remove") }
            },
            dismissButton = {
                TextButton(onClick = { supplierToRemove = null }) { Text("Cancel") }
            },
        )
    }

    Scaffold(
        topBar = {
            TopAppBar(title = { Text("My suppliers") })
        },
        floatingActionButton = {
            FloatingActionButton(onClick = { showConnectSheet = true }) {
                Icon(Icons.Rounded.Add, contentDescription = stringResource(R.string.mobile_retailer_ui_connect_vendor))
            }
        },
    ) { innerPadding ->
    PullToRefreshBox(
        isRefreshing = uiState.isLoading || uiState.isRefreshing,
        onRefresh = viewModel::refresh,
        modifier = Modifier.fillMaxSize().padding(innerPadding),
    ) {
        Column(modifier = Modifier.fillMaxSize()) {
            if (uiState.loadIssue != null || uiState.isRefreshing) {
                val loadIssue = uiState.loadIssue
                val syncMessage = when {
                    loadIssue != null -> uiState.error ?: uiState.syncMessage.orEmpty()
                    else -> "Syncing suppliers..."
                }
                val containerColor = when (loadIssue) {
                    SuppliersLoadIssue.RESTRICTED -> MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.5f)
                    SuppliersLoadIssue.OFFLINE -> MaterialTheme.colorScheme.tertiaryContainer.copy(alpha = 0.5f)
                    SuppliersLoadIssue.ERROR -> MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.35f)
                    null -> MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.4f)
                }
                val contentColor = when (loadIssue) {
                    SuppliersLoadIssue.RESTRICTED -> MaterialTheme.colorScheme.onErrorContainer
                    SuppliersLoadIssue.OFFLINE -> MaterialTheme.colorScheme.onTertiaryContainer
                    SuppliersLoadIssue.ERROR -> MaterialTheme.colorScheme.onErrorContainer
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
                        TextButton(onClick = viewModel::refresh) {
                            Text("Retry", color = contentColor)
                        }
                    }
                }
            }

            if (uiState.analytics != null && !uiState.isLoading) {
                AnalyticsOverview(uiState.analytics!!)
            }

            if (uiState.isLoading && uiState.suppliers.isEmpty()) {
                SupplierSkeletonGrid()
            } else if (uiState.suppliers.isEmpty() && !uiState.isLoading && !uiState.isRefreshing) {
                PegasusEmptyState(
                    icon = Icons.Rounded.Business,
                    title = when (uiState.loadIssue) {
                        SuppliersLoadIssue.RESTRICTED -> "Supplier Access Restricted"
                        SuppliersLoadIssue.OFFLINE -> "Suppliers Offline"
                        SuppliersLoadIssue.ERROR -> "Suppliers Unavailable"
                        null -> "No Suppliers Yet"
                    },
                    message = uiState.error ?: "Suppliers with repeated orders will appear here automatically",
                    actionLabel = if (uiState.loadIssue != null) "Retry" else "Connect vendor",
                    onAction = {
                        if (uiState.loadIssue != null) viewModel.refresh() else showConnectSheet = true
                    },
                )
            } else {
                LazyVerticalGrid(
                    columns = GridCells.Fixed(2),
                    contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
                    horizontalArrangement = Arrangement.spacedBy(14.dp),
                    verticalArrangement = Arrangement.spacedBy(14.dp),
                ) {
                    itemsIndexed(uiState.suppliers, key = { _, s -> s.id }) { _, supplier ->
                        SupplierCard(
                            supplier = supplier,
                            onClick = { onSupplierCash(supplier) },
                            onLongClick = { supplierToRemove = supplier },
                        )
                    }
                }
            }
        }
    }
    }
}

@Composable
private fun SupplierSkeletonGrid() {
    LazyVerticalGrid(
        columns = GridCells.Fixed(2),
        contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
        horizontalArrangement = Arrangement.spacedBy(14.dp),
        verticalArrangement = Arrangement.spacedBy(14.dp),
    ) {
        items(6) {
            Surface(
                modifier = Modifier
                    .fillMaxWidth()
                    .shadow(4.dp, SoftSquircleShape, ambientColor = Color.Black.copy(alpha = 0.04f), spotColor = Color.Black.copy(alpha = 0.04f)),
                shape = SoftSquircleShape,
                color = MaterialTheme.colorScheme.surface,
            ) {
                Column(modifier = Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Box(
                            modifier = Modifier
                                .size(48.dp)
                                .clip(SquircleShape)
                                .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.45f)),
                        )
                        Spacer(modifier = Modifier.weight(1f))
                        Box(
                            modifier = Modifier
                                .size(14.dp)
                                .clip(CircleShape)
                                .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.45f)),
                        )
                    }
                    Box(
                        modifier = Modifier
                            .fillMaxWidth(0.78f)
                            .height(14.dp)
                            .clip(RoundedCornerShape(6.dp))
                            .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.45f)),
                    )
                    Box(
                        modifier = Modifier
                            .fillMaxWidth(0.5f)
                            .height(10.dp)
                            .clip(RoundedCornerShape(6.dp))
                            .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.35f)),
                    )
                    Box(
                        modifier = Modifier
                            .width(92.dp)
                            .height(20.dp)
                            .clip(PillShape)
                            .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.4f)),
                    )
                }
            }
        }
    }
}

@OptIn(ExperimentalFoundationApi::class)
@Composable
private fun SupplierCard(
    supplier: Supplier,
    onClick: () -> Unit,
    onLongClick: () -> Unit,
) {
    Surface(
        modifier = Modifier.fillMaxWidth()
            .shadow(4.dp, SoftSquircleShape, ambientColor = Color.Black.copy(alpha = 0.06f), spotColor = Color.Black.copy(alpha = 0.06f))
            .combinedClickable(onClick = onClick, onLongClick = onLongClick),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Column(modifier = Modifier.padding(14.dp)) {
            // Avatar row + auto-order indicator
            Row(verticalAlignment = Alignment.CenterVertically) {
                Box(
                    modifier = Modifier.size(48.dp).clip(SquircleShape)
                        .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.3f)),
                    contentAlignment = Alignment.Center,
                ) {
                    Text(
                        supplier.initials,
                        style = MaterialTheme.typography.labelMedium.copy(fontWeight = FontWeight.Bold),
                        color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
                    )
                }
                Spacer(modifier = Modifier.weight(1f))
                // Auto-order indicator (placeholder — always show icon if supplier has orders)
                if (supplier.orderCount > 3) {
                    Icon(
                        Icons.Rounded.Sync,
                        contentDescription = stringResource(R.string.mobile_retailer_ui_auto_order_active),
                        modifier = Modifier.size(14.dp),
                        tint = StatusGreen,
                    )
                }
            }

            Spacer(modifier = Modifier.height(10.dp))

            // Name
            Text(
                supplier.name,
                style = MaterialTheme.typography.titleSmall.copy(fontWeight = FontWeight.SemiBold),
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )

            // Category
            if (!supplier.displayCategory.isNullOrBlank()) {
                Spacer(modifier = Modifier.height(2.dp))
                Text(
                    supplier.displayCategory.orEmpty(),
                    style = MaterialTheme.typography.bodySmall.copy(fontSize = 11.sp),
                    color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.4f),
                    maxLines = 1,
                )
            }

            Spacer(modifier = Modifier.height(8.dp))

            // Order count pill
            Text(
                stringResource(R.string.mobile_retailer_ui_ordercount_orders_2, supplier.orderCount),
                style = MaterialTheme.typography.labelSmall.copy(fontSize = 10.sp, fontWeight = FontWeight.Medium),
                color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f),
                modifier = Modifier
                    .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.3f), PillShape)
                    .padding(horizontal = 8.dp, vertical = 3.dp),
            )
        }
    }
}



@Composable
private fun AnalyticsOverview(analytics: com.pegasusx.retailer.data.model.RetailerAnalytics) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 8.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp)
    ) {
        Surface(
            modifier = Modifier.fillMaxWidth().shadow(4.dp, SoftSquircleShape, ambientColor = Color.Black.copy(alpha = 0.04f), spotColor = Color.Black.copy(alpha = 0.04f)),
            shape = SoftSquircleShape,
            color = MaterialTheme.colorScheme.primaryContainer,
        ) {
            Column(modifier = Modifier.padding(16.dp)) {
                Text(
                    text = stringResource(R.string.mobile_retailer_ui_mtd_settlement),
                    style = MaterialTheme.typography.labelMedium.copy(fontWeight = FontWeight.SemiBold, letterSpacing = 1.sp),
                    color = MaterialTheme.colorScheme.onPrimaryContainer.copy(alpha = 0.7f),
                    modifier = Modifier.padding(bottom = 8.dp)
                )
                Text(
                    text = stringResource(R.string.mobile_retailer_ui_totalthismonth, analytics.totalThisMonth),
                    style = MaterialTheme.typography.displaySmall.copy(fontWeight = FontWeight.Light),
                    color = MaterialTheme.colorScheme.onPrimaryContainer,
                )
            }
        }

        if (analytics.topSuppliers.isNotEmpty()) {
            Surface(
                modifier = Modifier.fillMaxWidth().shadow(4.dp, SoftSquircleShape, ambientColor = Color.Black.copy(alpha = 0.04f), spotColor = Color.Black.copy(alpha = 0.04f)),
                shape = SoftSquircleShape,
                color = MaterialTheme.colorScheme.surface,
            ) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text(
                        text = stringResource(R.string.mobile_retailer_ui_trade_breakdown),
                        style = MaterialTheme.typography.labelSmall.copy(letterSpacing = 1.sp, fontWeight = FontWeight.SemiBold),
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(bottom = 12.dp)
                    )
                    
                    analytics.topSuppliers.take(5).forEachIndexed { index, supplier ->
                        Row(
                            verticalAlignment = Alignment.CenterVertically,
                            modifier = Modifier.fillMaxWidth().padding(vertical = 6.dp)
                        ) {
                            Box(
                                modifier = Modifier
                                    .size(24.dp)
                                    .clip(RoundedCornerShape(6.dp))
                                    .background(MaterialTheme.colorScheme.surfaceVariant),
                                contentAlignment = Alignment.Center
                            ) {
                                Text("${index + 1}", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                            }
                            Spacer(modifier = Modifier.width(12.dp))
                            Column(modifier = Modifier.weight(1f)) {
                                Text(supplier.supplierName, style = MaterialTheme.typography.bodyMedium, maxLines = 1, overflow = TextOverflow.Ellipsis)
                                Text(stringResource(R.string.mobile_retailer_ui_ordercount_trades, supplier.orderCount), style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                            }
                            Text(stringResource(R.string.mobile_retailer_ui_total, supplier.total), style = MaterialTheme.typography.bodyMedium.copy(fontWeight = FontWeight.Medium))
                        }
                    }
                }
            }
        }
    }
}
