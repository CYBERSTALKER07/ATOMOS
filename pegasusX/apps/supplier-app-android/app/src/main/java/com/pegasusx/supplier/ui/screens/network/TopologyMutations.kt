package com.pegasusx.supplier.ui.screens.network

import com.pegasusx.supplier.data.model.SupplierTopologyFactoryInput
import com.pegasusx.supplier.data.model.SupplierTopologyResponse
import com.pegasusx.supplier.data.model.SupplierTopologyUpdateRequest
import com.pegasusx.supplier.data.model.SupplierTopologyWarehouseInput
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository
import com.pegasusx.supplier.ui.components.AddressLocationValue

fun defaultWarehouseCoordinates(): Pair<Double, Double> {
    val c = com.pegasus.design.network.sessionMapCenter()
    return (c?.lat ?: 0.0) to (c?.lng ?: 0.0)
}

suspend fun appendWarehouseNode(
    ops: SupplierOperationsRepository,
    name: String,
    location: AddressLocationValue,
    coverageRadiusKm: Double = 50.0,
): Result<SupplierTopologyResponse> = runCatching {
    val topology = ops.getTopology().body() ?: error("topology_unavailable")
    val warehouses = topology.warehouses.map { node ->
        SupplierTopologyWarehouseInput(
            warehouseId = node.warehouseId.takeIf { it.isNotBlank() },
            name = node.name,
            address = node.address.takeIf { it.isNotBlank() },
            placeId = node.placeId,
            lat = node.lat,
            lng = node.lng,
            coverageRadiusKm = node.coverageRadiusKm,
            isActive = node.isActive,
            isOnShift = node.isOnShift,
            transferMode = node.transferMode.ifBlank { "TRUCK" },
            coLocateWithFactoryId = node.coLocateWithFactoryId,
            primaryFactoryId = node.primaryFactoryId,
            secondaryFactoryId = node.secondaryFactoryId,
            assignedFactoryIds = node.assignedFactoryIds.takeIf { it.isNotEmpty() },
            countryCode = node.countryCode.takeIf { it.isNotBlank() },
            coverageCities = node.coverageCities.takeIf { it.isNotEmpty() },
        )
    } + SupplierTopologyWarehouseInput(
        warehouseId = null,
        name = name.trim(),
        address = location.address.takeIf { it.isNotBlank() },
        placeId = location.placeId,
        lat = location.lat,
        lng = location.lng,
        coverageRadiusKm = coverageRadiusKm,
        isActive = true,
        isOnShift = true,
        transferMode = "TRUCK",
    )
    val factories = topology.factories.map { node ->
        SupplierTopologyFactoryInput(
            factoryId = node.factoryId.takeIf { it.isNotBlank() },
            name = node.name,
            address = node.address.takeIf { it.isNotBlank() },
            placeId = node.placeId,
            lat = node.lat,
            lng = node.lng,
            isActive = node.isActive,
            countryCode = node.countryCode.takeIf { it.isNotBlank() },
        )
    }
    val resp = ops.updateTopology(SupplierTopologyUpdateRequest(warehouses = warehouses, factories = factories))
    if (!resp.isSuccessful) error("save_failed_${resp.code()}")
    resp.body() ?: error("topology_empty")
}

suspend fun appendFactoryNode(
    ops: SupplierOperationsRepository,
    name: String,
    location: AddressLocationValue,
): Result<SupplierTopologyResponse> = runCatching {
    val topology = ops.getTopology().body() ?: error("topology_unavailable")
    if (topology.warehouses.isEmpty()) error("Add at least one warehouse first.")
    val warehouses = topology.warehouses.map { node ->
        SupplierTopologyWarehouseInput(
            warehouseId = node.warehouseId.takeIf { it.isNotBlank() },
            name = node.name,
            address = node.address.takeIf { it.isNotBlank() },
            placeId = node.placeId,
            lat = node.lat,
            lng = node.lng,
            coverageRadiusKm = node.coverageRadiusKm,
            isActive = node.isActive,
            isOnShift = node.isOnShift,
            transferMode = node.transferMode.ifBlank { "TRUCK" },
            coLocateWithFactoryId = node.coLocateWithFactoryId,
            primaryFactoryId = node.primaryFactoryId,
            secondaryFactoryId = node.secondaryFactoryId,
            assignedFactoryIds = node.assignedFactoryIds.takeIf { it.isNotEmpty() },
            countryCode = node.countryCode.takeIf { it.isNotBlank() },
            coverageCities = node.coverageCities.takeIf { it.isNotEmpty() },
        )
    }
    val factories = topology.factories.map { node ->
        SupplierTopologyFactoryInput(
            factoryId = node.factoryId.takeIf { it.isNotBlank() },
            name = node.name,
            address = node.address.takeIf { it.isNotBlank() },
            placeId = node.placeId,
            lat = node.lat,
            lng = node.lng,
            isActive = node.isActive,
            countryCode = node.countryCode.takeIf { it.isNotBlank() },
        )
    } + SupplierTopologyFactoryInput(
        factoryId = null,
        name = name.trim(),
        address = location.address.takeIf { it.isNotBlank() },
        placeId = location.placeId,
        lat = location.lat,
        lng = location.lng,
        isActive = true,
    )
    val resp = ops.updateTopology(SupplierTopologyUpdateRequest(warehouses = warehouses, factories = factories))
    if (!resp.isSuccessful) error("save_failed_${resp.code()}")
    resp.body() ?: error("topology_empty")
}
