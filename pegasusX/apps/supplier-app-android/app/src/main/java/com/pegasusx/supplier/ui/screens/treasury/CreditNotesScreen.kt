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
import com.pegasusx.supplier.data.model.CreditNoteRow
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import java.util.UUID
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CreditNotesScreen(ops: SupplierOperationsRepository, onBack: () -> Unit) {
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var rows by remember { mutableStateOf<List<CreditNoteRow>>(emptyList()) }
    var busyId by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = ops.getCreditNotes()
                rows = if (resp.isSuccessful) resp.body()?.creditNotes ?: emptyList() else emptyList()
                if (!resp.isSuccessful) error = "Failed (${resp.code()})"
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    fun issue(id: String) {
        scope.launch {
            busyId = id
            try {
                val key = "supplier-credit-note-issue:$id:${UUID.randomUUID()}"
                val resp = ops.issueCreditNote(id, key)
                if (!resp.isSuccessful) error = "Issue failed (${resp.code()})"
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
                title = { Text("Credit notes") },
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
                body = "Draft credit notes",
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
                headline = "No draft credit notes",
                body = "Draft credit notes ready to issue will appear here.",
                modifier = Modifier.padding(padding),
                actionLabel = "Refresh",
                onAction = { load() },
            )
            else -> LazyColumn(
                modifier = Modifier.padding(padding),
                contentPadding = PaddingValues(PegasusSpacing.md),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                items(rows, key = { it.creditNoteId }) { row ->
                    ElevatedCard(modifier = Modifier.fillMaxWidth()) {
                        Column(
                            Modifier.padding(PegasusSpacing.md),
                            verticalArrangement = Arrangement.spacedBy(4.dp),
                        ) {
                            Text(row.creditNoteId, style = MaterialTheme.typography.labelMedium)
                            Text("Order ${row.orderId} · ${row.status}")
                            Text("${row.totalGrossMinor} minor")
                            if (row.status.equals("DRAFT", ignoreCase = true)) {
                                Button(
                                    onClick = { issue(row.creditNoteId) },
                                    enabled = busyId != row.creditNoteId,
                                    modifier = Modifier.padding(top = PegasusSpacing.xs),
                                ) {
                                    Text("Issue")
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
