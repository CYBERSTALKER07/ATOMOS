package com.pegasusx.retailer.ui.screens.settings

import android.content.Intent
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
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
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.core.content.FileProvider
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.ViewModel
import com.pegasusx.retailer.R
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.json.asInt
import com.pegasusx.retailer.data.json.asJsonObject
import com.pegasusx.retailer.data.json.asLong
import com.pegasusx.retailer.data.json.asString
import com.pegasusx.retailer.data.json.getAsJsonArray
import dagger.hilt.android.lifecycle.HiltViewModel
import java.io.File
import javax.inject.Inject
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

@HiltViewModel
class ReportsViewModel @Inject constructor(val api: PegasusApi) : ViewModel()

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ReportsScreen(
    onNavigateBack: () -> Unit,
    viewModel: ReportsViewModel = hiltViewModel(),
) {
    val scope = rememberCoroutineScope()
    val context = LocalContext.current
    var salesMinor by remember { mutableStateOf(0L) }
    var saleCount by remember { mutableStateOf(0) }
    var onHand by remember { mutableStateOf(0) }
    var lowStock by remember { mutableStateOf(0) }
    var topLine by remember { mutableStateOf("—") }
    var banner by remember { mutableStateOf<String?>(null) }
    var loadError by remember { mutableStateOf<String?>(null) }
    var summaryReady by remember { mutableStateOf(false) }
    var exporting by remember { mutableStateOf(false) }

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
                summaryReady = true
                loadError = null
                banner = "REPORTS_PRO enabled on first view"
            } catch (_: Exception) {
                loadError = "reports_failed"
            }
        }
    }

    fun shareSalesCsv() {
        if (exporting) return
        exporting = true
        scope.launch {
            try {
                val body = viewModel.api.exportReportsCsv(report = "sales")
                val file = withContext(Dispatchers.IO) {
                    val out = File(context.cacheDir, "sales.csv")
                    body.byteStream().use { input ->
                        out.outputStream().use { output -> input.copyTo(output) }
                    }
                    out
                }
                val uri = FileProvider.getUriForFile(
                    context,
                    "${context.packageName}.fileprovider",
                    file,
                )
                val share = Intent(Intent.ACTION_SEND).apply {
                    type = "text/csv"
                    putExtra(Intent.EXTRA_STREAM, uri)
                    putExtra(Intent.EXTRA_SUBJECT, "sales.csv")
                    addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
                }
                context.startActivity(Intent.createChooser(share, "Export sales CSV"))
                banner = "Sales CSV ready to share"
            } catch (e: Exception) {
                banner = e.message ?: "Export failed"
            } finally {
                exporting = false
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
            loadError?.let { item { Text(it, color = MaterialTheme.colorScheme.error) } }
            banner?.let { if (loadError == null) item { Text(it, color = MaterialTheme.colorScheme.primary) } }
            if (summaryReady && loadError == null) {
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
            item {
                Button(
                    onClick = { shareSalesCsv() },
                    enabled = !exporting,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text(if (exporting) "Exporting…" else "Export sales CSV")
                }
            }
        }
    }
}
