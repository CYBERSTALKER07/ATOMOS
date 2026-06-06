package com.pegasusx.supplier.ui.screens.treasury

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.PaymentLedgerEntry
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun LedgerScreen(ops: SupplierOperationsRepository, onBack: () -> Unit) {
    var rows by remember { mutableStateOf<List<PaymentLedgerEntry>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = ops.getPaymentLedger()
                rows = if (resp.isSuccessful) resp.body()?.items.orEmpty() else emptyList()
                if (!resp.isSuccessful) error = "Failed (${resp.code()})"
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
                title = { Text("Payment ledger") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> SupplierLoadingState("Loading ledger…", "Treasury entries")
            error != null -> SupplierStatePane(
                kind = SupplierStateKind.Error,
                headline = "Ledger unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            rows.isEmpty() -> SupplierStatePane(
                kind = SupplierStateKind.Empty,
                headline = "No ledger entries",
                body = "Payment movements will appear here.",
                modifier = Modifier.padding(padding),
            )
            else -> LazyColumn(
                modifier = Modifier.padding(padding).fillMaxSize(),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                items(rows, key = { it.ledgerEntryId }) { row ->
                    ElevatedCard(Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(PegasusSpacing.lg)) {
                            Text(row.entryType, style = MaterialTheme.typography.titleMedium)
                            Text("${row.currency} ${row.amountMinor}", style = MaterialTheme.typography.bodyMedium)
                            row.orderId?.let { Text("Order $it", style = MaterialTheme.typography.bodySmall) }
                            Text(row.occurredAt, style = MaterialTheme.typography.bodySmall)
                        }
                    }
                }
            }
        }
    }
}
