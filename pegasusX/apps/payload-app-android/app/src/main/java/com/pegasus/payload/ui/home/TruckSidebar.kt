package com.pegasus.payload.ui.home

import androidx.compose.ui.res.stringResource

import androidx.compose.animation.animateColorAsState
import com.pegasus.payload.ui.components.PayloadSectionTitle
import com.pegasus.payload.ui.components.ExplainStatusBanner
import androidx.compose.animation.core.Spring
import androidx.compose.animation.core.spring
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.LocalShipping
import androidx.compose.material.icons.filled.Menu
import androidx.compose.material.icons.filled.MenuOpen
import androidx.compose.material3.Button
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.pegasus.payload.data.model.Truck
import com.pegasus.payload.data.model.SealCompletedManifestResult
import com.pegasus.design.PegasusStatePane
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusLoadingState
import com.pegasus.payload.R

@Composable
fun TruckListPane(
    trucks: List<Truck>,
    selectedTruckId: String?,
    loading: Boolean,
    error: String?,
    batchReadyCount: Int,
    batchSealing: Boolean,
    batchSealFailures: List<SealCompletedManifestResult> = emptyList(),
    isExpanded: Boolean,
    onToggleExpanded: () -> Unit,
    onFinalizeBatch: () -> Unit,
    onSelect: (String) -> Unit,
) {
    Surface(
        color = MaterialTheme.colorScheme.surface,
        modifier = Modifier
            .width(if (isExpanded) 320.dp else 72.dp)
            .fillMaxHeight(),
    ) {
        Column(Modifier.fillMaxSize()) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 12.dp, vertical = 12.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                if (isExpanded) {
                    PayloadSectionTitle(
                        title = "Manifest board",
                        subtitle = "DRAFT · LOADING · SEALED · DISPATCHED",
                        modifier = Modifier.weight(1f),
                    )
                }
                IconButton(onClick = onToggleExpanded) {
                    Icon(
                        if (isExpanded) Icons.Default.MenuOpen else Icons.Default.Menu,
                        contentDescription = if (isExpanded) "Collapse truck list" else "Expand truck list",
                    )
                }
            }
            if (!isExpanded) {
                LazyVerticalGrid(
                    columns = GridCells.Adaptive(minSize = 340.dp),
                    contentPadding = PaddingValues(vertical = 8.dp),
                ) {
                    items(trucks, key = { it.id }) { truck ->
                        val selected = truck.id == selectedTruckId
                        IconButton(onClick = { onSelect(truck.id) }) {
                            Icon(
                                Icons.Default.LocalShipping,
                                contentDescription = truck.label.ifBlank { truck.licensePlate },
                                tint = if (selected) {
                                    MaterialTheme.colorScheme.onPrimaryContainer
                                } else {
                                    MaterialTheme.colorScheme.onSurfaceVariant
                                },
                            )
                        }
                    }
                }
                return@Column
            }
            if (loading) LinearProgressIndicator(Modifier.fillMaxWidth())
            if (batchReadyCount > 1) {
                ElevatedCard(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 12.dp, vertical = 8.dp),
                ) {
                    Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text(
                            stringResource(R.string.mobile_payload_ui_batchreadycount_trucks_ready_to_finalize, batchReadyCount),
                            style = MaterialTheme.typography.titleSmall,
                        )
                        Button(
                            onClick = onFinalizeBatch,
                            enabled = !batchSealing,
                            modifier = Modifier.fillMaxWidth(),
                        ) {
                            Text(if (batchSealing) "Finalizing…" else "Seal all trucks")
                        }
                        batchSealFailures.forEach { row ->
                            Column(
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .padding(top = 4.dp),
                            ) {
                                Text(stringResource(R.string.mobile_payload_ui_manifestid_status, row.manifestId, row.status),
                                    style = MaterialTheme.typography.labelMedium,
                                )
                                row.explain?.let { explain ->
                                    ExplainStatusBanner(
                                        explain = explain,
                                        fallbackTitle = row.status,
                                        modifier = Modifier.padding(top = 4.dp),
                                    )
                                }
                            }
                        }
                    }
                }
            }
            if (error != null) {
                Text(
                    error,
                    color = MaterialTheme.colorScheme.error,
                    style = MaterialTheme.typography.bodySmall,
                    modifier = Modifier.padding(horizontal = 20.dp, vertical = 8.dp),
                )
            }
            if (loading && trucks.isEmpty() && error == null) {
                PegasusLoadingState(
                    title = stringResource(R.string.mobile_payload_ui_loading_vehicles),
                    body = "Refreshing supplier fleet availability for this shift.",
                )
            } else if (!loading && trucks.isEmpty() && error == null) {
                PegasusStatePane(
                    kind = PegasusStateKind.Empty,
                    headline = "No vehicles available",
                    body = "Pull to refresh once dispatch assigns trucks.",
                )
            }
            val columns = ManifestBoard.group(trucks)
            val unassigned = ManifestBoard.unassigned(trucks)
            LazyVerticalGrid(
                columns = GridCells.Adaptive(minSize = 340.dp),
                contentPadding = PaddingValues(horizontal = 12.dp, vertical = 4.dp)
            ) {
                columns.forEach { column ->
                    item(key = "col-${column.state}", span = { GridItemSpan(maxLineSpan) }) {
                        Text(
                            text = "${column.state} · ${if (column.trucks.isEmpty()) "empty" else column.trucks.size.toString()}",
                            style = MaterialTheme.typography.labelMedium,
                            fontWeight = FontWeight.Bold,
                            modifier = Modifier.padding(top = 10.dp, bottom = 4.dp),
                        )
                    }
                    if (column.trucks.isEmpty()) {
                        item(key = "empty-${column.state}", span = { GridItemSpan(maxLineSpan) }) {
                            Text(
                                "No ${column.state.lowercase()} manifests",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                modifier = Modifier.padding(bottom = 6.dp),
                            )
                        }
                    } else {
                        items(column.trucks, key = { "${column.state}-${it.id}" }) { truck ->
                            TruckRow(truck, selected = truck.id == selectedTruckId, onClick = { onSelect(truck.id) })
                            Spacer(Modifier.height(6.dp))
                        }
                    }
                }
                if (unassigned.isNotEmpty()) {
                    item(key = "unassigned-head", span = { GridItemSpan(maxLineSpan) }) {
                        Text(
                            "No open manifest",
                            style = MaterialTheme.typography.labelMedium,
                            modifier = Modifier.padding(top = 10.dp, bottom = 4.dp),
                        )
                    }
                    items(unassigned, key = { "none-${it.id}" }) { truck ->
                        TruckRow(truck, selected = truck.id == selectedTruckId, onClick = { onSelect(truck.id) })
                    }
                }
            }
        }
    }
}

