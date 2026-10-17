package com.pegasusx.factory.ui.screens.payload

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
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
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
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.factory.data.model.Manifest
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.data.remote.FactoryRealtimeEventType
import com.pegasusx.factory.ui.realtime.FactoryRealtimeReloadEffect
import com.pegasusx.factory.ui.screens.manifest.applyManifestLifecycle
import com.pegasusx.factory.ui.screens.manifest.nextManifestLifecycleStep
import com.pegasusx.factory.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PayloadLoadScreen(
    api: FactoryApi,
    onBack: () -> Unit,
) {
    var manifests by remember { mutableStateOf<List<Manifest>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var actingId by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load(silent: Boolean = false) {
        if (!silent) loading = true
        error = null
        scope.launch {
            try {
                val resp = api.getManifests()
                if (resp.isSuccessful && resp.body() != null) {
                    manifests = resp.body()!!.manifests.filter { it.state == "DRAFT" || it.state == "LOADING" }
                } else {
                    error = "Failed (${resp.code()})"
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
            } finally {
                if (!silent) loading = false
            }
        }
    }

    LaunchedEffect(Unit) { load() }
    FactoryRealtimeReloadEffect(
        eventTypes = setOf(FactoryRealtimeEventType.ManifestUpdate, FactoryRealtimeEventType.TransferUpdate),
    ) { load(silent = true) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column {
                        Text("Payload / Load")
                        Text(
                            "Factory start-loading and seal only",
                            style = MaterialTheme.typography.labelMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                },
                navigationIcon = {
                    IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") }
                },
                actions = {
                    IconButton(onClick = { load() }) { Icon(Icons.Default.Refresh, "Refresh") }
                },
            )
        },
    ) { padding ->
        when {
            loading && manifests.isEmpty() -> PegasusLoadingState(
                title = "Loading payloads",
                body = "FactoryTruckManifests drafts and loading bays.",
                modifier = Modifier.fillMaxSize().padding(padding),
            )
            error != null && manifests.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Unable to load payloads",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier.fillMaxSize().padding(padding),
            )
            manifests.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Empty,
                headline = "No factory payloads",
                body = "AUTO dispatch creates drafts. Start loading and seal them here. Last-mile lists are not merged.",
                modifier = Modifier.fillMaxSize().padding(padding),
            )
            else -> LazyColumn(
                modifier = Modifier.fillMaxSize().padding(padding),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                items(manifests, key = { it.id }) { manifest ->
                    val next = nextManifestLifecycleStep(manifest.state)
                    val allowed = next != null && (next.path == "start-loading" || next.path == "seal")
                    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                            Text(manifest.id.take(16), style = MaterialTheme.typography.titleSmall)
                            Text("${manifest.state} · ${manifest.totalVolumeVU.toInt()} VU", style = MaterialTheme.typography.bodySmall)
                            if (allowed && next != null) {
                                Button(
                                    onClick = {
                                        actingId = manifest.id
                                        scope.launch {
                                            try {
                                                val resp = applyManifestLifecycle(api, manifest.id, next)
                                                if (!resp.isSuccessful) {
                                                    error = "Failed (${resp.code()})"
                                                } else {
                                                    load(silent = true)
                                                }
                                            } catch (e: Exception) {
                                                error = e.message
                                            } finally {
                                                actingId = null
                                            }
                                        }
                                    },
                                    enabled = actingId != manifest.id,
                                ) {
                                    Text(if (actingId == manifest.id) "Applying…" else next.label)
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
