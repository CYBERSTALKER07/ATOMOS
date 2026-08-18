package com.pegasusx.supplier.ui.screens.treasury

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
import com.pegasusx.supplier.data.model.PaymentLedgerEntry
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import com.pegasusx.supplier.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ClaimChargebacksScreen(
    ops: SupplierOperationsRepository,
    onBack: () -> Unit,
) {
    var items by remember { mutableStateOf<List<PaymentLedgerEntry>>(emptyList()) }
    var orderFilter by remember { mutableStateOf("") }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = ops.listClaimChargebacks(
                    limit = 100,
                    orderId = orderFilter.trim().ifBlank { null },
                )
                items = if (resp.isSuccessful) resp.body()?.items.orEmpty() else emptyList()
                if (!resp.isSuccessful) error = "Failed (${resp.code()})"
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    val total = items.sumOf { it.amountMinor }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Claim chargebacks") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
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
            OutlinedTextField(
                value = orderFilter,
                onValueChange = { orderFilter = it },
                label = { Text("Filter order id") },
                modifier = Modifier.fillMaxWidth(),
                singleLine = true,
            )
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(onClick = { load() }) { Text("Refresh") }
                Text(
                    stringResource(R.string.mobile_supplier_ui_size_rows_total_total_minor, items.size, total),
                    style = MaterialTheme.typography.bodySmall,
                    modifier = Modifier.padding(top = 12.dp),
                )
            }

            when {
                loading -> PegasusLoadingState("Loading claim chargebacks…", "Ledger rows from claim approve")
                error != null -> PegasusStatePane(
                    kind = PegasusStateKind.Error,
                    headline = "Unavailable",
                    body = error!!,
                    actionLabel = "Retry",
                    onAction = { load() },
                )
                items.isEmpty() -> PegasusStatePane(
                    kind = PegasusStateKind.Empty,
                    headline = "No claim chargebacks",
                    body = "Approve a logistics claim to create chargeback_clm_* ledger rows.",
                    actionLabel = "Refresh",
                    onAction = { load() },
                )
                else -> LazyColumn(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                    items(items, key = { it.ledgerEntryId.ifBlank { it.referenceId ?: it.occurredAt } }) { row ->
                        Card(Modifier.fillMaxWidth()) {
                            Column(Modifier.padding(PegasusSpacing.md), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                                Text(
                                    "${row.amountMinor} ${com.pegasus.design.moneyCurrency(row.currency)}",
                                    style = MaterialTheme.typography.titleSmall,
                                )
                                Text(stringResource(R.string.mobile_supplier_ui_order_orderid_2, row.orderId ?: "—"), style = MaterialTheme.typography.bodySmall)
                                Text(stringResource(R.string.mobile_supplier_ui_ref_referenceid, row.referenceId ?: "—"),
                                    style = MaterialTheme.typography.labelSmall,
                                )
                                Text(
                                    row.source ?: row.entryType,
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                                )
                                Text(
                                    row.occurredAt.ifBlank { row.createdAt },
                                    style = MaterialTheme.typography.labelSmall,
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}
