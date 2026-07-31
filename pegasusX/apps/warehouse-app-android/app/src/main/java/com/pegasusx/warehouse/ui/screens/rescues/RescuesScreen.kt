package com.pegasusx.warehouse.ui.screens.rescues

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
import androidx.compose.material3.CircularProgressIndicator
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
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import com.pegasusx.warehouse.data.model.AvailableDriver
import com.pegasusx.warehouse.data.model.RescueOption
import com.pegasusx.warehouse.data.model.RescuePreviewRequest
import com.pegasusx.warehouse.data.model.RescueProposeRequest
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import com.pegasusx.warehouse.util.WarehouseIdempotencyKeys
import java.util.UUID
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RescuesScreen(
    api: WarehouseApi,
    onBack: (() -> Unit)? = null,
) {
    var brokenDrivers by remember { mutableStateOf<List<AvailableDriver>>(emptyList()) }
    var selectedBroken by remember { mutableStateOf<AvailableDriver?>(null) }
    var rescueOptions by remember { mutableStateOf<List<RescueOption>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var previewLoading by remember { mutableStateOf(false) }
    var proposeLoading by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var statusMessage by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun loadDrivers() {
        loading = true
        error = null
        scope.launch {
            try {
                val resp = api.getDispatchPreview()
                if (resp.isSuccessful && resp.body() != null) {
                    val body = resp.body()!!
                    brokenDrivers = (body.availableDrivers + body.unavailableDrivers)
                        .filter { it.truckStatus.equals("NEEDS_RESCUE", ignoreCase = true) }
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

    fun findRescue(driver: AvailableDriver) {
        selectedBroken = driver
        rescueOptions = emptyList()
        previewLoading = true
        statusMessage = null
        scope.launch {
            try {
                val resp = api.previewRescue(RescuePreviewRequest(brokenDriverId = driver.driverId))
                if (resp.isSuccessful && resp.body() != null) {
                    rescueOptions = resp.body()!!.rescueOptions
                } else {
                    statusMessage = "Preview failed (${resp.code()})"
                    selectedBroken = null
                }
            } catch (e: Exception) {
                statusMessage = e.message
                selectedBroken = null
            } finally {
                previewLoading = false
            }
        }
    }

    fun propose(option: RescueOption) {
        val broken = selectedBroken ?: return
        proposeLoading = true
        scope.launch {
            try {
                val rescueId = UUID.randomUUID().toString()
                val key = WarehouseIdempotencyKeys.rescuePropose(
                    rescueId = rescueId,
                    brokenDriverId = broken.driverId,
                    rescueDriverId = option.driverId,
                )
                val resp = api.proposeRescue(
                    idempotencyKey = key,
                    body = RescueProposeRequest(
                        rescueId = rescueId,
                        brokenDriverId = broken.driverId,
                        rescueDriverId = option.driverId,
                    ),
                )
                if (!resp.isSuccessful) {
                    statusMessage = resp.body()?.error ?: "Propose failed (${resp.code()})"
                } else {
                    statusMessage = "Rescue proposed"
                    selectedBroken = null
                    rescueOptions = emptyList()
                    loadDrivers()
                }
            } catch (e: Exception) {
                statusMessage = e.message
            } finally {
                proposeLoading = false
            }
        }
    }

    LaunchedEffect(Unit) { loadDrivers() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Fleet Rescues") },
                navigationIcon = {
                    if (onBack != null) {
                        IconButton(onClick = onBack) {
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back")
                        }
                    }
                },
                actions = {
                    IconButton(onClick = { loadDrivers() }) {
                        Icon(Icons.Default.Refresh, "Refresh")
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
                    Spacer(Modifier.height(PegasusSpacing.lg))
                    Button(onClick = { loadDrivers() }) { Text("Retry") }
                }
            }
            else -> LazyColumn(
                modifier = Modifier.fillMaxSize().padding(innerPadding),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                item {
                    Text("Needs Rescue", style = MaterialTheme.typography.titleMedium)
                    statusMessage?.let {
                        Text(it, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.primary)
                    }
                }
                if (brokenDrivers.isEmpty()) {
                    item {
                        Text(
                            "No trucks currently require a rescue.",
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                } else {
                    items(brokenDrivers, key = { it.driverId }) { d ->
                        ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                            Row(
                                Modifier.padding(PegasusSpacing.lg).fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically,
                            ) {
                                Column(Modifier.weight(1f)) {
                                    Text(d.name.ifBlank { d.driverId }, style = MaterialTheme.typography.titleSmall)
                                    Text(
                                        "${d.vehicleLabel.ifBlank { "—" }} · ${d.truckStatus}",
                                        style = MaterialTheme.typography.bodySmall,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                                    )
                                }
                                Button(onClick = { findRescue(d) }, enabled = !previewLoading) {
                                    Text("Find Rescue")
                                }
                            }
                        }
                    }
                }
                selectedBroken?.let { broken ->
                    item {
                        Spacer(Modifier.height(PegasusSpacing.md))
                        Text(
                            "Rescue Options for ${broken.name.ifBlank { broken.driverId }}",
                            style = MaterialTheme.typography.titleMedium,
                        )
                    }
                    if (previewLoading) {
                        item { CircularProgressIndicator() }
                    } else if (rescueOptions.isEmpty()) {
                        item {
                            Text(
                                "No rescuers available.",
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                    } else {
                        items(rescueOptions, key = { it.driverId }) { opt ->
                            ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                                Row(
                                    Modifier.padding(PegasusSpacing.lg).fillMaxWidth(),
                                    horizontalArrangement = Arrangement.SpaceBetween,
                                    verticalAlignment = Alignment.CenterVertically,
                                ) {
                                    Column(Modifier.weight(1f)) {
                                        Text(opt.name.ifBlank { opt.driverId }, style = MaterialTheme.typography.titleSmall)
                                        Text(
                                            "${opt.licensePlate} · Capacity: ${"%.1f".format(opt.effectiveCapacityVu)} VU",
                                            style = MaterialTheme.typography.bodySmall,
                                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                                        )
                                        if (opt.isCapacityExceeded) {
                                            Text(
                                                "Insufficient capacity",
                                                style = MaterialTheme.typography.labelSmall,
                                                color = MaterialTheme.colorScheme.error,
                                            )
                                        }
                                    }
                                    Button(
                                        onClick = { propose(opt) },
                                        enabled = !opt.isCapacityExceeded && !proposeLoading,
                                    ) {
                                        Text("Propose")
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
