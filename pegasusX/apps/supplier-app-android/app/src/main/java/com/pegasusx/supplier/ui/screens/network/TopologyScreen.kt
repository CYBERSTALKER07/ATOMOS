package com.pegasusx.supplier.ui.screens.network

import androidx.compose.ui.res.stringResource

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.ElevatedCard
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.SupplierTopologyFactoryInput
import com.pegasusx.supplier.data.model.SupplierTopologyResponse
import com.pegasusx.supplier.data.model.SupplierTopologyUpdateRequest
import com.pegasusx.supplier.data.model.SupplierTopologyWarehouseInput
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import com.pegasusx.supplier.R



@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun TopologyScreen(ops: SupplierOperationsRepository, onBack: () -> Unit) {
    var topology by remember { mutableStateOf<SupplierTopologyResponse?>(null) }
    var loading by remember { mutableStateOf(true) }
    var saving by remember { mutableStateOf(false) }
    var editing by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val warehouseDrafts = remember { mutableStateListOf<WarehouseDraft>() }
    val factoryDrafts = remember { mutableStateListOf<FactoryDraft>() }
    val scope = rememberCoroutineScope()

    fun applyDrafts(data: SupplierTopologyResponse) {
        warehouseDrafts.clear()
        factoryDrafts.clear()
        data.warehouses.forEachIndexed { index, node ->
            warehouseDrafts.add(
                WarehouseDraft(
                    key = node.warehouseId.ifEmpty { "wh-$index" },
                    warehouseId = node.warehouseId.takeIf { it.isNotBlank() },
                    name = node.name,
                    lat = node.lat.toString(),
                    lng = node.lng.toString(),
                    coverageRadiusKm = node.coverageRadiusKm.toString(),
                    isActive = node.isActive,
                    isOnShift = node.isOnShift,
                    transferMode = node.transferMode?.takeIf { it.isNotBlank() } ?: "TRUCK",
                ),
            )
        }
        data.factories.forEachIndexed { index, node ->
            factoryDrafts.add(
                FactoryDraft(
                    key = node.factoryId.ifEmpty { "fc-$index" },
                    factoryId = node.factoryId.takeIf { it.isNotBlank() },
                    name = node.name,
                    lat = node.lat.toString(),
                    lng = node.lng.toString(),
                    isActive = node.isActive,
                ),
            )
        }
    }

    fun load() {
        scope.launch {
            loading = true
            error = null
            try {
                val resp = ops.getTopology()
                if (resp.isSuccessful) {
                    val body = resp.body()
                    topology = body
                    if (body != null) applyDrafts(body)
                } else {
                    error = "Failed (${resp.code()})"
                }
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    fun save() {
        scope.launch {
            saving = true
            error = null
            try {
                if (warehouseDrafts.isEmpty()) {
                    error = "At least one warehouse is required"
                    return@launch
                }
                val request = SupplierTopologyUpdateRequest(
                    warehouses = warehouseDrafts.map { draft ->
                        val existing = topology?.warehouses?.firstOrNull { it.warehouseId == draft.warehouseId }
                        SupplierTopologyWarehouseInput(
                            warehouseId = draft.warehouseId,
                            name = draft.name.trim(),
                            lat = draft.lat.toDoubleOrNull() ?: throw IllegalArgumentException("Invalid latitude"),
                            lng = draft.lng.toDoubleOrNull() ?: throw IllegalArgumentException("Invalid longitude"),
                            coverageRadiusKm = draft.coverageRadiusKm.toDoubleOrNull() ?: 50.0,
                            isActive = draft.isActive,
                            isOnShift = draft.isOnShift,
                            transferMode = draft.transferMode,
                            coLocateWithFactoryId = existing?.coLocateWithFactoryId,
                            primaryFactoryId = existing?.primaryFactoryId,
                            secondaryFactoryId = existing?.secondaryFactoryId,
                            assignedFactoryIds = existing?.assignedFactoryIds?.takeIf { it.isNotEmpty() },
                            countryCode = existing?.countryCode?.takeIf { it.isNotBlank() },
                            coverageCities = existing?.coverageCities?.takeIf { it.isNotEmpty() },
                        )
                    },
                    factories = factoryDrafts.map { draft ->
                        val existing = topology?.factories?.firstOrNull { it.factoryId == draft.factoryId }
                        SupplierTopologyFactoryInput(
                            factoryId = draft.factoryId,
                            name = draft.name.trim(),
                            lat = draft.lat.toDoubleOrNull() ?: throw IllegalArgumentException("Invalid latitude"),
                            lng = draft.lng.toDoubleOrNull() ?: throw IllegalArgumentException("Invalid longitude"),
                            isActive = draft.isActive,
                            countryCode = existing?.countryCode?.takeIf { it.isNotBlank() },
                        )
                    },
                )
                val resp = ops.updateTopology(request)
                if (resp.isSuccessful) {
                    val body = resp.body()
                    topology = body
                    if (body != null) applyDrafts(body)
                    editing = false
                } else {
                    error = "Save failed (${resp.code()})"
                }
            } catch (e: Exception) {
                error = e.message ?: "save_failed"
            } finally {
                saving = false
            }
        }
    }

    LaunchedEffect(Unit) { load() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Factories & warehouses") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.common_action_back))
                    }
                },
                actions = {
                    if (!loading && error == null) {
                        if (editing) {
                            TextButton(enabled = !saving, onClick = {
                                topology?.let { applyDrafts(it) }
                                editing = false
                            }) { Text("Cancel") }
                            TextButton(enabled = !saving, onClick = { save() }) {
                                Text(if (saving) "Saving…" else "Save")
                            }
                        } else {
                            TextButton(onClick = { editing = true }) { Text("Edit") }
                        }
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> PegasusLoadingState("Loading topology…", "Node topology")
            error != null && topology == null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Topology unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { load() },
            )
            editing -> LazyColumn(
                modifier = Modifier.padding(padding).fillMaxSize(),
                contentPadding = PaddingValues(PegasusSpacing.lg),
                verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
            ) {
                if (error != null) {
                    item { Text(error!!, color = MaterialTheme.colorScheme.error) }
                }
                item { SectionLabel("Warehouses") }
                items(warehouseDrafts, key = { it.key }) { draft ->
                    ElevatedCard(Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                            DraftField("Name", draft.name) { value ->
                                val index = warehouseDrafts.indexOfFirst { it.key == draft.key }
                                if (index >= 0) warehouseDrafts[index] = draft.copy(name = value)
                            }
                            DraftField("Latitude", draft.lat) { value ->
                                val index = warehouseDrafts.indexOfFirst { it.key == draft.key }
                                if (index >= 0) warehouseDrafts[index] = draft.copy(lat = value)
                            }
                            DraftField("Longitude", draft.lng) { value ->
                                val index = warehouseDrafts.indexOfFirst { it.key == draft.key }
                                if (index >= 0) warehouseDrafts[index] = draft.copy(lng = value)
                            }
                            DraftField("Coverage km", draft.coverageRadiusKm) { value ->
                                val index = warehouseDrafts.indexOfFirst { it.key == draft.key }
                                if (index >= 0) warehouseDrafts[index] = draft.copy(coverageRadiusKm = value)
                            }
                        }
                    }
                }
                item {
                    Button(onClick = {
                        warehouseDrafts.add(
                            WarehouseDraft(
                                key = "new-wh-${System.currentTimeMillis()}",
                                warehouseId = null,
                                name = "Warehouse ${warehouseDrafts.size + 1}",
                                lat = "41.2995",
                                lng = "69.2401",
                                coverageRadiusKm = "50",
                                isActive = true,
                                isOnShift = true,
                                transferMode = "TRUCK",
                            ),
                        )
                    }) { Text("Add warehouse") }
                }
                item { SectionLabel("Factories") }
                items(factoryDrafts, key = { it.key }) { draft ->
                    ElevatedCard(Modifier.fillMaxWidth()) {
                        Column(Modifier.padding(PegasusSpacing.lg), verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                            DraftField("Name", draft.name) { value ->
                                val index = factoryDrafts.indexOfFirst { it.key == draft.key }
                                if (index >= 0) factoryDrafts[index] = draft.copy(name = value)
                            }
                            DraftField("Latitude", draft.lat) { value ->
                                val index = factoryDrafts.indexOfFirst { it.key == draft.key }
                                if (index >= 0) factoryDrafts[index] = draft.copy(lat = value)
                            }
                            DraftField("Longitude", draft.lng) { value ->
                                val index = factoryDrafts.indexOfFirst { it.key == draft.key }
                                if (index >= 0) factoryDrafts[index] = draft.copy(lng = value)
                            }
                        }
                    }
                }
                item {
                    Button(onClick = {
                        factoryDrafts.add(
                            FactoryDraft(
                                key = "new-fc-${System.currentTimeMillis()}",
                                factoryId = null,
                                name = "Factory ${factoryDrafts.size + 1}",
                                lat = "41.3111",
                                lng = "69.2797",
                                isActive = true,
                            ),
                        )
                    }) { Text("Add factory") }
                }
            }
            topology == null || (topology!!.warehouses.isEmpty() && topology!!.factories.isEmpty()) -> PegasusStatePane(
                kind = PegasusStateKind.Empty,
                headline = "No nodes",
                body = "No warehouses or factories configured.",
                modifier = Modifier.padding(padding),
                actionLabel = "Configure",
                onAction = { editing = true },
            )
            else -> {
                val data = topology!!
                LazyColumn(
                    modifier = Modifier.padding(padding).fillMaxSize(),
                    contentPadding = PaddingValues(PegasusSpacing.lg),
                    verticalArrangement = Arrangement.spacedBy(PegasusSpacing.md),
                ) {
                    item { SectionLabel("Warehouses (${data.warehouses.size})") }
                    items(data.warehouses, key = { it.warehouseId }) { node ->
                        NodeCard(node.name, node.lat, node.lng, "${node.coverageRadiusKm} km coverage")
                    }
                    item { SectionLabel("Factories (${data.factories.size})") }
                    items(data.factories, key = { it.factoryId }) { node ->
                        NodeCard(node.name, node.lat, node.lng, if (node.isActive) "Active" else "Inactive")
                    }
                }
            }
        }
    }
}

