package com.pegasus.factory.ui.screens.supply

import androidx.compose.foundation.horizontalScroll
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
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.Badge
import androidx.compose.material3.Button
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.CircularProgressIndicator
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
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import com.pegasus.factory.data.model.SupplyRequest
import com.pegasus.factory.data.model.SupplyRequestTransitionRequest
import com.pegasus.factory.data.remote.FactoryApi
import com.pegasus.factory.data.remote.FactoryRealtimeEventType
import com.pegasus.factory.ui.realtime.FactoryRealtimeReloadEffect
import com.pegasus.factory.ui.theme.PegasusSpacing
import java.text.DateFormat
import java.util.Date
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch

private data class RequestActionSpec(
    val label: String,
    val action: String,
    val emphasized: Boolean,
)

private val requestFilters = listOf("ALL", "SUBMITTED", "ACKNOWLEDGED", "IN_PRODUCTION", "READY", "FULFILLED", "CANCELLED")

private fun actionsForState(state: String): List<RequestActionSpec> = when (state) {
    "SUBMITTED" -> listOf(
        RequestActionSpec("Acknowledge", "ACKNOWLEDGE", true),
        RequestActionSpec("Cancel", "CANCEL", false),
    )
    "ACKNOWLEDGED" -> listOf(
        RequestActionSpec("Start production", "START_PRODUCTION", true),
        RequestActionSpec("Cancel", "CANCEL", false),
    )
    "IN_PRODUCTION" -> listOf(
        RequestActionSpec("Mark ready", "MARK_READY", true),
    )
    "READY" -> listOf(
        RequestActionSpec("Fulfill", "FULFILL", true),
    )
    else -> emptyList()
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SupplyRequestsScreen(
    api: FactoryApi,
    onBack: () -> Unit,
) {
    var requests by remember { mutableStateOf<List<SupplyRequest>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var filter by remember { mutableStateOf("ALL") }
    var transitioningId by remember { mutableStateOf<String?>(null) }
    var refreshing by remember { mutableStateOf(false) }
    var staleMessage by remember { mutableStateOf<String?>(null) }
    var lastSyncedAt by remember { mutableStateOf<Long?>(null) }
    val scope = rememberCoroutineScope()
    val snackbarHostState = remember { SnackbarHostState() }
    val lifecycleOwner = LocalLifecycleOwner.current

    fun load(background: Boolean = false) {
        if (background) {
            refreshing = true
        } else if (requests.isEmpty()) {
            loading = true
            error = null
        }

        scope.launch {
            try {
                val resp = api.getSupplyRequests()
                if (resp.isSuccessful && resp.body() != null) {
                    requests = resp.body()!!
                    lastSyncedAt = System.currentTimeMillis()
                    staleMessage = null
                    error = null
                } else {
                    val message = "Failed (${resp.code()})"
                    if (requests.isEmpty()) {
                        error = message
                    } else {
                        staleMessage = "Showing last synced queue. $message"
                    }
                }
            } catch (e: Exception) {
                val message = e.message ?: "Network error"
                if (requests.isEmpty()) {
                    error = message
                } else {
                    staleMessage = "Showing last synced queue. $message"
                }
            } finally {
                loading = false
                refreshing = false
            }
        }
    }

    fun transition(request: SupplyRequest, action: String) {
        transitioningId = request.id
        scope.launch {
            try {
                val resp = api.transitionSupplyRequest(
                    request.id,
                    SupplyRequestTransitionRequest(action = action),
                )
                if (resp.isSuccessful) {
                    snackbarHostState.showSnackbar("${requestLabel(request)} moved to ${resp.body()?.state ?: action}")
                    load(background = true)
                } else {
                    snackbarHostState.showSnackbar("Transition failed (${resp.code()})")
                }
            } catch (e: Exception) {
                snackbarHostState.showSnackbar(e.message ?: "Transition failed")
            } finally {
                transitioningId = null
            }
        }
    }

    LaunchedEffect(Unit) {
        load()
        while (isActive) {
            delay(30_000)
            if (transitioningId == null) {
                load(background = true)
            }
        }
    }

    DisposableEffect(lifecycleOwner) {
        val observer = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_RESUME) {
                load(background = requests.isNotEmpty())
            }
        }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose {
            lifecycleOwner.lifecycle.removeObserver(observer)
        }
    }

    FactoryRealtimeReloadEffect(
        eventTypes = setOf(FactoryRealtimeEventType.SupplyRequestUpdate),
    ) {
        if (transitioningId == null) {
            load(background = requests.isNotEmpty())
        }
    }

    val filteredRequests = if (filter == "ALL") requests else requests.filter { it.state == filter }
    val runtimeStatus = when {
        refreshing -> "Refreshing live queue — last sync ${formatSyncTime(lastSyncedAt)}"
        staleMessage != null -> staleMessage!!
        lastSyncedAt != null -> "Live sync active — last sync ${formatSyncTime(lastSyncedAt)}"
        else -> "Waiting for first sync"
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Supply Requests") },
                navigationIcon = {
                    IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back") }
                },
                actions = {
                    IconButton(onClick = { load(background = requests.isNotEmpty()) }) { Icon(Icons.Default.Refresh, "Refresh") }
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
            filteredRequests.isEmpty() -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                Text(
                    text = if (filter == "ALL") "No supply requests in queue." else "No $filter requests right now.",
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            else -> LazyColumn(
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            ) {
                item {
                    FilterRow(
                        selected = filter,
                        onSelect = { filter = it },
                    )
                }
                item {
                    SupplySummaryCard(
                        total = requests.size,
                        visible = filteredRequests.size,
                        runtimeStatus = runtimeStatus,
                        stale = staleMessage != null,
                    )
                }
                items(filteredRequests, key = { it.id }) { request ->
                    SupplyRequestCard(
                        request = request,
                        transitioning = transitioningId == request.id,
                        onAction = { action -> transition(request, action) },
                    )
                }
            }
        }
    }
}

@Composable
private fun FilterRow(
    selected: String,
    onSelect: (String) -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .horizontalScroll(rememberScrollState()),
        horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
    ) {
        requestFilters.forEach { item ->
            FilterChip(
                selected = selected == item,
                onClick = { onSelect(item) },
                label = { Text(item.replace('_', ' ')) },
            )
        }
    }
}

