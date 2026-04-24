package com.thelab.retailer.ui.screens.dashboard

import androidx.compose.foundation.background
import androidx.compose.foundation.cashable
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
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.rounded.AddShoppingCart
import androidx.compose.material.icons.rounded.Check
import androidx.compose.material.icons.rounded.History
import androidx.compose.material.icons.rounded.Inventory2
import androidx.compose.material.icons.rounded.Search
import androidx.compose.material.icons.rounded.ShoppingBag
import androidx.compose.material.icons.rounded.AutoAwesome
import androidx.compose.material.icons.rounded.TrendingUp
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.MultiChoiceSegmentedButtonRow
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SegmentedButton
import androidx.compose.material3.SegmentedButtonDefaults
import androidx.compose.material3.SingleChoiceSegmentedButtonRow
import androidx.compose.material3.Snackbar
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import com.thelab.retailer.data.model.DemandForecast
import com.thelab.retailer.data.model.Product
import com.thelab.retailer.ui.theme.HexagonShape
import com.thelab.retailer.ui.theme.ScallopShape
import com.thelab.retailer.ui.theme.SoftSquircleShape
import com.thelab.retailer.ui.theme.SquircleShape
import com.thelab.retailer.ui.theme.StatusGreen
import com.thelab.retailer.ui.theme.StatusOrange
import com.thelab.retailer.ui.theme.StatusRed

