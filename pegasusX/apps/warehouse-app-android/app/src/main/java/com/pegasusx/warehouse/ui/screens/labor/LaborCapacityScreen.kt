package com.pegasusx.warehouse.ui.screens.labor

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
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
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
import com.pegasusx.warehouse.data.model.LaborDriverAvailabilityRequest
import com.pegasusx.warehouse.data.model.LaborDriverScore
import com.pegasusx.warehouse.data.model.LaborZoneCapacity
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import java.time.LocalDate

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LaborCapacityScreen(
    api: WarehouseApi,
    onBack: (() -> Unit)? = null,
) {
    var date by remember { mutableStateOf(LocalDate.now().toString()) }
    var zones by remember { mutableStateOf<List<LaborZoneCapacity>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var driverId by remember { mutableStateOf("") }
    var score by remember { mutableStateOf<LaborDriverScore?>(null) }
    var availHours by remember { mutableStateOf("8") }
    var availStatus by remember { mutableStateOf("AVAILABLE") }
    var zoneH3 by remember { mutableStateOf("") }
    var saving by remember { mutableStateOf(false) }
    var statusMessage by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun loadZones() {
        loading = true
        error = null
        scope.launch {
            try {
                val resp = api.listLaborZoneCapacity(date.trim())
                if (resp.isSuccessful && resp.body() != null) {
                    zones = resp.body()!!.resolvedZones()
                } else {
                    zones = emptyList()
                    error = "Failed to load zone capacity (${resp.code()})"
                }
            } catch (e: Exception) {
                zones = emptyList()
                error = e.message ?: "Network error"
            } finally {
                loading = false
            }
        }
    }

    fun loadScore() {
        val id = driverId.trim()
        if (id.isEmpty()) {
            error = "Driver ID required"
            return
        }
        scope.launch {
            try {
                val resp = api.getLaborDriverScore(id)
                if (resp.isSuccessful && resp.body() != null) {
                    score = resp.body()
                    error = null
                } else {
                    score = null
                    error = "Driver score not found (${resp.code()})"
                }
            } catch (e: Exception) {
                score = null
                error = e.message ?: "Network error"
            }
        }
    }

    fun saveAvailability() {
        val id = driverId.trim()
        if (id.isEmpty()) {
            error = "Driver ID required"
            return
        }
        saving = true
        statusMessage = null
        scope.launch {
            try {
                val resp = api.setLaborDriverAvailability(
                    LaborDriverAvailabilityRequest(
                        driverId = id,
                        date = date.trim(),
                        availableHours = availHours.trim().toDoubleOrNull() ?: 0.0,
                        status = availStatus.trim().ifBlank { "AVAILABLE" },
                        zoneH3 = zoneH3.trim().ifBlank { null },
                    ),
                )
                if (resp.isSuccessful) {
                    statusMessage = "Availability saved"
                    loadZones()
                } else {
                    error = "Save failed (${resp.code()})"
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
            } finally {
                saving = false
            }
        }
    }

    LaunchedEffect(Unit) { loadZones() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Labor capacity") },
                navigationIcon = {
                    if (onBack != null) {
                        IconButton(onClick = onBack) {
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back")
                        }
                    }
                },
                actions = {
                    IconButton(onClick = { loadZones() }, enabled = !loading) {
                        Icon(Icons.Default.Refresh, "Refresh")
                    }
                },
            )
        },
    ) { innerPadding ->
        LazyColumn(
            modifier = Modifier.fillMaxSize().padding(innerPadding),
            contentPadding = PaddingValues(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            item {
                Text(
                    "Zone delivery capacity and driver reliability scores.",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            item {
                OutlinedTextField(
                    value = date,
                    onValueChange = { date = it },
                    label = { Text("Date (YYYY-MM-DD)") },
                    modifier = Modifier.fillMaxWidth(),
                    singleLine = true,
                )
            }
            item {
                Button(onClick = { loadZones() }, enabled = !loading) {
                    Text("Refresh zones")
                }
            }
            if (error != null) {
                item { Text(error!!, color = MaterialTheme.colorScheme.error) }
            }
            if (statusMessage != null) {
                item { Text(statusMessage!!, color = MaterialTheme.colorScheme.primary) }
            }
            when {
                loading -> item {
                    Box(Modifier.fillMaxWidth(), contentAlignment = Alignment.Center) {
                        CircularProgressIndicator()
                    }
                }
                zones.isEmpty() -> item {
                    Text(
                        "No zone capacity rows. Workers populate ZoneCapacity after availability is set.",
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                else -> items(zones, key = { "${it.zoneH3}-${it.date}" }) { z ->
                    val util = if (z.totalCapacity > 0) (z.usedCapacity / z.totalCapacity) * 100 else 0.0
                    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(PegasusSpacing.lg)) {
                            Text(z.zoneH3, style = MaterialTheme.typography.titleSmall)
                            Text(
                                "Total ${"%.1f".format(z.totalCapacity)} · Used ${"%.1f".format(z.usedCapacity)} · ${"%.0f".format(util)}%",
                                style = MaterialTheme.typography.bodyMedium,
                            )
                            if (z.date.isNotBlank()) {
                                Text(z.date, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                            }
                        }
                    }
                }
            }
            item {
                ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                    Column(
                        Modifier.padding(PegasusSpacing.lg),
                        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                    ) {
                        Text("Driver score & availability", style = MaterialTheme.typography.titleSmall)
                        OutlinedTextField(
                            value = driverId,
                            onValueChange = { driverId = it },
                            label = { Text("Driver ID") },
                            modifier = Modifier.fillMaxWidth(),
                            singleLine = true,
                        )
                        OutlinedButton(onClick = { loadScore() }) { Text("Load score") }
                        score?.let { s ->
                            Text("Score ${"%.1f".format(s.score)}")
                            Text(
                                "On-time ${(s.onTimeRate * 100).toInt()}% · Completion ${(s.completionRate * 100).toInt()}% · Stops/hr ${"%.1f".format(s.stopsPerHour)}",
                                style = MaterialTheme.typography.bodySmall,
                            )
                        }
                        Spacer(Modifier.height(PegasusSpacing.sm))
                        OutlinedTextField(
                            value = availHours,
                            onValueChange = { availHours = it },
                            label = { Text("Hours") },
                            modifier = Modifier.fillMaxWidth(),
                            singleLine = true,
                        )
                        OutlinedTextField(
                            value = availStatus,
                            onValueChange = { availStatus = it },
                            label = { Text("Status (AVAILABLE / LIMITED / OFF)") },
                            modifier = Modifier.fillMaxWidth(),
                            singleLine = true,
                        )
                        OutlinedTextField(
                            value = zoneH3,
                            onValueChange = { zoneH3 = it },
                            label = { Text("Zone H3 (optional)") },
                            modifier = Modifier.fillMaxWidth(),
                            singleLine = true,
                        )
                        Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                            Button(onClick = { saveAvailability() }, enabled = !saving) {
                                Text(if (saving) "Saving…" else "Save availability")
                            }
                        }
                    }
                }
            }
        }
    }
}
