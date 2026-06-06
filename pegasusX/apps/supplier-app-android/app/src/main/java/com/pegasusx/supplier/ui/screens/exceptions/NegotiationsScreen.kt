package com.pegasusx.supplier.ui.screens.exceptions

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.NegotiationProposalRow
import com.pegasusx.supplier.data.model.NegotiationResolveRequest
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun NegotiationsScreen(ops: SupplierOperationsRepository, onBack: () -> Unit) {
    var rows by remember { mutableStateOf<List<NegotiationProposalRow>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var busyId by remember { mutableStateOf<String?>(null) }
    val snackbar = remember { SnackbarHostState() }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = ops.getNegotiationsPending()
                rows = if (resp.isSuccessful) resp.body()?.data.orEmpty() else emptyList()
                if (!resp.isSuccessful) error = "Failed (${resp.code()})"
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    fun resolve(proposalId: String, action: String) {
        busyId = proposalId
        scope.launch {
            try {
                val resp = ops.resolveNegotiation(NegotiationResolveRequest(proposalId, action, null))
                if (resp.isSuccessful) {
                    snackbar.showSnackbar("Negotiation $action")
                    load()
                } else {
                    snackbar.showSnackbar("Failed (${resp.code()})")
                }
            } catch (e: Exception) {
                snackbar.showSnackbar(e.message ?: "Network error")
            } finally {
                busyId = null
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Negotiations") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbar) },
    ) { padding ->
        when {
            loading -> SupplierLoadingState("Loading negotiations…", "Pending proposals")
            error != null -> SupplierStatePane(
                kind = SupplierStateKind.Error,
                headline = "Negotiations unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            rows.isEmpty() -> SupplierStatePane(
                kind = SupplierStateKind.Empty,
                headline = "No pending negotiations",
                body = "Driver quantity proposals appear here.",
                modifier = Modifier.padding(padding),
            )
            else -> LazyColumn(
                modifier = Modifier.padding(padding).fillMaxSize(),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                items(rows, key = { it.proposalId }) { row ->
                    val busy = busyId == row.proposalId
                    ElevatedCard(Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                            Text(row.orderId, style = MaterialTheme.typography.titleMedium)
                            Text("${row.items.size} line items · Driver ${row.driverId}", style = MaterialTheme.typography.bodySmall)
                            Row(horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.xs)) {
                                TextButton(onClick = { resolve(row.proposalId, "APPROVE") }, enabled = !busy) { Text("Approve") }
                                TextButton(onClick = { resolve(row.proposalId, "REJECT") }, enabled = !busy) { Text("Reject") }
                            }
                        }
                    }
                }
            }
        }
    }
}
