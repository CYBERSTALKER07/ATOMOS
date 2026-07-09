package com.pegasusx.supplier.ui.screens.treasury

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.ReconciliationMismatchRow
import com.pegasusx.supplier.data.model.SettlementAuthorityResponse
import com.pegasusx.supplier.data.model.SettlementAuthorityRow
import com.pegasusx.supplier.data.model.SettlementCurrencyTotal
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import java.text.NumberFormat
import java.util.Locale

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PaymentsScreen(ops: SupplierOperationsRepository, onBack: () -> Unit) {
    var authority by remember { mutableStateOf<SettlementAuthorityResponse?>(null) }
    var mismatches by remember { mutableStateOf<List<ReconciliationMismatchRow>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val fmt = remember { NumberFormat.getInstance(Locale("uz", "UZ")) }

    fun money(minor: Long, currency: String) = "${fmt.format(minor)} $currency"

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val authorityResp = ops.getPaymentSettlementAuthority()
                val mismatchResp = ops.getPaymentReconciliationMismatches()
                if (authorityResp.isSuccessful) authority = authorityResp.body()
                if (mismatchResp.isSuccessful) mismatches = mismatchResp.body()?.items ?: emptyList()
                if (!authorityResp.isSuccessful && !mismatchResp.isSuccessful) {
                    error = "Failed to load settlement authority"
                }
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Payments") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        val data = authority
        when {
            loading -> PegasusLoadingState("Loading settlement authority…", "Payment settlement")
            error != null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Payment authority unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            data == null -> PegasusStatePane(
                kind = PegasusStateKind.Empty,
                headline = "No data",
                body = "No settlement authority data available.",
                modifier = Modifier.padding(padding),
            )
            else -> LazyColumn(
                modifier = Modifier.padding(padding).fillMaxSize(),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                item { SectionLabel("Scope") }
                item {
                    ElevatedCard(Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                            KeyValue("Supplier scope", data.supplierId.ifEmpty { "(global)" })
                            KeyValue("Grouped rows", data.count.toString())
                            KeyValue("Total entries", data.entryCountTotal.toString())
                            KeyValue("Reconciliation groups", mismatches.size.toString())
                        }
                    }
                }

                item { SectionLabel("Totals by currency") }
                if (data.totalsByCurrency.isEmpty()) {
                    item { EmptyLine("No totals available.") }
                } else {
                    items(data.totalsByCurrency, key = { it.currency }) { row -> TotalRow(row, ::money) }
                }

                item { SectionLabel("Reconciliation mismatches") }
                if (mismatches.isEmpty()) {
                    item { EmptyLine("No non-zero mismatches detected.") }
                } else {
                    items(mismatches, key = { "${it.gateway}-${it.currency}" }) { row -> MismatchRow(row, ::money) }
                }

                item { SectionLabel("Settlement groups") }
                if (data.items.isEmpty()) {
                    item { EmptyLine("No settlement groups found.") }
                } else {
                    items(data.items, key = { "${it.gateway}-${it.entryType}-${it.currency}" }) { row -> GroupRow(row, ::money) }
                }
            }
        }
    }
}

@Composable
private fun SectionLabel(title: String) {
    Text(title, style = MaterialTheme.typography.titleSmall, color = MaterialTheme.colorScheme.primary)
}

@Composable
private fun EmptyLine(text: String) {
    Text(text, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.outline)
}

@Composable
private fun KeyValue(label: String, value: String) {
    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
        Text(label, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.outline)
        Text(value, style = MaterialTheme.typography.bodyMedium)
    }
}

@Composable
private fun TotalRow(row: SettlementCurrencyTotal, money: (Long, String) -> String) {
    ElevatedCard(Modifier.fillMaxWidth()) {
        Row(
            Modifier.fillMaxWidth().padding(PegasusSpacing.lg),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            Text(row.currency, style = MaterialTheme.typography.titleMedium)
            Column(horizontalAlignment = androidx.compose.ui.Alignment.End) {
                Text(money(row.amountMinorTotal, row.currency), style = MaterialTheme.typography.bodyLarge)
                Text("${row.entryCount} entries", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.outline)
            }
        }
    }
}

@Composable
private fun MismatchRow(row: ReconciliationMismatchRow, money: (Long, String) -> String) {
    ElevatedCard(Modifier.fillMaxWidth()) {
        Column(Modifier.padding(PegasusSpacing.lg)) {
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                Text("${row.gateway} · ${row.currency}", style = MaterialTheme.typography.titleMedium)
                Text(money(row.netAmountMinor, row.currency), style = MaterialTheme.typography.bodyLarge, color = MaterialTheme.colorScheme.error)
            }
            Text(
                "Credit ${money(row.creditAmountMinorTotal, row.currency)} · Debit ${money(row.debitAmountMinorTotal, row.currency)} · ${row.entryCountTotal} entries",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.outline,
            )
        }
    }
}

@Composable
private fun GroupRow(row: SettlementAuthorityRow, money: (Long, String) -> String) {
    ElevatedCard(Modifier.fillMaxWidth()) {
        Column(Modifier.padding(PegasusSpacing.lg)) {
            Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                Text("${row.gateway} · ${row.entryType}", style = MaterialTheme.typography.titleMedium)
                Text(money(row.amountMinorTotal, row.currency), style = MaterialTheme.typography.bodyLarge)
            }
            Text(
                "${row.currency} · ${row.entryCount} entries",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.outline,
            )
        }
    }
}
