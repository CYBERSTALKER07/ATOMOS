package com.pegasusx.warehouse.ui.screens.payment

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Cancel
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.pegasusx.warehouse.data.model.PaymentGateway
import com.pegasusx.warehouse.data.remote.WarehouseOperationsRepository
import com.pegasusx.warehouse.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import com.pegasusx.warehouse.R

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PaymentConfigScreen(
    opsRepository: WarehouseOperationsRepository,
    onBack: (() -> Unit)? = null,
) {
    var gateways by remember { mutableStateOf<List<PaymentGateway>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = opsRepository.getPaymentConfig()
                gateways = if (resp.isSuccessful) resp.body()?.gateways.orEmpty() else emptyList()
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
                title = { Text("Payment config") },
                navigationIcon = { if (onBack != null) { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back)) } } },
                actions = {
                    TextButton(onClick = { load() }) { Text("Refresh") }
                },
            )
        },
    ) { padding ->
        when {
            loading -> Box(
                Modifier.fillMaxSize().padding(padding),
                contentAlignment = Alignment.Center,
            ) { CircularProgressIndicator() }

            error != null -> Box(
                Modifier.fillMaxSize().padding(padding),
                contentAlignment = Alignment.Center,
            ) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(error!!, color = MaterialTheme.colorScheme.error)
                    Spacer(Modifier.height(PegasusSpacing.md))
                    Button(onClick = { load() }) { Text("Retry") }
                }
            }

            gateways.isEmpty() -> Box(
                Modifier.fillMaxSize().padding(padding),
                contentAlignment = Alignment.Center,
            ) { Text("No payment gateways configured") }

            else -> LazyVerticalGrid(
                columns = GridCells.Adaptive(minSize = 340.dp),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                horizontalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                modifier = Modifier.fillMaxSize().padding(padding),
            ) {
                items(gateways, key = { it.gatewayName }) { gw ->
                    ListItem(
                        headlineContent = { Text(gw.gatewayName) },
                        supportingContent = { Text(stringResource(R.string.mobile_warehouse_ui_provider_mode, gw.provider, gw.mode)) },
                        trailingContent = {
                            Icon(
                                if (gw.isActive) Icons.Default.CheckCircle else Icons.Default.Cancel,
                                contentDescription = null,
                                tint = if (gw.isActive) {
                                    MaterialTheme.colorScheme.primary
                                } else {
                                    MaterialTheme.colorScheme.onSurfaceVariant
                                },
                            )
                        },
                    )
                    HorizontalDivider()
                }
            }
        }
    }
}
