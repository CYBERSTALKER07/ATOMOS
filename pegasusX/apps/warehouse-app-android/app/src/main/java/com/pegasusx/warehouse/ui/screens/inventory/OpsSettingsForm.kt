package com.pegasusx.warehouse.ui.screens.inventory

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.selection.selectable
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import com.pegasusx.warehouse.R

data class FeeTierDraft(
    val maxKm: String = "",
    val feeMinor: String = "0",
)

@Composable
fun OpsSettingsForm(
    preorderMinLeadDays: String,
    onPreorderMinLeadDaysChange: (String) -> Unit,
    preorderMaxLeadDays: String,
    onPreorderMaxLeadDaysChange: (String) -> Unit,
    policy: String,
    onPolicyChange: (String) -> Unit,
    showStockCounts: Boolean,
    onShowStockCountsChange: (Boolean) -> Unit,
    clearOrderLineMin: Boolean,
    onClearOrderLineMinChange: (Boolean) -> Unit,
    orderLineMin: String,
    onOrderLineMinChange: (String) -> Unit,
    clearOrderLineMax: Boolean,
    onClearOrderLineMaxChange: (Boolean) -> Unit,
    orderLineMax: String,
    onOrderLineMaxChange: (String) -> Unit,
    expressEnabled: Boolean,
    onExpressEnabledChange: (Boolean) -> Unit,
    expressStockFloor: String,
    onExpressStockFloorChange: (String) -> Unit,
    clearFeeRules: Boolean,
    onClearFeeRulesChange: (Boolean) -> Unit,
    feeBaseMinor: String,
    onFeeBaseMinorChange: (String) -> Unit,
    feeCurrency: String,
    onFeeCurrencyChange: (String) -> Unit,
    feeTiers: List<FeeTierDraft>,
    onFeeTiersChange: (List<FeeTierDraft>) -> Unit,
    enforceOrderAcceptance: Boolean,
    onEnforceOrderAcceptanceChange: (Boolean) -> Unit,
    scheduleIs24h: Boolean,
    onScheduleIs24hChange: (Boolean) -> Unit,
    scheduleTimezone: String,
    onScheduleTimezoneChange: (String) -> Unit,
    weekdayOpen: String,
    onWeekdayOpenChange: (String) -> Unit,
    weekdayClose: String,
    onWeekdayCloseChange: (String) -> Unit,
    scheduleJSON: String,
    onScheduleJSONChange: (String) -> Unit,
    scheduleError: String?,
) {
    SettingsCard(title = stringResource(R.string.warehouse_portal_settings_ops_settings_form_text_pre_order_lead_window)) {
        Text(
            "Retailers can request delivery between these lead days from today.",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
            OutlinedTextField(
                value = preorderMinLeadDays,
                onValueChange = onPreorderMinLeadDaysChange,
                label = { Text("Min days") },
                singleLine = true,
                modifier = Modifier.weight(1f),
            )
            OutlinedTextField(
                value = preorderMaxLeadDays,
                onValueChange = onPreorderMaxLeadDaysChange,
                label = { Text("Max days") },
                singleLine = true,
                modifier = Modifier.weight(1f),
            )
        }
    }

    SettingsCard(title = stringResource(R.string.warehouse_portal_settings_ops_settings_form_text_out_of_stock_orders)) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text("Accept when out of stock")
            Switch(
                checked = policy == "ACCEPT_BACKORDER",
                onCheckedChange = { onPolicyChange(if (it) "ACCEPT_BACKORDER" else "REJECT") },
            )
        }
        PolicyOption(
            label = stringResource(R.string.supplier_portal_residual_text_reject_orders_when_out_of_stock),
            selected = policy == "REJECT",
            onSelect = { onPolicyChange("REJECT") },
        )
        PolicyOption(
            label = stringResource(R.string.mobile_warehouse_ui_accept_orders_warn_retailer_fulfill_when_stock_arrives),
            selected = policy == "ACCEPT_BACKORDER",
            onSelect = { onPolicyChange("ACCEPT_BACKORDER") },
        )
    }

    SettingsCard(title = stringResource(R.string.mobile_warehouse_ui_retailer_catalog_display)) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text("Show stock counts to retailers")
            Switch(checked = showStockCounts, onCheckedChange = onShowStockCountsChange)
        }
    }

    SettingsCard(title = stringResource(R.string.mobile_warehouse_ui_order_line_quantity_limits)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Checkbox(checked = clearOrderLineMin, onCheckedChange = onClearOrderLineMinChange)
            Text("No minimum quantity", style = MaterialTheme.typography.bodyMedium)
        }
        if (!clearOrderLineMin) {
            OutlinedTextField(
                value = orderLineMin,
                onValueChange = onOrderLineMinChange,
                label = { Text("Minimum quantity") },
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
            )
        }
        Row(verticalAlignment = Alignment.CenterVertically) {
            Checkbox(checked = clearOrderLineMax, onCheckedChange = onClearOrderLineMaxChange)
            Text("No maximum quantity", style = MaterialTheme.typography.bodyMedium)
        }
        if (!clearOrderLineMax) {
            OutlinedTextField(
                value = orderLineMax,
                onValueChange = onOrderLineMaxChange,
                label = { Text("Maximum quantity") },
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
            )
        }
    }

    SettingsCard(title = stringResource(R.string.mobile_warehouse_ui_express_delivery)) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text("Express enabled")
            Switch(checked = expressEnabled, onCheckedChange = onExpressEnabledChange)
        }
        OutlinedTextField(
            value = expressStockFloor,
            onValueChange = onExpressStockFloorChange,
            label = { Text("Express stock floor") },
            singleLine = true,
            modifier = Modifier.fillMaxWidth(),
        )
    }

    SettingsCard(title = stringResource(R.string.mobile_warehouse_ui_delivery_fee_rules)) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Checkbox(checked = clearFeeRules, onCheckedChange = onClearFeeRulesChange)
            Text("No delivery fee rules", style = MaterialTheme.typography.bodyMedium)
        }
        if (!clearFeeRules) {
            Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                OutlinedTextField(
                    value = feeBaseMinor,
                    onValueChange = onFeeBaseMinorChange,
                    label = { Text("Base fee (minor)") },
                    singleLine = true,
                    modifier = Modifier.weight(1f),
                )
                OutlinedTextField(
                    value = feeCurrency,
                    onValueChange = {},
                    readOnly = true,
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
                            onFeeTiersChange(feeTiers.toMutableList().also { it[index] = tier.copy(maxKm = value) })
                        },
                        label = { Text("Max km") },
                        singleLine = true,
                        modifier = Modifier.weight(1f),
                    )
                    OutlinedTextField(
                        value = tier.feeMinor,
                        onValueChange = { value ->
                            onFeeTiersChange(feeTiers.toMutableList().also { it[index] = tier.copy(feeMinor = value) })
                        },
                        label = { Text("Fee (minor)") },
                        singleLine = true,
                        modifier = Modifier.weight(1f),
                    )
                    TextButton(
                        enabled = feeTiers.size > 1,
                        onClick = { onFeeTiersChange(feeTiers.filterIndexed { i, _ -> i != index }) },
                    ) { Text("Remove") }
                }
            }
            TextButton(onClick = { onFeeTiersChange(feeTiers + FeeTierDraft()) }) {
                Text("Add tier")
            }
        }
    }

    SettingsCard(title = stringResource(R.string.warehouse_portal_settings_ops_settings_form_text_order_acceptance_hours)) {
        Text(
            "When enforcement is on, retailers cannot preview or create orders outside the window.",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text("Enforce order acceptance hours")
            Switch(checked = enforceOrderAcceptance, onCheckedChange = onEnforceOrderAcceptanceChange)
        }
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text("Open 24 hours")
            Switch(checked = scheduleIs24h, onCheckedChange = onScheduleIs24hChange)
        }
        OutlinedTextField(
            value = scheduleTimezone,
            onValueChange = onScheduleTimezoneChange,
            label = { Text("Timezone") },
            singleLine = true,
            modifier = Modifier.fillMaxWidth(),
        )
        Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
            OutlinedTextField(
                value = weekdayOpen,
                onValueChange = onWeekdayOpenChange,
                label = { Text("Weekday open") },
                singleLine = true,
                modifier = Modifier.weight(1f),
            )
            OutlinedTextField(
                value = weekdayClose,
                onValueChange = onWeekdayCloseChange,
                label = { Text("Weekday close") },
                singleLine = true,
                modifier = Modifier.weight(1f),
            )
        }
        Text("Advanced JSON", style = MaterialTheme.typography.labelMedium)
        OutlinedTextField(
            value = scheduleJSON,
            onValueChange = onScheduleJSONChange,
            modifier = Modifier.fillMaxWidth().heightIn(min = 140.dp),
            label = { Text("Schedule JSON") },
        )
        if (scheduleError != null) {
            Text(scheduleError, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
        }
    }
}

@Composable
internal fun SettingsCard(title: String, content: @Composable ColumnScope.() -> Unit) {
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
internal fun PolicyOption(label: String, selected: Boolean, onSelect: () -> Unit) {
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
