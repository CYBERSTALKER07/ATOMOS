package com.pegasusx.supplier.ui.screens.settings

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.supplier.data.model.SupplierReturnPolicy
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.util.SupplierIdempotencyKeys
import kotlinx.coroutines.launch
import com.pegasusx.supplier.R

private val HOUR_PRESETS = listOf(8L, 24L, 48L, 72L)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ReturnPolicySettingsScreen(
    ops: SupplierOperationsRepository,
    onBack: () -> Unit,
) {
    var loading by remember { mutableStateOf(true) }
    var saving by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var saved by remember { mutableStateOf(false) }
    var hours by remember { mutableLongStateOf(48L) }
    var hoursText by remember { mutableStateOf("48") }
    var concealed by remember { mutableStateOf("") }
    var requirePhoto by remember { mutableStateOf(true) }
    var allowExpired by remember { mutableStateOf(false) }
    var sourceHint by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()

    fun applyHours(value: Long) {
        hours = value
        hoursText = value.toString()
    }

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = ops.getReturnPolicy()
                if (resp.isSuccessful && resp.body() != null) {
                    val p = resp.body()!!
                    applyHours(p.defaultWindowHours.takeIf { it > 0 } ?: 48L)
                    concealed = p.concealedDamageWindowHours?.takeIf { it > 0 }?.toString().orEmpty()
                    requirePhoto = p.requirePhoto
                    allowExpired = p.allowExpiredClaims
                    sourceHint = p.policySourceHint.orEmpty()
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
        val parsedHours = hoursText.toLongOrNull() ?: hours
        if (parsedHours < 1 || parsedHours > 168) {
            error = "Default window must be between 1 and 168 hours"
            return
        }
        scope.launch {
            saving = true
            error = null
            saved = false
            val concealedNum = concealed.trim().toLongOrNull()?.takeIf { it > 0 }
            val body = SupplierReturnPolicy(
                defaultWindowHours = parsedHours,
                concealedDamageWindowHours = concealedNum,
                requirePhoto = requirePhoto,
                allowExpiredClaims = allowExpired,
            )
            try {
                val key = SupplierIdempotencyKeys.returnPolicyPut(
                    SupplierIdempotencyKeys.supplierScopeId(),
                    parsedHours,
                )
                val resp = ops.putReturnPolicy(body, key)
                if (resp.isSuccessful) {
                    sourceHint = resp.body()?.policySourceHint ?: "SUPPLIER"
                    applyHours(parsedHours)
                    saved = true
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
                title = { Text("Return policy") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
                actions = {
                    TextButton(onClick = { load() }) { Text("Refresh") }
                },
            )
        },
    ) { padding ->
        when {
            loading -> PegasusLoadingState(
                title = stringResource(R.string.warehouse_portal_bins_text_loading),
                body = "Return policy",
                modifier = Modifier.padding(padding),
            )
            error != null && hoursText.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Policy unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            else -> Column(
                modifier = Modifier
                    .padding(padding)
                    .padding(PegasusSpacing.lg)
                    .verticalScroll(rememberScrollState())
                    .fillMaxSize(),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                Text(
                    "Claim filing windows applied when orders complete. Retailers see the countdown from this policy.",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                if (sourceHint.isNotBlank()) {
                    Text(
                        stringResource(R.string.mobile_supplier_ui_source_hint_sourcehint_2, sourceHint),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                if (error != null) {
                    Text(error!!, color = MaterialTheme.colorScheme.error)
                }
                if (saved) {
                    Text("Return policy saved.", color = MaterialTheme.colorScheme.primary)
                }

                Text("Default claim window (hours)", style = MaterialTheme.typography.labelLarge)
                Row(
                    horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    HOUR_PRESETS.forEach { preset ->
                        FilterChip(
                            selected = hours == preset,
                            onClick = { applyHours(preset) },
                            label = { Text(stringResource(R.string.mobile_supplier_ui_preseth, preset)) },
                        )
                    }
                }
                OutlinedTextField(
                    value = hoursText,
                    onValueChange = {
                        hoursText = it.filter { ch -> ch.isDigit() }
                        hoursText.toLongOrNull()?.let { hours = it }
                    },
                    label = { Text("Custom hours (1–168)") },
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                    modifier = Modifier.fillMaxWidth(),
                    singleLine = true,
                )
                Text(stringResource(R.string.mobile_supplier_ui_preview_retailers_may_file_claims_for_ifblankh_after_delivery, hoursText.ifBlank { "—" }),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )

                OutlinedTextField(
                    value = concealed,
                    onValueChange = { concealed = it.filter { ch -> ch.isDigit() } },
                    label = { Text("Concealed damage window (optional)") },
                    placeholder = { Text("Same as default") },
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                    modifier = Modifier.fillMaxWidth(),
                    singleLine = true,
                )

                Row(
                    modifier = Modifier.fillMaxWidth(),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.SpaceBetween,
                ) {
                    Text("Require photo evidence on claims", modifier = Modifier.weight(1f))
                    Switch(checked = requirePhoto, onCheckedChange = { requirePhoto = it })
                }
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.SpaceBetween,
                ) {
                    Text("Allow filing after window expires", modifier = Modifier.weight(1f))
                    Switch(checked = allowExpired, onCheckedChange = { allowExpired = it })
                }

                Button(
                    onClick = { save() },
                    enabled = !saving,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    Text(if (saving) "Saving…" else "Save return policy")
                }
            }
        }
    }
}
