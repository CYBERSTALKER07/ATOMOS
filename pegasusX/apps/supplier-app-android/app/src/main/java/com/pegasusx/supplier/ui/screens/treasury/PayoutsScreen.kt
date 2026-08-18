package com.pegasusx.supplier.ui.screens.treasury

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material3.Button
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
import androidx.compose.ui.Modifier
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.supplier.data.model.PayoutBatch
import com.pegasusx.supplier.data.model.PayoutBatchGenerateRequest
import com.pegasusx.supplier.data.model.PayoutRailInfo
import com.pegasusx.supplier.data.model.SupplierPayoutPolicy
import com.pegasusx.supplier.data.model.SupplierPayoutPolicyPatch
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import com.pegasusx.supplier.util.SupplierIdempotencyKeys
import kotlinx.coroutines.launch
import java.text.NumberFormat
import java.util.Locale

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PayoutsScreen(
    ops: SupplierOperationsRepository,
    onBack: () -> Unit,
) {
    var rail by remember { mutableStateOf<PayoutRailInfo?>(null) }
    var policy by remember { mutableStateOf<SupplierPayoutPolicy?>(null) }
    var draftMode by remember { mutableStateOf("HQ_SUPPLIER") }
    var policyReason by remember { mutableStateOf("") }
    var batches by remember { mutableStateOf<List<PayoutBatch>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var status by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    var periodStart by remember { mutableStateOf("") }
    var periodEnd by remember { mutableStateOf("") }
    val scope = rememberCoroutineScope()
    val fmt = remember { NumberFormat.getInstance(Locale("uz", "UZ")) }

    fun load() {
        loading = true
        error = null
        scope.launch {
            try {
                val railResp = ops.getPayoutRail()
                val listResp = ops.listPayoutBatches()
                val policyResp = ops.getPayoutPolicy()
                if (railResp.isSuccessful) rail = railResp.body()
                if (policyResp.isSuccessful) {
                    policy = policyResp.body()
                    draftMode = policyResp.body()?.payoutMode?.ifBlank { "HQ_SUPPLIER" } ?: "HQ_SUPPLIER"
                }
                if (listResp.isSuccessful) {
                    batches = listResp.body()?.batches.orEmpty()
                } else {
                    error = "Failed (${listResp.code()})"
                    batches = emptyList()
                }
            } catch (e: Exception) {
                error = e.message ?: "Network error"
                batches = emptyList()
            } finally {
                loading = false
            }
        }
    }

    fun generate() {
        if (periodStart.isBlank() || periodEnd.isBlank()) {
            status = "Period start and end are required"
            return
        }
        busy = true
        scope.launch {
            try {
                val key = SupplierIdempotencyKeys.payoutGenerate(
                    SupplierIdempotencyKeys.supplierScopeId(),
                    periodStart.trim(),
                    periodEnd.trim(),
                )
                val resp = ops.generatePayoutBatch(
                    PayoutBatchGenerateRequest(periodStart = periodStart.trim(), periodEnd = periodEnd.trim()),
                    key,
                )
                status = if (resp.isSuccessful) {
                    "Batch generated — export CSV, process at bank, then mark-paid."
                } else {
                    "Generate failed (${resp.code()})"
                }
                load()
            } catch (e: Exception) {
                status = e.message
            } finally {
                busy = false
            }
        }
    }

    fun exportCsv(id: String) {
        busy = true
        scope.launch {
            try {
                val resp = ops.exportPayoutBatch(id)
                status = if (resp.isSuccessful) {
                    val bytes = resp.body()?.bytes()?.size ?: 0
                    "CSV exported ($bytes bytes). Process at bank, then mark-paid."
                } else {
                    "Export failed (${resp.code()})"
                }
                load()
            } catch (e: Exception) {
                status = e.message
            } finally {
                busy = false
            }
        }
    }

    fun markPaid(id: String) {
        busy = true
        scope.launch {
            try {
                val resp = ops.markPayoutBatchPaid(id)
                status = if (resp.isSuccessful) {
                    resp.body()?.message?.ifBlank { "Marked paid" } ?: "Marked paid"
                } else {
                    "Mark-paid failed (${resp.code()})"
                }
                load()
            } catch (e: Exception) {
                status = e.message
            } finally {
                busy = false
            }
        }
    }

    fun dispatchLive(id: String) {
        busy = true
        scope.launch {
            try {
                val resp = ops.dispatchPayoutBatch(id, true)
                val body = resp.body()
                val errText = resp.errorBody()?.string().orEmpty()
                status = when {
                    body?.code == "no_live_rail" || body?.error == "no_live_rail" ->
                        body.message.ifBlank { "no_live_rail — export CSV, then mark-paid" }
                    errText.contains("no_live_rail") ->
                        "no_live_rail — export CSV, process at bank, then mark-paid."
                    resp.isSuccessful -> body?.message?.ifBlank { "Dispatch attempted" } ?: "Dispatch attempted"
                    else -> "Dispatch failed (${resp.code()})"
                }
                load()
            } catch (e: Exception) {
                status = e.message
            } finally {
                busy = false
            }
        }
    }

    fun savePolicy() {
        if (policyReason.isBlank()) {
            status = "Reason is required to change payout mode"
            return
        }
        busy = true
        scope.launch {
            try {
                val resp = ops.patchPayoutPolicy(
                    SupplierPayoutPolicyPatch(payoutMode = draftMode, reason = policyReason.trim()),
                )
                status = if (resp.isSuccessful) {
                    policyReason = ""
                    "Payout mode saved. Bank-file rail is unchanged (no_live_rail)."
                } else {
                    "Policy save failed (${resp.code()})"
                }
                load()
            } catch (e: Exception) {
                status = e.message
            } finally {
                busy = false
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Payouts") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
                actions = {
                    IconButton(onClick = { load() }) {
                        Icon(Icons.Default.Refresh, contentDescription = "Refresh")
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading && batches.isEmpty() && rail == null ->
                PegasusLoadingState("Loading payouts…", body = "", modifier = Modifier.padding(padding))
            error != null && batches.isEmpty() && rail == null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Payouts unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            else -> LazyColumn(
                modifier = Modifier.padding(padding),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                item {
                    ElevatedCard(Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                            Text("Rail honesty", style = MaterialTheme.typography.titleSmall)
                            Text(
                                rail?.message?.ifBlank { null }
                                    ?: "Bank-file rail: generate → export CSV → mark-paid. Not a live bank.",
                                style = MaterialTheme.typography.bodySmall,
                            )
                            Text(
                                "Live: ${if (rail?.isLive == true) "yes" else "no"}",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                    }
                }
                item {
                    ElevatedCard(Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                            Text("Payout policy", style = MaterialTheme.typography.titleSmall)
                            Text(
                                "Mode ${policy?.payoutMode ?: "HQ_SUPPLIER"} · source ${policy?.source ?: "DEFAULT"}. Does not enable a live PSP.",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                            Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                                if (draftMode == "HQ_SUPPLIER") {
                                    Button(onClick = { draftMode = "HQ_SUPPLIER" }, enabled = !busy) { Text("HQ_SUPPLIER") }
                                } else {
                                    OutlinedButton(onClick = { draftMode = "HQ_SUPPLIER" }, enabled = !busy) { Text("HQ_SUPPLIER") }
                                }
                                if (draftMode == "WAREHOUSE_LOCAL") {
                                    Button(onClick = { draftMode = "WAREHOUSE_LOCAL" }, enabled = !busy) { Text("WAREHOUSE_LOCAL") }
                                } else {
                                    OutlinedButton(onClick = { draftMode = "WAREHOUSE_LOCAL" }, enabled = !busy) { Text("WAREHOUSE_LOCAL") }
                                }
                            }
                            OutlinedTextField(
                                value = policyReason,
                                onValueChange = { policyReason = it },
                                label = { Text("Reason (required)") },
                                modifier = Modifier.fillMaxWidth(),
                            )
                            Button(onClick = { savePolicy() }, enabled = !busy && policyReason.isNotBlank()) {
                                Text("Save mode")
                            }
                        }
                    }
                }
                item {
                    ElevatedCard(Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                            Text("Generate period", style = MaterialTheme.typography.titleSmall)
                            OutlinedTextField(
                                value = periodStart,
                                onValueChange = { periodStart = it },
                                label = { Text("Start YYYY-MM-DD") },
                                modifier = Modifier.fillMaxWidth(),
                            )
                            OutlinedTextField(
                                value = periodEnd,
                                onValueChange = { periodEnd = it },
                                label = { Text("End YYYY-MM-DD") },
                                modifier = Modifier.fillMaxWidth(),
                            )
                            Button(onClick = { generate() }, enabled = !busy) { Text("Generate") }
                        }
                    }
                }
                status?.let {
                    item { Text(it, style = MaterialTheme.typography.bodySmall) }
                }
                if (batches.isEmpty()) {
                    item {
                        PegasusStatePane(
                            kind = PegasusStateKind.Empty,
                            headline = "No payout batches",
                            body = "Generate a period to create a bank-file batch.",
                        )
                    }
                }
                items(batches, key = { it.batchId }) { batch ->
                    ElevatedCard(Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                            Text("${batch.status} · ${fmt.format(batch.netPayoutMinor)} ${batch.currency}")
                            Text(
                                "${batch.periodStart} → ${batch.periodEnd}",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                            Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                                OutlinedButton(onClick = { exportCsv(batch.batchId) }, enabled = !busy) { Text("Export") }
                                Button(onClick = { markPaid(batch.batchId) }, enabled = !busy) { Text("Mark paid") }
                                OutlinedButton(onClick = { dispatchLive(batch.batchId) }, enabled = !busy) { Text("Live") }
                            }
                        }
                    }
                }
            }
        }
    }
}
