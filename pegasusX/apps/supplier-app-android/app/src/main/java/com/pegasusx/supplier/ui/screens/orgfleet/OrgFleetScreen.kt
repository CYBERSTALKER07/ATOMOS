package com.pegasusx.supplier.ui.screens.orgfleet

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.*
import com.pegasusx.supplier.data.remote.SupplierApi
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.SupplierLoadingState
import com.pegasusx.supplier.ui.components.SupplierStateKind
import com.pegasusx.supplier.ui.components.SupplierStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch
import java.util.UUID

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun OrgFleetScreen(
    api: SupplierApi,
    ops: SupplierOperationsRepository,
    onBack: () -> Unit,
) {
    var tab by remember { mutableIntStateOf(0) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }
    var topology by remember { mutableStateOf<SupplierTopologyResponse?>(null) }
    var drivers by remember { mutableStateOf<List<FleetDriver>>(emptyList()) }
    var vehicles by remember { mutableStateOf<List<FleetVehicle>>(emptyList()) }
    var orgMembers by remember { mutableStateOf<List<SupplierOrgMember>>(emptyList()) }
    var showDriverDialog by remember { mutableStateOf(false) }
    var showVehicleDialog by remember { mutableStateOf(false) }
    var showOrgDialog by remember { mutableStateOf(false) }
    var memberActionId by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()

    fun reload() {
        scope.launch {
            loading = true
            error = null
            try {
                val topo = ops.getTopology()
                val driverResp = api.getFleetDrivers()
                val vehicleResp = api.getFleetVehicles()
                val orgResp = ops.getOrgMembers()
                if (topo.isSuccessful) topology = topo.body()
                if (driverResp.isSuccessful) drivers = driverResp.body()?.items.orEmpty()
                if (vehicleResp.isSuccessful) vehicles = vehicleResp.body()?.items.orEmpty()
                if (orgResp.isSuccessful) orgMembers = orgResp.body()?.items.orEmpty()
                if (!topo.isSuccessful) error = "Failed to load topology"
            } catch (e: Exception) {
                error = e.message
            } finally {
                loading = false
            }
        }
    }

    LaunchedEffect(Unit) { reload() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Org & fleet") },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
                    }
                },
                actions = {
                    IconButton(onClick = {
                        when (tab) {
                            0 -> showDriverDialog = true
                            1 -> showVehicleDialog = true
                            else -> showOrgDialog = true
                        }
                    }) {
                        Icon(Icons.Default.Add, contentDescription = "Create")
                    }
                },
            )
        },
    ) { padding ->
        when {
            loading -> SupplierLoadingState("Loading org & fleet…", "Topology and rosters")
            error != null -> SupplierStatePane(
                kind = SupplierStateKind.Error,
                headline = "Onboarding unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { reload() },
            )
            else -> Column(Modifier.padding(padding).fillMaxSize()) {
                TabRow(selectedTabIndex = tab) {
                    Tab(selected = tab == 0, onClick = { tab = 0 }, text = { Text("Drivers (${drivers.size})") })
                    Tab(selected = tab == 1, onClick = { tab = 1 }, text = { Text("Vehicles (${vehicles.size})") })
                    Tab(selected = tab == 2, onClick = { tab = 2 }, text = { Text("Org (${orgMembers.size})") })
                }
                when (tab) {
                    0 -> DriverRoster(drivers, topology)
                    1 -> VehicleRoster(vehicles, topology)
                    else -> OrgRoster(
                        members = orgMembers,
                        actionId = memberActionId,
                        onDeactivate = { userId ->
                            scope.launch {
                                memberActionId = userId
                                try {
                                    val resp = ops.deactivateOrgMember(userId, UUID.randomUUID().toString())
                                    if (resp.isSuccessful) orgMembers = resp.body()?.items.orEmpty()
                                } finally {
                                    memberActionId = null
                                }
                            }
                        },
                    )
                }
            }
        }
    }

    val topo = topology
    if (showDriverDialog && topo != null) {
        CreateDriverDialog(
            topology = topo,
            vehicles = vehicles,
            onDismiss = { showDriverDialog = false },
            onCreate = { request ->
                scope.launch {
                    ops.createFleetDriver(request, UUID.randomUUID().toString())
                    showDriverDialog = false
                    reload()
                }
            },
        )
    }
    if (showVehicleDialog && topo != null) {
        CreateVehicleDialog(
            topology = topo,
            onDismiss = { showVehicleDialog = false },
            onCreate = { request ->
                scope.launch {
                    ops.createFleetVehicle(request, UUID.randomUUID().toString())
                    showVehicleDialog = false
                    reload()
                }
            },
        )
    }
    if (showOrgDialog && topo != null) {
        CreateOrgMemberDialog(
            topology = topo,
            onDismiss = { showOrgDialog = false },
            onCreate = { request ->
                scope.launch {
                    ops.createOrgMember(request, UUID.randomUUID().toString())
                    showOrgDialog = false
                    reload()
                }
            },
        )
    }
}

