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
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierOpsListCard
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun FactoriesScreen(
    ops: SupplierOperationsRepository,
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
            loading -> SupplierLoadingState("Loading factories…", "Topology nodes", Modifier.padding(padding))
            error != null && factories.isEmpty() -> SupplierStatePane(
                kind = SupplierStateKind.Error,
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
                SupplierStatePane(
                    kind = SupplierStateKind.Empty,
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
                        supporting = "${factory.lat}, ${factory.lng}",
                        status = if (factory.isActive) "ACTIVE" else "INACTIVE",
                    )
                }
            }
        }
    }

    if (showAdd) {
        AddFactoryDialog(
            onDismiss = { showAdd = false },
            onSave = { name, lat, lng ->
                scope.launch {
                    appendFactoryNode(ops, name, lat, lng)
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
    onDismiss: () -> Unit,
    onSave: (String, Double, Double) -> Unit,
) {
    var name by remember { mutableStateOf("") }
    var lat by remember { mutableStateOf("41.3111") }
    var lng by remember { mutableStateOf("69.2797") }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Add factory") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                OutlinedTextField(value = name, onValueChange = { name = it }, label = { Text("Name") }, singleLine = true)
                OutlinedTextField(value = lat, onValueChange = { lat = it }, label = { Text("Latitude") }, keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal), singleLine = true)
                OutlinedTextField(value = lng, onValueChange = { lng = it }, label = { Text("Longitude") }, keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal), singleLine = true)
            }
        },
        confirmButton = {
            TextButton(
                onClick = {
                    val latValue = lat.toDoubleOrNull()
                    val lngValue = lng.toDoubleOrNull()
                    if (name.isNotBlank() && latValue != null && lngValue != null) {
                        onSave(name, latValue, lngValue)
                    }
                },
                enabled = name.isNotBlank(),
            ) { Text("Save") }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}
