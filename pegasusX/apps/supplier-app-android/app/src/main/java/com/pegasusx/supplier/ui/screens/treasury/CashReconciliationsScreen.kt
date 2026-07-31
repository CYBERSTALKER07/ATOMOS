package com.pegasusx.supplier.ui.screens.treasury

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.supplier.data.model.CashReconciliationRow
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CashReconciliationsScreen(ops: SupplierOperationsRepository, onBack: () -> Unit) {
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var rows by remember { mutableStateOf<List<CashReconciliationRow>>(emptyList()) }
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

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Cash reconciliations") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> PegasusLoadingState("Loading…", modifier = Modifier.padding(padding))
            error != null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            else -> Column(Modifier.padding(padding).padding(PegasusSpacing.md)) {
                rows.forEach { row ->
                    ElevatedCard(Modifier.fillMaxWidth().padding(vertical = PegasusSpacing.xs)) {
                        Column(Modifier.padding(PegasusSpacing.md)) {
                            Text(row.reconciliationId, style = MaterialTheme.typography.labelMedium)
                            Text("Driver ${row.driverId} · ${row.status}")
                            Text("Diff ${row.differenceMinor} minor")
                        }
                    }
                }
            }
        }
    }
}
