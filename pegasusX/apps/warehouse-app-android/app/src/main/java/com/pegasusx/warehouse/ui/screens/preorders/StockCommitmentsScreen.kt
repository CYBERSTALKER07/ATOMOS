package com.pegasusx.warehouse.ui.screens.preorders

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.GridItemSpan
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.remote.WarehouseApi
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun StockCommitmentsScreen(api: WarehouseApi, onBack: (() -> Unit)? = null) {
    var rows by remember { mutableStateOf<List<com.pegasusx.warehouse.data.model.StockCommitmentRow>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    val scope = rememberCoroutineScope()

    LaunchedEffect(Unit) {
        scope.launch {
            loading = true
            try {
                val resp = api.getStockCommitments()
                rows = if (resp.isSuccessful) resp.body()?.items.orEmpty() else emptyList()
            } finally {
                loading = false
            }
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Stock commitments") },
                navigationIcon = {
                    if (onBack != null) {
                        IconButton(onClick = onBack) {
                            Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                        }
                    }
                },
            )
        },
    ) { padding ->
        if (loading) {
            Box(Modifier.fillMaxSize().padding(padding), contentAlignment = androidx.compose.ui.Alignment.Center) {
                CircularProgressIndicator()
            }
        } else {
            LazyVerticalGrid(
                columns = GridCells.Adaptive(minSize = 340.dp),
                modifier = Modifier.padding(padding).padding(16.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                items(rows, key = { it.skuId }) { row ->
                    ElevatedCard(Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(12.dp)) {
                            Text(row.name ?: row.skuId, style = MaterialTheme.typography.titleSmall)
                            Text(stringResource(R.string.mobile_warehouse_ui_available_availableqty_asap_reservedasap_scheduled_reservedscheduled, row.availableQty, row.reservedAsap, row.reservedScheduled))
                            Text(stringResource(R.string.mobile_warehouse_ui_on_hand_onhand, row.onHand), style = MaterialTheme.typography.bodySmall)
                            if (row.deficitQty > 0) Text(stringResource(R.string.mobile_warehouse_ui_short_deficitqty, row.deficitQty), color = MaterialTheme.colorScheme.error)
                        }
                    }
                }
            }
        }
    }
}
