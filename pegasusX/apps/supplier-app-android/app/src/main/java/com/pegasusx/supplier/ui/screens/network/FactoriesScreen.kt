package com.pegasusx.supplier.ui.screens.network

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.remote.GeocodeApi
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.AddressLocationField
import com.pegasusx.supplier.ui.components.AddressLocationValue
import com.pegasus.design.PegasusLoadingState
import com.pegasusx.supplier.ui.components.SupplierOpsListCard
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
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
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
                actions = {
                    IconButton(onClick = { showAdd = true }) {
                        Icon(Icons.Default.Add, contentDescription = "Add factory")
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
            else -> LazyColumn(
                modifier = Modifier.padding(padding),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                items(factories, key = { it.factoryId }) { factory ->
                    SupplierOpsListCard(
                        headline = factory.name.ifBlank { factory.factoryId },
                        supporting = factory.address.ifBlank { "Coordinates on file" },
                        status = if (factory.isActive) "ACTIVE" else "INACTIVE",
                    )
                }
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

@Composable
private fun AddFactoryDialog(
    geocodeApi: GeocodeApi,
    onDismiss: () -> Unit,
    onSave: (String, AddressLocationValue) -> Unit,
) {
    var name by remember { mutableStateOf("") }
    var location by remember { mutableStateOf(AddressLocationValue(lat = 41.3111, lng = 69.2797)) }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Add factory") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                OutlinedTextField(value = name, onValueChange = { name = it }, label = { Text("Name") }, singleLine = true)
                AddressLocationField(
                    geocodeApi = geocodeApi,
                    value = location,
                    onValueChange = { location = it },
                    label = "Factory address",
                )
            }
        },
        confirmButton = {
            TextButton(
                onClick = {
                    if (name.isNotBlank() && location.address.isNotBlank() && location.lat != 0.0 && location.lng != 0.0) {
                        onSave(name, location)
                    }
                },
                enabled = name.isNotBlank() && location.address.isNotBlank(),
            ) { Text("Save") }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}
