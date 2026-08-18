package com.pegasusx.supplier.ui.screens.exceptions

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.supplier.data.model.ApproveClaimRequest
import com.pegasusx.supplier.data.model.RejectClaimRequest
import com.pegasusx.supplier.data.model.SupplierClaim
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import com.pegasusx.supplier.R

private val settlementModes = listOf(
    "LEDGER_ONLY" to "Ledger only",
    "STORE_CREDIT" to "Store credit",
    "GATEWAY_REFUND" to "Card refund (GP)",
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ClaimsScreen(
    ops: SupplierOperationsRepository,
    onBack: () -> Unit,
    onOpenClaimChargebacks: () -> Unit = {},
) {
    var claims by remember { mutableStateOf<List<SupplierClaim>>(emptyList()) }
    var statusFilter by remember { mutableStateOf("OPEN") }
    var settlementMode by remember { mutableStateOf("LEDGER_ONLY") }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var busyId by remember { mutableStateOf<String?>(null) }
    var lastSettlement by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val status = statusFilter.ifBlank { null }
                val resp = ops.listSupplierClaims(status = status, limit = 50)
                claims = if (resp.isSuccessful) resp.body()?.claims.orEmpty() else emptyList()
                if (!resp.isSuccessful) error = "Failed (${resp.code()})"
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(statusFilter) { load() }

    fun approve(claimId: String) {
        scope.launch {
            busyId = claimId
            lastSettlement = null
            try {
                val resp = ops.approveClaim(
                    claimId,
                    ApproveClaimRequest(
                        resolutionNote = "approved_via_supplier_android",
                        settlementMode = settlementMode,
                        skipGatewayRefund = settlementMode != "GATEWAY_REFUND",
                    ),
                )
                if (!resp.isSuccessful) {
                    error = "Approve failed (${resp.code()})"
                } else {
                    val s = resp.body()?.settlement
                    if (s != null) {
                        lastSettlement =
                            "${s.mode} · ${s.amountMinor} · refund=${s.gatewayRefunded} · id=${s.chargebackId ?: "—"}"
                    }
                    load()
                }
            } catch (e: Exception) {
                error = e.message
            } finally {
                busyId = null
            }
        }
    }

    fun reject(claimId: String) {
        scope.launch {
            busyId = claimId
            try {
                val resp = ops.rejectClaim(
                    claimId,
                    RejectClaimRequest(resolutionNote = "rejected_via_supplier_android"),
                )
                if (!resp.isSuccessful) error = "Reject failed (${resp.code()})"
                else load()
            } catch (e: Exception) {
                error = e.message
            } finally {
                busyId = null
            }
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Claims") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
                actions = {
                    TextButton(onClick = onOpenClaimChargebacks) { Text("Chargebacks") }
                },
            )
        },
    ) { padding ->
        Column(
            Modifier
                .padding(padding)
                .fillMaxSize()
                .padding(PegasusSpacing.md),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
        ) {
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                FilterChip(
                    selected = statusFilter == "OPEN",
                    onClick = { statusFilter = "OPEN" },
                    label = { Text("OPEN") },
                )
                FilterChip(
                    selected = statusFilter == "UNDER_REVIEW",
                    onClick = { statusFilter = "UNDER_REVIEW" },
                    label = { Text("Review") },
                )
                FilterChip(
                    selected = statusFilter.isEmpty(),
                    onClick = { statusFilter = "" },
                    label = { Text("All") },
                )
            }

            Text("Settlement mode", style = MaterialTheme.typography.labelMedium)
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                settlementModes.forEach { (value, label) ->
                    FilterChip(
                        selected = settlementMode == value,
                        onClick = { settlementMode = value },
                        label = { Text(label) },
                    )
                }
            }
            lastSettlement?.let {
                Text(it, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.primary)
            }

            when {
                loading -> PegasusLoadingState("Loading claims…", "Logistics claim queue")
                error != null -> PegasusStatePane(
                    kind = PegasusStateKind.Error,
                    headline = "Claims unavailable",
                    body = error!!,
                    actionLabel = "Retry",
                    onAction = { load() },
                )
                claims.isEmpty() -> PegasusStatePane(
                    kind = PegasusStateKind.Empty,
                    headline = "No claims",
                    body = "Post-delivery retailer claims appear here within the 48h window.",
                    actionLabel = "Refresh",
                    onAction = { load() },
                )
                else -> LazyColumn(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                    items(claims, key = { it.claimId }) { claim ->
                        ClaimCard(
                            claim = claim,
                            busy = busyId == claim.claimId,
                            settlementLabel = settlementModes.find { it.first == settlementMode }?.second
                                ?: settlementMode,
                            onApprove = { approve(claim.claimId) },
                            onReject = { reject(claim.claimId) },
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun ClaimCard(
    claim: SupplierClaim,
    busy: Boolean,
    settlementLabel: String,
    onApprove: () -> Unit,
    onReject: () -> Unit,
) {
    Card(Modifier.fillMaxWidth()) {
        Column(
            Modifier.padding(PegasusSpacing.md),
            verticalArrangement = Arrangement.spacedBy(4.dp),
        ) {
            Text(stringResource(R.string.mobile_supplier_ui_claimtype_status, claim.claimType, claim.status), style = MaterialTheme.typography.titleSmall)
            Text(claim.claimId, style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.primary)
            Text(stringResource(R.string.mobile_supplier_ui_order_orderid_retailer_retailerid, claim.orderId, claim.retailerId), style = MaterialTheme.typography.bodySmall)
            Text(
                "Amount ${claim.amountMinor ?: 0} ${com.pegasus.design.moneyCurrency(claim.currency)}",
                style = MaterialTheme.typography.bodyMedium,
            )
            claim.description?.takeIf { it.isNotBlank() }?.let {
                Text(it, style = MaterialTheme.typography.bodySmall)
            }
            if (claim.status == "OPEN" || claim.status == "UNDER_REVIEW") {
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    Button(onClick = onApprove, enabled = !busy) {
                        Text(stringResource(R.string.mobile_supplier_ui_approve_settlementlabel, settlementLabel))
                    }
                    OutlinedButton(onClick = onReject, enabled = !busy) {
                        Text("Reject")
                    }
                }
            }
        }
    }
}
