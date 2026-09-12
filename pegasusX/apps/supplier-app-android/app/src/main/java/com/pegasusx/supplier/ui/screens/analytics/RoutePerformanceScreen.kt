package com.pegasusx.supplier.ui.screens.analytics

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasus.design.ui.PegasusLoadingState
import com.pegasusx.supplier.data.model.RoutePerformanceRow
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import com.pegasusx.supplier.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RoutePerformanceScreen(ops: SupplierOperationsRepository, onBack: () -> Unit) {
    var loading by remember { mutableStateOf(true) }
    var rows by remember { mutableStateOf<List<RoutePerformanceRow>>(emptyList()) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(Unit) {
        scope.launch {
            loading = true
            try {
                val resp = ops.getRoutePerformance()
                rows = if (resp.isSuccessful) resp.body()?.routes ?: emptyList() else emptyList()
            } finally {
                loading = false
            }
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Route performance") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
            )
        },
    ) { padding ->
        if (loading) {
            PegasusLoadingState(
                title = stringResource(R.string.warehouse_portal_bins_text_loading),
                body = "Fetching route performance metrics.",
                modifier = Modifier.padding(padding),
            )
        } else {
            Column(Modifier.padding(padding).padding(PegasusSpacing.md)) {
                rows.forEach { row ->
                    Text(stringResource(R.string.mobile_supplier_ui_route_routeid_driver_driverid_orderscompleted_orders, row.routeId, row.driverId, row.ordersCompleted))
                }
            }
        }
    }
}
