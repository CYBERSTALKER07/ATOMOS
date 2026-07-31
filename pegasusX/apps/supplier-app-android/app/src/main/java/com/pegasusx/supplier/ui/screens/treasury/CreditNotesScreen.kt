package com.pegasusx.supplier.ui.screens.treasury

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasus.design.PegasusLoadingState
import com.pegasusx.supplier.data.model.CreditNoteRow
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CreditNotesScreen(ops: SupplierOperationsRepository, onBack: () -> Unit) {
    var loading by remember { mutableStateOf(true) }
    var rows by remember { mutableStateOf<List<CreditNoteRow>>(emptyList()) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(Unit) {
        scope.launch {
            loading = true
            try {
                val resp = ops.getCreditNotes()
                rows = if (resp.isSuccessful) resp.body()?.creditNotes ?: emptyList() else emptyList()
            } finally {
                loading = false
            }
        }
    }

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
        if (loading) {
            PegasusLoadingState("Loading…", modifier = Modifier.padding(padding))
        } else {
            Column(Modifier.padding(padding).padding(PegasusSpacing.md)) {
                rows.forEach { row ->
                    Text("${row.creditNoteId} · order ${row.orderId} · ${row.totalGrossMinor} minor")
                }
            }
        }
    }
}
