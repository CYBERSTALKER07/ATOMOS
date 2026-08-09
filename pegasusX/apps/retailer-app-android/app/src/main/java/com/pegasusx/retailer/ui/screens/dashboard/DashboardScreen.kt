package com.pegasusx.retailer.ui.screens.dashboard

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.items

import androidx.compose.foundation.lazy.grid.GridItemSpan

import androidx.compose.foundation.lazy.grid.LazyVerticalGrid

import androidx.compose.foundation.lazy.grid.GridCells

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
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.rounded.TrendingUp
import androidx.compose.material.icons.rounded.AddShoppingCart
import androidx.compose.material.icons.rounded.ShoppingCart
import androidx.compose.material.icons.rounded.AutoAwesome
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material.icons.rounded.History
import androidx.compose.material.icons.rounded.Inventory2
import androidx.compose.material.icons.rounded.Map
import androidx.compose.material.icons.rounded.Search
import androidx.compose.material.icons.outlined.ShoppingBag
import androidx.compose.material.icons.outlined.DeviceHub
import androidx.compose.material.icons.outlined.Insights
import androidx.compose.material.icons.rounded.AccountCircle
import androidx.compose.material.icons.rounded.Storefront
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilledIconButton
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SegmentedButton
import androidx.compose.material3.SegmentedButtonDefaults
import androidx.compose.material3.SingleChoiceSegmentedButtonRow
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import com.pegasusx.retailer.data.model.DemandForecast
import com.pegasusx.retailer.data.model.Product
import com.pegasusx.retailer.ui.components.RetailerMetricTile
import com.pegasus.design.PegasusRuntimeBanner
import com.pegasus.design.PegasusRuntimeTone
import com.pegasusx.retailer.ui.components.RetailerSectionHeader
import com.pegasusx.retailer.ui.screens.dashboard.components.DashboardOverviewCard
import com.pegasusx.retailer.ui.screens.dashboard.components.PredictionCard
import com.pegasusx.retailer.ui.screens.dashboard.components.QuickReorderRow
import com.pegasusx.retailer.ui.screens.dashboard.components.ServiceGrid
import com.pegasusx.retailer.ui.components.modifiers.bounceCash
import com.pegasusx.retailer.ui.theme.PegasusSpacing
import com.pegasusx.retailer.ui.theme.HexagonShape
import com.pegasusx.retailer.ui.theme.PillShape
import com.pegasusx.retailer.ui.theme.StatusGreen
import com.pegasusx.retailer.ui.theme.StatusOrange
import com.pegasusx.retailer.ui.theme.StatusRed
import com.pegasusx.retailer.ui.theme.SquircleShape
import com.pegasusx.retailer.R

private val timeRanges = listOf("Day", "Week", "Month")

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DashboardScreen(
    viewModel: DashboardViewModel = hiltViewModel(),
    onOpenCatalog: () -> Unit = {},
    onOpenOrders: () -> Unit = {},
    onOpenDeliveries: () -> Unit = {},
    onOpenInsights: () -> Unit = {},
    onOpenSuppliers: () -> Unit = {},
    onOpenProcurement: () -> Unit = {},
    onOpenProfile: () -> Unit = {},
    onOpenControlTower: () -> Unit = {},
    onQuickReorder: (Product) -> Unit = {},
) {
    val uiState by viewModel.uiState.collectAsState()
    var selectedRange by rememberSaveable { mutableIntStateOf(0) }

    Scaffold(
        containerColor = MaterialTheme.colorScheme.background,
    ) { innerPadding ->
        PullToRefreshBox(
            isRefreshing = uiState.isLoading,
            onRefresh = viewModel::refresh,
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding),
        ) {
            LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        
                contentPadding = PaddingValues(horizontal = 16.dp, vertical = 16.dp),
                verticalArrangement = Arrangement.spacedBy(24.dp),
                modifier = Modifier.fillMaxSize(),
        horizontalArrangement = Arrangement.spacedBy(24.dp)
    ) {
                if (uiState.loadIssue != null || uiState.isLoading) {
                    item {
                        val loadIssue = uiState.loadIssue
                        val syncMessage = when {
                            loadIssue != null -> uiState.error ?: uiState.syncMessage.orEmpty()
                            else -> "Syncing dashboard data..."
                        }
                        val tone = when (loadIssue) {
                            DashboardLoadIssue.OFFLINE -> PegasusRuntimeTone.Offline
                            DashboardLoadIssue.RESTRICTED, DashboardLoadIssue.ERROR -> PegasusRuntimeTone.Warning
                            null -> PegasusRuntimeTone.Refreshing
                        }
                        PegasusRuntimeBanner(
                            tone = tone,
                            message = syncMessage,
                            onRetry = loadIssue?.let { { viewModel.refresh() } },
                        )
                    }
                }

                item {
                    DashboardOverviewCard(
                        activeOrderCount = uiState.activeOrders.size,
                        predictionCount = uiState.predictions.size,
                        recentProductCount = uiState.recentProducts.size,
                    )
                }

                item {
                    ServiceGrid(
                        activeOrderCount = uiState.activeOrders.size,
                        predictionCount = uiState.predictions.size,
                        onOpenCatalog = onOpenCatalog,
                        onOpenOrders = onOpenOrders,
                        onOpenDeliveries = onOpenDeliveries,
                        onOpenInsights = onOpenInsights,
                        onOpenSuppliers = onOpenSuppliers,
                        onOpenProcurement = onOpenProcurement,
                        onOpenProfile = onOpenProfile,
                        onOpenControlTower = onOpenControlTower,
                    )
                }

                if (uiState.recentProducts.isNotEmpty()) {
                    item {
                        RetailerSectionHeader(title = stringResource(R.string.mobile_retailer_ui_quick_reorder), icon = Icons.Rounded.History)
                        Spacer(modifier = Modifier.height(PegasusSpacing.md))
                        QuickReorderRow(
                            products = uiState.recentProducts,
                            onReorder = onQuickReorder,
                        )
                    }
                }

                if (uiState.predictions.isNotEmpty()) {
                    item {
                        RetailerSectionHeader(
                            title = stringResource(R.string.mobile_retailer_ui_ai_predictions),
                            icon = Icons.Rounded.AutoAwesome,
                            count = uiState.predictions.size,
                        )
                    }

                    item {
                        SingleChoiceSegmentedButtonRow(
                            modifier = Modifier.fillMaxWidth(),
                        ) {
                            timeRanges.forEachIndexed { index, label ->
                                SegmentedButton(
                                    selected = selectedRange == index,
                                    onClick = { selectedRange = index },
                                    shape = SegmentedButtonDefaults.itemShape(
                                        index = index,
                                        count = timeRanges.size,
                                    ),
                                    icon = {
                                        SegmentedButtonDefaults.Icon(active = selectedRange == index) {
                                            Icon(
                                                imageVector = Icons.Rounded.Check,
                                                contentDescription = null,
                                                modifier = Modifier.size(SegmentedButtonDefaults.IconSize),
                                            )
                                        }
                                    },
                                ) {
                                    Text(label)
                                }
                            }
                        }
                    }

                    items(uiState.predictions, key = { it.id }) { forecast ->
                        PredictionCard(
                            forecast = forecast,
                            onPreorder = { viewModel.requestPreorder(forecast) },
                        )
                    }
                }

                item(span = { GridItemSpan(maxLineSpan) }) { Spacer(modifier = Modifier.height(32.dp)) }
            }
        }
    }
}

