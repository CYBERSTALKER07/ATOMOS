package com.pegasusx.retailer.ui.screens.hq

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
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
import com.pegasusx.retailer.data.json.asInt
import com.pegasusx.retailer.data.json.asJsonObject
import com.pegasusx.retailer.data.json.asLong
import com.pegasusx.retailer.data.json.asString
import com.pegasusx.retailer.data.json.getAsJsonArray
import dagger.hilt.android.lifecycle.HiltViewModel
import java.time.Instant
import java.time.ZoneOffset
import java.time.format.DateTimeFormatter
import javax.inject.Inject
import kotlinx.coroutines.launch
import kotlinx.serialization.json.jsonObject

data class HqLocRow(
    val locationId: String,
    val netMinor: Long,
    val qtySold: Int,
)

@HiltViewModel
class HqViewModel @Inject constructor(val api: PegasusApi) : ViewModel()

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun HqScreen(
    onNavigateBack: () -> Unit,
    viewModel: HqViewModel = hiltViewModel(),
) {
    val scope = rememberCoroutineScope()
    val day = remember {
        DateTimeFormatter.ISO_LOCAL_DATE.format(Instant.now().atZone(ZoneOffset.UTC))
    }
    var summaryLine by remember { mutableStateOf("—") }
    var locs by remember { mutableStateOf<List<HqLocRow>>(emptyList()) }
    var banner by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(Unit) {
        scope.launch {
            try {
                val s = viewModel.api.getHqSummary(day).asJsonObject
                val net = s["net_minor"]?.asLong ?: 0L
                val locsCount = s["location_count"]?.asInt ?: 0
                val skus = s["sku_count"]?.asInt ?: 0
                summaryLine = "$day · $locsCount locations · $skus SKUs · net ${net / 100.0}"
                val locJson = viewModel.api.getHqSalesByLocation(day).asJsonObject
                val arr = locJson.getAsJsonArray("items")
                locs = buildList {
                    if (arr != null) {
                        for (el in arr) {
                            val o = el.jsonObject
                            add(
                                HqLocRow(
                                    locationId = o["location_id"]?.asString ?: "—",
                                    netMinor = o["net_minor"]?.asLong ?: 0L,
                                    qtySold = o["qty_sold"]?.asInt ?: 0,
                                ),
                            )
                        }
                    }
                }
                banner = null
            } catch (e: Exception) {
                banner = e.message ?: "HQ unavailable"
            }
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("HQ multi-store") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding),
            contentPadding = PaddingValues(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            banner?.let {
                item {
                    Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
                }
            }
            item {
                Text(summaryLine, style = MaterialTheme.typography.bodyMedium)
            }
            item {
                Text("Sales by location", style = MaterialTheme.typography.titleMedium)
            }
            if (locs.isEmpty()) {
                item { Text("No HQ rows for this day.", style = MaterialTheme.typography.bodySmall) }
            } else {
                items(locs, key = { it.locationId }) { row ->
                    Card(modifier = Modifier.fillMaxWidth()) {
                        Column(Modifier = Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                            Text(row.locationId, style = MaterialTheme.typography.labelMedium)
                            Text("Qty ${row.qtySold} · Net ${row.netMinor / 100.0}")
                        }
                    }
                }
            }
        }
    }
}