private val timeRanges = listOf("Day", "Week", "Month")

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DashboardScreen(
    viewModel: DashboardViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()
    var selectedRange by rememberSaveable { mutableIntStateOf(0) }
    val snackbarHostState = remember { SnackbarHostState() }

    LaunchedEffect(uiState.error) {
        uiState.error?.let {
            snackbarHostState.showSnackbar(it)
            viewModel.clearError()
        }
    }

    Scaffold(
        snackbarHost = { SnackbarHost(snackbarHostState) },
        containerColor = MaterialTheme.colorScheme.background,
    ) { innerPadding ->
    PullToRefreshBox(
        isRefreshing = uiState.isLoading,
        onRefresh = viewModel::refresh,
        modifier = Modifier.fillMaxSize().padding(innerPadding),
    ) {
        LazyColumn(
            contentPadding = PaddingValues(horizontal = 16.dp, vertical = 12.dp),
            verticalArrangement = Arrangement.spacedBy(24.dp),
            modifier = Modifier.fillMaxSize(),
        ) {
            // ── Bento Service Grid ──
            item { ServiceGrid(activeOrderCount = uiState.activeOrders.size, predictionCount = uiState.predictions.size) }

            // ── Quick Reorder ──
            if (uiState.recentProducts.isNotEmpty()) {
                item {
                    SectionHeader(title = "Quick Reorder", icon = Icons.Rounded.History)
                    Spacer(modifier = Modifier.height(12.dp))
                    QuickReorderRow(products = uiState.recentProducts)
                }
            }

            // ── AI Predictions ──
            if (uiState.predictions.isNotEmpty()) {
                item {
                    SectionHeader(title = "AI Predictions", icon = Icons.Rounded.AutoAwesome, count = uiState.predictions.size)
                }

                // ── M3 Segmented Button (Day / Week / Month) ──
                item {
                    SingleChoiceSegmentedButtonRow(
                        modifier = Modifier.fillMaxWidth().padding(vertical = 8.dp),
                    ) {
                        timeRanges.forEachIndexed { index, label ->
                            SegmentedButton(
                                selected = selectedRange == index,
                                onCash = { selectedRange = index },
                                shape = SegmentedButtonDefaults.itemShape(index = index, count = timeRanges.size),
                                icon = {
                                    SegmentedButtonDefaults.Icon(active = selectedRange == index) {
                                        Icon(
                                            Icons.Rounded.Check,
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
                itemsIndexed(uiState.predictions, key = { _, f -> f.id }) { _, forecast ->
                    PredictionCard(forecast = forecast, onPreorder = { viewModel.requestPreorder(forecast) })
                }
            }

            item { Spacer(modifier = Modifier.height(32.dp)) }
        }
    }
    } // Scaffold
}

// ── Bento Service Grid (Yandex Go style) ──

@Composable
private fun ServiceGrid(activeOrderCount: Int, predictionCount: Int) {
    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
        // Row 1: two big tiles
        Row(horizontalArrangement = Arrangement.spacedBy(12.dp), modifier = Modifier.fillMaxWidth()) {
            ServiceTile("Catalog", Icons.Rounded.ShoppingBag, "Browse products", Modifier.weight(1f).height(130.dp))
            ServiceTile("AI Insights", Icons.Rounded.AutoAwesome, "$predictionCount predictions", Modifier.weight(1f).height(130.dp))
        }
        // Row 2: one wide + two small stacked
        Row(horizontalArrangement = Arrangement.spacedBy(12.dp), modifier = Modifier.fillMaxWidth()) {
            ServiceTile("Orders", Icons.Rounded.Inventory2, "$activeOrderCount active", Modifier.weight(1f).height(120.dp))
            Column(verticalArrangement = Arrangement.spacedBy(12.dp), modifier = Modifier.weight(1f)) {
                ServiceTile("Inbox", Icons.Rounded.Inventory2, null, Modifier.fillMaxWidth().height(54.dp))
                ServiceTile("History", Icons.Rounded.History, null, Modifier.fillMaxWidth().height(54.dp))
            }
        }
        // Row 3: three equal small tiles
        Row(horizontalArrangement = Arrangement.spacedBy(12.dp), modifier = Modifier.fillMaxWidth()) {
            ServiceTileSmall("Procurement", Icons.Rounded.TrendingUp, Modifier.weight(1f))
            ServiceTileSmall("Search", Icons.Rounded.Search, Modifier.weight(1f))
            ServiceTileSmall("Profile", Icons.Rounded.History, Modifier.weight(1f))
        }
    }
}

@Composable
private fun ServiceTile(title: String, icon: ImageVector, subtitle: String?, modifier: Modifier = Modifier) {
    Surface(
        modifier = modifier
            .shadow(4.dp, SoftSquircleShape, ambientColor = Color.Black.copy(alpha = 0.06f), spotColor = Color.Black.copy(alpha = 0.06f)),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Column(
            modifier = Modifier.fillMaxSize().padding(12.dp),
            verticalArrangement = Arrangement.Bottom,
        ) {
            Icon(icon, contentDescription = null, modifier = Modifier.size(28.dp), tint = MaterialTheme.colorScheme.onSurface)
            Spacer(modifier = Modifier.height(8.dp))
            Text(title, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.SemiBold)
            if (subtitle != null) {
                Text(subtitle, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f))
            }
        }
    }
}

@Composable
private fun ServiceTileSmall(title: String, icon: ImageVector, modifier: Modifier = Modifier) {
    Surface(
        modifier = modifier.height(80.dp)
            .shadow(4.dp, SquircleShape, ambientColor = Color.Black.copy(alpha = 0.06f), spotColor = Color.Black.copy(alpha = 0.06f)),
        shape = SquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Column(
            modifier = Modifier.fillMaxSize(),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center,
        ) {
            Icon(icon, contentDescription = null, modifier = Modifier.size(22.dp), tint = MaterialTheme.colorScheme.onSurface)
            Spacer(modifier = Modifier.height(6.dp))
            Text(title, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.7f))
        }
    }
}

// ── Quick Reorder Row ──

@Composable
private fun QuickReorderRow(products: List<Product>) {
    LazyRow(
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        items(products.size) { idx ->
            val product = products[idx]
            Column(
                horizontalAlignment = Alignment.CenterHorizontally,
                modifier = Modifier.width(80.dp).cashable { /* add to cart */ },
            ) {
                Box(
                    modifier = Modifier.size(64.dp)
                        .clip(HexagonShape)
                        .background(MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.3f)),
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(Icons.Rounded.Inventory2, contentDescription = null, modifier = Modifier.size(24.dp), tint = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.3f))
                }
                Spacer(modifier = Modifier.height(6.dp))
                Text(product.name, style = MaterialTheme.typography.labelSmall, maxLines = 1, overflow = TextOverflow.Ellipsis)
                Text(product.displayPrice, style = MaterialTheme.typography.labelSmall, fontWeight = FontWeight.Bold)
            }
        }
    }
}

// ── AI Prediction Card ──

@Composable
private fun PredictionCard(forecast: DemandForecast, onPreorder: () -> Unit) {
    Surface(
        modifier = Modifier.fillMaxWidth()
            .shadow(4.dp, SoftSquircleShape, ambientColor = Color.Black.copy(alpha = 0.06f), spotColor = Color.Black.copy(alpha = 0.06f)),
        shape = SoftSquircleShape,
        color = MaterialTheme.colorScheme.surface,
    ) {
        Row(
            modifier = Modifier.padding(16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            // Confidence ring
            ConfidenceRing(confidence = forecast.confidence, size = 44)

            Spacer(modifier = Modifier.width(12.dp))

            Column(modifier = Modifier.weight(1f)) {
                Text(forecast.productName, style = MaterialTheme.typography.titleSmall, fontWeight = FontWeight.SemiBold, maxLines = 1, overflow = TextOverflow.Ellipsis)
                Spacer(modifier = Modifier.height(2.dp))
                Text(forecast.reasoning, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f), maxLines = 2, overflow = TextOverflow.Ellipsis)
            }

            Spacer(modifier = Modifier.width(12.dp))

            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Text("${forecast.predictedQuantity}", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold)
                Text("units", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.5f))
                Spacer(modifier = Modifier.height(6.dp))
                Box(
                    modifier = Modifier.size(32.dp)
                        .clip(CircleShape)
                        .background(MaterialTheme.colorScheme.primary)
                        .cashable { onPreorder() },
                    contentAlignment = Alignment.Center,
                ) {
                    Icon(Icons.Rounded.AddShoppingCart, contentDescription = "Pre-order", modifier = Modifier.size(16.dp), tint = MaterialTheme.colorScheme.onPrimary)
                }
            }
        }
    }
}

