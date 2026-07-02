package com.pegasusx.warehouse.ui.screens.tomorrowboard

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.WarehouseOpsBoardOrder
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.data.remote.WarehouseRealtimeSignals
import kotlinx.coroutines.launch
import java.time.LocalDate

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TomorrowBoardScreen(
    api: WarehouseApi,
    realtimeSignals: WarehouseRealtimeSignals,
    onBack: (() -> Unit)? = null,
) {
    var date by remember { mutableStateOf(LocalDate.now().plusDays(1).toString()) }
    var preorders by remember { mutableStateOf<List<WarehouseOpsBoardOrder>>(emptyList()) }
    var deliverBefore by remember { mutableStateOf<List<WarehouseOpsBoardOrder>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load(silent: Boolean = false) {
        scope.launch {
            if (!silent) loading = true
            error = null
            try {
                val resp = api.getOpsBoard(date)
                if (resp.isSuccessful) {
                    val body = resp.body()
                    preorders = body?.preorders.orEmpty()
                    deliverBefore = body?.deliverBefore.orEmpty()
                } else if (!silent) {
                    error = "Failed (${resp.code()})"
                }
            } catch (e: Exception) {
                if (!silent) error = e.message
            } finally {
                if (!silent) loading = false
            }
        }
    }

    LaunchedEffect(date) { load() }
    LaunchedEffect(Unit) {
        realtimeSignals.refreshTick.collect { load(silent = true) }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Tomorrow board") },
                navigationIcon = {
                    if (onBack != null) {
                        IconButton(onClick = onBack) {
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                        }
                    }
                },
            )
        },
    ) { innerPadding ->
        if (loading) {
            Box(
                Modifier.fillMaxSize().padding(innerPadding),
                contentAlignment = androidx.compose.ui.Alignment.Center,
            ) {
                CircularProgressIndicator()
            }
            return@Scaffold
        }

        val rows = preorders.map { it to "Pre-order" } + deliverBefore.map { it to "Deliver by" }

        LazyColumn(
            modifier = Modifier.fillMaxSize().padding(innerPadding).padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            item {
                OutlinedTextField(
                    value = date,
                    onValueChange = { date = it },
                    label = { Text("Date (YYYY-MM-DD)") },
                    modifier = Modifier.fillMaxWidth(),
                    singleLine = true,
                )
            }
            if (error != null) {
                item { Text(error!!, color = MaterialTheme.colorScheme.error) }
            }
            if (rows.isEmpty()) {
                item { Text("No orders scheduled for this date.") }
            } else {
                items(rows, key = { (order, lane) -> "$lane-${order.orderId}" }) { (order, lane) ->
                    Card(modifier = Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                            Text(lane, style = MaterialTheme.typography.titleSmall)
                            Text(order.orderId, style = MaterialTheme.typography.bodyMedium)
                            Text(
                                order.deliveryExpectation?.targetLabel ?: order.status,
                                style = MaterialTheme.typography.bodySmall,
                            )
                        }
                    }
                }
            }
        }
    }
}
