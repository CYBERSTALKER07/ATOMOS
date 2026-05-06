package com.pegasus.factory.ui.screens.override

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.RadioButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalLifecycleOwner
import androidx.compose.ui.unit.dp
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import com.pegasus.factory.data.model.Manifest
import com.pegasus.factory.data.model.ManifestCancelRequest
import com.pegasus.factory.data.model.ManifestCancelTransferRequest
import com.pegasus.factory.data.model.ManifestRebalanceRequest
import com.pegasus.factory.data.model.ManifestTransfer
import com.pegasus.factory.data.remote.FactoryApi
import com.pegasus.factory.data.remote.FactoryRealtimeEventType
import com.pegasus.factory.ui.realtime.FactoryRealtimeReloadEffect
import com.pegasus.factory.ui.theme.PegasusSpacing
import java.text.DateFormat
import java.util.Date
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch

private data class MoveTransferCandidate(
    val sourceManifestId: String,
    val transfer: ManifestTransfer,
)

private data class CancelTransferCandidate(
    val manifestId: String,
    val transfer: ManifestTransfer,
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PayloadOverrideScreen(
    api: FactoryApi,
    onBack: () -> Unit,
) {
    var manifests by remember { mutableStateOf<List<Manifest>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var actingKey by remember { mutableStateOf<String?>(null) }
    var refreshing by remember { mutableStateOf(false) }
    var staleMessage by remember { mutableStateOf<String?>(null) }
    var lastSyncedAt by remember { mutableStateOf<Long?>(null) }
    var moveCandidate by remember { mutableStateOf<MoveTransferCandidate?>(null) }
    var cancelTransferCandidate by remember { mutableStateOf<CancelTransferCandidate?>(null) }
    var cancelManifestCandidate by remember { mutableStateOf<Manifest?>(null) }
    var selectedTargetManifestId by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()
    val snackbarHostState = remember { SnackbarHostState() }
    val lifecycleOwner = LocalLifecycleOwner.current

    fun load(background: Boolean = false) {
        if (background) {
            refreshing = true
        } else if (manifests.isEmpty()) {
            loading = true
            error = null
        }
        scope.launch {
            try {
                val resp = api.getManifests(state = "LOADING")
                if (resp.isSuccessful && resp.body() != null) {
                    manifests = resp.body()!!.manifests.filter { it.state == "LOADING" }
                    staleMessage = null
                    error = null
                    lastSyncedAt = System.currentTimeMillis()
                } else {
                    val message = "Failed (${resp.code()})"
                    if (manifests.isEmpty()) {
                        error = message
                    } else {
                        staleMessage = "Showing last synced manifests. $message"
                    }
                }
            } catch (e: Exception) {
                val message = e.message ?: "Network error"
                if (manifests.isEmpty()) {
                    error = message
                } else {
                    staleMessage = "Showing last synced manifests. $message"
                }
            } finally {
                loading = false
                refreshing = false
            }
        }
    }

    fun rebalance(candidate: MoveTransferCandidate, targetManifestId: String) {
        actingKey = candidate.transfer.transferId
        scope.launch {
            try {
                val resp = api.rebalanceManifest(
                    ManifestRebalanceRequest(
                        sourceManifestId = candidate.sourceManifestId,
                        targetManifestId = targetManifestId,
                        transferIds = listOf(candidate.transfer.transferId),
                    )
                )
                if (resp.isSuccessful) {
                    snackbarHostState.showSnackbar("Moved ${candidate.transfer.transferId.take(8)}")
                    moveCandidate = null
                    load(background = true)
                } else {
                    snackbarHostState.showSnackbar("Move failed (${resp.code()})")
                }
            } catch (e: Exception) {
                snackbarHostState.showSnackbar(e.message ?: "Move failed")
            } finally {
                actingKey = null
            }
        }
    }

    fun cancelTransfer(candidate: CancelTransferCandidate) {
        actingKey = candidate.transfer.transferId
        scope.launch {
            try {
                val resp = api.cancelManifestTransfer(
                    ManifestCancelTransferRequest(
                        manifestId = candidate.manifestId,
                        transferId = candidate.transfer.transferId,
                    )
                )
                if (resp.isSuccessful) {
                    snackbarHostState.showSnackbar("Released ${candidate.transfer.transferId.take(8)}")
                    cancelTransferCandidate = null
                    load(background = true)
                } else {
                    snackbarHostState.showSnackbar("Release failed (${resp.code()})")
                }
            } catch (e: Exception) {
                snackbarHostState.showSnackbar(e.message ?: "Release failed")
            } finally {
                actingKey = null
            }
        }
    }

    fun cancelManifest(manifest: Manifest) {
        actingKey = manifest.id
        scope.launch {
            try {
                val resp = api.cancelManifest(ManifestCancelRequest(manifest.id))
                if (resp.isSuccessful) {
                    snackbarHostState.showSnackbar("Cancelled manifest ${manifest.id.take(8)}")
                    cancelManifestCandidate = null
                    load(background = true)
                } else {
                    snackbarHostState.showSnackbar("Cancel failed (${resp.code()})")
                }
            } catch (e: Exception) {
                snackbarHostState.showSnackbar(e.message ?: "Cancel failed")
            } finally {
                actingKey = null
            }
        }
    }

    LaunchedEffect(Unit) {
        load()
        while (isActive) {
            delay(30_000)
            if (actingKey == null) {
                load(background = true)
            }
        }
    }
    LaunchedEffect(moveCandidate?.sourceManifestId) {
        selectedTargetManifestId = ""
    }

    DisposableEffect(lifecycleOwner) {
        val observer = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_RESUME) {
                load(background = manifests.isNotEmpty())
            }
        }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose {
            lifecycleOwner.lifecycle.removeObserver(observer)
        }
    }

    FactoryRealtimeReloadEffect(
        eventTypes = setOf(
            FactoryRealtimeEventType.TransferUpdate,
            FactoryRealtimeEventType.ManifestUpdate,
        ),
    ) {
        if (actingKey == null) {
            load(background = manifests.isNotEmpty())
        }
    }

    val runtimeStatus = when {
        refreshing -> "Refreshing live manifests — last sync ${formatOverrideSyncTime(lastSyncedAt)}"
        staleMessage != null -> staleMessage!!
        lastSyncedAt != null -> "Live sync active — last sync ${formatOverrideSyncTime(lastSyncedAt)}"
        else -> "Waiting for first sync"
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Payload Override") },
                navigationIcon = {
                    IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") }
                },
                actions = {
                    IconButton(onClick = { load(background = manifests.isNotEmpty()) }) { Icon(Icons.Default.Refresh, "Refresh") }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { innerPadding ->
        when {
            loading -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                CircularProgressIndicator()
            }
            error != null -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(error!!, color = MaterialTheme.colorScheme.error)
                    Spacer(Modifier.height(PegasusSpacing.lg))
                    Button(onClick = { load() }) { Text("Retry") }
                }
            }
            manifests.isEmpty() -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                Text(
                    text = "No manifests are currently loading.",
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            else -> LazyColumn(
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            ) {
                item {
                    OverrideSummaryCard(
                        manifests = manifests,
                        runtimeStatus = runtimeStatus,
                        stale = staleMessage != null,
                    )
                }
                items(manifests, key = { it.id }) { manifest ->
                    OverrideManifestCard(
                        manifest = manifest,
                        hasMoveTargets = manifests.any { it.id != manifest.id },
                        actingKey = actingKey,
                        onMove = { transfer -> moveCandidate = MoveTransferCandidate(manifest.id, transfer) },
                        onRemove = { transfer -> cancelTransferCandidate = CancelTransferCandidate(manifest.id, transfer) },
                        onCancelManifest = { cancelManifestCandidate = manifest },
                    )
                }
            }
        }
    }

    moveCandidate?.let { candidate ->
        val targetOptions = manifests.filter { it.id != candidate.sourceManifestId }
        AlertDialog(
            onDismissRequest = { moveCandidate = null },
            title = { Text("Move transfer") },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                    Text("Select the loading manifest that should receive transfer ${candidate.transfer.transferId.take(8)}.")
                    targetOptions.forEach { manifest ->
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .clickable { selectedTargetManifestId = manifest.id }
                                .padding(vertical = PegasusSpacing.xs),
                            verticalAlignment = Alignment.CenterVertically,
                            horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                        ) {
                            RadioButton(
                                selected = selectedTargetManifestId == manifest.id,
                                onClick = { selectedTargetManifestId = manifest.id },
                            )
                            Column {
                                Text(manifest.truckPlate.ifBlank { manifest.truckId.take(8) })
                                Text(
                                    text = "${trimDecimal(manifest.totalVolumeVU)} / ${trimDecimal(manifest.maxCapacityVU)} VU",
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                            }
                        }
                    }
                    if (targetOptions.isEmpty()) {
                        Text(
                            text = "Create or keep another loading manifest active before moving this transfer.",
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                }
            },
            confirmButton = {
                TextButton(
                    onClick = { rebalance(candidate, selectedTargetManifestId) },
                    enabled = selectedTargetManifestId.isNotBlank() && actingKey == null,
                ) {
                    Text("Move")
                }
            },
            dismissButton = {
                TextButton(onClick = { moveCandidate = null }) { Text("Cancel") }
            },
        )
    }

    cancelTransferCandidate?.let { candidate ->
        AlertDialog(
            onDismissRequest = { cancelTransferCandidate = null },
            title = { Text("Remove transfer") },
            text = { Text("Release transfer ${candidate.transfer.transferId.take(8)} back to APPROVED so it can be reassigned.") },
            confirmButton = {
                TextButton(
                    onClick = { cancelTransfer(candidate) },
                    enabled = actingKey == null,
                ) { Text("Release") }
            },
            dismissButton = {
                TextButton(onClick = { cancelTransferCandidate = null }) { Text("Keep") }
            },
        )
    }

    cancelManifestCandidate?.let { manifest ->
        AlertDialog(
            onDismissRequest = { cancelManifestCandidate = null },
            title = { Text("Cancel manifest") },
            text = { Text("Cancel manifest ${manifest.id.take(8)} and return all linked transfers to APPROVED.") },
            confirmButton = {
                TextButton(
                    onClick = { cancelManifest(manifest) },
                    enabled = actingKey == null,
                ) { Text("Cancel manifest") }
            },
            dismissButton = {
                TextButton(onClick = { cancelManifestCandidate = null }) { Text("Keep") }
            },
        )
    }
}

