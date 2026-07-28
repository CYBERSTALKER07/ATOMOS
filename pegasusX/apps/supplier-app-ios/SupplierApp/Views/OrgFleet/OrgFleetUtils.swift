import Foundation

enum OrgFleetUtils {
    static func nodeLabel(type: String, id: String, topology: SupplierTopologyResponse?) -> String {
        guard let topology = topology else { return id }
        if type == "FACTORY" {
            return topology.factories.first { $0.factoryId == id }?.name ?? id
        }
        return topology.warehouses.first { $0.warehouseId == id }?.name ?? id
    }
}
