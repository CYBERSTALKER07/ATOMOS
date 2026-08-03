import Foundation

func nodeLabel(topology: SupplierTopologyResponse?, type: String, id: String) -> String {
    guard let topology else { return id }
    if type == "FACTORY" {
        return topology.factories.first { $0.factoryId == id }?.name ?? id
    }
    return topology.warehouses.first { $0.warehouseId == id }?.name ?? id
}
