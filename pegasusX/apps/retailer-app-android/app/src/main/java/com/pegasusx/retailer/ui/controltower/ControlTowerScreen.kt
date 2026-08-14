package com.pegasusx.retailer.ui.controltower

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.ViewModel
import com.pegasusx.retailer.data.api.PegasusApi
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.launch
import javax.inject.Inject
import com.pegasusx.retailer.R
import com.pegasusx.retailer.data.json.*
import com.pegasusx.retailer.ui.components.PegasusTab

data class PulseTile(val label: String, val value: String, val route: String)

@HiltViewModel
class ControlTowerViewModel @Inject constructor(val api: PegasusApi) : ViewModel()

@Composable
fun ControlTowerScreen(
    viewModel: ControlTowerViewModel = hiltViewModel(),
    onNavigate: (String) -> Unit = {},
) {
    val scope = rememberCoroutineScope()
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var empty by remember { mutableStateOf(true) }
    var generatedAt by remember { mutableStateOf("") }
    var packs by remember { mutableStateOf("") }
    var tiles by remember { mutableStateOf(listOf<PulseTile>()) }

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val p = viewModel.api.getControlTowerPulse().asJsonObject
                empty = p.get("empty")?.asBoolean != false
                generatedAt = p.get("generated_at")?.asString?.take(19) ?: ""
                val caps = p.getAsJsonArray("capabilities")
                packs = if (caps != null) caps.joinToString { it.asString } else "CORE"
                tiles = listOf(
                    PulseTile("Open orders", "${p.get("open_orders")?.asInt ?: 0}", PegasusTab.ORDERS.name),
                    PulseTile("Fulfillment", "${p.get("active_fulfillments")?.asInt ?: 0}", PegasusTab.MAP.name),
                    PulseTile("Dock pending", "${p.get("dock_pending")?.asInt ?: 0}", "DOCK"),
                    PulseTile("POS sessions", "${p.get("pos_open_sessions")?.asInt ?: 0}", "POS"),
                    PulseTile("Open shifts", "${p.get("open_shifts")?.asInt ?: 0}", "SHIFTS"),
                    PulseTile("Assist", "${p.get("open_assist_tickets")?.asInt ?: 0}", "ASSIST"),
                    PulseTile("Low stock", "${p.get("low_stock_sku_bins")?.asInt ?: 0}", "STORE_STOCK"),
                    PulseTile("Variances", "${p.get("shift_variances_7d")?.asInt ?: 0}", "SHIFTS"),
                    PulseTile(
                        "Sales 7d",
                        "${(p.get("sales_minor_7d")?.asLong ?: 0L) / 100.0}",
                        "REPORTS_PRO",
                    ),
                )
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    Column(
        Modifier
            .fillMaxSize()
            .background(Color(0xFF0A0A0A))
            .padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Text("Retailer ops pulse", style = MaterialTheme.typography.headlineMedium, color = Color.White)
        Text(
            "Live counts for your shop — never demo charts.",
            style = MaterialTheme.typography.bodySmall,
            color = Color.Gray,
        )
        if (generatedAt.isNotEmpty()) {
            Text(stringResource(R.string.mobile_retailer_ui_updated_generatedat_packs_2, generatedAt, packs), style = MaterialTheme.typography.labelSmall, color = Color.DarkGray)
        }
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            Button(onClick = { load() }) { Text("Refresh") }
        }
        error?.let { Text(it, color = Color(0xFFF87171)) }
        if (loading && tiles.isEmpty()) {
            CircularProgressIndicator(Modifier.align(Alignment.CenterHorizontally), color = Color(0xFF34D399))
        } else if (empty && error == null) {
            Card(
                colors = CardDefaults.cardColors(containerColor = Color.White.copy(alpha = 0.06f)),
                modifier = Modifier.fillMaxWidth(),
            ) {
                Column(Modifier.padding(20.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text("No live ops signals yet", color = Color.White, style = MaterialTheme.typography.titleMedium)
                    Text(
                        "Place orders, enable stock/POS, open a shift, or create an assist ticket. This stays empty until real activity exists.",
                        color = Color.Gray,
                        style = MaterialTheme.typography.bodyMedium,
                    )
                }
            }
        } else {
            LazyVerticalGrid(
                columns = GridCells.Adaptive(140.dp),
                contentPadding = PaddingValues(vertical = 4.dp),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
                modifier = Modifier.fillMaxSize(),
            ) {
                items(tiles) { tile ->
                    Card(
                        colors = CardDefaults.cardColors(containerColor = Color.White.copy(alpha = 0.06f)),
                        modifier = Modifier.clickable { onNavigate(tile.route) },
                    ) {
                        Column(Modifier.padding(12.dp)) {
                            Text(tile.label, color = Color.Gray, style = MaterialTheme.typography.labelSmall)
                            Text(tile.value, color = Color.White, style = MaterialTheme.typography.titleLarge)
                        }
                    }
                }
            }
        }
    }
}
