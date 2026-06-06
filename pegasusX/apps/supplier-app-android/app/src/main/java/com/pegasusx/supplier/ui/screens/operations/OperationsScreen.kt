package com.pegasusx.supplier.ui.screens.operations

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun OperationsScreen(ops: SupplierOperationsRepository, onBack: () -> Unit) {
    var busy by remember { mutableStateOf(false) }
    val snackbar = remember { SnackbarHostState() }
    val scope = rememberCoroutineScope()

    fun triggerReplenishment() {
        busy = true
        scope.launch {
            try {
                val resp = ops.triggerReplenishment()
                if (resp.isSuccessful) {
                    val body = resp.body()
                    snackbar.showSnackbar("Replenishment ${body?.status ?: "queued"}")
                } else {
                    snackbar.showSnackbar("Failed (${resp.code()})")
                }
            } catch (e: Exception) {
                snackbar.showSnackbar(e.message ?: "Network error")
            } finally {
                busy = false
            }
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Operations") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbar) },
    ) { padding ->
        Column(
            modifier = Modifier
                .padding(padding)
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(PegasusSpacing.lg),
            verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
        ) {
            Text(
                "Supplier operator actions. Broadcast and payment-bypass remain portal-only in v1.",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Button(onClick = { triggerReplenishment() }, enabled = !busy, modifier = Modifier.fillMaxWidth()) {
                Text("Trigger replenishment")
            }
        }
    }
}
