package com.pegasusx.warehouse.ui.screens.transfers

import androidx.compose.ui.res.stringResource

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
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import com.pegasus.barcode.DataWedgeBarcodeEffect
import com.pegasus.barcode.EanBarcodeScannerPreview
import com.pegasus.barcode.KeyboardWedgeBarcodeField
import com.pegasusx.warehouse.R
import com.pegasusx.warehouse.data.model.EmergencyTransferRequest
import com.pegasusx.warehouse.data.model.ForceReceiveRequest
import com.pegasusx.warehouse.data.model.PickTask
import com.pegasusx.warehouse.data.model.PickWave
import com.pegasusx.warehouse.data.model.StockLotPutawayRequest
import com.pegasusx.warehouse.data.model.WarehouseBinCreateRequest
import com.pegasusx.warehouse.data.remote.WarehouseOperationsRepository
import com.pegasusx.warehouse.data.remote.WarehouseRealtimeSignals
import com.pegasusx.warehouse.ui.components.WarehouseSectionTitle
import com.pegasusx.warehouse.ui.realtime.WAREHOUSE_RECONNECT_RECOVERY_HINT
import com.pegasusx.warehouse.ui.realtime.WarehouseReconnectRecoveryEffect
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TransferActionsScreen(
    opsRepository: WarehouseOperationsRepository,
    realtimeSignals: WarehouseRealtimeSignals,
    onBack: (() -> Unit)? = null,
) {
    var volumeInput by remember { mutableStateOf("20") }
    var transferIdInput by remember { mutableStateOf("") }
    var notesInput by remember { mutableStateOf("") }
    var putawayProduct by remember { mutableStateOf("") }
    var putawayLocation by remember { mutableStateOf("") }
    var putawayQty by remember { mutableStateOf("1") }
    var putawayExpiry by remember { mutableStateOf("") }
    var pickManifestId by remember { mutableStateOf("") }
    var pickScan by remember { mutableStateOf("") }
    var pickScannedQty by remember { mutableLongStateOf(0L) }
    var activeWave by remember { mutableStateOf<PickWave?>(null) }
    var busy by remember { mutableStateOf(false) }
    val snackbarHostState = remember { SnackbarHostState() }
    val scope = rememberCoroutineScope()

    WarehouseReconnectRecoveryEffect(
        realtimeSignals = realtimeSignals,
        isBusy = { busy },
    ) { hadInFlight ->
        if (hadInFlight) {
            busy = false
            scope.launch { snackbarHostState.showSnackbar(WAREHOUSE_RECONNECT_RECOVERY_HINT) }
        }
    }

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

    fun matchPending(code: String): PickTask? {
        val trimmed = code.trim()
        if (trimmed.isEmpty()) return null
        return activeWave?.tasks?.firstOrNull { t ->
            t.status == "PENDING" && (
                t.productId.equals(trimmed, ignoreCase = true) ||
                    t.lotId.equals(trimmed, ignoreCase = true)
                )
        }
    }

    fun confirmPick(task: PickTask, qty: Long) {
        val wave = activeWave ?: return
        busy = true
        scope.launch {
            try {
                val res = opsRepository.confirmPickTask(wave.waveId, task.taskId, qty)
                if (res.isSuccessful) {
                    activeWave = res.body()
                    pickScan = ""
                    pickScannedQty = 0L
                    snackbarHostState.showSnackbar("Confirmed · ${activeWave?.status}")
                } else {
                    snackbarHostState.showSnackbar("Confirm failed (${res.code()})")
                }
            } catch (e: Exception) {
                snackbarHostState.showSnackbar(e.message ?: "Confirm failed")
            } finally {
                busy = false
            }
        }
    }

    fun onPickBarcode(code: String) {
        val trimmed = code.trim()
        if (trimmed.isEmpty() || busy) return
        pickScan = trimmed
        if (putawayProduct.isBlank()) {
            putawayProduct = trimmed
        }
        val task = matchPending(trimmed) ?: return
        val next = pickScannedQty + 1L
        pickScannedQty = next
        if (next >= task.quantityRequested) {
            confirmPick(task, task.quantityRequested)
        } else {
            scope.launch {
                snackbarHostState.showSnackbar("Scanned $next / ${task.quantityRequested}")
            }
        }
    }

    DataWedgeBarcodeEffect(onBarcode = ::onPickBarcode)

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Transfer actions") },
                navigationIcon = {
                    if (onBack != null) {
                        IconButton(onClick = onBack) {
                            Icon(
                                Icons.AutoMirrored.Filled.ArrowBack,
                                contentDescription = stringResource(R.string.common_action_back),
                            )
                        }
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
            WarehouseSectionTitle("Inbound transfer controls")
            Text(
                "Factory inbound transfer controls for warehouse operators.",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            WarehouseSectionTitle("Create or force receive")
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
            WarehouseSectionTitle("Receive by transfer ID")
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

            WarehouseSectionTitle("WMS putaway (lots)")
            OutlinedTextField(
                value = putawayProduct,
                onValueChange = { putawayProduct = it },
                label = { Text("Product ID") },
                modifier = Modifier.fillMaxWidth(),
                enabled = !busy,
                singleLine = true,
            )
            OutlinedTextField(
                value = putawayLocation,
                onValueChange = { putawayLocation = it },
                label = { Text("Location / bin ID") },
                modifier = Modifier.fillMaxWidth(),
                enabled = !busy,
                singleLine = true,
            )
            OutlinedTextField(
                value = putawayQty,
                onValueChange = { putawayQty = it },
                label = { Text("Quantity") },
                modifier = Modifier.fillMaxWidth(),
                enabled = !busy,
                singleLine = true,
            )
            OutlinedTextField(
                value = putawayExpiry,
                onValueChange = { putawayExpiry = it },
                label = { Text("Expiry YYYY-MM-DD (perishable)") },
                modifier = Modifier.fillMaxWidth(),
                enabled = !busy,
                singleLine = true,
            )
            Button(
                onClick = {
                    runAction("Create STAGE bin") {
                        opsRepository.createBin(
                            WarehouseBinCreateRequest(
                                locationId = putawayLocation.ifBlank { null },
                                zone = "RECV",
                                locationType = "STAGE",
                            ),
                        )
                    }
                },
                enabled = !busy,
                modifier = Modifier.fillMaxWidth(),
            ) { Text("Ensure bin") }
            Button(
                onClick = {
                    val pid = putawayProduct.trim()
                    val lid = putawayLocation.trim()
                    val qty = putawayQty.trim().toLongOrNull() ?: 0L
                    if (pid.isEmpty() || lid.isEmpty() || qty <= 0L) {
                        scope.launch { snackbarHostState.showSnackbar("Product, location, qty required") }
                        return@Button
                    }
                    runAction("Putaway lot") {
                        opsRepository.putawayLot(
                            StockLotPutawayRequest(
                                productId = pid,
                                locationId = lid,
                                quantity = qty,
                                expiryDate = putawayExpiry.trim().ifEmpty { null },
                            ),
                        )
                    }
                },
                enabled = !busy,
                modifier = Modifier.fillMaxWidth(),
            ) { Text("Putaway lot") }

            WarehouseSectionTitle("WMS pick waves")
            OutlinedTextField(
                value = pickManifestId,
                onValueChange = { pickManifestId = it },
                label = { Text("Manifest ID") },
                modifier = Modifier.fillMaxWidth(),
                enabled = !busy,
                singleLine = true,
            )
            Button(
                onClick = {
                    val mid = pickManifestId.trim()
                    if (mid.isEmpty()) {
                        scope.launch { snackbarHostState.showSnackbar("Manifest ID required") }
                        return@Button
                    }
                    busy = true
                    scope.launch {
                        try {
                            val res = opsRepository.createPickWave(mid)
                            if (res.isSuccessful) {
                                activeWave = res.body()
                                pickScannedQty = 0L
                                snackbarHostState.showSnackbar("Pick wave ${activeWave?.status ?: "created"}")
                            } else {
                                snackbarHostState.showSnackbar("Create wave failed (${res.code()})")
                            }
                        } catch (e: Exception) {
                            snackbarHostState.showSnackbar(e.message ?: "Create wave failed")
                        } finally {
                            busy = false
                        }
                    }
                },
                enabled = !busy,
                modifier = Modifier.fillMaxWidth(),
            ) { Text("Create pick wave") }
            Button(
                onClick = {
                    busy = true
                    scope.launch {
                        try {
                            val res = opsRepository.listPickWaves(pickManifestId.trim().ifEmpty { null })
                            val wave = res.body()?.waves?.firstOrNull()
                            if (res.isSuccessful && wave != null) {
                                val detail = opsRepository.getPickWave(wave.waveId)
                                activeWave = detail.body() ?: wave
                                pickScannedQty = 0L
                                snackbarHostState.showSnackbar("Loaded ${activeWave?.tasks?.size ?: 0} tasks")
                            } else {
                                snackbarHostState.showSnackbar("No pick waves")
                            }
                        } catch (e: Exception) {
                            snackbarHostState.showSnackbar(e.message ?: "List failed")
                        } finally {
                            busy = false
                        }
                    }
                },
                enabled = !busy,
                modifier = Modifier.fillMaxWidth(),
            ) { Text("Load pick waves") }

            EanBarcodeScannerPreview(
                onBarcode = ::onPickBarcode,
                enabled = !busy,
                previewHeightDp = 180,
            )
            KeyboardWedgeBarcodeField(
                onBarcode = ::onPickBarcode,
                enabled = !busy,
            )
            OutlinedTextField(
                value = pickScan,
                onValueChange = { pickScan = it },
                label = { Text("Scan product / lot ID") },
                modifier = Modifier.fillMaxWidth(),
                enabled = !busy,
                singleLine = true,
            )
            val pending: PickTask? = activeWave?.tasks?.firstOrNull { t ->
                t.status == "PENDING" && (
                    pickScan.isBlank() ||
                        t.productId.equals(pickScan.trim(), ignoreCase = true) ||
                        t.lotId.equals(pickScan.trim(), ignoreCase = true)
                    )
            }
            if (activeWave != null) {
                Text(
                    stringResource(
                        R.string.mobile_warehouse_ui_wave_take_status,
                        activeWave!!.waveId.take(8),
                        activeWave!!.status,
                    ),
                    style = MaterialTheme.typography.bodyMedium,
                )
                pending?.let { task ->
                    Text(
                        stringResource(
                            R.string.mobile_warehouse_ui_next_productid_locationid_qty_quantityrequested,
                            task.productId,
                            task.locationId,
                            task.quantityRequested,
                        ),
                        style = MaterialTheme.typography.bodySmall,
                    )
                    Text(
                        "Scanned $pickScannedQty / ${task.quantityRequested}",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.primary,
                    )
                    Button(
                        onClick = {
                            val qty = if (pickScannedQty > 0L) {
                                minOf(pickScannedQty, task.quantityRequested)
                            } else {
                                task.quantityRequested
                            }
                            confirmPick(task, qty)
                        },
                        enabled = !busy,
                        modifier = Modifier.fillMaxWidth(),
                    ) { Text("Confirm pick") }
                }
            }

            WarehouseSectionTitle("WMS cycle counts")
            OutlinedTextField(
                value = putawayLocation,
                onValueChange = { putawayLocation = it },
                label = { Text("Count location ID") },
                modifier = Modifier.fillMaxWidth(),
                enabled = !busy,
                singleLine = true,
            )
            OutlinedTextField(
                value = putawayProduct,
                onValueChange = { putawayProduct = it },
                label = { Text("Count product ID") },
                modifier = Modifier.fillMaxWidth(),
                enabled = !busy,
                singleLine = true,
            )
            Button(
                onClick = {
                    val lid = putawayLocation.trim()
                    val pid = putawayProduct.trim()
                    if (lid.isEmpty() || pid.isEmpty()) {
                        scope.launch { snackbarHostState.showSnackbar("Location + product required") }
                        return@Button
                    }
                    busy = true
                    scope.launch {
                        try {
                            val res = opsRepository.createCycleCount(lid, pid, null)
                            if (res.isSuccessful) {
                                snackbarHostState.showSnackbar("Count ${res.body()?.countId?.take(8)}…")
                            } else {
                                snackbarHostState.showSnackbar("Create count failed (${res.code()})")
                            }
                        } catch (e: Exception) {
                            snackbarHostState.showSnackbar(e.message ?: "Create count failed")
                        } finally {
                            busy = false
                        }
                    }
                },
                enabled = !busy,
                modifier = Modifier.fillMaxWidth(),
            ) { Text("Create cycle count") }
            Button(
                onClick = {
                    busy = true
                    scope.launch {
                        try {
                            val res = opsRepository.listCycleCounts()
                            val open = res.body()?.counts?.firstOrNull { it.status == "OPEN" }
                            if (open != null) {
                                val sub = opsRepository.submitCycleCount(open.countId, open.expectedQty)
                                if (sub.isSuccessful) {
                                    snackbarHostState.showSnackbar("Submitted ${sub.body()?.status}")
                                } else {
                                    snackbarHostState.showSnackbar("Submit failed (${sub.code()})")
                                }
                            } else {
                                snackbarHostState.showSnackbar("No OPEN counts")
                            }
                        } catch (e: Exception) {
                            snackbarHostState.showSnackbar(e.message ?: "List/submit failed")
                        } finally {
                            busy = false
                        }
                    }
                },
                enabled = !busy,
                modifier = Modifier.fillMaxWidth(),
            ) { Text("Submit first OPEN count @ expected") }
        }
    }
}
