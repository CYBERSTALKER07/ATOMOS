package com.pegasusx.warehouse.ui.screens.transfers

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import com.pegasusx.warehouse.data.model.EmergencyTransferRequest
import com.pegasusx.warehouse.data.model.ForceReceiveRequest
import com.pegasusx.warehouse.data.remote.WarehouseOperationsRepository
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TransferActionsScreen(
    opsRepository: WarehouseOperationsRepository,
    onBack: () -> Unit,
) {
    var volumeInput by remember { mutableStateOf("20") }
    var transferIdInput by remember { mutableStateOf("") }
    var notesInput by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    val snackbarHostState = remember { SnackbarHostState() }
    val scope = rememberCoroutineScope()

    fun runAction(label: String, block: suspend () -> retrofit2.Response<*>) {
        busy = true
        scope.launch {
            try {
                val resp = block()
                if (resp.isSuccessful) {
                    snackbarHostState.showSnackbar("$label succeeded")
                } else {
                    snackbarHostState.showSnackbar("$label failed (${resp.code()})")
                }
            } catch (e: Exception) {
                snackbarHostState.showSnackbar(e.message ?: "Network error")
            } finally {
                busy = false
            }
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Transfer actions") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .padding(PegasusSpacing.lg)
                .verticalScroll(rememberScrollState()),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            Text(
                "Factory inbound transfer controls for warehouse operators.",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            OutlinedTextField(
                value = volumeInput,
                onValueChange = { volumeInput = it },
                label = { Text("Volume (VU)") },
                modifier = Modifier.fillMaxWidth(),
                enabled = !busy,
                singleLine = true,
            )
            OutlinedTextField(
                value = notesInput,
                onValueChange = { notesInput = it },
                label = { Text("Notes (optional)") },
                modifier = Modifier.fillMaxWidth(),
                enabled = !busy,
            )
            Button(
                onClick = {
                    val volume = volumeInput.toDoubleOrNull() ?: 20.0
                    runAction("Emergency transfer") {
                        opsRepository.emergencyTransfer(
                            EmergencyTransferRequest(totalVolumeVu = volume, notes = notesInput.ifBlank { null }),
                        )
                    }
                },
                enabled = !busy,
                modifier = Modifier.fillMaxWidth(),
            ) { Text("Create emergency transfer") }
            Button(
                onClick = {
                    val volume = volumeInput.toDoubleOrNull() ?: 20.0
                    runAction("Force receive") {
                        opsRepository.forceReceive(
                            ForceReceiveRequest(totalVolumeVu = volume, notes = notesInput.ifBlank { null }),
                        )
                    }
                },
                enabled = !busy,
                modifier = Modifier.fillMaxWidth(),
            ) { Text("Force receive payload") }
            OutlinedTextField(
                value = transferIdInput,
                onValueChange = { transferIdInput = it },
                label = { Text("Transfer ID to receive") },
                modifier = Modifier.fillMaxWidth(),
                enabled = !busy,
                singleLine = true,
            )
            Button(
                onClick = {
                    val id = transferIdInput.trim()
                    if (id.isEmpty()) {
                        scope.launch { snackbarHostState.showSnackbar("Transfer ID required") }
                        return@Button
                    }
                    runAction("Receive transfer") { opsRepository.receiveTransfer(id) }
                },
                enabled = !busy,
                modifier = Modifier.fillMaxWidth(),
            ) { Text("Receive transfer") }
        }
    }
}
