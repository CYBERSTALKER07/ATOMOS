package com.thelab.warehouse.ui.screens.returns

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
import com.thelab.warehouse.data.model.ReturnItem
import com.thelab.warehouse.data.remote.WarehouseApi
import com.thelab.warehouse.ui.theme.LabSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ReturnsScreen(
    api: WarehouseApi,
    onBack: () -> Unit,
) {
    var items by remember { mutableStateOf<List<ReturnItem>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        loading = true; error = null
        scope.launch {
            try {
                val resp = api.getReturns()
                if (resp.isSuccessful && resp.body() != null) items = resp.body()!!.returns
                else error = "Failed (${resp.code()})"
            } catch (e: Exception) { error = e.message ?: "Network error" }
            finally { loading = false }
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Returns") },
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
            items.isEmpty() -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                Text("No returns", color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            else -> LazyColumn(
                contentPadding = PaddingValues(LabSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(LabSpacing.md),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            ) {
                items(items, key = { it.returnId }) { r ->
                    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                        Column(modifier = Modifier.padding(LabSpacing.lg)) {
                            Row(verticalAlignment = Alignment.CenterVertically) {
                                Text(r.productName, style = MaterialTheme.typography.titleSmall, modifier = Modifier.weight(1f))
                                AssistChip(onClick = {}, label = { Text(r.reason, style = MaterialTheme.typography.labelSmall) })
                            }
                            Spacer(Modifier.height(LabSpacing.xs))
                            Text(
                                "Qty: ${r.quantity} · Order: ${r.orderId.take(8)} · ${r.createdAt}",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                    }
                }
            }
        }
    }
}