@Composable
private fun DriverRoster(drivers: List<FleetDriver>, topology: SupplierTopologyResponse?) {
    if (drivers.isEmpty()) {
        SupplierStatePane(SupplierStateKind.Empty, "No drivers", "Create a driver to start fleet onboarding.")
        return
    }
    LazyColumn(contentPadding = PaddingValues(PegasusSpacing.lg)) {
        items(drivers, key = { it.driverId }) { driver ->
            ListItem(
                headlineContent = { Text(driver.name) },
                supportingContent = {
                    Text("${nodeLabel(driver.homeNodeType, driver.homeNodeId, topology)} · ${driver.phone}")
                },
            )
        }
    }
}

@Composable
private fun VehicleRoster(vehicles: List<FleetVehicle>, topology: SupplierTopologyResponse?) {
    if (vehicles.isEmpty()) {
        SupplierStatePane(SupplierStateKind.Empty, "No vehicles", "Create a vehicle for driver assignment.")
        return
    }
    LazyColumn(contentPadding = PaddingValues(PegasusSpacing.lg)) {
        items(vehicles, key = { it.vehicleId }) { vehicle ->
            ListItem(
                headlineContent = { Text(vehicle.label ?: vehicle.licensePlate) },
                supportingContent = {
                    Text("${vehicle.licensePlate} · ${nodeLabel(vehicle.homeNodeType, vehicle.homeNodeId, topology)}")
                },
            )
        }
    }
}

@Composable
private fun OrgRoster(
    members: List<SupplierOrgMember>,
    onDeactivate: (String) -> Unit,
    actionId: String?,
) {
    if (members.isEmpty()) {
        SupplierStatePane(SupplierStateKind.Empty, "No org members", "Create warehouse, factory, or payload staff.")
        return
    }
    LazyColumn(contentPadding = PaddingValues(PegasusSpacing.lg)) {
        items(members, key = { it.userId }) { member ->
            ListItem(
                headlineContent = { Text(member.name) },
                supportingContent = {
                    Text("${member.supplierRole} · ${member.phone} · ${if (member.isActive) "Active" else "Inactive"}")
                },
                trailingContent = {
                    if (member.isActive) {
                        TextButton(
                            enabled = actionId != member.userId,
                            onClick = { onDeactivate(member.userId) },
                        ) { Text("Deactivate") }
                    }
                },
            )
        }
    }
}

private fun nodeLabel(type: String, id: String, topology: SupplierTopologyResponse?): String {
    if (topology == null) return id
    if (type == "FACTORY") {
        return topology.factories.find { it.factoryId == id }?.name ?: id
    }
    return topology.warehouses.find { it.warehouseId == id }?.name ?: id
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun CreateDriverDialog(
    topology: SupplierTopologyResponse,
    vehicles: List<FleetVehicle>,
    onDismiss: () -> Unit,
    onCreate: (FleetDriverCreateRequest) -> Unit,
) {
    var name by remember { mutableStateOf("") }
    var phone by remember { mutableStateOf("") }
    var pin by remember { mutableStateOf("") }
    var nodeType by remember { mutableStateOf("WAREHOUSE") }
    var nodeId by remember { mutableStateOf("") }
    var vehicleId by remember { mutableStateOf("") }
    val nodeOptions = if (nodeType == "FACTORY") {
        topology.factories.map { it.factoryId to it.name }
    } else {
        topology.warehouses.map { it.warehouseId to it.name }
    }
    val vehicleOptions = vehicles.filter { it.homeNodeType == nodeType && it.homeNodeId == nodeId }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Create driver") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                OutlinedTextField(name, { name = it }, label = { Text("Name") }, modifier = Modifier.fillMaxWidth())
                OutlinedTextField(phone, { phone = it }, label = { Text("Phone") }, modifier = Modifier.fillMaxWidth())
                OutlinedTextField(pin, { pin = it }, label = { Text("PIN") }, modifier = Modifier.fillMaxWidth())
                NodeTypePicker(nodeType) { nodeType = it; nodeId = ""; vehicleId = "" }
                NodePicker(nodeOptions, nodeId) { nodeId = it; vehicleId = "" }
                if (vehicleOptions.isNotEmpty()) {
                    VehiclePicker(vehicleOptions, vehicleId) { vehicleId = it }
                }
            }
        },
        confirmButton = {
            TextButton(onClick = {
                if (name.isBlank() || phone.isBlank() || pin.isBlank() || nodeId.isBlank()) return@TextButton
                onCreate(
                    FleetDriverCreateRequest(
                        name = name.trim(),
                        phone = phone.trim(),
                        pin = pin,
                        homeNodeType = nodeType,
                        homeNodeId = nodeId,
                        vehicleId = vehicleId.ifBlank { null },
                    ),
                )
            }) { Text("Create") }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}

