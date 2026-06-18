package com.pegasusx.factory.ui.screens.exceptions

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
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
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
import com.pegasusx.factory.data.model.ManifestException
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.data.remote.FactoryRealtimeEventType
import com.pegasusx.factory.ui.components.FactoryLoadingState
import com.pegasusx.factory.ui.components.FactoryStateKind
import com.pegasusx.factory.ui.components.FactoryStatePane
import com.pegasusx.factory.ui.realtime.FactoryRealtimeReloadEffect
import com.pegasusx.factory.ui.theme.PegasusSpacing
import java.text.DateFormat
import java.util.Date
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ManifestExceptionsScreen(
    api: FactoryApi,
    onBack: () -> Unit,
) {
    var exceptions by remember { mutableStateOf<List<ManifestException>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var escalatedOnly by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()

    fun load(silent: Boolean = false) {
        if (!silent) {
            loading = true
        }
        error = null
        scope.launch {
            try {
                val resp = api.getManifestExceptions(
                    escalated = if (escalatedOnly) "true" else null,
                )
                if (resp.isSuccessful && resp.body() != null) {
                    exceptions = resp.body()!!.exceptions
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

    LaunchedEffect(escalatedOnly) { load() }

    FactoryRealtimeReloadEffect(
        eventTypes = setOf(
            FactoryRealtimeEventType.ManifestUpdate,
            FactoryRealtimeEventType.TransferUpdate,
        ),
    ) {
        load(silent = true)
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Loading Gate Exceptions") },
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
            loading && exceptions.isEmpty() -> FactoryLoadingState(
                title = "Loading exceptions",
                body = "Fetching transfers removed from manifests during loading.",
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            error != null -> FactoryStatePane(
                kind = FactoryStateKind.Error,
                headline = "Unable to load exceptions",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            else -> LazyColumn(
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            ) {
                item {
                    Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                        FilterChip(
                            selected = !escalatedOnly,
                            onClick = { escalatedOnly = false },
                            label = { Text("All") },
                        )
                        FilterChip(
                            selected = escalatedOnly,
                            onClick = { escalatedOnly = true },
                            label = { Text("Escalated only") },
                        )
                    }
                }
                if (exceptions.isEmpty()) {
                    item {
                        FactoryStatePane(
                            kind = FactoryStateKind.Empty,
                            headline = if (escalatedOnly) "No escalated exceptions" else "No exceptions",
                            body = if (escalatedOnly) {
                                "No transfers have hit the DLQ threshold (3+ overflows)."
                            } else {
                                "All manifest loading operations completed without exceptions."
                            },
                            modifier = Modifier.fillMaxWidth(),
                        )
                    }
                } else {
                    items(exceptions, key = { it.exceptionId }) { exception ->
                        ExceptionCard(exception = exception)
                    }
                }
            }
        }
    }
}

@Composable
private fun ExceptionCard(exception: ManifestException) {
    val isDlq = exception.attemptCount >= 3
    ElevatedCard(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.elevatedCardColors(
            containerColor = if (isDlq) {
                MaterialTheme.colorScheme.errorContainer
            } else {
                MaterialTheme.colorScheme.surfaceContainer
            },
        ),
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
            ) {
                Text(exception.reason, style = MaterialTheme.typography.titleSmall)
                if (exception.escalated) {
                    Surface(
                        shape = MaterialTheme.shapes.small,
                        color = MaterialTheme.colorScheme.error,
                    ) {
                        Text(
                            text = "Escalated",
                            modifier = Modifier.padding(horizontal = PegasusSpacing.sm, vertical = PegasusSpacing.xs),
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.onError,
                        )
                    }
                }
            }
            Text(
                text = "Transfer ${shortId(exception.transferId)} · Manifest ${shortId(exception.manifestId)}",
                style = MaterialTheme.typography.bodyMedium,
                fontFamily = FontFamily.Monospace,
            )
            Text(
                text = buildString {
                    append("Attempts: ${exception.attemptCount}")
                    if (isDlq) append(" — DLQ")
                },
                style = MaterialTheme.typography.bodySmall,
                color = if (isDlq) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Text(
                text = formatTimestamp(exception.createdAt),
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}

private fun shortId(id: String): String = if (id.length > 12) "${id.take(8)}…" else id

private fun formatTimestamp(raw: String): String {
    if (raw.isBlank()) return "—"
    return runCatching {
        DateFormat.getDateTimeInstance(DateFormat.SHORT, DateFormat.SHORT).format(Date.from(java.time.Instant.parse(raw)))
    }.getOrDefault(raw)
}
