package com.pegasusx.warehouse.ui.screens.inventory

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.WarehouseOpsSettingsPatchRequest
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import com.pegasusx.warehouse.util.WarehouseIdempotencyKeys
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun OpsSettingsScreen(
    api: WarehouseApi,
    onBack: (() -> Unit)? = null,
) {
    var policy by remember { mutableStateOf("REJECT") }
    var scheduleJSON by remember { mutableStateOf("{\n  \"is_24h\": true\n}") }
    var loading by remember { mutableStateOf(true) }
    var saving by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var scheduleError by remember { mutableStateOf<String?>(null) }
    var saveMessage by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val snackbarHostState = remember { SnackbarHostState() }

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = api.getOpsSettings()
                if (resp.isSuccessful && resp.body() != null) {
                    val body = resp.body()!!
                    policy = body.defaultOutOfStockPolicy
                    body.operatingSchedule?.let { scheduleJSON = Json { prettyPrint = true }.encodeToString(JsonElement.serializer(), it) }
                } else {
                    error = "Failed (${resp.code()})"
                }
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    fun save() {
        scope.launch {
            saving = true
            saveMessage = null
            scheduleError = null
            val schedule: JsonElement = try {
                Json.parseToJsonElement(scheduleJSON)
            } catch (_: Exception) {
                scheduleError = "Operating schedule must be valid JSON"
                saving = false
                return@launch
            }
            try {
                val resp = api.patchOpsSettings(
                    WarehouseOpsSettingsPatchRequest(
                        defaultOutOfStockPolicy = policy,
                        operatingSchedule = schedule,
                    ),
                    WarehouseIdempotencyKeys.opsSettings(),
                )
                if (resp.isSuccessful) {
                    saveMessage = "Warehouse settings saved"
                    snackbarHostState.showSnackbar("Warehouse settings saved")
                    load()
                } else {
                    saveMessage = "Save failed (${resp.code()})"
                }
            } catch (e: Exception) {
                saveMessage = e.message
            } finally {
                saving = false
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Ops settings") },
                navigationIcon = {
                    if (onBack != null) {
                        IconButton(onClick = onBack) {
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                        }
                    }
                },
                actions = {
                    TextButton(onClick = { load() }) { Text("Refresh") }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { padding ->
        when {
            loading -> Box(
                Modifier.fillMaxSize().padding(padding),
                contentAlignment = Alignment.Center,
            ) { CircularProgressIndicator() }

            error != null -> Box(
                Modifier.fillMaxSize().padding(padding),
                contentAlignment = Alignment.Center,
            ) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(error!!, color = MaterialTheme.colorScheme.error)
                    Spacer(Modifier.height(PegasusSpacing.md))
                    Button(onClick = { load() }) { Text("Retry") }
                }
            }

            else -> Column(
                modifier = Modifier
                    .padding(padding)
                    .padding(PegasusSpacing.lg)
                    .verticalScroll(rememberScrollState())
                    .fillMaxSize(),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.lg),
            ) {
                Text(
                    "Stock policy for retailer checkout and display operating hours.",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )

                ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                    Column(Modifier.padding(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                        Text("Out-of-stock orders", style = MaterialTheme.typography.titleSmall)
                        PolicyOption(
                            label = "Reject orders when out of stock",
                            selected = policy == "REJECT",
                            onSelect = { policy = "REJECT" },
                        )
                        PolicyOption(
                            label = "Accept orders — warn retailer, fulfill when stock arrives",
                            selected = policy == "ACCEPT_BACKORDER",
                            onSelect = { policy = "ACCEPT_BACKORDER" },
                        )
                    }
                }

                ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                    Column(Modifier.padding(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                        Text("Operating hours (display only)", style = MaterialTheme.typography.titleSmall)
                        Text(
                            "Shown to retailers for planning. Dispatch and delivery are not blocked outside these hours.",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                        OutlinedTextField(
                            value = scheduleJSON,
                            onValueChange = { scheduleJSON = it },
                            modifier = Modifier.fillMaxWidth().heightIn(min = 140.dp),
                            label = { Text("Schedule JSON") },
                        )
                        if (scheduleError != null) {
                            Text(scheduleError!!, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
                        }
                    }
                }

                if (saveMessage != null) {
                    Text(saveMessage!!, style = MaterialTheme.typography.bodySmall)
                }

                Button(
                    onClick = { save() },
                    enabled = !saving,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text(if (saving) "Saving…" else "Save settings")
                }
            }
        }
    }
}

@Composable
private fun PolicyOption(label: String, selected: Boolean, onSelect: () -> Unit) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .selectable(selected = selected, onClick = onSelect, role = Role.RadioButton)
            .padding(vertical = PegasusSpacing.xs),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        RadioButton(selected = selected, onClick = null)
        Spacer(Modifier.width(PegasusSpacing.sm))
        Text(label, style = MaterialTheme.typography.bodyMedium)
    }
}
