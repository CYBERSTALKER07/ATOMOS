package com.pegasusx.factory.ui.screens.override.components

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasus.design.PegasusRuntimeTone
import com.pegasusx.factory.data.model.Manifest
import com.pegasusx.factory.data.model.ManifestTransfer
import com.pegasusx.factory.ui.theme.PegasusSpacing

@Composable
fun PayloadList(
    manifests: List<Manifest>,
    runtimeStatus: String,
    runtimeTone: PegasusRuntimeTone,
    actingKey: String?,
    onMove: (String, ManifestTransfer) -> Unit,
    onRemove: (String, ManifestTransfer) -> Unit,
    onCancelManifest: (Manifest) -> Unit,
    modifier: Modifier = Modifier,
) {
    LazyVerticalGrid(
        columns = GridCells.Adaptive(minSize = 340.dp),
        contentPadding = PaddingValues(PegasusSpacing.lg),
        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        modifier = modifier.fillMaxSize(),
        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md)
    ) {
        item {
            OverrideSummaryCard(
                manifests = manifests,
                runtimeStatus = runtimeStatus,
                runtimeTone = runtimeTone,
            )
        }
        items(manifests, key = { it.id }) { manifest ->
            OverrideManifestCard(
                manifest = manifest,
                hasMoveTargets = manifests.any { it.id != manifest.id },
                actingKey = actingKey,
                onMove = { transfer -> onMove(manifest.id, transfer) },
                onRemove = { transfer -> onRemove(manifest.id, transfer) },
                onCancelManifest = { onCancelManifest(manifest) },
            )
        }
    }
}
