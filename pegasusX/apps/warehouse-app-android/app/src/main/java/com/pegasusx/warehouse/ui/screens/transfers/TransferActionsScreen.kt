package com.pegasusx.warehouse.ui.screens.transfers

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.pegasus.barcode.DataWedgeBarcodeEffect
import com.pegasus.barcode.EanBarcodeScannerPreview
import com.pegasus.barcode.KeyboardWedgeBarcodeField
import com.pegasusx.warehouse.R
import com.pegasusx.warehouse.data.model.EmergencyTransferRequest
import com.pegasusx.warehouse.data.model.ForceReceiveRequest
import com.pegasusx.warehouse.data.model.PickTask
import com.pegasusx.warehouse.data.model.PickWave
import com.pegasusx.warehouse.data.model.StockLot
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
    var cycleExpectedInput by remember { mutableStateOf("") }
    var wmsState by remember { mutableStateOf(TransferWmsState()) }
    val wmsController = remember(opsRepository) {
        TransferActionsOperationsController(
            opsRepository = opsRepository,
            getStateOverride = { wmsState },
            setStateOverride = { wmsState = it },
        )
    }
    var busy by remember { mutableStateOf(false) }
    val snackbarHostState = remember { SnackbarHostState() }
    val scope = rememberCoroutineScope()

    LaunchedEffect(wmsController) {
        try {
            wmsController.refreshBinsAndLots()
            wmsController.refreshCycleCountsAndAdjustments()
        } catch (e: Exception) {
            snackbarHostState.showSnackbar(e.message ?: "Unable to refresh cycle counts")
        }
    }

    LaunchedEffect(wmsState.activeWave?.manifestId) {
        val mid = wmsState.activeWave?.manifestId.orEmpty()
        val wave = wmsState.activeWave
        if (mid.isNotEmpty() && (wave?.coldLotMigratedWithoutLiveBaseline == true || wave?.coldChainBaselineMissing == true || TransferWmsHelpers.coldWarningLots(wmsState.lots, wmsState.bins).isNotEmpty())) {
            wmsController.refreshManifestTempReadiness(mid)
        }
    }

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
        return wmsState.activeWave?.tasks?.firstOrNull { t ->
            (t.status == "PENDING" || t.status == "SHORT") && (
                t.productId.equals(trimmed, ignoreCase = true) ||
                    t.lotId.equals(trimmed, ignoreCase = true)
                )
        }
    }

    fun confirmPickWithQty(task: PickTask, quantityPicked: Long) {
        busy = true
        scope.launch {
            try {
                val res = wmsController.confirmPickExplicit(task, quantityPicked)
                res.onSuccess { updated ->
                    pickScan = ""
                    pickScannedQty = 0L
                    snackbarHostState.showSnackbar("Confirmed · ${updated.status}")
                }.onFailure { err ->
                    snackbarHostState.showSnackbar(err.message ?: "Confirm failed")
                }
            } finally {
                busy = false
            }
        }
    }

    fun confirmPickFromInput(task: PickTask) {
        busy = true
        scope.launch {
            try {
                val res = wmsController.confirmPickWithInput(task, wmsState.pickQtyText)
                res.onSuccess { updated ->
                    pickScan = ""
                    pickScannedQty = 0L
                    snackbarHostState.showSnackbar("Confirmed · ${updated.status}")
                }.onFailure { err ->
                    snackbarHostState.showSnackbar(err.message ?: "Confirm failed")
                }
            } finally {
                busy = false
            }
        }
    }

    fun waiveActiveShorts() {
        busy = true
        scope.launch {
            try {
                val res = wmsController.waivePickShorts()
                res.onSuccess { updated ->
                    snackbarHostState.showSnackbar("Shorts waived · ${updated.status}")
                }.onFailure { err ->
                    snackbarHostState.showSnackbar(err.message ?: "Waive shorts failed")
                }
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
        wmsController.updatePickQtyText(minOf(next, task.quantityRequested).toString())
        if (next >= task.quantityRequested) {
            confirmPickWithQty(task, task.quantityRequested)
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
            val coldAlerts: List<StockLot> = remember(wmsState.lots, wmsState.bins) {
                val cw = TransferWmsHelpers.coldWarningLots(wmsState.lots, wmsState.bins)
                cw.ifEmpty { TransferWmsHelpers.stagingPutawayLots(wmsState.lots, wmsState.bins) }
            }
            if (coldAlerts.isNotEmpty()) {
                Surface(
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(10.dp),
                    color = MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.45f),
                    border = BorderStroke(1.dp, MaterialTheme.colorScheme.error.copy(alpha = 0.5f)),
                ) {
                    Column(
                        modifier = Modifier.padding(12.dp),
                        verticalArrangement = Arrangement.spacedBy(6.dp),
                    ) {
                        Text(
                            stringResource(R.string.mobile_warehouse_cold_lot_banner),
                            style = MaterialTheme.typography.titleSmall,
                            color = MaterialTheme.colorScheme.onErrorContainer,
                        )
                        Text(
                            "cold_lot_migrated_without_live_baseline",
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.onErrorContainer,
                        )
                        coldAlerts.take(5).forEach { lot ->
                            OutlinedButton(
                                onClick = {
                                    putawayProduct = lot.productId
                                    putawayQty = maxOf(1L, lot.quantityOnHand).toString()
                                    putawayExpiry = lot.expiryDate.orEmpty()
                                    TransferWmsHelpers.recommendedBinForLot(lot, wmsState.bins)?.let {
                                        putawayLocation = it
                                    }
                                },
                                modifier = Modifier.fillMaxWidth(),
                            ) {
                                Column(modifier = Modifier.fillMaxWidth()) {
                                    Text(
                                        "${lot.productName ?: lot.productId} · ${lot.locationId} (${lot.quantityOnHand})",
                                        style = MaterialTheme.typography.labelMedium,
                                    )
                                    Text(
                                        buildString {
                                            append(
                                                stringResource(
                                                    R.string.mobile_warehouse_cold_lot_lot_label,
                                                    lot.lotCode ?: lot.lotId.take(8),
                                                ),
                                            )
                                            lot.expiryDate?.takeIf { it.isNotBlank() }?.let { exp ->
                                                append(" · ")
                                                append(stringResource(R.string.mobile_warehouse_cold_lot_expiry_label, exp))
                                            }
                                            TransferWmsHelpers.requiredColdZone(lot)?.let { zone ->
                                                append(" · ")
                                                append(zone)
                                            }
                                        },
                                        style = MaterialTheme.typography.labelSmall,
                                    )
                                }
                            }
                        }
                    }
                }
            }
            val mismatchZone = remember(putawayProduct, putawayLocation, wmsState.lots, wmsState.bins) {
                TransferWmsHelpers.putawayZoneMismatch(putawayProduct, putawayLocation, wmsState.lots, wmsState.bins)
            }
            if (mismatchZone != null) {
                Text(
                    stringResource(R.string.mobile_warehouse_cold_lot_zone_mismatch, mismatchZone),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.error,
                )
            }
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
                    busy = true
                    scope.launch {
                        try {
                            val res = wmsController.createPickWave(pickManifestId)
                            res.onSuccess { created ->
                                pickScannedQty = 0L
                                snackbarHostState.showSnackbar("Pick wave ${created.status.ifEmpty { "created" }}")
                            }.onFailure { err ->
                                snackbarHostState.showSnackbar(err.message ?: "Create wave failed")
                            }
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
                            val res = wmsController.loadPickWaves(pickManifestId)
                            res.onSuccess { loaded ->
                                pickScannedQty = 0L
                                snackbarHostState.showSnackbar("Loaded ${loaded.tasks.size} tasks")
                            }.onFailure { err ->
                                snackbarHostState.showSnackbar(err.message ?: "List failed")
                            }
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
            val activeWave = wmsState.activeWave
            val editableTask: PickTask? = activeWave?.tasks?.firstOrNull { t ->
                (t.status == "PENDING" || t.status == "SHORT") && (
                    pickScan.isBlank() ||
                        t.productId.equals(pickScan.trim(), ignoreCase = true) ||
                        t.lotId.equals(pickScan.trim(), ignoreCase = true)
                    )
            }
            if (activeWave != null) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        stringResource(
                            R.string.mobile_warehouse_ui_wave_take_status,
                            activeWave.waveId.take(8),
                            activeWave.status,
                        ),
                        style = MaterialTheme.typography.bodyMedium,
                    )
                    if (TransferWmsHelpers.hasShortTasks(activeWave)) {
                        OutlinedButton(
                            onClick = { waiveActiveShorts() },
                            enabled = !busy,
                        ) {
                            Text("Waive shorts")
                        }
                    }
                }
                if (TransferWmsHelpers.isWavePreSealColdWarning(wmsState)) {
                    Surface(
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(8.dp),
                        color = MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.4f),
                        border = BorderStroke(1.dp, MaterialTheme.colorScheme.error.copy(alpha = 0.5f)),
                    ) {
                        Column(
                            modifier = Modifier.padding(10.dp),
                            verticalArrangement = Arrangement.spacedBy(2.dp),
                        ) {
                            Text(
                                stringResource(R.string.mobile_warehouse_cold_lot_banner),
                                style = MaterialTheme.typography.labelMedium,
                                color = MaterialTheme.colorScheme.onErrorContainer,
                            )
                            Text(
                                "cold_lot_migrated_without_live_baseline · Pre-seal cold-chain check",
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.onErrorContainer,
                            )
                        }
                    }
                }
                activeWave.tasks.forEach { task ->
                    Surface(
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(8.dp),
                        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant),
                    ) {
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(horizontal = 12.dp, vertical = 8.dp),
                            horizontalArrangement = Arrangement.SpaceBetween,
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Column(modifier = Modifier.weight(1f)) {
                                Text(
                                    "${task.productId} · ${task.locationId.ifEmpty { "—" }}",
                                    style = MaterialTheme.typography.bodySmall,
                                )
                                Text(
                                    "Picked ${task.quantityPicked}/${task.quantityRequested}",
                                    style = MaterialTheme.typography.labelSmall,
                                )
                            }
                            PickStatusChip(status = task.status)
                        }
                    }
                }
                editableTask?.let { task ->
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
                    OutlinedTextField(
                        value = wmsState.pickQtyText,
                        onValueChange = { wmsController.updatePickQtyText(it) },
                        label = { Text("Picked quantity (0 = SHORT)") },
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                        modifier = Modifier.fillMaxWidth(),
                        enabled = !busy,
                        singleLine = true,
                    )
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        Button(
                            onClick = { confirmPickFromInput(task) },
                            enabled = !busy,
                            modifier = Modifier.weight(1f),
                        ) { Text("Confirm pick") }
                        OutlinedButton(
                            onClick = { confirmPickWithQty(task, 0L) },
                            enabled = !busy,
                            modifier = Modifier.weight(1f),
                        ) { Text("Report short (0)") }
                    }
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
            OutlinedTextField(
                value = cycleExpectedInput,
                onValueChange = { cycleExpectedInput = it },
                label = { Text("Expected qty (optional)") },
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                modifier = Modifier.fillMaxWidth(),
                enabled = !busy,
                singleLine = true,
            )
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Button(
                    onClick = {
                        busy = true
                        scope.launch {
                            try {
                                val res = wmsController.createCycleCount(
                                    locationId = putawayLocation,
                                    productId = putawayProduct,
                                    expectedQtyText = cycleExpectedInput,
                                )
                                res.onSuccess { created ->
                                    cycleExpectedInput = ""
                                    snackbarHostState.showSnackbar("Count ${created.countId.take(8)}…")
                                }.onFailure { err ->
                                    snackbarHostState.showSnackbar(err.message ?: "Create count failed")
                                }
                            } finally {
                                busy = false
                            }
                        }
                    },
                    enabled = !busy,
                    modifier = Modifier.weight(1f),
                ) { Text("Create cycle count") }
                OutlinedButton(
                    onClick = {
                        busy = true
                        scope.launch {
                            try {
                                val res = wmsController.refreshCycleCountsAndAdjustments()
                                res.onSuccess {
                                    snackbarHostState.showSnackbar("Refreshed ${wmsState.cycleCounts.size} counts")
                                }.onFailure { err ->
                                    snackbarHostState.showSnackbar(err.message ?: "Refresh failed")
                                }
                            } finally {
                                busy = false
                            }
                        }
                    },
                    enabled = !busy,
                    modifier = Modifier.weight(1f),
                ) { Text("Refresh counts") }
            }

            wmsState.cycleCounts.forEach { count ->
                val variance = TransferWmsHelpers.computeVariance(count)
                val isSelected = count.countId == TransferWmsHelpers.selectedOpenCount(wmsState)?.countId
                Surface(
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(8.dp),
                    border = BorderStroke(
                        width = if (isSelected) 2.dp else 1.dp,
                        color = if (isSelected) {
                            MaterialTheme.colorScheme.primary
                        } else {
                            MaterialTheme.colorScheme.outlineVariant
                        },
                    ),
                ) {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(horizontal = 12.dp, vertical = 8.dp),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Column(modifier = Modifier.weight(1f)) {
                            Text(
                                "${count.locationId} · ${count.productId} (${count.status})",
                                style = MaterialTheme.typography.bodySmall,
                            )
                            val qtySummary = if (count.countedQty != null) {
                                val deltaLabel = variance?.let { " · Δ${TransferWmsHelpers.formatSignedDelta(it)}" }.orEmpty()
                                "Counted ${count.countedQty}/${count.expectedQty}$deltaLabel"
                            } else {
                                "Expected ${count.expectedQty}"
                            }
                            Text(qtySummary, style = MaterialTheme.typography.labelSmall)
                        }
                        if (count.status == "OPEN") {
                            Spacer(modifier = Modifier.width(8.dp))
                            OutlinedButton(
                                onClick = { wmsController.selectOpenCount(count) },
                                enabled = !busy,
                            ) { Text(if (isSelected) "Selected" else "Select") }
                        }
                    }
                }
            }

            val selectedOpen = TransferWmsHelpers.selectedOpenCount(wmsState)
            if (selectedOpen != null) {
                Text(
                    "Submit count ${selectedOpen.countId.take(8)}… (${selectedOpen.productId} @ ${selectedOpen.locationId})",
                    style = MaterialTheme.typography.bodySmall,
                )
                OutlinedTextField(
                    value = wmsState.countedQtyText,
                    onValueChange = { wmsController.updateCountedQtyText(it) },
                    label = { Text("Actual counted qty (0 allowed)") },
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                    modifier = Modifier.fillMaxWidth(),
                    enabled = !busy,
                    singleLine = true,
                )
                OutlinedTextField(
                    value = wmsState.cycleReasonCode,
                    onValueChange = { wmsController.updateCycleReasonCode(it) },
                    label = { Text("Reason code (optional)") },
                    modifier = Modifier.fillMaxWidth(),
                    enabled = !busy,
                    singleLine = true,
                )
                Button(
                    onClick = {
                        busy = true
                        scope.launch {
                            try {
                                val res = wmsController.submitCycleCount(
                                    countId = selectedOpen.countId,
                                    countedQtyText = wmsState.countedQtyText,
                                    reasonCode = wmsState.cycleReasonCode,
                                )
                                res.onSuccess { submitted ->
                                    snackbarHostState.showSnackbar("Submitted · ${submitted.status}")
                                }.onFailure { err ->
                                    snackbarHostState.showSnackbar(err.message ?: "Submit failed")
                                }
                            } finally {
                                busy = false
                            }
                        }
                    },
                    enabled = !busy,
                    modifier = Modifier.fillMaxWidth(),
                ) { Text("Submit cycle count") }
            }

            if (wmsState.adjustments.isNotEmpty()) {
                WarehouseSectionTitle("Inventory adjustments")
                wmsState.adjustments.forEach { adj ->
                    Surface(
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(8.dp),
                        border = BorderStroke(1.dp, MaterialTheme.colorScheme.outlineVariant),
                    ) {
                        Column(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(horizontal = 12.dp, vertical = 8.dp),
                            verticalArrangement = Arrangement.spacedBy(6.dp),
                        ) {
                            Text(
                                "${adj.adjustmentId.take(8)}… · ${adj.productId} · Δ${TransferWmsHelpers.formatSignedDelta(adj.deltaQty)} · ${adj.status}",
                                style = MaterialTheme.typography.bodySmall,
                            )
                            if (adj.status == "PENDING") {
                                Row(
                                    modifier = Modifier.fillMaxWidth(),
                                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                                ) {
                                    Button(
                                        onClick = {
                                            busy = true
                                            scope.launch {
                                                try {
                                                    val res = wmsController.approveAdjustment(adj.adjustmentId)
                                                    res.onSuccess {
                                                        snackbarHostState.showSnackbar("Adjustment approved")
                                                    }.onFailure { err ->
                                                        snackbarHostState.showSnackbar(err.message ?: "Approve failed")
                                                    }
                                                } finally {
                                                    busy = false
                                                }
                                            }
                                        },
                                        enabled = !busy,
                                        modifier = Modifier.weight(1f),
                                    ) { Text("Approve") }
                                    OutlinedButton(
                                        onClick = {
                                            busy = true
                                            scope.launch {
                                                try {
                                                    val res = wmsController.rejectAdjustment(adj.adjustmentId)
                                                    res.onSuccess {
                                                        snackbarHostState.showSnackbar("Adjustment rejected")
                                                    }.onFailure { err ->
                                                        snackbarHostState.showSnackbar(err.message ?: "Reject failed")
                                                    }
                                                } finally {
                                                    busy = false
                                                }
                                            }
                                        },
                                        enabled = !busy,
                                        modifier = Modifier.weight(1f),
                                    ) { Text("Reject") }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun PickStatusChip(status: String) {
    val containerColor = when (status) {
        "CONFIRMED" -> MaterialTheme.colorScheme.primaryContainer
        "SHORT" -> MaterialTheme.colorScheme.errorContainer
        "SHORT_WAIVED" -> MaterialTheme.colorScheme.secondaryContainer
        else -> MaterialTheme.colorScheme.surfaceVariant
    }
    val contentColor = when (status) {
        "CONFIRMED" -> MaterialTheme.colorScheme.onPrimaryContainer
        "SHORT" -> MaterialTheme.colorScheme.onErrorContainer
        "SHORT_WAIVED" -> MaterialTheme.colorScheme.onSecondaryContainer
        else -> MaterialTheme.colorScheme.onSurfaceVariant
    }
    Surface(
        shape = RoundedCornerShape(6.dp),
        color = containerColor,
    ) {
        Text(
            text = status,
            color = contentColor,
            style = MaterialTheme.typography.labelSmall,
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
        )
    }
}
