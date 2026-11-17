package com.pegasusx.supplier.ui.screens.exceptions

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.data.remote.SupplierRealtimeSignals
import com.pegasusx.supplier.ui.realtime.SupplierReconnectRecoveryEffect
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.util.SUPPLIER_RECONNECT_RECOVERY_HINT
import com.pegasusx.supplier.util.SupplierIdempotencyKeys
import kotlinx.coroutines.launch
import com.pegasusx.supplier.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun EarlyCompleteScreen(
    ops: SupplierOperationsRepository,
    realtimeSignals: SupplierRealtimeSignals,
    onBack: () -> Unit,
) {
    var driverId by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var success by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun approve() {
        val trimmed = driverId.trim()
        if (trimmed.isEmpty()) {
            error = "Driver ID is required."
            return
        }
        scope.launch {
            busy = true
            error = null
            success = null
            try {
                val key = SupplierIdempotencyKeys.approveEarlyComplete(trimmed)
                val resp = ops.approveEarlyComplete(trimmed, key)
                if (resp.isSuccessful) {
                    success = "Early route complete approved for driver ${trimmed.take(12)}…"
                    driverId = ""
                } else {
                    error = "Failed (${resp.code()})"
                }
            } catch (e: Exception) {
                error = e.message
            } finally {
                busy = false
            }
        }
    }

    SupplierReconnectRecoveryEffect(
        realtimeSignals = realtimeSignals,
        isBusy = { busy },
    ) { hadInFlight ->
        if (hadInFlight) {
            busy = false
            error = SUPPLIER_RECONNECT_RECOVERY_HINT
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Early route complete") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
            )
        },
    ) { padding ->
        Column(
            modifier = Modifier.padding(padding).padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            Text(
                "Approve a driver request to finish the route before all stops are completed.",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.outline,
            )
            OutlinedTextField(
                value = driverId,
                onValueChange = { driverId = it },
                label = { Text("Driver ID") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
            )
            Button(
                onClick = { approve() },
                enabled = !busy && driverId.trim().isNotEmpty(),
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(if (busy) "Approving…" else "Approve early complete")
            }
            error?.let { Text(it, color = MaterialTheme.colorScheme.error) }
            success?.let { Text(it, color = MaterialTheme.colorScheme.primary) }
        }
    }
}
