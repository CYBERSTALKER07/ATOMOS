package com.pegasusx.retailer.ui.screens.settings

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
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

data class LocalSkuRow(
    val id: String,
    val name: String,
    val barcode: String,
    val priceMinor: Long,
    val active: Boolean,
)

@HiltViewModel
class LocalSkusViewModel @Inject constructor(val api: PegasusApi) : ViewModel()

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LocalSkusScreen(
    onNavigateBack: () -> Unit,
    viewModel: LocalSkusViewModel = hiltViewModel(),
) {
    val scope = rememberCoroutineScope()
    var name by remember { mutableStateOf("") }
    var barcode by remember { mutableStateOf("") }
    var price by remember { mutableStateOf("5000") }
    var banner by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    val rows = remember { mutableStateListOf<LocalSkuRow>() }

    fun refresh() {
        scope.launch {
            try {
                val items = viewModel.api.getLocalSkus().asJsonObject.getAsJsonArray("items")
                rows.clear()
                if (items != null) {
                    for (el in items) {
                        val o = el.asJsonObject
                        rows.add(
                            LocalSkuRow(
                                id = o.get("local_sku_id")?.asString ?: continue,
                                name = o.get("name")?.asString ?: "",
                                barcode = o.get("barcode")?.asString ?: "",
                                priceMinor = o.get("default_price_minor")?.asLong ?: 0L,
                                active = o.get("is_active")?.asBoolean ?: true,
                            ),
                        )
                    }
                }
            } catch (e: Exception) {
                banner = e.message
            }
        }
    }

    LaunchedEffect(Unit) { refresh() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Local SKUs") },
                navigationIcon = {
                    IconButton(onClick = onNavigateBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
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
                    Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text("Non-Pegasus goods for POS (local: prefix)", style = MaterialTheme.typography.titleSmall)
                        OutlinedTextField(value = name, onValueChange = { name = it }, label = { Text("Name") }, modifier = Modifier.fillMaxWidth())
                        OutlinedTextField(value = barcode, onValueChange = { barcode = it }, label = { Text("Barcode") }, modifier = Modifier.fillMaxWidth())
                        OutlinedTextField(value = price, onValueChange = { price = it }, label = { Text("Price (minor)") }, modifier = Modifier.fillMaxWidth())
                        Button(enabled = !busy && name.isNotBlank(), onClick = {
                            scope.launch {
                                busy = true
                                try {
                                    viewModel.api.createLocalSku(
                                        body = mapOf(
                                            "name" to name.trim(),
                                            "barcode" to barcode.trim(),
                                            "default_price_minor" to (price.toLongOrNull() ?: 0L),
                                        ),
                                        idempotencyKey = "local-sku-${System.currentTimeMillis()}",
                                    )
                                    name = ""
                                    barcode = ""
                                    banner = "Created"
                                    refresh()
                                } catch (e: Exception) {
                                    banner = e.message
                                } finally {
                                    busy = false
                                }
                            }
                        }) { Text("Add local SKU") }
                    }
                }
            }
            items(rows, key = { it.id }) { row ->
                Card {
                    Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                        Text(row.name, style = MaterialTheme.typography.titleSmall)
                        Text("${row.id} · ${row.priceMinor} minor · ${if (row.active) "active" else "inactive"}", style = MaterialTheme.typography.bodySmall)
                        if (row.barcode.isNotBlank()) {
                            Text("Barcode ${row.barcode}", style = MaterialTheme.typography.bodySmall)
                        }
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            OutlinedButton(onClick = {
                                scope.launch {
                                    try {
                                        viewModel.api.patchLocalSku(row.id, mapOf("is_active" to !row.active))
                                        refresh()
                                    } catch (e: Exception) {
                                        banner = e.message
                                    }
                                }
                            }) { Text(if (row.active) "Disable" else "Enable") }
                        }
                    }
                }
            }
        }
    }
}
