package com.pegasusx.factory.ui.screens.override

import androidx.compose.foundation.lazy.grid.items

import androidx.compose.foundation.lazy.grid.GridItemSpan

import androidx.compose.foundation.lazy.grid.LazyVerticalGrid

import androidx.compose.foundation.lazy.grid.GridCells

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
import com.pegasusx.factory.data.model.Manifest
import com.pegasusx.factory.data.model.ManifestCancelRequest
import com.pegasusx.factory.data.model.ManifestCancelTransferRequest
import com.pegasusx.factory.data.model.ManifestRebalanceRequest
import com.pegasusx.factory.data.model.ManifestTransfer
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.data.remote.FactoryRealtimeEventType
import com.pegasusx.factory.data.remote.FactoryRealtimeStatus
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusRuntimeBanner
import com.pegasus.design.PegasusRuntimeTone
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.factory.ui.realtime.FactoryRealtimeReloadEffect
import com.pegasusx.factory.ui.screens.override.components.CancelManifestDialog
import com.pegasusx.factory.ui.screens.override.components.CancelTransferDialog
import com.pegasusx.factory.ui.screens.override.components.MoveTransferDialog
import com.pegasusx.factory.ui.screens.override.components.PayloadList
import com.pegasusx.factory.util.FactoryIdempotencyKeys
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
    var realtimeStatus by remember { mutableStateOf(FactoryRealtimeStatus.IDLE) }
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
                val transferId = candidate.transfer.transferId
                val resp = api.rebalanceManifest(
                    ManifestRebalanceRequest(
                        sourceManifestId = candidate.sourceManifestId,
                        targetManifestId = targetManifestId,
                        transferIds = listOf(transferId),
                    ),
                    FactoryIdempotencyKeys.rebalance(
                        manifestId = candidate.sourceManifestId,
                        transferId = transferId,
                        targetManifestId = targetManifestId,
                    ),
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
                val transferId = candidate.transfer.transferId
                val resp = api.cancelManifestTransfer(
                    ManifestCancelTransferRequest(
                        manifestId = candidate.manifestId,
                        transferId = transferId,
                    ),
                    FactoryIdempotencyKeys.cancelTransfer(candidate.manifestId, transferId),
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
                val resp = api.cancelManifest(
                    ManifestCancelRequest(manifest.id),
                    FactoryIdempotencyKeys.cancelManifest(manifest.id),
                )
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
        onStatusChange = { status ->
            realtimeStatus = status
        },
    ) {
        if (actingKey == null) {
            load(background = manifests.isNotEmpty())
        }
    }

    val runtimeStatus = when {
        staleMessage != null -> staleMessage!!
        realtimeStatus == FactoryRealtimeStatus.OFFLINE -> "Offline — showing last sync ${formatOverrideSyncTime(lastSyncedAt)}"
        realtimeStatus == FactoryRealtimeStatus.RECONNECTING -> "Reconnecting live manifests — last sync ${formatOverrideSyncTime(lastSyncedAt)}"
        realtimeStatus == FactoryRealtimeStatus.CONNECTING -> "Connecting to live manifests…"
        refreshing -> "Refreshing live manifests — last sync ${formatOverrideSyncTime(lastSyncedAt)}"
        lastSyncedAt != null -> "Live sync active — last sync ${formatOverrideSyncTime(lastSyncedAt)}"
        else -> "Waiting for first sync"
    }
    val runtimeTone = when {
        staleMessage != null && realtimeStatus == FactoryRealtimeStatus.OFFLINE -> PegasusRuntimeTone.Offline
        staleMessage != null -> PegasusRuntimeTone.Warning
        realtimeStatus == FactoryRealtimeStatus.OFFLINE -> PegasusRuntimeTone.Offline
        realtimeStatus == FactoryRealtimeStatus.RECONNECTING || realtimeStatus == FactoryRealtimeStatus.CONNECTING -> PegasusRuntimeTone.Refreshing
        refreshing -> PegasusRuntimeTone.Refreshing
        else -> PegasusRuntimeTone.Live
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
            loading -> PegasusLoadingState(
                title = "Loading payload override",
                body = "Fetching live loading manifests that can be rebalanced or released.",
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            error != null -> PegasusStatePane(
                kind = if (realtimeStatus == FactoryRealtimeStatus.OFFLINE) PegasusStateKind.Offline else PegasusStateKind.Error,
                headline = if (realtimeStatus == FactoryRealtimeStatus.OFFLINE) "Payload override unavailable offline" else "Unable to load loading manifests",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            manifests.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Empty,
                headline = "No manifests are currently loading",
                body = "Payload override becomes available when at least one manifest reaches the loading state.",
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            else -> PayloadList(
                manifests = manifests,
                runtimeStatus = runtimeStatus,
                runtimeTone = runtimeTone,
                actingKey = actingKey,
                onMove = { sourceManifestId, transfer ->
                    moveCandidate = MoveTransferCandidate(sourceManifestId, transfer)
                },
                onRemove = { manifestId, transfer ->
                    cancelTransferCandidate = CancelTransferCandidate(manifestId, transfer)
                },
                onCancelManifest = { manifest ->
                    cancelManifestCandidate = manifest
                },
                modifier = Modifier.padding(innerPadding)
            )
        }
    }

    moveCandidate?.let { candidate ->
        MoveTransferDialog(
            sourceManifestId = candidate.sourceManifestId,
            transfer = candidate.transfer,
            manifests = manifests,
            selectedTargetManifestId = selectedTargetManifestId,
            onTargetSelected = { selectedTargetManifestId = it },
            actingKey = actingKey,
            onConfirm = { targetManifestId -> rebalance(candidate, targetManifestId) },
            onDismiss = { moveCandidate = null }
        )
    }

    cancelTransferCandidate?.let { candidate ->
        CancelTransferDialog(
            transfer = candidate.transfer,
            actingKey = actingKey,
            onConfirm = { cancelTransfer(candidate) },
            onDismiss = { cancelTransferCandidate = null }
        )
    }

    cancelManifestCandidate?.let { manifest ->
        CancelManifestDialog(
            manifest = manifest,
            actingKey = actingKey,
            onConfirm = { cancelManifest(manifest) },
            onDismiss = { cancelManifestCandidate = null }
        )
    }
}

private fun formatOverrideSyncTime(value: Long?): String {
    if (value == null) return "waiting"
    return DateFormat.getTimeInstance(DateFormat.SHORT).format(Date(value))
}
