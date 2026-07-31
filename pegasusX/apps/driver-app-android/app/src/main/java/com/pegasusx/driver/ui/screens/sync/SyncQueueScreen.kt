package com.pegasusx.driver.ui.screens.sync

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
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import com.pegasusx.driver.data.model.PendingMutationEntity
import com.pegasusx.driver.ui.theme.PegasusSpacing

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SyncQueueScreen(
    onBack: (() -> Unit)? = null,
    viewModel: SyncQueueViewModel = hiltViewModel(),
) {
    val pending by viewModel.pending.collectAsState()
    val dead by viewModel.dead.collectAsState()
    val flushing by viewModel.flushing.collectAsState()
    val status by viewModel.statusMessage.collectAsState()

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Sync Queue") },
                navigationIcon = {
                    if (onBack != null) {
                        IconButton(onClick = onBack) {
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back")
                        }
                    }
                },
                actions = {
                    IconButton(onClick = { viewModel.flushNow() }, enabled = !flushing) {
                        Icon(Icons.Default.Refresh, "Flush")
                    }
                },
            )
        },
    ) { inner ->
        LazyColumn(
            modifier = Modifier.fillMaxSize().padding(inner),
            contentPadding = PaddingValues(PegasusSpacing.s16),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.s12),
        ) {
            item {
                Row(
                    Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        "${pending.size} pending · ${dead.size} dead-letter",
                        style = MaterialTheme.typography.titleSmall,
                    )
                    Button(onClick = { viewModel.flushNow() }, enabled = !flushing && pending.isNotEmpty()) {
                        Text(if (flushing) "Flushing…" else "Flush now")
                    }
                }
                status?.let {
                    Text(it, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.primary)
                }
            }
            item {
                Text("Pending", style = MaterialTheme.typography.titleMedium)
            }
            if (pending.isEmpty()) {
                item {
                    Text("No pending offline actions.", color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            } else {
                items(pending, key = { it.id }) { row ->
                    MutationCard(row)
                }
            }
            item {
                Spacer(Modifier.height(8.dp))
                Row(
                    Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text("Dead letter", style = MaterialTheme.typography.titleMedium)
                    if (dead.isNotEmpty()) {
                        TextButton(onClick = { viewModel.dismissDead() }) { Text("Dismiss all") }
                    }
                }
            }
            if (dead.isEmpty()) {
                item {
                    Text("No dead-lettered actions.", color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            } else {
                items(dead, key = { it.id }) { row ->
                    MutationCard(row, dead = true)
                }
            }
        }
    }
}

@Composable
private fun MutationCard(row: PendingMutationEntity, dead: Boolean = false) {
    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
        Column(Modifier.padding(PegasusSpacing.s16)) {
            Text(row.endpoint, style = MaterialTheme.typography.titleSmall)
            Text(
                "Order ${row.orderId.ifBlank { "—" }} · attempts ${row.attemptCount}",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            if (row.clientTimestampIso.isNotBlank()) {
                Text(
                    "client_ts ${row.clientTimestampIso}",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            if (row.lastError.isNotBlank()) {
                Text(
                    row.lastError,
                    style = MaterialTheme.typography.bodySmall,
                    color = if (dead) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}
