package com.pegasusx.factory.ui.screens.staff

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.factory.data.model.StaffMember
import com.pegasusx.factory.data.remote.FactoryApi
import com.pegasusx.factory.data.remote.FactoryRealtimeEventType
import com.pegasusx.factory.ui.components.FactoryLoadingState
import com.pegasusx.factory.ui.components.FactoryMetricTile
import com.pegasusx.factory.ui.components.FactoryOpsListCard
import com.pegasusx.factory.ui.components.FactorySectionTitle
import com.pegasusx.factory.ui.components.FactoryStateKind
import com.pegasusx.factory.ui.components.FactoryStatePane
import com.pegasusx.factory.ui.realtime.FactoryRealtimeReloadEffect
import com.pegasusx.factory.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun StaffScreen(
    api: FactoryApi,
    onStaffClick: (String) -> Unit,
    onBack: () -> Unit,
) {
    var staff by remember { mutableStateOf<List<StaffMember>>(emptyList()) }
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
                val resp = api.getStaff()
                if (resp.isSuccessful && resp.body() != null) {
                    staff = resp.body()!!.staff
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
        eventTypes = setOf(
            FactoryRealtimeEventType.SupplyRequestUpdate,
            FactoryRealtimeEventType.TransferUpdate,
            FactoryRealtimeEventType.ManifestUpdate,
        ),
    ) {
        load(silent = true)
    }

    val onShift = staff.count { it.status.equals("ON_SHIFT", ignoreCase = true) || it.status.equals("ACTIVE", ignoreCase = true) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                        Text("Staff")
                        Text(
                            text = "Operator roster and shift status",
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
    ) { innerPadding ->
        when {
            loading && staff.isEmpty() -> FactoryLoadingState(
                title = "Loading staff",
                body = "Fetching the current factory operator roster.",
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            error != null -> FactoryStatePane(
                kind = FactoryStateKind.Error,
                headline = "Unable to load staff",
                body = error!!,
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            staff.isEmpty() -> FactoryStatePane(
                kind = FactoryStateKind.Empty,
                headline = "No staff on record",
                body = "There are no staff members registered for this factory yet.",
                modifier = Modifier
                    .fillMaxSize()
                    .padding(innerPadding),
            )
            else -> LazyColumn(
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            ) {
                item {
                    StaffSummaryCard(
                        total = staff.size,
                        onShift = onShift,
                    )
                }
                item {
                    FactorySectionTitle(title = "Operator roster")
                }
                items(staff, key = { it.id }) { member ->
                    FactoryOpsListCard(
                        headline = member.name,
                        supporting = "${member.role} · ${member.phone}",
                        status = member.status,
                        onClick = { onStaffClick(member.id) },
                    )
                }
            }
        }
    }
}

@Composable
private fun StaffSummaryCard(
    total: Int,
    onShift: Int,
) {
    ElevatedCard(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.elevatedCardColors(
            containerColor = MaterialTheme.colorScheme.surfaceContainerHigh,
        ),
    ) {
        Column(
            modifier = Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            Text(
                text = "Staffing snapshot",
                style = MaterialTheme.typography.titleLarge,
            )
            Text(
                text = "Operators currently registered and active on the factory floor.",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                FactoryMetricTile("Total", total.toString(), Modifier.weight(1f))
                FactoryMetricTile("On shift", onShift.toString(), Modifier.weight(1f))
                FactoryMetricTile("Off shift", (total - onShift).toString(), Modifier.weight(1f))
            }
        }
    }
}
