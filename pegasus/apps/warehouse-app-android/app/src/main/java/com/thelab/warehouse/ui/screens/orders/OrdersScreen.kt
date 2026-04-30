package com.pegasus.warehouse.ui.screens.orders

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.FilterList
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import com.pegasus.warehouse.data.model.Order
import com.pegasus.warehouse.data.remote.WarehouseApi
import com.pegasus.warehouse.ui.theme.LabSpacing
import kotlinx.coroutines.launch
import java.text.NumberFormat
import java.util.Locale

private val STATES = listOf("ALL", "PENDING", "LOADED", "IN_TRANSIT", "ARRIVED", "COMPLETED", "CANCELLED")

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun OrdersScreen(
    api: WarehouseApi,
    onOrderClick: (String) -> Unit,
    onBack: () -> Unit,
) {
    var orders by remember { mutableStateOf<List<Order>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var selectedState by remember { mutableStateOf("ALL") }
    var filterExpanded by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    val fmt = remember { NumberFormat.getInstance(Locale("uz", "UZ")) }

    fun load() {
        loading = true
        error = null
        scope.launch {
            try {
                val state = if (selectedState == "ALL") null else selectedState
                val resp = api.getOrders(state = state)
                if (resp.isSuccessful && resp.body() != null) {
                    orders = resp.body()!!.orders
                } else {
                    error = "Failed (${resp.code()})"
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(selectedState) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Orders") },
                navigationIcon = { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") } },
                actions = {
                    Box {
                        IconButton(onClick = { filterExpanded = true }) {
                            Icon(Icons.Default.FilterList, "Filter")
                        }
                        DropdownMenu(expanded = filterExpanded, onDismissRequest = { filterExpanded = false }) {
                            STATES.forEach { s ->
                                DropdownMenuItem(
                                    text = { Text(s) },
                                    onClick = { selectedState = s; filterExpanded = false },
                                    leadingIcon = {
                                        if (s == selectedState) Icon(Icons.Default.FilterList, null, tint = MaterialTheme.colorScheme.primary)
                                    },
                                )
                            }
                        }
                    }
                    IconButton(onClick = { load() }) { Icon(Icons.Default.Refresh, "Refresh") }
                },
            )
        },
    ) { innerPadding ->
        when {
            loading -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                CircularProgressIndicator()
            }
            error != null -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(error!!, color = MaterialTheme.colorScheme.error)
                    Spacer(Modifier.height(LabSpacing.lg))
                    Button(onClick = { load() }) { Text("Retry") }
                }
            }
            orders.isEmpty() -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                Text("No orders", style = MaterialTheme.typography.bodyLarge, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            else -> LazyColumn(
                contentPadding = PaddingValues(LabSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(LabSpacing.md),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            ) {
                items(orders, key = { it.orderId }) { order ->
                    ElevatedCard(
                        modifier = Modifier.fillMaxWidth().clickable { onOrderClick(order.orderId) },
                    ) {
                        Row(
                            modifier = Modifier.padding(LabSpacing.lg),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Column(modifier = Modifier.weight(1f)) {
                                Text(
                                    text = order.retailerName.ifBlank { order.orderId.take(8) },
                                    style = MaterialTheme.typography.titleSmall,
                                )
                                Spacer(Modifier.height(LabSpacing.xs))
                                Text(
                                    text = "${fmt.format(order.totalUzs)} UZS",
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                            }
                            AssistChip(
                                onClick = {},
                                label = { Text(order.state, style = MaterialTheme.typography.labelSmall) },
                            )
                        }
                    }
                }
            }
        }
    }
}