@Composable
private fun CreateVehicleDialog(
    topology: SupplierTopologyResponse,
    onDismiss: () -> Unit,
    onCreate: (FleetVehicleCreateRequest) -> Unit,
) {
    var label by remember { mutableStateOf("") }
    var plate by remember { mutableStateOf("") }
    var nodeType by remember { mutableStateOf("WAREHOUSE") }
    var nodeId by remember { mutableStateOf("") }
    val nodeOptions = if (nodeType == "FACTORY") {
        topology.factories.map { it.factoryId to it.name }
    } else {
        topology.warehouses.map { it.warehouseId to it.name }
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Create vehicle") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                OutlinedTextField(label, { label = it }, label = { Text("Label") }, modifier = Modifier.fillMaxWidth())
                OutlinedTextField(plate, { plate = it.uppercase() }, label = { Text("License plate") }, modifier = Modifier.fillMaxWidth())
                NodeTypePicker(nodeType) { nodeType = it; nodeId = "" }
                NodePicker(nodeOptions, nodeId) { nodeId = it }
            }
        },
        confirmButton = {
            TextButton(onClick = {
                if (plate.isBlank() || nodeId.isBlank()) return@TextButton
                onCreate(
                    FleetVehicleCreateRequest(
                        label = label.ifBlank { null },
                        licensePlate = plate.trim(),
                        homeNodeType = nodeType,
                        homeNodeId = nodeId,
                    ),
                )
            }) { Text("Create") }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}

