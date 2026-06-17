package com.pegasusx.warehouse.ui.screens.preorders

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.remote.WarehouseApi
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PreordersScreen(api: WarehouseApi, onBack: (() -> Unit)? = null) {
    var rows by remember { mutableStateOf<List<com.pegasusx.warehouse.data.model.WarehousePreorderRow>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(Unit) {
        scope.launch {
            loading = true
            try {
                val resp = api.getPreorders()
                rows = if (resp.isSuccessful) resp.body()?.items.orEmpty() else emptyList()
                if (!resp.isSuccessful) error = "Failed (${resp.code()})"
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Pre-orders") },
                navigationIcon = {
                    if (onBack != null) {
                        IconButton(onClick = onBack) {
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                        }
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> Box(Modifier.fillMaxSize().padding(padding), contentAlignment = androidx.compose.ui.Alignment.Center) {
                CircularProgressIndicator()
            }
            error != null -> Text(error!!, modifier = Modifier.padding(padding).padding(16.dp))
            rows.isEmpty() -> Text("No scheduled pre-orders", modifier = Modifier.padding(padding).padding(16.dp))
            else -> LazyColumn(Modifier.padding(padding).padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                items(rows, key = { it.orderId }) { row ->
                    ElevatedCard(Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(12.dp)) {
                            Text(row.orderId, style = MaterialTheme.typography.titleSmall)
                            Text("Status: ${row.status}")
                            row.requestedDeliveryDate?.let { Text("Delivery: $it") }
                        }
                    }
                }
            }
        }
    }
}
