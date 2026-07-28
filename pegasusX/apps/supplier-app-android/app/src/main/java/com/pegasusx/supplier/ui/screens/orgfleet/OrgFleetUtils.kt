package com.pegasusx.supplier.ui.screens.orgfleet

import com.pegasusx.supplier.data.model.SupplierTopologyResponse

fun nodeLabel(type: String, id: String, topology: SupplierTopologyResponse?): String {
    if (topology == null) return id
    if (type == "FACTORY") {
        return topology.factories.find { it.factoryId == id }?.name ?: id
    }
    return topology.warehouses.find { it.warehouseId == id }?.name ?: id
}
