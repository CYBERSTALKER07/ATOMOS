package com.pegasusx.supplier.ui.screens.network

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import com.pegasusx.supplier.data.remote.GeocodeApi
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.AddressLocationField
import com.pegasusx.supplier.ui.components.AddressLocationValue
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierOpsListCard
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun WarehousesScreen(
    ops: SupplierOperationsRepository,
    geocodeApi: GeocodeApi,
    onBack: () -> Unit,
) {
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var warehouses by remember { mutableStateOf(emptyList<com.pegasusx.supplier.data.model.SupplierTopologyWarehouse>()) }
    var showAdd by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = ops.getTopology()
                warehouses = if (resp.isSuccessful) resp.body()?.warehouses.orEmpty() else emptyList()
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
                title = { Text("Warehouses") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
                actions = {
                    IconButton(onClick = { showAdd = true }) {
                        Icon(Icons.Default.Add, contentDescription = "Add warehouse")
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> SupplierLoadingState("Loading warehouses…", "Topology nodes", Modifier.padding(padding))
            error != null && warehouses.isEmpty() -> SupplierStatePane(
                kind = SupplierStateKind.Error,
                headline = "Warehouses unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            warehouses.isEmpty() -> Box(
                modifier = Modifier
                    .padding(padding)
                    .fillMaxSize(),
                contentAlignment = Alignment.Center,
            ) {
                SupplierStatePane(
                    kind = SupplierStateKind.Empty,
                    headline = "No warehouses",
                    body = "Add your first distribution node to start fulfilling orders.",
                    actionLabel = "Add first warehouse",
                    onAction = { showAdd = true },
                )
            }
            else -> LazyColumn(
                modifier = Modifier.padding(padding),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                items(warehouses, key = { it.warehouseId }) { warehouse ->
                    val locationLabel = warehouse.address.ifBlank { "Coordinates on file" }
                    SupplierOpsListCard(
                        headline = warehouse.name.ifBlank { warehouse.warehouseId },
                        supporting = "Radius ${warehouse.coverageRadiusKm} km · $locationLabel",
                        status = if (warehouse.isActive) "ACTIVE" else "INACTIVE",
                    )
                }
            }
        }
    }

    if (showAdd) {
        AddWarehouseDialog(
            geocodeApi = geocodeApi,
            onDismiss = { showAdd = false },
            onSave = { name, location, radius ->
                scope.launch {
                    appendWarehouseNode(ops, name, location, radius)
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
private fun AddWarehouseDialog(
    geocodeApi: GeocodeApi,
    onDismiss: () -> Unit,
    onSave: (String, AddressLocationValue, Double) -> Unit,
) {
    val (defaultLat, defaultLng) = defaultWarehouseCoordinates()
    var name by remember { mutableStateOf("") }
    var location by remember {
        mutableStateOf(AddressLocationValue(lat = defaultLat, lng = defaultLng))
    }
    var radius by remember { mutableStateOf("50") }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Add warehouse") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                OutlinedTextField(value = name, onValueChange = { name = it }, label = { Text("Name") }, singleLine = true)
                AddressLocationField(
                    geocodeApi = geocodeApi,
                    value = location,
                    onValueChange = { location = it },
                    label = "Warehouse address",
                )
                OutlinedTextField(value = radius, onValueChange = { radius = it }, label = { Text("Coverage km") }, keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal), singleLine = true)
            }
        },
        confirmButton = {
            TextButton(
                onClick = {
                    if (name.isNotBlank() && location.address.isNotBlank() && location.lat != 0.0 && location.lng != 0.0) {
                        onSave(name, location, radius.toDoubleOrNull() ?: 50.0)
                    }
                },
                enabled = name.isNotBlank() && location.address.isNotBlank(),
            ) { Text("Save") }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}