@Composable
fun TruckRow(truck: Truck, selected: Boolean, onClick: () -> Unit) {
    val bgColor by animateColorAsState(
        targetValue = if (selected) MaterialTheme.colorScheme.primaryContainer else MaterialTheme.colorScheme.surfaceContainerHigh,
        animationSpec = spring(dampingRatio = Spring.DampingRatioMediumBouncy, stiffness = Spring.StiffnessMedium),
        label = "truck-bg",
    )
    val fgColor by animateColorAsState(
        targetValue = if (selected) MaterialTheme.colorScheme.onPrimaryContainer else MaterialTheme.colorScheme.onSurface,
        label = "truck-fg",
    )
    Surface(
        color = bgColor,
        contentColor = fgColor,
        shape = RoundedCornerShape(14.dp),
        modifier = Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(14.dp))
            .clickable(onClick = onClick),
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.padding(14.dp),
        ) {
            Icon(Icons.Filled.LocalShipping, contentDescription = null)
            Spacer(Modifier.size(12.dp))
            Column(Modifier.fillMaxWidth()) {
                Text(
                    truck.label.ifBlank { truck.licensePlate.ifBlank { truck.id.take(8) } },
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Medium,
                )
                Text(
                    listOfNotNull(
                        truck.licensePlate.takeIf { it.isNotBlank() },
                        truck.vehicleClass.takeIf { it.isNotBlank() },
                        ManifestBoard.canonicalState(truck.truckStatus).takeIf { it.isNotEmpty() },
                        if (truck.maxVolumeVu > 0) "${truck.usedVolumeVu}/${truck.maxVolumeVu} VU" else null,
                    ).joinToString(" • "),
                    style = MaterialTheme.typography.bodySmall,
                )
            }
        }
    }
}
