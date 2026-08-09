package com.pegasusx.supplier.ui.screens.exceptions

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.SupplierExceptionRow
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import kotlinx.coroutines.launch
import com.pegasusx.supplier.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ExceptionsScreen(
    ops: SupplierOperationsRepository,
    onBack: () -> Unit,
    onOpenClaims: () -> Unit = {},
) {
    var rows by remember { mutableStateOf<List<SupplierExceptionRow>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var busyKey by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = ops.getExceptions()
                rows = if (resp.isSuccessful) resp.body()?.exceptions.orEmpty() else emptyList()
                if (!resp.isSuccessful) error = "Failed (${resp.code()})"
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    fun resolve(row: SupplierExceptionRow) {
        scope.launch {
            val kind = row.kind.uppercase()
            val key = "$kind:${row.orderId}"
            busyKey = key
            try {
                val resolveId = when (kind) {
                    "CREDIT_NOTE_DRAFT" -> row.note?.takeIf { it.isNotBlank() } ?: row.orderId
                    else -> row.orderId
                }
                val body = when (kind) {
                    "CREDIT_NOTE_DRAFT" ->
                        row.note?.takeIf { it.isNotBlank() }?.let { mapOf("credit_note_id" to it) }
                            ?: emptyMap()
                    else -> emptyMap()
                }
                val resp = ops.resolveException(kind, resolveId, body)
                if (!resp.isSuccessful) error = "Resolve failed (${resp.code()})"
                else load()
            } catch (e: Exception) {
                error = e.message
            } finally {
                busyKey = null
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Exceptions") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
                actions = {
                    TextButton(onClick = onOpenClaims) { Text("Claims") }
                },
            )
        },
    ) { padding ->
        when {
            loading -> PegasusLoadingState("Loading exceptions…", "Supplier exception queue")
            error != null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Exceptions unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            rows.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Empty,
                headline = "No exceptions",
                body = "Operational exceptions will appear here. Use Claims for post-delivery OS&D.",
                modifier = Modifier.padding(padding),
                actionLabel = "Open claims",
                onAction = onOpenClaims,
            )
            else -> Box(Modifier.padding(padding)) {
                ExceptionsList(rows = rows, busyKey = busyKey, onResolve = ::resolve)
            }
        }
    }
}
