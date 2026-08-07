package com.pegasusx.factory.ui.screens.manifest

import androidx.compose.ui.res.stringResource

import androidx.compose.ui.unit.dp

import androidx.compose.foundation.lazy.grid.items

import androidx.compose.foundation.lazy.grid.GridItemSpan

import androidx.compose.foundation.lazy.grid.LazyVerticalGrid

import androidx.compose.foundation.lazy.grid.GridCells

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.CardDefaults
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
import com.pegasusx.factory.data.model.Manifest
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.data.remote.FactoryRealtimeEventType
import com.pegasus.design.PegasusLoadingState
import com.pegasusx.factory.ui.components.FactoryOpsListCard
import com.pegasusx.factory.ui.components.FactorySectionTitle
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.factory.ui.realtime.FactoryRealtimeReloadEffect
import com.pegasusx.factory.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ManifestListScreen(
    api: FactoryApi,
    onManifestClick: (String) -> Unit,
    onBack: () -> Unit,
) {
    var manifests by remember { mutableStateOf<List<Manifest>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load(silent: Boolean = false) {
        if (!silent) {
            loading = true
        }
        error = null
        scope.launch {
            try {
                val resp = api.getManifests()
                if (resp.isSuccessful && resp.body() != null) {
                    manifests = resp.body()!!.manifests
                } else {
                    error = "Failed (${resp.code()})"
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
            } finally {
                if (!silent) {
                    loading = false
                }
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    FactoryRealtimeReloadEffect(
        api = api,
        eventTypes = setOf(
            FactoryRealtimeEventType.ManifestUpdate,
            FactoryRealtimeEventType.TransferUpdate,
        ),
        onEvent = { load(silent = true) },
        onReconnect = { load(silent = true) },
    )

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                        Text("Manifests")
                        Text(
                            text = stringResource(R.string.mobile_factory_ui_draft_through_dispatch_lifecycle),
                            style = MaterialTheme.typography.labelMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                },
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
            loading && manifests.isEmpty() -> PegasusLoadingState(
                title = stringResource(R.string.mobile_factory_ui_loading_manifests),
                body = "Fetching manifest pipeline for the loading gate.",
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            )
            error != null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Unable to load manifests",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            )
            manifests.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Empty,
                headline = "No manifests",
                body = "Dispatch transfers to create a manifest draft.",
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            )
            else -> LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md)
    ) {
                item {
                    ManifestListSummary(count = manifests.size)
                }
                item {
                    FactorySectionTitle(title = stringResource(R.string.factory_portal_manifests_text_manifest_pipeline))
                }
                items(manifests, key = { it.id }) { manifest ->
                    val next = nextManifestLifecycleStep(manifest.state)
                    FactoryOpsListCard(
                        headline = manifest.id.take(12),
                        supporting = buildString {
                            append("${manifest.totalVolumeVU.toInt()} VU")
                            next?.let { append(" · Next: ${it.label}") } ?: append(" · Terminal state")
                        },
                        status = manifest.state,
                        onClick = { onManifestClick(manifest.id) },
                    )
                }
            }
        }
    }
}

@Composable
private fun ManifestListSummary(count: Int) {
    ElevatedCard(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.elevatedCardColors(
            containerColor = MaterialTheme.colorScheme.surfaceContainerHigh,
        ),
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
        ) {
            Text(
                text = stringResource(R.string.mobile_factory_ui_count_manifests_in_pipeline),
                style = MaterialTheme.typography.titleLarge,
            )
            Text(
                text = stringResource(R.string.mobile_factory_ui_advance_manifests_through_draft_loading_sealed_dispatched_and_co),
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}
