package com.pegasusx.factory.ui.screens.manifest

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import com.pegasusx.factory.data.model.ManifestDetailResponse
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.data.remote.FactoryRealtimeEventType
import com.pegasusx.factory.ui.components.FactoryLoadingState
import com.pegasusx.factory.ui.components.FactoryStateKind
import com.pegasusx.factory.ui.components.FactoryStatePane
import com.pegasusx.factory.ui.realtime.FactoryRealtimeReloadEffect
import com.pegasusx.factory.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ManifestDetailScreen(
    api: FactoryApi,
    manifestId: String,
    onBack: () -> Unit,
) {
    var detail by remember { mutableStateOf<ManifestDetailResponse?>(null) }
    var loading by remember { mutableStateOf(true) }
    var acting by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val snackbarHostState = remember { SnackbarHostState() }
    val scope = rememberCoroutineScope()

    fun load() {
        loading = true
        error = null
        scope.launch {
            try {
                val resp = api.getManifestDetail(manifestId)
                if (resp.isSuccessful && resp.body() != null) {
                    detail = resp.body()!!
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

    fun runLifecycle() {
        val current = detail ?: return
        val step = nextManifestLifecycleStep(current.manifest.state) ?: return
        acting = true
        scope.launch {
            try {
                val resp = applyManifestLifecycle(api, manifestId, step)
                if (resp.isSuccessful) {
                    snackbarHostState.showSnackbar("${step.label} applied")
                    load()
                } else {
                    snackbarHostState.showSnackbar("Failed (${resp.code()})")
                }
            } catch (e: Exception) {
                snackbarHostState.showSnackbar(e.message ?: "Transition failed")
            } finally {
                acting = false
            }
        }
    }

    LaunchedEffect(manifestId) { load() }

    FactoryRealtimeReloadEffect(
        api = api,
        eventTypes = setOf(FactoryRealtimeEventType.ManifestUpdate),
        onEvent = { load() },
        onReconnect = {
            acting = false
            load()
        },
    )

    Scaffold(
        snackbarHost = { SnackbarHost(snackbarHostState) },
        topBar = {
            TopAppBar(
                title = { Text("Manifest") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back")
                    }
                },
                actions = {
                    IconButton(onClick = { load() }) {
                        Icon(Icons.Default.Refresh, "Refresh")
                    }
                },
            )
        },
    ) { innerPadding ->
        when {
            loading -> FactoryLoadingState(
                title = "Loading manifest",
                body = "Fetching manifest detail and lifecycle history.",
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            )
            error != null -> FactoryStatePane(
                kind = FactoryStateKind.Error,
                headline = "Unable to load manifest",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            )
            detail != null -> {
                val manifest = detail!!.manifest
                val next = nextManifestLifecycleStep(manifest.state)
                LazyColumn(
                    contentPadding = PaddingValues(PegasusSpacing.lg),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                    modifier = Modifier.fillMaxSize().padding(innerPadding),
                ) {
                    item {
                        Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                            Text(manifest.id, style = MaterialTheme.typography.titleMedium, fontFamily = FontFamily.Monospace)
                            Text("State: ${manifest.state}", style = MaterialTheme.typography.bodyLarge)
                            Text(
                                "Route ${detail!!.routeId.ifBlank { "—" }} · ${detail!!.orderCount} orders",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                    }
                    if (next != null) {
                        item {
                            Button(
                                onClick = { runLifecycle() },
                                enabled = !acting,
                                modifier = Modifier.fillMaxWidth(),
                            ) {
                                Text(if (acting) "Applying…" else next.label)
                            }
                        }
                    }
                    item {
                        Text("Transfers", style = MaterialTheme.typography.titleSmall)
                    }
                    if (detail!!.transfers.isEmpty()) {
                        item {
                            Text("No transfers on this manifest.", style = MaterialTheme.typography.bodySmall)
                        }
                    } else {
                        items(detail!!.transfers, key = { it.transferId }) { transfer ->
                            Text(
                                "${transfer.transferId} · ${transfer.state}",
                                style = MaterialTheme.typography.bodySmall,
                                fontFamily = FontFamily.Monospace,
                            )
                        }
                    }
                    item {
                        Text("Transitions", style = MaterialTheme.typography.titleSmall)
                    }
                    if (detail!!.transitions.isEmpty()) {
                        item {
                            Text("No transitions recorded.", style = MaterialTheme.typography.bodySmall)
                        }
                    } else {
                        items(detail!!.transitions, key = { "${it.action}-${it.at}" }) { transition ->
                            Text(
                                "${transition.action}: ${transition.fromState} → ${transition.toState}",
                                style = MaterialTheme.typography.bodySmall,
                            )
                        }
                    }
                }
            }
        }
    }
}