@Composable
private fun SupplySummaryCard(
    total: Int,
    visible: Int,
    runtimeStatus: String,
    stale: Boolean,
) {
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
                text = "Warehouse demand queue",
                style = MaterialTheme.typography.titleLarge,
            )
            Text(
                text = "$visible requests in view, $total total across the factory queue.",
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
private fun SupplyRequestCard(
    request: SupplyRequest,
    transitioning: Boolean,
    onAction: (String) -> Unit,
) {
    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                verticalAlignment = Alignment.Top,
            ) {
                Column(
                    modifier = Modifier.weight(1f),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
                ) {
                    Text(
                        text = requestLabel(request),
                        style = MaterialTheme.typography.titleMedium,
                    )
                    Text(
                        text = "Request ${request.id.take(8)}",
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                Column(
                    horizontalAlignment = Alignment.End,
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
                ) {
                    RequestTag(
                        text = request.state,
                        containerColor = MaterialTheme.colorScheme.secondaryContainer,
                        contentColor = MaterialTheme.colorScheme.onSecondaryContainer,
                    )
                    RequestTag(
                        text = request.priority.ifBlank { "NORMAL" },
                        containerColor = MaterialTheme.colorScheme.surfaceContainerHighest,
                        contentColor = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }

            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                SupplyMetric("Volume", "${trimDecimal(request.totalVolumeVU)} VU", Modifier.weight(1f))
                SupplyMetric("Created", formatDate(request.createdAt), Modifier.weight(1f))
                SupplyMetric("Delivery", formatDate(request.requestedDeliveryDate), Modifier.weight(1f))
            }

            if (request.notes.isNotBlank()) {
                Surface(
                    modifier = Modifier.fillMaxWidth(),
                    shape = MaterialTheme.shapes.medium,
                    color = MaterialTheme.colorScheme.surfaceContainerLowest,
                ) {
                    Text(
                        text = request.notes,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(PegasusSpacing.md),
                    )
                }
            }

            val actions = actionsForState(request.state)
            if (actions.isEmpty()) {
                Text(
                    text = "No manual action is available for the current state.",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            } else {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                ) {
                    actions.forEach { action ->
                        val buttonModifier = Modifier.weight(1f)
                        if (action.emphasized) {
                            FilledTonalButton(
                                onClick = { onAction(action.action) },
                                enabled = !transitioning,
                                modifier = buttonModifier,
                            ) {
                                Text(if (transitioning) "Working…" else action.label)
                            }
                        } else {
                            Button(
                                onClick = { onAction(action.action) },
                                enabled = !transitioning,
                                modifier = buttonModifier,
                            ) {
                                Text(if (transitioning) "Working…" else action.label)
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun SupplyMetric(
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
            Text(
                text = value,
                style = MaterialTheme.typography.titleSmall,
            )
            Text(
                text = label,
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}

@Composable
private fun RequestTag(
    text: String,
    containerColor: androidx.compose.ui.graphics.Color,
    contentColor: androidx.compose.ui.graphics.Color,
) {
    Surface(
        shape = MaterialTheme.shapes.small,
        color = containerColor,
        contentColor = contentColor,
    ) {
        Text(
            text = text.replace('_', ' '),
            style = MaterialTheme.typography.labelMedium,
            modifier = Modifier.padding(horizontal = PegasusSpacing.sm, vertical = PegasusSpacing.xs),
        )
    }
}

private fun requestLabel(request: SupplyRequest): String =
    request.warehouseId.takeIf { it.isNotBlank() }?.take(8)?.let { "Warehouse $it" } ?: "Warehouse"

private fun formatDate(value: String?): String {
    if (value.isNullOrBlank()) return "Unscheduled"
    return value.substringBefore('T')
}

private fun trimDecimal(value: Double): String =
    if (value % 1.0 == 0.0) value.toInt().toString() else String.format("%.1f", value)

private fun formatSyncTime(value: Long?): String {
    if (value == null) return "waiting"
    return DateFormat.getTimeInstance(DateFormat.SHORT).format(Date(value))
}
