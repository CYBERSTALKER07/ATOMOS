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
import com.pegasusx.warehouse.data.model.DeliveryFeeRules
import com.pegasusx.warehouse.data.model.DeliveryFeeTier
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
    var showStockCounts by remember { mutableStateOf(false) }
    var preorderMinLeadDays by remember { mutableStateOf("3") }
    var preorderMaxLeadDays by remember { mutableStateOf("90") }
    var orderLineMin by remember { mutableStateOf("") }
    var orderLineMax by remember { mutableStateOf("") }
    var clearOrderLineMin by remember { mutableStateOf(true) }
    var clearOrderLineMax by remember { mutableStateOf(true) }
    var expressEnabled by remember { mutableStateOf(false) }
    var expressStockFloor by remember { mutableStateOf("0") }
    var feeBaseMinor by remember { mutableStateOf("0") }
    var feeCurrency by remember { mutableStateOf("UZS") }
    var feeTiers by remember { mutableStateOf(listOf(FeeTierDraft(maxKm = "5", feeMinor = "0"))) }
    var clearFeeRules by remember { mutableStateOf(true) }
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
                    showStockCounts = body.showStockCountsToRetailers
                    preorderMinLeadDays = body.preorderMinLeadDays.toString()
                    preorderMaxLeadDays = body.preorderMaxLeadDays.toString()
                    if (body.orderLineMinQuantity != null) {
                        orderLineMin = body.orderLineMinQuantity.toString()
                        clearOrderLineMin = false
                    } else {
                        orderLineMin = ""
                        clearOrderLineMin = true
                    }
                    if (body.orderLineMaxQuantity != null) {
                        orderLineMax = body.orderLineMaxQuantity.toString()
                        clearOrderLineMax = false
                    } else {
                        orderLineMax = ""
                        clearOrderLineMax = true
                    }
                    expressEnabled = body.expressEnabled
                    expressStockFloor = body.expressStockFloor.toString()
                    body.deliveryFeeRules?.let { rules ->
                        feeBaseMinor = rules.baseFeeMinor.toString()
                        feeCurrency = rules.currency.ifBlank { "UZS" }
                        feeTiers = rules.tiers.map {
                            FeeTierDraft(
                                maxKm = it.maxKm?.toString() ?: "",
                                feeMinor = it.feeMinor.toString(),
                            )
                        }.ifEmpty { listOf(FeeTierDraft()) }
                        clearFeeRules = false
                    } ?: run {
                        feeBaseMinor = "0"
                        feeCurrency = "UZS"
                        feeTiers = listOf(FeeTierDraft(maxKm = "5", feeMinor = "0"))
                        clearFeeRules = true
                    }
                    body.operatingSchedule?.let {
                        scheduleJSON = Json { prettyPrint = true }.encodeToString(JsonElement.serializer(), it)
                    }
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
            val minLead = preorderMinLeadDays.toLongOrNull()
            val maxLead = preorderMaxLeadDays.toLongOrNull()
            if (minLead == null || maxLead == null) {
                saveMessage = "Pre-order lead days must be valid numbers"
                saving = false
                return@launch
            }
            val patch = WarehouseOpsSettingsPatchRequest(
                defaultOutOfStockPolicy = policy,
                showStockCountsToRetailers = showStockCounts,
                operatingSchedule = schedule,
                preorderMinLeadDays = minLead,
                preorderMaxLeadDays = maxLead,
                expressEnabled = expressEnabled,
                expressStockFloor = expressStockFloor.toLongOrNull() ?: 0L,
                clearOrderLineMinQuantity = if (clearOrderLineMin) true else null,
                clearOrderLineMaxQuantity = if (clearOrderLineMax) true else null,
                orderLineMinQuantity = if (clearOrderLineMin) null else orderLineMin.toLongOrNull(),
                orderLineMaxQuantity = if (clearOrderLineMax) null else orderLineMax.toLongOrNull(),
                clearDeliveryFeeRules = if (clearFeeRules) true else null,
                deliveryFeeRules = if (clearFeeRules) {
                    null
                } else {
                    DeliveryFeeRules(
                        baseFeeMinor = feeBaseMinor.toLongOrNull() ?: 0L,
                        currency = feeCurrency.trim().ifBlank { "UZS" },
                        tiers = feeTiers.map { draft ->
                            DeliveryFeeTier(
                                maxKm = draft.maxKm.trim().takeIf { it.isNotEmpty() }?.toDoubleOrNull(),
                                feeMinor = draft.feeMinor.toLongOrNull() ?: 0L,
                            )
                        },
                    )
                },
            )
            try {
                val resp = api.patchOpsSettings(patch, WarehouseIdempotencyKeys.opsSettings())
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
                    "Checkout policy, pre-orders, delivery fees, and retailer catalog display.",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )

                SettingsCard(title = "Pre-order lead window") {
                    Text(
                        "Retailers can request delivery between these lead days from today.",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                    Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                        OutlinedTextField(
                            value = preorderMinLeadDays,
                            onValueChange = { preorderMinLeadDays = it },
                            label = { Text("Min days") },
                            singleLine = true,
                            modifier = Modifier.weight(1f),
                        )
                        OutlinedTextField(
                            value = preorderMaxLeadDays,
                            onValueChange = { preorderMaxLeadDays = it },
                            label = { Text("Max days") },
                            singleLine = true,
                            modifier = Modifier.weight(1f),
                        )
                    }
                }

                SettingsCard(title = "Out-of-stock orders") {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text("Accept when out of stock")
                        Switch(
                            checked = policy == "ACCEPT_BACKORDER",
                            onCheckedChange = { policy = if (it) "ACCEPT_BACKORDER" else "REJECT" },
                        )
                    }
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

                SettingsCard(title = "Retailer catalog display") {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text("Show stock counts to retailers")
                        Switch(checked = showStockCounts, onCheckedChange = { showStockCounts = it })
                    }
                }

                SettingsCard(title = "Order line quantity limits") {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Checkbox(checked = clearOrderLineMin, onCheckedChange = { clearOrderLineMin = it })
                        Text("No minimum quantity", style = MaterialTheme.typography.bodyMedium)
                    }
                    if (!clearOrderLineMin) {
                        OutlinedTextField(
                            value = orderLineMin,
                            onValueChange = { orderLineMin = it },
                            label = { Text("Minimum quantity") },
                            singleLine = true,
                            modifier = Modifier.fillMaxWidth(),
                        )
                    }
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Checkbox(checked = clearOrderLineMax, onCheckedChange = { clearOrderLineMax = it })
                        Text("No maximum quantity", style = MaterialTheme.typography.bodyMedium)
                    }
                    if (!clearOrderLineMax) {
                        OutlinedTextField(
                            value = orderLineMax,
                            onValueChange = { orderLineMax = it },
                            label = { Text("Maximum quantity") },
                            singleLine = true,
                            modifier = Modifier.fillMaxWidth(),
                        )
                    }
                }

                SettingsCard(title = "Express delivery") {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text("Express enabled")
                        Switch(checked = expressEnabled, onCheckedChange = { expressEnabled = it })
                    }
                    OutlinedTextField(
                        value = expressStockFloor,
                        onValueChange = { expressStockFloor = it },
                        label = { Text("Express stock floor") },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth(),
                    )
                }

                SettingsCard(title = "Delivery fee rules") {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Checkbox(checked = clearFeeRules, onCheckedChange = { clearFeeRules = it })
                        Text("No delivery fee rules", style = MaterialTheme.typography.bodyMedium)
                    }
                    if (!clearFeeRules) {
                        Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                            OutlinedTextField(
                                value = feeBaseMinor,
                                onValueChange = { feeBaseMinor = it },
                                label = { Text("Base fee (minor)") },
                                singleLine = true,
                                modifier = Modifier.weight(1f),
                            )
                            OutlinedTextField(
                                value = feeCurrency,
                                onValueChange = { feeCurrency = it },
                                label = { Text("Currency") },
                                singleLine = true,
                                modifier = Modifier.weight(1f),
                            )
                        }
                        feeTiers.forEachIndexed { index, tier ->
                            Row(
                                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
                                verticalAlignment = Alignment.Bottom,
                            ) {
                                OutlinedTextField(
                                    value = tier.maxKm,
                                    onValueChange = { value ->
                                        feeTiers = feeTiers.toMutableList().also { it[index] = tier.copy(maxKm = value) }
                                    },
                                    label = { Text("Max km") },
                                    singleLine = true,
                                    modifier = Modifier.weight(1f),
                                )
                                OutlinedTextField(
                                    value = tier.feeMinor,
                                    onValueChange = { value ->
                                        feeTiers = feeTiers.toMutableList().also { it[index] = tier.copy(feeMinor = value) }
                                    },
                                    label = { Text("Fee (minor)") },
                                    singleLine = true,
                                    modifier = Modifier.weight(1f),
                                )
                                TextButton(
                                    enabled = feeTiers.size > 1,
                                    onClick = { feeTiers = feeTiers.filterIndexed { i, _ -> i != index } },
                                ) { Text("Remove") }
                            }
                        }
                        TextButton(onClick = { feeTiers = feeTiers + FeeTierDraft() }) {
                            Text("Add tier")
                        }
                    }
                }

                SettingsCard(title = "Operating hours (display only)") {
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

private data class FeeTierDraft(
    val maxKm: String = "",
    val feeMinor: String = "0",
)

@Composable
private fun SettingsCard(title: String, content: @Composable ColumnScope.() -> Unit) {
    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
        Column(
            Modifier.padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            content = {
                Text(title, style = MaterialTheme.typography.titleSmall)
                content()
            },
        )
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
