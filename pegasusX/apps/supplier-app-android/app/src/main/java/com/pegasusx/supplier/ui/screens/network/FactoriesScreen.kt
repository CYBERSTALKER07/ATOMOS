package com.pegasusx.supplier.ui.screens.network

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.remote.GeocodeApi
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun FactoriesScreen(
    ops: SupplierOperationsRepository,
    geocodeApi: GeocodeApi,
    onBack: () -> Unit,
) {
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var factories by remember { mutableStateOf(emptyList<com.pegasusx.supplier.data.model.SupplierTopologyFactory>()) }
    var showAdd by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = ops.getTopology()
                factories = if (resp.isSuccessful) resp.body()?.factories.orEmpty() else emptyList()
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
                title = { Text("Factories") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
                actions = {
                    IconButton(onClick = { showAdd = true }) {
                        Icon(Icons.Default.Add, contentDescription = stringResource(R.string.supplier_portal_factories_components_factory_form_text_add_factory))
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> PegasusLoadingState("Loading factories…", "Topology nodes", Modifier.padding(padding))
            error != null && factories.isEmpty() -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Factories unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            factories.isEmpty() -> Box(
                modifier = Modifier
                    .padding(padding)
                    .fillMaxSize(),
                contentAlignment = Alignment.Center,
            ) {
                PegasusStatePane(
                    kind = PegasusStateKind.Empty,
                    headline = "No factories",
                    body = "Add a production node linked to your warehouse network.",
                    actionLabel = "Add first factory",
                    onAction = { showAdd = true },
                )
            }
            else -> Box(modifier = Modifier.padding(padding)) {
                FactoryList(factories = factories)
            }
        }
    }

    if (showAdd) {
        AddFactoryDialog(
            geocodeApi = geocodeApi,
            onDismiss = { showAdd = false },
            onSave = { name, location ->
                scope.launch {
                    appendFactoryNode(ops, name, location)
                        .onSuccess {
                            showAdd = false
                            load()
                        }
                        .onFailure { error = it.message }
                }
            },
        )
    }
}
