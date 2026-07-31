package com.pegasusx.supplier.ui.screens.treasury

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
import com.pegasusx.supplier.data.model.CreditProfileRow
import com.pegasusx.supplier.data.model.RetailerCreditProfilePatchRequest
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import java.util.UUID
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CreditProfilesScreen(ops: SupplierOperationsRepository, onBack: () -> Unit) {
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var rows by remember { mutableStateOf<List<CreditProfileRow>>(emptyList()) }
    var busyId by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = ops.getCreditProfiles()
                rows = if (resp.isSuccessful) resp.body()?.profiles ?: emptyList() else emptyList()
                if (!resp.isSuccessful) error = "Failed (${resp.code()})"
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    fun setStatus(row: CreditProfileRow, status: String) {
        scope.launch {
            busyId = row.retailerId
            try {
                val key = "supplier-credit-profile:$status:${row.retailerId}:${UUID.randomUUID()}"
                val resp = ops.patchRetailerCreditProfile(
                    RetailerCreditProfilePatchRequest(
                        retailerId = row.retailerId,
                        creditLimitMinor = row.creditLimitMinor,
                        status = status,
                        reason = "collections_desk",
                    ),
                    key,
                )
                if (!resp.isSuccessful) error = "Update failed (${resp.code()})"
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
                title = { Text("Credit profiles") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> PegasusLoadingState(
                title = "Loading…",
                body = "Retailer credit profiles",
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
                headline = "No credit profiles",
                body = "Retailer credit lines for this supplier will appear here.",
                modifier = Modifier.padding(padding),
                actionLabel = "Refresh",
                onAction = { load() },
            )
            else -> LazyColumn(
                modifier = Modifier.padding(padding),
                contentPadding = PaddingValues(PegasusSpacing.md),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                items(rows, key = { it.retailerId }) { row ->
                    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                        Column(
                            Modifier.padding(PegasusSpacing.md),
                            verticalArrangement = Arrangement.spacedBy(4.dp),
                        ) {
                            Text(row.retailerId, style = MaterialTheme.typography.labelMedium)
                            Text("${row.status} · risk ${row.riskTier.ifBlank { "—" }}")
                            Text(
                                "Limit ${row.creditLimitMinor} · bal ${row.currentBalanceMinor} · avail ${row.availableCreditMinor}",
                            )
                            val busy = busyId == row.retailerId
                            Row(
                                horizontalArrangement = Arrangement.spacedBy(8.dp),
                                modifier = Modifier.padding(top = PegasusSpacing.xs),
                            ) {
                                when (row.status.uppercase()) {
                                    "ACTIVE" -> OutlinedButton(
                                        onClick = { setStatus(row, "FROZEN") },
                                        enabled = !busy,
                                    ) { Text("Freeze") }
                                    "FROZEN" -> Button(
                                        onClick = { setStatus(row, "ACTIVE") },
                                        enabled = !busy,
                                    ) { Text("Unfreeze") }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
