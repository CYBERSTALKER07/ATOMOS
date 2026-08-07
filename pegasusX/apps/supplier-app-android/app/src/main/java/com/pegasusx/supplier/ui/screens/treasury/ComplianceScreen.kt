package com.pegasusx.supplier.ui.screens.treasury

import androidx.compose.ui.res.stringResource

import android.content.Intent
import android.net.Uri
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.supplier.data.model.ComplianceDashboardResponse
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ComplianceScreen(ops: SupplierOperationsRepository, onBack: () -> Unit) {
    var data by remember { mutableStateOf<ComplianceDashboardResponse?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val context = LocalContext.current

    fun openReceipt(orderId: String) {
        scope.launch {
            try {
                val resp = ops.getOrderReceipt(orderId)
                val meta = resp.body()
                val url = meta?.htmlUrl?.ifBlank { null }
                    ?: meta?.qrUrl?.ifBlank { null }
                    ?: meta?.pdfUrl?.ifBlank { null }
                if (resp.isSuccessful && !url.isNullOrBlank()) {
                    context.startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
                }
            } catch (_: Exception) {
                /* ignore — receipt may not exist for open fiscal rows */
            }
        }
    }

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = ops.getComplianceDashboard(100)
                if (resp.isSuccessful) {
                    data = resp.body()
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

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Compliance audit") },
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
            loading -> PegasusLoadingState("Loading compliance…", "Fiscal & credit audit")
            error != null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Compliance unavailable",
                body = error ?: "",
                actionLabel = "Retry",
                onAction = { load() },
                modifier = Modifier
                    .fillMaxSize()
                    .padding(padding),
            )
            else -> {
                val summary = data?.summary
                LazyColumn(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(padding),
                    contentPadding = PaddingValues(PegasusSpacing.lg),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                ) {
                    item {
                        ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                            Column(
                                modifier = Modifier.padding(PegasusSpacing.lg),
                                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.xs),
                            ) {
                                Text("Summary", style = MaterialTheme.typography.titleMedium)
                                Text(stringResource(R.string.mobile_supplier_ui_open_fiscal_openfiscalcount_0, summary?.openFiscalCount ?: 0))
                                Text(stringResource(R.string.mobile_supplier_ui_force_completes_forcecompletecount_0, summary?.forceCompleteCount ?: 0))
                                Text(stringResource(R.string.mobile_supplier_ui_claim_mismatches_claimmismatchcount_0, summary?.claimMismatchCount ?: 0))
                                Text(stringResource(R.string.mobile_supplier_ui_credit_freezes_creditfreezecount_0, summary?.creditFreezeCount ?: 0))
                            }
                        }
                    }
                    item {
                        Text("Open fiscal", style = MaterialTheme.typography.titleSmall)
                    }
                    items(data?.openFiscal.orEmpty(), key = { it.orderId }) { row ->
                        ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                            Column(modifier = Modifier.padding(PegasusSpacing.md)) {
                                Text(row.orderId, style = MaterialTheme.typography.titleSmall)
                                Text(stringResource(R.string.mobile_supplier_ui_status_fiscalstatus, row.status, row.fiscalStatus))
                                Text(stringResource(R.string.mobile_supplier_ui_totalminor_currency, row.totalMinor, row.currency))
                                TextButton(onClick = { openReceipt(row.orderId) }) {
                                    Text("View receipt")
                                }
                            }
                        }
                    }
                    item {
                        Text("Force-completes", style = MaterialTheme.typography.titleSmall)
                    }
                    items(data?.forceCompletes.orEmpty(), key = { it.orderId + it.completedAt }) { row ->
                        ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                            Column(modifier = Modifier.padding(PegasusSpacing.md)) {
                                Text(row.orderId, style = MaterialTheme.typography.titleSmall)
                                Text(stringResource(R.string.mobile_supplier_ui_reason_ifblank_actor_ifblank_2, row.reasonCode.ifBlank { "—" }, row.actorId.ifBlank { "—" }))
                                Row {
                                    TextButton(onClick = { openReceipt(row.orderId) }) {
                                        Text("View receipt")
                                    }
                                }
                            }
                        }
                    }
                    item {
                        Text("Credit freezes", style = MaterialTheme.typography.titleSmall)
                    }
                    items(data?.creditFreezes.orEmpty(), key = { it.retailerId + it.status }) { row ->
                        ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                            Column(modifier = Modifier.padding(PegasusSpacing.md)) {
                                Text(row.retailerId, style = MaterialTheme.typography.titleSmall)
                                Text(row.status)
                                Text(stringResource(R.string.mobile_supplier_ui_balance_currentbalanceminor_limit_creditlimitminor, row.currentBalanceMinor, row.creditLimitMinor))
                            }
                        }
                    }
                    item {
                        Text("Claim mismatches", style = MaterialTheme.typography.titleSmall)
                    }
                    items(data?.claimMismatches.orEmpty(), key = { it.claimId }) { row ->
                        ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                            Column(modifier = Modifier.padding(PegasusSpacing.md)) {
                                Text(row.claimId, style = MaterialTheme.typography.titleSmall)
                                Text(stringResource(R.string.mobile_supplier_ui_order_orderid_2, row.orderId))
                                Text(row.mismatchReason)
                            }
                        }
                    }
                }
            }
        }
    }
}
