package com.pegasusx.warehouse.ui.screens.inventory

import androidx.compose.ui.res.stringResource

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
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.put

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
    var enforceOrderAcceptance by remember { mutableStateOf(false) }
    var scheduleIs24h by remember { mutableStateOf(true) }
    var scheduleTimezone by remember { mutableStateOf("UTC") }
    var weekdayOpen by remember { mutableStateOf("09:00") }
    var weekdayClose by remember { mutableStateOf("17:00") }
    var loading by remember { mutableStateOf(true) }
    var saving by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var scheduleError by remember { mutableStateOf<String?>(null) }
    var saveMessage by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val snackbarHostState = remember { SnackbarHostState() }


    fun applyScheduleFields(element: JsonElement) {
        val obj = element.jsonObject
        enforceOrderAcceptance = obj["enforce_order_acceptance"]?.jsonPrimitive?.booleanOrNull ?: false
        scheduleIs24h = obj["is_24h"]?.jsonPrimitive?.booleanOrNull ?: true
        scheduleTimezone = obj["timezone"]?.jsonPrimitive?.contentOrNull ?: "UTC"
        obj["schedules"]?.jsonObject?.get("monday")?.jsonObject?.let { mon ->
            weekdayOpen = mon["open"]?.jsonPrimitive?.contentOrNull ?: "09:00"
            weekdayClose = mon["close"]?.jsonPrimitive?.contentOrNull ?: "17:00"
        }
    }

    fun buildScheduleForSave(): JsonElement {
        val base = try {
            Json.parseToJsonElement(scheduleJSON).jsonObject
        } catch (_: Exception) {
            buildJsonObject {}
        }
        val weekdayWindow = buildJsonObject {
            put("open", weekdayOpen)
            put("close", weekdayClose)
        }
        return buildJsonObject {
            base.forEach { (key, value) ->
                if (key !in setOf("enforce_order_acceptance", "is_24h", "timezone", "schedules")) {
                    put(key, value)
                }
            }
            put("enforce_order_acceptance", enforceOrderAcceptance)
            put("is_24h", scheduleIs24h)
            put("timezone", scheduleTimezone)
            put("schedules", buildJsonObject {
                listOf("monday", "tuesday", "wednesday", "thursday", "friday").forEach { day ->
                    put(day, weekdayWindow)
                }
            })
        }
    }

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
                        applyScheduleFields(it)
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
                buildScheduleForSave()
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
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
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

                OpsSettingsForm(
                    preorderMinLeadDays = preorderMinLeadDays,
                    onPreorderMinLeadDaysChange = { preorderMinLeadDays = it },
                    preorderMaxLeadDays = preorderMaxLeadDays,
                    onPreorderMaxLeadDaysChange = { preorderMaxLeadDays = it },
                    policy = policy,
                    onPolicyChange = { policy = it },
                    showStockCounts = showStockCounts,
                    onShowStockCountsChange = { showStockCounts = it },
                    clearOrderLineMin = clearOrderLineMin,
                    onClearOrderLineMinChange = { clearOrderLineMin = it },
                    orderLineMin = orderLineMin,
                    onOrderLineMinChange = { orderLineMin = it },
                    clearOrderLineMax = clearOrderLineMax,
                    onClearOrderLineMaxChange = { clearOrderLineMax = it },
                    orderLineMax = orderLineMax,
                    onOrderLineMaxChange = { orderLineMax = it },
                    expressEnabled = expressEnabled,
                    onExpressEnabledChange = { expressEnabled = it },
                    expressStockFloor = expressStockFloor,
                    onExpressStockFloorChange = { expressStockFloor = it },
                    clearFeeRules = clearFeeRules,
                    onClearFeeRulesChange = { clearFeeRules = it },
                    feeBaseMinor = feeBaseMinor,
                    onFeeBaseMinorChange = { feeBaseMinor = it },
                    feeCurrency = feeCurrency,
                    onFeeCurrencyChange = { feeCurrency = it },
                    feeTiers = feeTiers,
                    onFeeTiersChange = { feeTiers = it },
                    enforceOrderAcceptance = enforceOrderAcceptance,
                    onEnforceOrderAcceptanceChange = { enforceOrderAcceptance = it },
                    scheduleIs24h = scheduleIs24h,
                    onScheduleIs24hChange = { scheduleIs24h = it },
                    scheduleTimezone = scheduleTimezone,
                    onScheduleTimezoneChange = { scheduleTimezone = it },
                    weekdayOpen = weekdayOpen,
                    onWeekdayOpenChange = { weekdayOpen = it },
                    weekdayClose = weekdayClose,
                    onWeekdayCloseChange = { weekdayClose = it },
                    scheduleJSON = scheduleJSON,
                    onScheduleJSONChange = { scheduleJSON = it },
                    scheduleError = scheduleError,
                )

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


