package com.pegasusx.supplier.ui.screens.manifests

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.SupplierManifestExceptionRow
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ManifestExceptionsScreen(
    ops: SupplierOperationsRepository,
    onBack: () -> Unit,
    onOpenManifest: (String) -> Unit,
) {
    var rows by remember { mutableStateOf<List<SupplierManifestExceptionRow>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var escalatedOnly by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = ops.getManifestExceptions(escalatedOnly)
                rows = if (resp.isSuccessful) resp.body()?.exceptions.orEmpty() else emptyList()
                if (!resp.isSuccessful) error = "Failed (${resp.code()})"
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(escalatedOnly) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Gate exceptions") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> SupplierLoadingState("Loading exceptions…", "Manifest gate")
            error != null -> SupplierStatePane(
                kind = SupplierStateKind.Error,
                headline = "Exceptions unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            rows.isEmpty() -> SupplierStatePane(
                kind = SupplierStateKind.Empty,
                headline = "No exceptions",
                body = "No manifest gate exceptions in the current window.",
                modifier = Modifier.padding(padding),
            )
            else -> LazyColumn(
                modifier = Modifier.padding(padding).fillMaxSize(),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm),
            ) {
                item {
                    Row(verticalAlignment = androidx.compose.ui.Alignment.CenterVertically) {
                        Checkbox(checked = escalatedOnly, onCheckedChange = { escalatedOnly = it })
                        Text("Escalated only")
                    }
                }
                items(rows, key = { it.exceptionId }) { row ->
                    ElevatedCard(
                        Modifier
                            .fillMaxWidth()
                            .clickable { onOpenManifest(row.manifestId) },
                    ) {
                        Column(Modifier.padding(PegasusSpacing.lg)) {
                            Text(row.reason, style = MaterialTheme.typography.titleMedium)
                            Text("Manifest ${row.manifestId.take(8)}…", style = MaterialTheme.typography.bodySmall)
                            Text("Order ${row.orderId.take(8)}… · attempts ${row.attemptCount}", style = MaterialTheme.typography.bodySmall)
                            if (row.escalated) Text("Escalated", color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.labelSmall)
                        }
                    }
                }
            }
        }
    }
}
