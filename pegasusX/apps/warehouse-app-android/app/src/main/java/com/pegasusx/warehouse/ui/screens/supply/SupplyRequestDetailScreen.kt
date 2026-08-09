package com.pegasusx.warehouse.ui.screens.supply

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import com.pegasusx.warehouse.data.model.WarehouseSupplyRequest
import com.pegasusx.warehouse.data.model.WarehouseSupplyRequestTransitionRequest
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.data.remote.WarehouseOperationsRepository
import com.pegasusx.warehouse.util.WarehouseIdempotencyKeys
import com.pegasusx.warehouse.data.remote.WarehouseRealtimeSignals
import com.pegasusx.warehouse.ui.realtime.WAREHOUSE_RECONNECT_RECOVERY_HINT
import com.pegasusx.warehouse.ui.realtime.WarehouseReconnectRecoveryEffect
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import com.pegasusx.warehouse.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SupplyRequestDetailScreen(
    api: WarehouseApi,
    opsRepository: WarehouseOperationsRepository,
    realtimeSignals: WarehouseRealtimeSignals,
    requestId: String,
    onBack: (() -> Unit)? = null,
) {
    var request by remember { mutableStateOf<WarehouseSupplyRequest?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    var statusMessage by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = opsRepository.getSupplyRequest(requestId)
                if (resp.isSuccessful) {
                    request = resp.body()
                } else {
                    error = "Failed (${resp.code()})"
                }
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    fun cancelRequest() {
        busy = true
        scope.launch {
            try {
                val key = WarehouseIdempotencyKeys.supplyRequestTransition(requestId, "CANCEL")
                val resp = api.transitionSupplyRequest(
                    requestId,
                    key,
                    WarehouseSupplyRequestTransitionRequest(action = "CANCEL"),
                )
                if (resp.isSuccessful) {
                    statusMessage = "Request cancelled"
                    load()
                } else {
                    statusMessage = "Cancel failed (${resp.code()})"
                }
            } catch (e: Exception) {
                statusMessage = e.message
            } finally {
                busy = false
            }
        }
    }

    LaunchedEffect(requestId) { load() }

    WarehouseReconnectRecoveryEffect(
        realtimeSignals = realtimeSignals,
        isBusy = { busy },
    ) { hadInFlight ->
        if (hadInFlight) {
            busy = false
            statusMessage = WAREHOUSE_RECONNECT_RECOVERY_HINT
        }
        load()
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Supply request") },
                navigationIcon = { if (onBack != null) { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back)) } } },
                actions = {
                    TextButton(onClick = { load() }) { Text("Refresh") }
                },
            )
        },
    ) { padding ->
        when {
            loading -> Box(
                Modifier.fillMaxSize().padding(padding),
                contentAlignment = Alignment.Center,
            ) { CircularProgressIndicator() }

            error != null -> Box(
                Modifier.fillMaxSize().padding(padding),
                contentAlignment = Alignment.Center,
            ) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(error!!, color = MaterialTheme.colorScheme.error)
                    Spacer(Modifier.height(PegasusSpacing.md))
                    Button(onClick = { load() }) { Text("Retry") }
                }
            }

            request == null -> Box(
                Modifier.fillMaxSize().padding(padding),
                contentAlignment = Alignment.Center,
            ) { Text("Request not found") }

            else -> {
                val row = request!!
                Column(
                    modifier = Modifier
                        .padding(padding)
                        .padding(PegasusSpacing.lg)
                        .fillMaxSize()
                        .verticalScroll(rememberScrollState()),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                ) {
                    statusMessage?.let { Text(it, color = MaterialTheme.colorScheme.primary) }
                    DetailRow("Request ID", row.requestId)
                    DetailRow("State", row.state)
                    DetailRow("Priority", row.priority)
                    DetailRow("Factory", row.factoryId)
                    DetailRow("Volume (VU)", row.totalVolumeVu.toString())
                    DetailRow("Transfer order", row.transferOrderId ?: "—")
                    DetailRow("Created", row.createdAt)
                    if (row.notes.isNotBlank()) {
                        DetailRow("Notes", row.notes)
                    }
                    if (row.state.equals("OPEN", ignoreCase = true)) {
                        Button(
                            onClick = { cancelRequest() },
                            enabled = !busy,
                            colors = ButtonDefaults.buttonColors(
                                containerColor = MaterialTheme.colorScheme.error,
                            ),
                        ) {
                            Text("Cancel request")
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun DetailRow(label: String, value: String) {
    Column {
        Text(label, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
        Text(value, style = MaterialTheme.typography.bodyLarge)
    }
}