@Composable
private fun CreateOrgMemberDialog(
    topology: SupplierTopologyResponse,
    onDismiss: () -> Unit,
    onCreate: (SupplierOrgMemberCreateRequest) -> Unit,
) {
    var name by remember { mutableStateOf("") }
    var email by remember { mutableStateOf("") }
    var phone by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var role by remember { mutableStateOf("WAREHOUSE_ADMIN") }
    var nodeType by remember { mutableStateOf("WAREHOUSE") }
    var nodeId by remember { mutableStateOf("") }
    val nodeOptions = when (role) {
        "FACTORY_ADMIN" -> topology.factories.map { it.factoryId to it.name }
        "ADMIN" -> emptyList()
        else -> if (nodeType == "FACTORY") topology.factories.map { it.factoryId to it.name }
        else topology.warehouses.map { it.warehouseId to it.name }
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Create org member") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                OutlinedTextField(name, { name = it }, label = { Text("Name") }, modifier = Modifier.fillMaxWidth())
                OutlinedTextField(email, { email = it }, label = { Text("Email") }, modifier = Modifier.fillMaxWidth())
                OutlinedTextField(phone, { phone = it }, label = { Text("Phone") }, modifier = Modifier.fillMaxWidth())
                OutlinedTextField(password, { password = it }, label = { Text("Password") }, modifier = Modifier.fillMaxWidth())
                RolePicker(role) { role = it; nodeId = "" }
                if (role == "PAYLOAD") NodeTypePicker(nodeType) { nodeType = it; nodeId = "" }
                if (role != "ADMIN" && nodeOptions.isNotEmpty()) NodePicker(nodeOptions, nodeId) { nodeId = it }
            }
        },
        confirmButton = {
            TextButton(onClick = {
                if (name.isBlank() || phone.isBlank() || password.isBlank()) return@TextButton
                if (role != "ADMIN" && nodeId.isBlank()) return@TextButton
                val warehouseId = if (role == "WAREHOUSE_ADMIN" || (role == "PAYLOAD" && nodeType == "WAREHOUSE")) nodeId else null
                val factoryId = if (role == "FACTORY_ADMIN" || (role == "PAYLOAD" && nodeType == "FACTORY")) nodeId else null
                onCreate(
                    SupplierOrgMemberCreateRequest(
                        name = name.trim(),
                        email = email.ifBlank { null },
                        phone = phone.trim(),
                        password = password,
                        supplierRole = role,
                        assignedWarehouseId = warehouseId,
                        assignedFactoryId = factoryId,
                    ),
                )
            }) { Text("Create") }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun NodeTypePicker(selected: String, onSelect: (String) -> Unit) {
    var expanded by remember { mutableStateOf(false) }
    ExposedDropdownMenuBox(expanded, { expanded = !expanded }) {
        OutlinedTextField(
            value = if (selected == "FACTORY") "Factory" else "Warehouse",
            onValueChange = {},
            readOnly = true,
            label = { Text("Node type") },
            modifier = Modifier.menuAnchor().fillMaxWidth(),
        )
        ExposedDropdownMenu(expanded, { expanded = false }) {
            DropdownMenuItem(text = { Text("Warehouse") }, onClick = { onSelect("WAREHOUSE"); expanded = false })
            DropdownMenuItem(text = { Text("Factory") }, onClick = { onSelect("FACTORY"); expanded = false })
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun NodePicker(options: List<Pair<String, String>>, selected: String, onSelect: (String) -> Unit) {
    var expanded by remember { mutableStateOf(false) }
    val label = options.find { it.first == selected }?.second ?: "Select node"
    ExposedDropdownMenuBox(expanded, { expanded = !expanded }) {
        OutlinedTextField(
            value = label,
            onValueChange = {},
            readOnly = true,
            label = { Text("Home node") },
            modifier = Modifier.menuAnchor().fillMaxWidth(),
        )
        ExposedDropdownMenu(expanded, { expanded = false }) {
            options.forEach { (id, name) ->
                DropdownMenuItem(text = { Text(name) }, onClick = { onSelect(id); expanded = false })
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun VehiclePicker(vehicles: List<FleetVehicle>, selected: String, onSelect: (String) -> Unit) {
    var expanded by remember { mutableStateOf(false) }
    val label = vehicles.find { it.vehicleId == selected }?.licensePlate ?: "Assign later"
    ExposedDropdownMenuBox(expanded, { expanded = !expanded }) {
        OutlinedTextField(
            value = label,
            onValueChange = {},
            readOnly = true,
            label = { Text("Vehicle") },
            modifier = Modifier.menuAnchor().fillMaxWidth(),
        )
        ExposedDropdownMenu(expanded, { expanded = false }) {
            DropdownMenuItem(text = { Text("Assign later") }, onClick = { onSelect(""); expanded = false })
            vehicles.forEach { vehicle ->
                DropdownMenuItem(
                    text = { Text(vehicle.licensePlate) },
                    onClick = { onSelect(vehicle.vehicleId); expanded = false },
                )
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun RolePicker(selected: String, onSelect: (String) -> Unit) {
    val roles = listOf(
        "WAREHOUSE_ADMIN" to "Warehouse admin",
        "FACTORY_ADMIN" to "Factory admin",
        "PAYLOAD" to "Payload staff",
        "ADMIN" to "Supplier operator",
    )
    var expanded by remember { mutableStateOf(false) }
    val label = roles.find { it.first == selected }?.second ?: selected
    ExposedDropdownMenuBox(expanded, { expanded = !expanded }) {
        OutlinedTextField(
            value = label,
            onValueChange = {},
            readOnly = true,
            label = { Text("Role") },
            modifier = Modifier.menuAnchor().fillMaxWidth(),
        )
        ExposedDropdownMenu(expanded, { expanded = false }) {
            roles.forEach { (id, name) ->
                DropdownMenuItem(text = { Text(name) }, onClick = { onSelect(id); expanded = false })
            }
        }
    }
}
