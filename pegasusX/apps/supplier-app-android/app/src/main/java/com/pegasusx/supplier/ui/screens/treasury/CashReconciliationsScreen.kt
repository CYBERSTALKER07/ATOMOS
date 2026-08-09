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
import com.pegasusx.supplier.data.model.CashReconciliationRow
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import java.util.UUID
import kotlinx.coroutines.launch
import com.pegasusx.supplier.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CashReconciliationsScreen(ops: SupplierOperationsRepository, onBack: () -> Unit) {
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var rows by remember { mutableStateOf<List<CashReconciliationRow>>(emptyList()) }
    var busyId by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = ops.getCashReconciliations()
                rows = if (resp.isSuccessful) resp.body()?.reconciliations ?: emptyList() else emptyList()
                if (!resp.isSuccessful) error = "Failed (${resp.code()})"
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    fun accept(id: String) {
        scope.launch {
            busyId = id
            try {
                val key = "supplier-cash-recon-accept:$id:${UUID.randomUUID()}"
                val resp = ops.acceptCashReconciliation(id, key)
                if (!resp.isSuccessful) error = "Accept failed (${resp.code()})"
                else load()
            } catch (e: Exception) {
                error = e.message
            } finally {
                busyId = null
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Cash reconciliations") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> PegasusLoadingState(
                title = stringResource(R.string.warehouse_portal_bins_text_loading),
                body = "Driver cash reconciliations",
                modifier = Modifier.padding(padding),
            )
            error != null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            rows.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Empty,
                headline = "No open discrepancies",
                body = "Driver cash reconciliations appear here when declared cash differs from expected.",
                modifier = Modifier.padding(padding),
                actionLabel = "Refresh",
                onAction = { load() },
            )
            else -> LazyColumn(
                modifier = Modifier.padding(padding),
                contentPadding = PaddingValues(PegasusSpacing.md),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                items(rows, key = { it.reconciliationId }) { row ->
                    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                        Column(
                            Modifier.padding(PegasusSpacing.md),
                            verticalArrangement = Arrangement.spacedBy(4.dp),
                        ) {
                            Text(row.reconciliationId, style = MaterialTheme.typography.labelMedium)
                            Text(stringResource(R.string.mobile_supplier_ui_driver_driverid_status, row.driverId, row.status))
                            Text(stringResource(R.string.mobile_supplier_ui_diff_differenceminor_minor, row.differenceMinor))
                            val open = row.status.equals("PENDING", ignoreCase = true) ||
                                row.status.equals("DISPUTED", ignoreCase = true)
                            if (open) {
                                Button(
                                    onClick = { accept(row.reconciliationId) },
                                    enabled = busyId != row.reconciliationId,
                                    modifier = Modifier.padding(top = PegasusSpacing.xs),
                                ) {
                                    Text("Accept")
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
