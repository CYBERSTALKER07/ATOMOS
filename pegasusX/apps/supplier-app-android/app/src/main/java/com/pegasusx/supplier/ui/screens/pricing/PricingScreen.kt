package com.pegasusx.supplier.ui.screens.pricing

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.SupplierPricingRule
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PricingScreen(ops: SupplierOperationsRepository, onBack: () -> Unit) {
    var rule by remember { mutableStateOf<SupplierPricingRule?>(null) }
    var baseMarkupBps by remember { mutableStateOf("") }
    var retailerDiscountBps by remember { mutableStateOf("") }
    var minMarginBps by remember { mutableStateOf("") }
    var currency by remember { mutableStateOf("") }
    var loading by remember { mutableStateOf(true) }
    var saving by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun applyRule(loaded: SupplierPricingRule) {
        rule = loaded
        baseMarkupBps = loaded.baseMarkupBps.toString()
        retailerDiscountBps = loaded.retailerDiscountBps.toString()
        minMarginBps = loaded.minMarginBps.toString()
        currency = loaded.currency
    }

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = ops.getPricingRules()
                if (resp.isSuccessful) {
                    resp.body()?.let { applyRule(it) }
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
        val base = baseMarkupBps.toLongOrNull()
        val discount = retailerDiscountBps.toLongOrNull()
        val margin = minMarginBps.toLongOrNull()
        if (base == null || discount == null || margin == null || base < 0 || discount < 0 || margin < 0) {
            error = "Enter non-negative integer basis points"
            return
        }
        scope.launch {
            saving = true
            error = null
            try {
                val body = buildJsonObject {
                    put("base_markup_bps", JsonPrimitive(base))
                    put("retailer_discount_bps", JsonPrimitive(discount))
                    put("min_margin_bps", JsonPrimitive(margin))
                    val trimmedCurrency = currency.trim().uppercase()
                    if (trimmedCurrency.length == 3) {
                        put("currency", JsonPrimitive(trimmedCurrency))
                    }
                }
                val resp = ops.updatePricingRules(body)
                if (resp.isSuccessful) {
                    resp.body()?.let { applyRule(it) }
                } else {
                    error = "Save failed (${resp.code()})"
                }
            } catch (e: Exception) {
                error = e.message
            } finally {
                saving = false
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Pricing") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> SupplierLoadingState("Loading pricing…", "Markup and discount rules")
            error != null && rule == null -> SupplierStatePane(
                kind = SupplierStateKind.Error,
                headline = "Pricing unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            rule == null -> SupplierStatePane(
                kind = SupplierStateKind.Empty,
                headline = "No pricing rule",
                body = "Supplier pricing authority has not been configured.",
                modifier = Modifier.padding(padding),
            )
            else -> Column(
                modifier = Modifier.padding(padding).padding(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                OutlinedTextField(
                    value = baseMarkupBps,
                    onValueChange = { baseMarkupBps = it.filter(Char::isDigit) },
                    label = { Text("Base markup (bps)") },
                    modifier = Modifier.fillMaxWidth(),
                )
                OutlinedTextField(
                    value = retailerDiscountBps,
                    onValueChange = { retailerDiscountBps = it.filter(Char::isDigit) },
                    label = { Text("Retailer discount (bps)") },
                    modifier = Modifier.fillMaxWidth(),
                )
                OutlinedTextField(
                    value = minMarginBps,
                    onValueChange = { minMarginBps = it.filter(Char::isDigit) },
                    label = { Text("Min margin (bps)") },
                    modifier = Modifier.fillMaxWidth(),
                )
                OutlinedTextField(
                    value = currency,
                    onValueChange = { currency = it.uppercase().take(3) },
                    label = { Text("Currency") },
                    modifier = Modifier.fillMaxWidth(),
                )
                PricingMetric("Version", rule!!.ruleVersion.toString())
                if (error != null) {
                    Text(error!!, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
                }
                Button(onClick = { save() }, enabled = !saving, modifier = Modifier.fillMaxWidth()) {
                    Text(if (saving) "Saving…" else "Save pricing rule")
                }
            }
        }
    }
}

@Composable
private fun PricingMetric(label: String, value: String) {
    Column {
        Text(label, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.outline)
        Text(value, style = MaterialTheme.typography.titleMedium)
    }
}
