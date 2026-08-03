package com.pegasusx.supplier.ui.screens.orgfleet.components

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.*
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.supplier.ui.theme.PegasusSpacing

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun OrgFleetScreen(
    api: SupplierApi,
    ops: SupplierOperationsRepository,
    realtimeSignals: SupplierRealtimeSignals,
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
    var editingMember by remember { mutableStateOf<SupplierOrgMember?>(null) }
    var memberActionId by remember { mutableStateOf<String?>(null) }
    var fleetActionBusy by remember { mutableStateOf(false) }
    var fleetActionMessage by remember { mutableStateOf<String?>(null) }
    val scope = rememberCoroutineScope()
    val scopeId = TokenHolder.supplierId.orEmpty().ifBlank { "supplier" }

    SupplierReconnectRecoveryEffect(
        realtimeSignals = realtimeSignals,
        isBusy = { fleetActionBusy || memberActionId != null },
    ) { hadInFlight ->
        if (hadInFlight) {
            fleetActionBusy = false
            memberActionId = null
            fleetActionMessage = SUPPLIER_RECONNECT_RECOVERY_HINT
        }
    }

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
            loading -> PegasusLoadingState("Loading org & fleet…", "Topology and rosters")
            error != null -> PegasusStatePane(
                kind = PegasusStateKind.Error,
                headline = "Onboarding unavailable",
                body = error!!,
                modifier = Modifier.padding(padding),
                actionLabel = "Retry",
                onAction = { reload() },
            )
            else -> Column(Modifier.padding(padding).fillMaxSize()) {
                fleetActionMessage?.let { msg ->
                    Text(msg, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.primary, modifier = Modifier.padding(horizontal = PegasusSpacing.lg, vertical = PegasusSpacing.sm))
                }
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
                        onEdit = { editingMember = it },
                        onDeactivate = { userId ->
                            scope.launch {
                                memberActionId = userId
                                try {
                                    val resp = ops.deactivateOrgMember(
                                        userId,
                                        SupplierIdempotencyKeys.orgMemberDeactivate(scopeId, userId),
                                    )
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
                    fleetActionBusy = true
                    fleetActionMessage = null
                    try {
                        ops.createFleetDriver(
                            request,
                            SupplierIdempotencyKeys.fleetDriverCreate(scopeId, request.phone),
                        )
                        showDriverDialog = false
                        reload()
                    } finally {
                        fleetActionBusy = false
                    }
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
                    fleetActionBusy = true
                    fleetActionMessage = null
                    try {
                        ops.createFleetVehicle(
                            request,
                            SupplierIdempotencyKeys.fleetVehicleCreate(scopeId, request.licensePlate),
                        )
                        showVehicleDialog = false
                        reload()
                    } finally {
                        fleetActionBusy = false
                    }
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
                    fleetActionBusy = true
                    fleetActionMessage = null
                    try {
                        ops.createOrgMember(
                            request,
                            SupplierIdempotencyKeys.orgMemberCreate(scopeId, request.phone),
                        )
                        showOrgDialog = false
                        reload()
                    } finally {
                        fleetActionBusy = false
                    }
                }
            },
        )
    }
    editingMember?.let { member ->
        if (topo != null) {
            EditOrgMemberDialog(
                member = member,
                topology = topo,
                onDismiss = { editingMember = null },
                onSave = { request ->
                    scope.launch {
                        memberActionId = member.userId
                        try {
                            val revision = "${request.name}:${request.supplierRole}"
                            val resp = ops.updateOrgMember(
                                member.userId,
                                request,
                                SupplierIdempotencyKeys.orgMemberUpdate(scopeId, member.userId, revision),
                            )
                            if (resp.isSuccessful) {
                                orgMembers = resp.body()?.items.orEmpty()
                                editingMember = null
                            }
                        } finally {
                            memberActionId = null
                        }
                    }
                },
            )
        }
    }
}

@Composable
fun DriverRoster(drivers: List<FleetDriver>, topology: SupplierTopologyResponse?) {
    if (drivers.isEmpty()) {
        PegasusStatePane(PegasusStateKind.Empty, "No drivers", "Create a driver to start fleet onboarding.")
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
fun VehicleRoster(vehicles: List<FleetVehicle>, topology: SupplierTopologyResponse?) {
    if (vehicles.isEmpty()) {
        PegasusStatePane(PegasusStateKind.Empty, "No vehicles", "Create a vehicle for driver assignment.")
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
fun OrgRoster(
    members: List<SupplierOrgMember>,
    onEdit: (SupplierOrgMember) -> Unit,
    onDeactivate: (String) -> Unit,
    actionId: String?,
) {
    if (members.isEmpty()) {
        PegasusStatePane(PegasusStateKind.Empty, "No org members", "Create warehouse, factory, or payload staff.")
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
                    Row {
                        TextButton(
                            enabled = actionId != member.userId,
                            onClick = { onEdit(member) },
                        ) { Text("Edit") }
                        if (member.isActive) {
                            TextButton(
                                enabled = actionId != member.userId,
                                onClick = { onDeactivate(member.userId) },
                            ) { Text("Deactivate") }
                        }
                    }
                },
            )
        }
    }
}

fun nodeLabel(type: String, id: String, topology: SupplierTopologyResponse?): String {
    if (topology == null) return id
    if (type == "FACTORY") {
        return topology.factories.find { it.factoryId == id }?.name ?: id
    }
    return topology.warehouses.find { it.warehouseId == id }?.name ?: id
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun CreateDriverDialog(
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
fun CreateVehicleDialog(
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
fun CreateOrgMemberDialog(
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

@Composable
fun EditOrgMemberDialog(
    member: SupplierOrgMember,
    topology: SupplierTopologyResponse,
    onDismiss: () -> Unit,
    onSave: (SupplierOrgMemberUpdateRequest) -> Unit,
) {
    var name by remember { mutableStateOf(member.name) }
    var role by remember { mutableStateOf(member.supplierRole) }
    var nodeId by remember {
        mutableStateOf(member.assignedWarehouseId ?: member.assignedFactoryId ?: "")
    }
    val nodeOptions = when (role) {
        "FACTORY_ADMIN" -> topology.factories.map { it.factoryId to it.name }
        "ADMIN" -> emptyList()
        else -> topology.warehouses.map { it.warehouseId to it.name }
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Edit org member") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(PegasusSpacing.sm)) {
                OutlinedTextField(name, { name = it }, label = { Text("Name") }, modifier = Modifier.fillMaxWidth())
                RolePicker(role) { role = it; nodeId = "" }
                if (role != "ADMIN" && nodeOptions.isNotEmpty()) {
                    NodePicker(nodeOptions, nodeId) { nodeId = it }
                }
            }
        },
        confirmButton = {
            TextButton(onClick = {
                if (name.isBlank()) return@TextButton
                val warehouseId = if (role == "WAREHOUSE_ADMIN" || role == "PAYLOAD") nodeId else null
                val factoryId = if (role == "FACTORY_ADMIN") nodeId else null
                onSave(
                    SupplierOrgMemberUpdateRequest(
                        name = name.trim(),
                        supplierRole = role,
                        assignedWarehouseId = warehouseId,
                        assignedFactoryId = factoryId,
                    ),
                )
            }) { Text("Save") }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}