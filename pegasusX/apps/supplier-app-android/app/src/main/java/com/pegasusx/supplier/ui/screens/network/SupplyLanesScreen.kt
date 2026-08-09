package com.pegasusx.supplier.ui.screens.network

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.SupplierSupplyLaneRow
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import com.pegasusx.supplier.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SupplyLanesScreen(ops: SupplierOperationsRepository, onBack: () -> Unit) {
    var lanes by remember { mutableStateOf<List<SupplierSupplyLaneRow>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = ops.getSupplyLanes()
                if (resp.isSuccessful) lanes = resp.body()?.lanes ?: emptyList()
                else error = "Failed (${resp.code()})"
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
                title = { Text("Supply lanes") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> PegasusLoadingState("Loading supply lanes…", "Warehouse lanes")
            error != null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Supply lanes unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            lanes.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Empty,
                headline = "No lanes",
                body = "No active warehouse lanes. Configure nodes on topology.",
                modifier = Modifier.padding(padding),
            )
            else -> SupplyLanesList(
                lanes = lanes,
                modifier = Modifier.padding(padding)
            )
        }
    }
}