@Composable
private fun OverrideSummaryCard(
    manifests: List<Manifest>,
    runtimeStatus: String,
    stale: Boolean,
) {
    val transferCount = manifests.sumOf { it.transfers.size }
    ElevatedCard(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.elevatedCardColors(
            containerColor = MaterialTheme.colorScheme.surfaceContainerHigh,
        ),
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
        ) {
            Text(
                text = "Live manifest override",
                style = MaterialTheme.typography.titleLarge,
            )
            Text(
                text = "${manifests.size} loading manifests, $transferCount transfers available for rebalance or release.",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Surface(
                shape = MaterialTheme.shapes.medium,
                color = if (stale) MaterialTheme.colorScheme.errorContainer else MaterialTheme.colorScheme.surfaceContainer,
            ) {
                Text(
                    text = runtimeStatus,
                    style = MaterialTheme.typography.labelMedium,
                    color = if (stale) MaterialTheme.colorScheme.onErrorContainer else MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(horizontal = PegasusSpacing.md, vertical = PegasusSpacing.sm),
                )
            }
        }
    }
}

@Composable
private fun OverrideManifestCard(
    manifest: Manifest,
    hasMoveTargets: Boolean,
    actingKey: String?,
    onMove: (ManifestTransfer) -> Unit,
    onRemove: (ManifestTransfer) -> Unit,
    onCancelManifest: () -> Unit,
) {
    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Column(
                    modifier = Modifier.weight(1f),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
                ) {
                    Text(
                        text = manifest.truckPlate.ifBlank { manifest.truckId.take(8) },
                        style = MaterialTheme.typography.titleMedium,
                    )
                    Text(
                        text = "Manifest ${manifest.id.take(8)}",
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                FilledTonalButton(
                    onClick = onCancelManifest,
                    enabled = actingKey == null,
                ) {
                    Text("Cancel manifest")
                }
            }

            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                LinearProgressIndicator(
                    progress = {
                        val capacity = manifest.maxCapacityVU.takeIf { it > 0 } ?: 1.0
                        (manifest.totalVolumeVU / capacity).coerceIn(0.0, 1.0).toFloat()
                    },
                    modifier = Modifier.fillMaxWidth(),
                )
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                ) {
                    OverrideMetric("Volume", "${trimDecimal(manifest.totalVolumeVU)} VU", Modifier.weight(1f))
                    OverrideMetric("Capacity", "${trimDecimal(manifest.maxCapacityVU)} VU", Modifier.weight(1f))
                    OverrideMetric("Transfers", manifest.transfers.size.toString(), Modifier.weight(1f))
                }
            }

            if (manifest.transfers.isEmpty()) {
                Surface(
                    modifier = Modifier.fillMaxWidth(),
                    shape = MaterialTheme.shapes.medium,
                    color = MaterialTheme.colorScheme.surfaceContainerLowest,
                ) {
                    Text(
                        text = "No transfers are assigned to this manifest.",
                        modifier = Modifier.padding(PegasusSpacing.lg),
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            } else {
                manifest.transfers.forEach { transfer ->
                    OverrideTransferRow(
                        transfer = transfer,
                        canMove = hasMoveTargets,
                        busy = actingKey == transfer.transferId,
                        onMove = { onMove(transfer) },
                        onRemove = { onRemove(transfer) },
                    )
                }
            }
        }
    }
}