@Composable
private fun ConfidenceRing(confidence: Double, size: Int) {
    val color = when {
        confidence >= 0.8 -> StatusGreen
        confidence >= 0.6 -> StatusOrange
        else -> StatusRed
    }
    val trackColor = MaterialTheme.colorScheme.outlineVariant.copy(alpha = 0.3f)

    Box(
        modifier = Modifier.size(size.dp)
            .drawBehind {
                val strokeWidth = 3.dp.toPx()
                val arcSize = Size(this.size.width - strokeWidth, this.size.height - strokeWidth)
                val topLeft = Offset(strokeWidth / 2, strokeWidth / 2)
                drawArc(trackColor, 0f, 360f, false, topLeft = topLeft, size = arcSize, style = Stroke(strokeWidth))
                drawArc(color, -90f, (confidence * 360).toFloat(), false, topLeft = topLeft, size = arcSize, style = Stroke(strokeWidth, cap = StrokeCap.Round))
            },
        contentAlignment = Alignment.Center,
    ) {
        Text(
            "${(confidence * 100).toInt()}%",
            style = MaterialTheme.typography.labelSmall.copy(fontSize = 10.sp, fontWeight = FontWeight.Bold),
            color = color,
        )
    }
}

// ── Section Header ──

@Composable
private fun SectionHeader(title: String, icon: ImageVector, count: Int? = null) {
    Row(verticalAlignment = Alignment.CenterVertically) {
        Icon(icon, contentDescription = null, modifier = Modifier.size(14.dp), tint = MaterialTheme.colorScheme.onSurface.copy(alpha = 0.6f))
        Spacer(modifier = Modifier.width(6.dp))
        Text(title, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
        if (count != null) {
            Spacer(modifier = Modifier.width(8.dp))
            Text(
                "$count",
                style = MaterialTheme.typography.labelSmall.copy(fontWeight = FontWeight.ExtraBold),
                color = Color.White,
                modifier = Modifier
                    .size(20.dp)
                    .clip(CircleShape)
                    .background(MaterialTheme.colorScheme.primary)
                    .padding(2.dp),
                textAlign = androidx.compose.ui.text.style.TextAlign.Center,
            )
        }
        Spacer(modifier = Modifier.weight(1f))
    }
}
