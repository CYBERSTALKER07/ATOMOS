package com.pegasusx.retailer.ui.screens.settings

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Card
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.ViewModel
import com.pegasusx.retailer.data.api.PegasusApi
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.launch
import javax.inject.Inject
import com.pegasusx.retailer.R
import com.pegasusx.retailer.data.json.*

@HiltViewModel
class ReportsViewModel @Inject constructor(val api: PegasusApi) : ViewModel()

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ReportsScreen(
    onNavigateBack: () -> Unit,
    viewModel: ReportsViewModel = hiltViewModel(),
) {
    val scope = rememberCoroutineScope()
    var salesMinor by remember { mutableStateOf(0L) }
    var saleCount by remember { mutableStateOf(0) }
    var onHand by remember { mutableStateOf(0) }
    var lowStock by remember { mutableStateOf(0) }
    var topLine by remember { mutableStateOf("—") }
    var banner by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(Unit) {
        scope.launch {
            try {
                val s = viewModel.api.getReportsSummary().asJsonObject
                salesMinor = s.get("sales_minor")?.asLong ?: 0L
                saleCount = s.get("sale_count")?.asInt ?: 0
                onHand = s.get("on_hand_sku_count")?.asInt ?: 0
                lowStock = s.get("low_stock_count")?.asInt ?: 0
                val top = s.getAsJsonArray("top_skus")
                if (top != null && top.size > 0) {
                    val first = top[0].asJsonObject
                    topLine = "${first.get("sku")?.asString} · ${first.get("sales_minor")?.asLong?.div(100.0)}"
                }
                banner = "REPORTS_PRO enabled on first view"
            } catch (e: Exception) {
                banner = e.message
            }
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Reports Pro") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
            )
        },
    ) { padding ->
        LazyColumn(
            Modifier.fillMaxSize().padding(padding).padding(horizontal = 16.dp),
            contentPadding = PaddingValues(vertical = 12.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            banner?.let { item { Text(it, color = MaterialTheme.colorScheme.primary) } }
            item {
                Card {
                    Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                        Text("Last 7 days", style = MaterialTheme.typography.titleMedium)
                        Text(stringResource(R.string.mobile_retailer_ui_sales_n_0, salesMinor / 100.0))
                        Text(stringResource(R.string.mobile_retailer_ui_sale_count_salecount_2, saleCount))
                        Text(stringResource(R.string.mobile_retailer_ui_on_hand_skus_onhand_2, onHand))
                        Text(stringResource(R.string.mobile_retailer_ui_low_stock_bins_lowstock_2, lowStock))
                        Text(stringResource(R.string.mobile_retailer_ui_top_sku_topline_2, topLine))
                    }
                }
            }
        }
    }
}
