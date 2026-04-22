package com.thelab.warehouse.ui.screens.dispatch

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import com.thelab.warehouse.data.model.DispatchPreview
import com.thelab.warehouse.data.remote.WarehouseApi
import com.thelab.warehouse.ui.theme.LabSpacing
import kotlinx.coroutines.launch
import java.text.NumberFormat
import java.util.Locale

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DispatchScreen(
    api: WarehouseApi,
    onBack: () -> Unit,
) {
    var preview by remember { mutableStateOf<DispatchPreview?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var tab by remember { mutableIntStateOf(0) }
    val scope = rememberCoroutineScope()
    val fmt = remember { NumberFormat.getInstance(Locale("uz", "UZ")) }

    fun load() {
        loading = true; error = null
        scope.launch {
            try {
                val resp = api.getDispatchPreview()
                if (resp.isSuccessful && resp.body() != null) preview = resp.body()!!
                else error = "Failed (${resp.code()})"
            } catch (e: Exception) { error = e.message ?: "Network error" }
            finally { loading = false }
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Dispatch") },
                navigationIcon = { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") } },
                actions = { IconButton(onClick = { load() }) { Icon(Icons.Default.Refresh, "Refresh") } },
            )
        },
    ) { innerPadding ->
        when {
            loading -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
            error != null -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(error!!, color = MaterialTheme.colorScheme.error)
                    Spacer(Modifier.height(LabSpacing.lg))
                    Button(onClick = { load() }) { Text("Retry") }
                }
            }
            preview != null -> Column(modifier = Modifier.fillMaxSize().padding(innerPadding)) {
                TabRow(selectedTabIndex = tab) {
                    Tab(selected = tab == 0, onClick = { tab = 0 }, text = { Text("Orders (${preview!!.undispatchedOrders.size})") })
                    Tab(selected = tab == 1, onClick = { tab = 1 }, text = { Text("Drivers (${preview!!.availableDrivers.size})") })
                }
                when (tab) {
                    0 -> {
                        if (preview!!.undispatchedOrders.isEmpty()) {
                            Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                                Text("All orders dispatched", color = MaterialTheme.colorScheme.onSurfaceVariant)
                            }
                        } else {
                            LazyColumn(contentPadding = PaddingValues(LabSpacing.lg), verticalArrangement = Arrangement.spacedBy(LabSpacing.md)) {
                                items(preview!!.undispatchedOrders, key = { it.orderId }) { o ->
                                    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                                        Row(modifier = Modifier.padding(LabSpacing.lg), verticalAlignment = Alignment.CenterVertically) {
                                            Column(modifier = Modifier.weight(1f)) {
                                                Text(o.retailerName.ifBlank { o.orderId.take(8) }, style = MaterialTheme.typography.titleSmall)
                                                Text("${fmt.format(o.totalUzs)} UZS · ${o.itemCount} items", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                    1 -> {
                        if (preview!!.availableDrivers.isEmpty()) {
                            Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                                Text("No available drivers", color = MaterialTheme.colorScheme.onSurfaceVariant)
                            }
                        } else {
                            LazyColumn(contentPadding = PaddingValues(LabSpacing.lg), verticalArrangement = Arrangement.spacedBy(LabSpacing.md)) {
                                items(preview!!.availableDrivers, key = { it.driverId }) { d ->
                                    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                                        Row(modifier = Modifier.padding(LabSpacing.lg), verticalAlignment = Alignment.CenterVertically) {
                                            Column(modifier = Modifier.weight(1f)) {
                                                Text(d.name, style = MaterialTheme.typography.titleSmall)
                                                Text(d.vehicleLabel.ifBlank { "No vehicle" }, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
