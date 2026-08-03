package com.pegasusx.supplier.ui.screens.orgfleet

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.pegasusx.supplier.data.model.*
import com.pegasusx.supplier.data.remote.SupplierApi
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.data.remote.SupplierRealtimeSignals
import com.pegasusx.supplier.data.remote.TokenHolder
import com.pegasusx.supplier.util.SUPPLIER_RECONNECT_RECOVERY_HINT
import com.pegasusx.supplier.util.SupplierIdempotencyKeys
import com.pegasusx.supplier.ui.realtime.SupplierReconnectRecoveryEffect
import com.pegasus.design.PegasusLoadingState
import com.pegasus.design.PegasusStateKind
import com.pegasus.design.PegasusStatePane
import com.pegasusx.supplier.ui.screens.orgfleet.components.*
import com.pegasusx.supplier.ui.theme.PegasusSpacing
import kotlinx.coroutines.launch

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
                
                MetricsOverview(
                    driversCount = drivers.size,
                    vehiclesCount = vehicles.size,
                    orgMembersCount = orgMembers.size,
                    topology = topology
                )

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
<<<<<<< HEAD
}
=======
}
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8
