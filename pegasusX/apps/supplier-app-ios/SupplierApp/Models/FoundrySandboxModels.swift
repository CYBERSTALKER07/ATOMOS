//
//  FoundrySandboxModels.swift
//  SupplierAppIOS
//

import Foundation

// MARK: - Scenario Sandbox & CoW Patch Models

public struct ScenarioPatchRequest: Codable, Sendable {
    public let cellId: String
    public let supplierId: String
    public let objectType: String
    public let objectId: String
    public let operation: String
    public let patchProperties: [String: String]

    public init(
        cellId: String = "cell-uz",
        supplierId: String,
        objectType: String,
        objectId: String,
        operation: String = "SET",
        patchProperties: [String: String]
    ) {
        self.cellId = cellId
        self.supplierId = supplierId
        self.objectType = objectType
        self.objectId = objectId
        self.operation = operation
        self.patchProperties = patchProperties
    }

    enum CodingKeys: String, CodingKey {
        case cellId = "cell_id"
        case supplierId = "supplier_id"
        case objectType = "object_type"
        case objectId = "object_id"
        case operation
        case patchProperties = "patch_properties"
    }
}

public struct ScenarioWatermarkConflict: Codable, Sendable, Identifiable {
    public var id: String { "\(objectType):\(objectId)" }
    public let cellId: String
    public let objectType: String
    public let objectId: String
    public let baseVersion: Int64
    public let liveVersion: Int64

    public init(
        cellId: String,
        objectType: String,
        objectId: String,
        baseVersion: Int64,
        liveVersion: Int64
    ) {
        self.cellId = cellId
        self.objectType = objectType
        self.objectId = objectId
        self.baseVersion = baseVersion
        self.liveVersion = liveVersion
    }

    enum CodingKeys: String, CodingKey {
        case cellId = "cell_id"
        case objectType = "object_type"
        case objectId = "object_id"
        case baseVersion = "base_version"
        case liveVersion = "live_version"
    }
}

public struct ScenarioCommitResult: Codable, Sendable {
    public let scenarioId: String
    public let committedObjects: Int
    public let conflicts: [ScenarioWatermarkConflict]

    public init(
        scenarioId: String,
        committedObjects: Int,
        conflicts: [ScenarioWatermarkConflict] = []
    ) {
        self.scenarioId = scenarioId
        self.committedObjects = committedObjects
        self.conflicts = conflicts
    }

    enum CodingKeys: String, CodingKey {
        case scenarioId = "scenario_id"
        case committedObjects = "committed_objects"
        case conflicts
    }
}

// MARK: - Contaminated Lot Recall Models

public struct LotRecallRequest: Codable, Sendable {
    public let cellId: String
    public let supplierId: String
    public let rawMaterialLotId: String

    public init(
        cellId: String = "cell-uz",
        supplierId: String,
        rawMaterialLotId: String
    ) {
        self.cellId = cellId
        self.supplierId = supplierId
        self.rawMaterialLotId = rawMaterialLotId
    }

    enum CodingKeys: String, CodingKey {
        case cellId = "cell_id"
        case supplierId = "supplier_id"
        case rawMaterialLotId = "raw_material_lot_id"
    }
}

public struct RecallBlastRadiusReport: Codable, Sendable, Identifiable {
    public var id: String { contaminatedRawMaterialLot }
    public let cellId: String
    public let supplierId: String
    public let contaminatedRawMaterialLot: String
    public let affectedWorkOrders: [String]
    public let affectedFinishedLots: [String]
    public let quarantinedSsccPallets: [String]
    public let quarantinedWarehouseTotes: [String]
    public let inTransitDriverDropBlocks: [String]
    public let impactedSoliqFacturas: [String]

    public init(
        cellId: String,
        supplierId: String,
        contaminatedRawMaterialLot: String,
        affectedWorkOrders: [String] = [],
        affectedFinishedLots: [String] = [],
        quarantinedSsccPallets: [String] = [],
        quarantinedWarehouseTotes: [String] = [],
        inTransitDriverDropBlocks: [String] = [],
        impactedSoliqFacturas: [String] = []
    ) {
        self.cellId = cellId
        self.supplierId = supplierId
        self.contaminatedRawMaterialLot = contaminatedRawMaterialLot
        self.affectedWorkOrders = affectedWorkOrders
        self.affectedFinishedLots = affectedFinishedLots
        self.quarantinedSsccPallets = quarantinedSsccPallets
        self.quarantinedWarehouseTotes = quarantinedWarehouseTotes
        self.inTransitDriverDropBlocks = inTransitDriverDropBlocks
        self.impactedSoliqFacturas = impactedSoliqFacturas
    }

    enum CodingKeys: String, CodingKey {
        case cellId = "cell_id"
        case supplierId = "supplier_id"
        case contaminatedRawMaterialLot = "contaminated_raw_material_lot"
        case affectedWorkOrders = "affected_work_orders"
        case affectedFinishedLots = "affected_finished_lots"
        case quarantinedSsccPallets = "quarantined_sscc_pallets"
        case quarantinedWarehouseTotes = "quarantined_warehouse_totes"
        case inTransitDriverDropBlocks = "in_transit_driver_drop_blocks"
        case impactedSoliqFacturas = "impacted_soliq_facturas"
    }
}

// MARK: - Action Execution Models

public struct ActionExecutionRequest: Codable, Sendable {
    public let actionName: String
    public let actionType: String
    public let scenarioId: String?
    public let actorId: String
    public let actorRole: String
    public let targetObjectType: String
    public let targetObjectId: String
    public let idempotencyKey: String
    public let cellId: String
    public let supplierId: String

    public init(
        actionName: String,
        actionType: String = "MUTATION",
        scenarioId: String? = nil,
        actorId: String,
        actorRole: String,
        targetObjectType: String,
        targetObjectId: String,
        idempotencyKey: String,
        cellId: String = "cell-uz",
        supplierId: String
    ) {
        self.actionName = actionName
        self.actionType = actionType
        self.scenarioId = scenarioId
        self.actorId = actorId
        self.actorRole = actorRole
        self.targetObjectType = targetObjectType
        self.targetObjectId = targetObjectId
        self.idempotencyKey = idempotencyKey
        self.cellId = cellId
        self.supplierId = supplierId
    }

    enum CodingKeys: String, CodingKey {
        case actionName = "action_name"
        case actionType = "action_type"
        case scenarioId = "scenario_id"
        case actorId = "actor_id"
        case actorRole = "actor_role"
        case targetObjectType = "target_object_type"
        case targetObjectId = "target_object_id"
        case idempotencyKey = "idempotency_key"
        case cellId = "cell_id"
        case supplierId = "supplier_id"
    }
}

public struct ActionExecutionRecord: Codable, Sendable, Identifiable {
    public var id: String { executionId }
    public let executionId: String
    public let actionName: String
    public let actionType: String
    public let scenarioId: String?
    public let actorId: String
    public let actorRole: String
    public let targetObjectType: String
    public let targetObjectId: String
    public let idempotencyKey: String
    public let executedAt: String
    public let status: String
    public let cellId: String
    public let supplierId: String

    enum CodingKeys: String, CodingKey {
        case executionId = "execution_id"
        case actionName = "action_name"
        case actionType = "action_type"
        case scenarioId = "scenario_id"
        case actorId = "actor_id"
        case actorRole = "actor_role"
        case targetObjectType = "target_object_type"
        case targetObjectId = "target_object_id"
        case idempotencyKey = "idempotency_key"
        case executedAt = "executed_at"
        case status
        case cellId = "cell_id"
        case supplierId = "supplier_id"
    }
}