@Composable
private fun OverrideTransferRow(
    transfer: ManifestTransfer,
    canMove: Boolean,
    busy: Boolean,
    onMove: () -> Unit,
    onRemove: () -> Unit,
) {
    Surface(
        modifier = Modifier.fillMaxWidth(),
        shape = MaterialTheme.shapes.medium,
        color = MaterialTheme.colorScheme.surfaceContainerLowest,
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.md),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Column(
                    modifier = Modifier.weight(1f),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
                ) {
                    Text(
                        text = transfer.productName.ifBlank { "Transfer ${transfer.transferId.take(8)}" },
                        style = MaterialTheme.typography.titleSmall,
                    )
                    Text(
                        text = transfer.transferId.take(8),
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                OverrideStateTag(transfer.state)
            }

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                OverrideMetric("Qty", transfer.quantity.toString(), Modifier.weight(1f))
                OverrideMetric("Volume", "${trimDecimal(transfer.volumeVU)} VU", Modifier.weight(1f))
            }

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                FilledTonalButton(
                    onClick = onMove,
                    enabled = canMove && !busy,
                    modifier = Modifier.weight(1f),
                ) {
                    Text(if (busy) "Working…" else "Move")
                }
                Button(
                    onClick = onRemove,
                    enabled = !busy,
                    modifier = Modifier.weight(1f),
                ) {
                    Text(if (busy) "Working…" else "Release")
                }
            }
        }
    }
}

@Composable
private fun OverrideMetric(
    label: String,
    value: String,
    modifier: Modifier = Modifier,
) {
    Surface(
        modifier = modifier,
        shape = MaterialTheme.shapes.medium,
        color = MaterialTheme.colorScheme.surfaceContainer,
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.md),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
        ) {
            Text(value, style = MaterialTheme.typography.titleSmall)
            Text(
                text = label,
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}

@Composable
private fun OverrideStateTag(
    text: String,
) {
    Surface(
        shape = MaterialTheme.shapes.small,
        color = MaterialTheme.colorScheme.secondaryContainer,
        contentColor = MaterialTheme.colorScheme.onSecondaryContainer,
    ) {
        Text(
            text = text,
            style = MaterialTheme.typography.labelMedium,
            modifier = Modifier.padding(horizontal = PegasusSpacing.sm, vertical = PegasusSpacing.xs),
        )
    }
}

private fun trimDecimal(value: Double): String =
    if (value % 1.0 == 0.0) value.toInt().toString() else String.format("%.1f", value)

private fun formatOverrideSyncTime(value: Long?): String {
    if (value == null) return "waiting"
    return DateFormat.getTimeInstance(DateFormat.SHORT).format(Date(value))
}
