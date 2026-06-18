package com.pegasusx.supplier.ui.screens.network

import com.pegasusx.supplier.data.model.SupplierTopologyFactoryInput
import com.pegasusx.supplier.data.model.SupplierTopologyResponse
import com.pegasusx.supplier.data.model.SupplierTopologyUpdateRequest
import com.pegasusx.supplier.data.model.SupplierTopologyWarehouseInput
import com.pegasusx.supplier.data.remote.SupplierOperationsRepository

private const val DEFAULT_LAT = 41.2995
private const val DEFAULT_LNG = 69.2401

suspend fun appendWarehouseNode(
    ops: SupplierOperationsRepository,
    name: String,
    lat: Double,
    lng: Double,
    coverageRadiusKm: Double = 50.0,
): Result<SupplierTopologyResponse> = runCatching {
    val topology = ops.getTopology().body() ?: error("topology_unavailable")
    val warehouses = topology.warehouses.map { node ->
        SupplierTopologyWarehouseInput(
            warehouseId = node.warehouseId.takeIf { it.isNotBlank() },
            name = node.name,
            lat = node.lat,
            lng = node.lng,
            coverageRadiusKm = node.coverageRadiusKm,
            isActive = node.isActive,
            isOnShift = node.isOnShift,
            transferMode = node.transferMode.ifBlank { "TRUCK" },
        )
    } + SupplierTopologyWarehouseInput(
        warehouseId = null,
        name = name.trim(),
        lat = lat,
        lng = lng,
        coverageRadiusKm = coverageRadiusKm,
        isActive = true,
        isOnShift = true,
        transferMode = "TRUCK",
    )
    val factories = topology.factories.map { node ->
        SupplierTopologyFactoryInput(
            factoryId = node.factoryId.takeIf { it.isNotBlank() },
            name = node.name,
            lat = node.lat,
            lng = node.lng,
            isActive = node.isActive,
        )
    }
    val resp = ops.updateTopology(SupplierTopologyUpdateRequest(warehouses = warehouses, factories = factories))
    if (!resp.isSuccessful) error("save_failed_${resp.code()}")
    resp.body() ?: error("topology_empty")
}

suspend fun appendFactoryNode(
    ops: SupplierOperationsRepository,
    name: String,
    lat: Double,
    lng: Double,
): Result<SupplierTopologyResponse> = runCatching {
    val topology = ops.getTopology().body() ?: error("topology_unavailable")
    if (topology.warehouses.isEmpty()) error("Add at least one warehouse first.")
    val warehouses = topology.warehouses.map { node ->
        SupplierTopologyWarehouseInput(
            warehouseId = node.warehouseId.takeIf { it.isNotBlank() },
            name = node.name,
            lat = node.lat,
            lng = node.lng,
            coverageRadiusKm = node.coverageRadiusKm,
            isActive = node.isActive,
            isOnShift = node.isOnShift,
            transferMode = node.transferMode.ifBlank { "TRUCK" },
        )
    }
    val factories = topology.factories.map { node ->
        SupplierTopologyFactoryInput(
            factoryId = node.factoryId.takeIf { it.isNotBlank() },
            name = node.name,
            lat = node.lat,
            lng = node.lng,
            isActive = node.isActive,
        )
    } + SupplierTopologyFactoryInput(
        factoryId = null,
        name = name.trim(),
        lat = lat,
        lng = lng,
        isActive = true,
    )
    val resp = ops.updateTopology(SupplierTopologyUpdateRequest(warehouses = warehouses, factories = factories))
    if (!resp.isSuccessful) error("save_failed_${resp.code()}")
    resp.body() ?: error("topology_empty")
}

fun defaultWarehouseCoordinates(): Pair<Double, Double> = DEFAULT_LAT to DEFAULT_LNG
