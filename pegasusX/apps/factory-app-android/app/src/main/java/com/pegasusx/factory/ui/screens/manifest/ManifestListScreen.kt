package com.pegasusx.factory.ui.screens.manifest

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
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
import androidx.compose.ui.text.font.FontFamily
import com.pegasusx.factory.data.model.Manifest
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
fun ManifestListScreen(
    api: FactoryApi,
    onManifestClick: (String) -> Unit,
    onBack: () -> Unit,
) {
    var manifests by remember { mutableStateOf<List<Manifest>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        loading = true
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
                loading = false
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    FactoryRealtimeReloadEffect(
        eventTypes = setOf(
            FactoryRealtimeEventType.ManifestUpdate,
            FactoryRealtimeEventType.TransferUpdate,
        ),
    ) {
        load()
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Manifests") },
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
                title = "Loading manifests",
                body = "Fetching manifest pipeline for the loading gate.",
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            )
            error != null -> FactoryStatePane(
                kind = FactoryStateKind.Error,
                headline = "Unable to load manifests",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            )
            manifests.isEmpty() -> FactoryStatePane(
                kind = FactoryStateKind.Empty,
                headline = "No manifests",
                body = "Dispatch transfers to create a manifest draft.",
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            )
            else -> LazyColumn(
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            ) {
                items(manifests, key = { it.id }) { manifest ->
                    val next = nextManifestLifecycleStep(manifest.state)
                    ElevatedCard(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clickable { onManifestClick(manifest.id) },
                        colors = CardDefaults.elevatedCardColors(
                            containerColor = MaterialTheme.colorScheme.surfaceContainer,
                        ),
                    ) {
                        Column(
                            modifier = Modifier.padding(PegasusSpacing.lg),
                            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
                        ) {
                            Text(manifest.id, style = MaterialTheme.typography.titleSmall, fontFamily = FontFamily.Monospace)
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                            ) {
                                Text(manifest.state, style = MaterialTheme.typography.labelLarge)
                                Text("${manifest.totalVolumeVU.toInt()} VU", style = MaterialTheme.typography.bodySmall)
                            }
                            Text(
                                text = next?.label ?: "Terminal state",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                    }
                }
            }
        }
    }
}
