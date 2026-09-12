import Foundation

enum ManifestBoard {
    static let states = ["DRAFT", "LOADING", "SEALED", "DISPATCHED"]

    static func canonicalState(_ state: String?) -> String {
        let s = (state ?? "").trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
        return Self.states.contains(s) ? s : ""
    }

    static func isBoardState(_ state: String?) -> Bool {
        !canonicalState(state).isEmpty
    }

    struct Column: Equatable {
        let state: String
        let trucks: [Truck]
    }

    static func attach(trucks: [Truck], manifests: [Manifest]) -> [Truck] {
        trucks.map { truck in
            if !canonicalState(truck.truckStatus).isEmpty { return truck }
            let match = manifests
                .filter { $0.matchesTruck(truck.id) && isBoardState($0.state) }
                .max(by: { ($0.createdAt ?? "") < ($1.createdAt ?? "") })
            guard let match else { return truck }
            return truck.withBoard(
                status: canonicalState(match.state),
                used: match.totalVolumeVu,
                max: match.maxVolumeVu,
                stops: match.stopCount
            )
        }
    }

    static func group(_ trucks: [Truck]) -> [Column] {
        states.map { state in
            Column(state: state, trucks: trucks.filter { canonicalState($0.truckStatus) == state })
        }
    }

    static func unassigned(_ trucks: [Truck]) -> [Truck] {
        trucks.filter { canonicalState($0.truckStatus).isEmpty }
    }
}
