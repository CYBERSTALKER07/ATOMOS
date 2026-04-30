package com.pegasus.factory.ui.screens.dashboard

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ExitToApp
import androidx.compose.material.icons.automirrored.filled.List
import androidx.compose.material.icons.filled.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.unit.dp
import com.pegasus.factory.data.model.DashboardStats
import com.pegasus.factory.data.remote.FactoryApi
import com.pegasus.factory.ui.navigation.FactoryRoutes
import com.pegasus.factory.ui.theme.LabSpacing
import kotlinx.coroutines.launch

private data class KpiCard(
    val label: String,
    val icon: ImageVector,
    val route: String,
    val value: (DashboardStats) -> String,
)

private val kpiCards = listOf(
    KpiCard("Pending Transfers", Icons.Default.MoveToInbox, FactoryRoutes.TRANSFERS) { it.pendingTransfers.toString() },
    KpiCard("Now Loading", Icons.Default.LocalShipping, FactoryRoutes.LOADING_BAY) { it.loadingTransfers.toString() },
    KpiCard("Active Manifests", Icons.AutoMirrored.Filled.List, FactoryRoutes.LOADING_BAY) { it.activeManifests.toString() },
    KpiCard("Dispatched Today", Icons.Default.CheckCircle, FactoryRoutes.TRANSFERS) { it.dispatchedToday.toString() },
    KpiCard("Vehicles Total", Icons.Default.DirectionsCar, FactoryRoutes.FLEET) { it.vehiclesTotal.toString() },
    KpiCard("Available", Icons.Default.DirectionsCar, FactoryRoutes.FLEET) { it.vehiclesAvailable.toString() },
    KpiCard("Staff on Shift", Icons.Default.People, FactoryRoutes.STAFF) { it.staffOnShift.toString() },
    KpiCard("Critical Insights", Icons.Default.Warning, FactoryRoutes.INSIGHTS) { it.criticalInsights.toString() },
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DashboardScreen(
    api: FactoryApi,
    onNavigate: (String) -> Unit,
    onSignOut: () -> Unit,
) {
    var stats by remember { mutableStateOf(DashboardStats()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        loading = true
        error = null
        scope.launch {
            try {
                val resp = api.getDashboard()
                if (resp.isSuccessful && resp.body() != null) {
                    stats = resp.body()!!
                } else {
                    error = "Failed to load (${resp.code()})"
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Dashboard") },
                actions = {
                    IconButton(onClick = { load() }) {
                        Icon(Icons.Default.Refresh, "Refresh")
                    }
                    IconButton(onClick = onSignOut) {
                        Icon(Icons.AutoMirrored.Filled.ExitToApp, "Sign out")
                    }
                },
            )
        },
    ) { innerPadding ->
        when {
            loading -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                CircularProgressIndicator()
            }
            error != null -> Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(error!!, color = MaterialTheme.colorScheme.error)
                    Spacer(Modifier.height(LabSpacing.lg))
                    Button(onClick = { load() }) { Text("Retry") }
                }
            }
            else -> LazyVerticalGrid(
                columns = GridCells.Fixed(2),
                contentPadding = PaddingValues(LabSpacing.lg),
                horizontalArrangement = Arrangement.spacedBy(LabSpacing.md),
                verticalArrangement = Arrangement.spacedBy(LabSpacing.md),
                modifier = Modifier.fillMaxSize().padding(innerPadding),
            ) {
                items(kpiCards.size) { index ->
                    val card = kpiCards[index]
                    ElevatedCard(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clickable { onNavigate(card.route) },
                    ) {
                        Column(
                            modifier = Modifier.padding(LabSpacing.lg),
                        ) {
                            Icon(
                                imageVector = card.icon,
                                contentDescription = null,
                                tint = MaterialTheme.colorScheme.onSurfaceVariant,
                                modifier = Modifier.size(24.dp),
                            )
                            Spacer(Modifier.height(LabSpacing.md))
                            Text(
                                text = card.value(stats),
                                style = MaterialTheme.typography.headlineMedium,
                            )
                            Spacer(Modifier.height(LabSpacing.xs))
                            Text(
                                text = card.label,
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
