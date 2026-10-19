package com.pegasusx.warehouse.ui.screens.coldchain

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
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.TemperatureReading
import com.pegasusx.warehouse.data.model.TemperatureReadingIngestRequest
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ColdChainScreen(
    api: WarehouseApi,
    onBack: (() -> Unit)? = null,
) {
    var manifestId by remember { mutableStateOf("") }
    var sensorId by remember { mutableStateOf("") }
    var tempC by remember { mutableStateOf("") }
    var minC by remember { mutableStateOf("") }
    var maxC by remember { mutableStateOf("") }
    var readings by remember { mutableStateOf<List<TemperatureReading>>(emptyList()) }
    var enabled by remember { mutableStateOf(true) }
    var loading by remember { mutableStateOf(false) }
    var posting by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var statusMessage by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        val mid = manifestId.trim()
        if (mid.isEmpty()) {
            error = "Manifest ID required"
            return
        }
        loading = true
        error = null
        statusMessage = null
        scope.launch {
            try {
                val resp = api.listTemperatureReadings(mid)
                when {
                    resp.code() == 409 -> {
                        enabled = false
                        readings = emptyList()
                    }
                    resp.isSuccessful && resp.body() != null -> {
                        enabled = true
                        readings = resp.body()!!.readings
                    }
                    else -> error = "Failed (${resp.code()})"
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
            } finally {
                loading = false
            }
        }
    }

    fun ingest() {
        val mid = manifestId.trim()
        val temp = tempC.trim().toDoubleOrNull()
        if (mid.isEmpty() || temp == null) {
            error = "Manifest ID and temperature required"
            return
        }
        val minVal = minC.trim().takeIf { it.isNotEmpty() }?.toDoubleOrNull()
        val maxVal = maxC.trim().takeIf { it.isNotEmpty() }?.toDoubleOrNull()
        posting = true
        error = null
        statusMessage = null
        scope.launch {
            try {
                val resp = api.ingestTemperatureReading(
                    TemperatureReadingIngestRequest(
                        manifestId = mid,
                        tempC = temp,
                        sensorId = sensorId.trim().ifBlank { null },
                        minC = if (minVal != null && maxVal != null) minVal else null,
                        maxC = if (minVal != null && maxVal != null) maxVal else null,
                    ),
                )
                when {
                    resp.code() == 409 -> {
                        enabled = false
                        error = "Cold chain disabled on this environment"
                    }
                    resp.isSuccessful -> {
                        statusMessage = "Reading recorded"
                        tempC = ""
                        load()
                    }
                    else -> error = "Ingest failed (${resp.code()})"
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
            } finally {
                posting = false
            }
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Cold chain") },
                navigationIcon = {
                    if (onBack != null) {
                        IconButton(onClick = onBack) {
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, "Back")
                        }
                    }
                },
                actions = {
                    IconButton(onClick = { load() }, enabled = !loading) {
                        Icon(Icons.Default.Refresh, "Refresh")
                    }
                },
            )
        },
    ) { innerPadding ->
        if (!enabled) {
            Box(
                Modifier.fillMaxSize().padding(innerPadding),
                contentAlignment = Alignment.Center,
            ) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text("Cold chain disabled", style = MaterialTheme.typography.titleMedium)
                    Spacer(Modifier.height(PegasusSpacing.sm))
                    Text(
                        "Set WMS_COLD_CHAIN_ENABLED=true on the API to enable temperature ingest.",
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
            return@Scaffold
        }

        LazyColumn(
            modifier = Modifier.fillMaxSize().padding(innerPadding),
            contentPadding = PaddingValues(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            item {
                Text(
                    "Manifest temperature readings — excursions quarantine lots and raise system breaches.",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            item {
                OutlinedTextField(
                    value = manifestId,
                    onValueChange = { manifestId = it },
                    label = { Text("Manifest ID") },
                    modifier = Modifier.fillMaxWidth(),
                    singleLine = true,
                )
            }
            item {
                Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                    Button(onClick = { load() }, enabled = !loading) {
                        Text(if (loading) "Loading…" else "Load readings")
                    }
                }
            }
            item {
                ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                    Column(
                        Modifier.padding(PegasusSpacing.lg),
                        verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                    ) {
                        Text("Record reading", style = MaterialTheme.typography.titleSmall)
                        OutlinedTextField(
                            value = sensorId,
                            onValueChange = { sensorId = it },
                            label = { Text("Sensor ID") },
                            modifier = Modifier.fillMaxWidth(),
                            singleLine = true,
                        )
                        OutlinedTextField(
                            value = tempC,
                            onValueChange = { tempC = it },
                            label = { Text("Temp °C") },
                            modifier = Modifier.fillMaxWidth(),
                            singleLine = true,
                        )
                        Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                            OutlinedTextField(
                                value = minC,
                                onValueChange = { minC = it },
                                label = { Text("Min °C") },
                                modifier = Modifier.weight(1f),
                                singleLine = true,
                            )
                            OutlinedTextField(
                                value = maxC,
                                onValueChange = { maxC = it },
                                label = { Text("Max °C") },
                                modifier = Modifier.weight(1f),
                                singleLine = true,
                            )
                        }
                        OutlinedButton(onClick = { ingest() }, enabled = !posting) {
                            Text(if (posting) "Recording…" else "Record reading")
                        }
                    }
                }
            }
            if (error != null) {
                item {
                    Text(error!!, color = MaterialTheme.colorScheme.error)
                }
            }
            if (statusMessage != null) {
                item {
                    Text(statusMessage!!, color = MaterialTheme.colorScheme.primary)
                }
            }
            if (!loading && readings.isEmpty()) {
                item {
                    Text(
                        "No readings — load a manifest or record the first sample.",
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
            items(readings, key = { it.readingId.ifBlank { "${it.recordedAt}-${it.tempC}" } }) { row ->
                ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                    Column(Modifier.padding(PegasusSpacing.lg)) {
                        Text(
                            "%.2f °C".format(row.tempC),
                            style = MaterialTheme.typography.titleMedium,
                            color = if (row.excursion) {
                                MaterialTheme.colorScheme.error
                            } else {
                                MaterialTheme.colorScheme.onSurface
                            },
                        )
                        Text(
                            if (row.excursion) "EXCURSION" else "OK",
                            style = MaterialTheme.typography.labelMedium,
                            color = if (row.excursion) {
                                MaterialTheme.colorScheme.error
                            } else {
                                MaterialTheme.colorScheme.primary
                            },
                        )
                        Spacer(Modifier.height(4.dp))
                        Text(row.recordedAt, style = MaterialTheme.typography.bodySmall)
                        val band = when {
                            row.minC != null && row.maxC != null -> "${row.minC}…${row.maxC}"
                            else -> "—"
                        }
                        Text("Band $band · Sensor ${row.sensorId ?: "—"}", style = MaterialTheme.typography.bodySmall)
                    }
                }
            }
            if (loading) {
                item {
                    Box(Modifier.fillMaxWidth(), contentAlignment = Alignment.Center) {
                        CircularProgressIndicator()
                    }
                }
            }
        }
    }
}
