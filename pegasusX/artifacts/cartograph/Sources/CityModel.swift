import Foundation
import SwiftUI

enum Verdict: String, CaseIterable, Identifiable {
    case real = "REAL"
    case partial = "PARTIAL"
    case theatre = "THEATRE"
    case gone = "GONE"
    case gated = "GATED"
    case absent = "ABSENT"

    var id: String { rawValue }

    var gloss: String {
        switch self {
        case .real: return "Mounted path with a durable write or a live read this session verified on origin/main."
        case .partial: return "Mounted, but a downstream rail, secret, or client is missing."
        case .theatre: return "Returns a success-shaped object (redirect URL, receipt id) without the real rail."
        case .gone: return "Explicit 410/501 or the file is not on this commit."
        case .gated: return "Terraform/kustomize declares it behind a flag; this map does not claim it is applied."
        case .absent: return "Not in origin/main. Later working-tree work exists; it is not this city."
        }
    }
}

enum District: String, CaseIterable, Identifiable {
    case clients = "Client shore"
    case gate = "Auth gate"
    case spine = "Chi colonnade"
    case domain = "Domain halls"
    case canal = "Event canal"
    case vaults = "Vaults"
    case money = "Money court"
    case yard = "Infra yard"

    var id: String { rawValue }
}

enum BuildingKind: String {
    case tower, warehouse, silo, vault, mast, plant, shed, dock, kiosk, lock, lot, stack
}

enum LaneKind: String, CaseIterable, Identifiable {
    case control = "Control"
    case data = "Data"
    case money = "Money"
    case event = "Event"
    case live = "Live UX"
    case theatre = "Theatre"

    var id: String { rawValue }

    var tint: Color { Ink.fg }

    var dash: [CGFloat] {
        switch self {
        case .control: return []
        case .data: return [10, 5]
        case .money: return [2, 4]
        case .event: return [14, 4, 3, 4]
        case .live: return [22, 4]
        case .theatre: return [1, 5]
        }
    }

    var caption: String {
        switch self {
        case .control: return "HTTP + JWT. Role from claims, never body IDs."
        case .data: return "Spanner txn, cache invalidate, H3 lookup."
        case .money: return "Checkout, webhook, ledger, fiscal receipt."
        case .event: return "OutboxEvents → relay → Kafka → consumer."
        case .live: return "GET /v1/ws + Redis fan-out ws:<hub>:fanout."
        case .theatre: return "Hosted-redirect string with no Charge API."
        }
    }
}

struct Cite: Identifiable, Hashable {
    let file: String
    let line: Int
    var note: String = ""
    var id: String { "\(file):\(line)" }
    var label: String { "\(file):\(line)" }
}

struct Building: Identifiable {
    let id: String
    let name: String
    let kind: BuildingKind
    let district: District
    let gx: Double
    let gy: Double
    let w: Double
    let d: Double
    let h: Double
    let verdict: Verdict
    let blurb: String
    let cites: [Cite]
    var tags: [String] = []
    var code: String {
        Atlas.codes[id] ?? String(name.prefix(2)).uppercased()
    }
}

struct Trace: Identifiable {
    let id: String
    let title: String
    let lane: LaneKind
    let verdict: Verdict
    let payload: String
    let hops: [String]
    let waypoints: [(Double, Double)]
    let cites: [Cite]
    let explainer: String
}

enum Atlas {
    static let commit = "fbfd134e"
    static let commitFull = "fbfd134ebb75a6de4904a5ce775320f544c091f3"
    static let branch = "origin/main"
    static let repo = "CYBERSTALKER07/ATOMOS"
    static let tree = "pegasusX/"
    static let surveyed = "2026-08-16"

    static let headline = "Single-supplier cargo district, surveyed from live mounts"

    static let thesis = """
    This city is origin/main only. The composition root is bootstrap.NewApp; \
    chi mounts live in main.go. JWT role ADMIN is the supplier. MarketPack, \
    home cells, and tenant self-serve are empty lots — those files are not on this commit.
    """

    static let buildings: [Building] = [
        Building(
            id: "supplier-portal", name: "Supplier HQ", kind: .tower,
            district: .clients, gx: 1.1, gy: 2.0, w: 0.85, d: 0.85, h: 3.4,
            verdict: .partial,
            blurb: "Web portal for JWT ADMIN (product language: supplier). Cookie supplier_jwt. No RoleSupplier on this commit.",
            cites: [
                Cite(file: "apps/backend-go/auth/claims.go", line: 20, note: "RoleAdmin = ADMIN — supplier portal session"),
                Cite(file: "apps/backend-go/auth/jwt.go", line: 22, note: "cookie name supplier_jwt"),
                Cite(file: "apps/supplier-portal", line: 1, note: "role-row web surface")
            ],
            tags: ["ADMIN", "web"]
        ),
        Building(
            id: "supplier-native", name: "Supplier iOS · Android · desktop", kind: .shed,
            district: .clients, gx: 0.05, gy: 2.15, w: 0.9, d: 0.7, h: 1.05,
            verdict: .partial,
            blurb: "Native twins in the ADMIN row. They speak HTTP + GET /v1/ws, not a separate API.",
            cites: [
                Cite(file: "apps/supplier-app-ios", line: 1),
                Cite(file: "apps/supplier-app-android", line: 1),
                Cite(file: "apps/supplier-app-desktop", line: 1)
            ]
        ),
        Building(
            id: "admin-portal", name: "Admin portal", kind: .shed,
            district: .clients, gx: 1.15, gy: 0.3, w: 0.8, d: 0.7, h: 1.15,
            verdict: .partial,
            blurb: "Present under apps/. No dedicated JWT. Not a platform-admin identity on this commit.",
            cites: [Cite(file: "apps/admin-portal", line: 1, note: "no platform-admin hub / role")]
        ),
        Building(
            id: "retailer-row", name: "Retailer row", kind: .warehouse,
            district: .clients, gx: 0.05, gy: 4.4, w: 1.9, d: 0.85, h: 1.25,
            verdict: .real,
            blurb: "Desktop + iOS + Android. Create uses claims.Subject as retailer id. Pay-at-delivery is here, not pre-delivery checkout.",
            cites: [
                Cite(file: "apps/retailer-app-desktop", line: 1),
                Cite(file: "apps/retailer-app-ios", line: 1),
                Cite(file: "apps/retailer-app-android", line: 1),
                Cite(file: "apps/backend-go/order/service.go", line: 2342, note: "Create(ctx, claims.Subject, req)")
            ],
            tags: ["RETAILER"]
        ),
        Building(
            id: "warehouse-row", name: "Warehouse row", kind: .warehouse,
            district: .clients, gx: 0.05, gy: 6.0, w: 1.9, d: 0.8, h: 1.35,
            verdict: .partial,
            blurb: "Portal + iOS + Android. JWT WAREHOUSE / WAREHOUSE_ADMIN. Home-node scoped.",
            cites: [
                Cite(file: "apps/warehouse-portal", line: 1),
                Cite(file: "apps/warehouse-app-ios", line: 1),
                Cite(file: "apps/warehouse-app-android", line: 1),
                Cite(file: "apps/backend-go/auth/claims.go", line: 27)
            ]
        ),
        Building(
            id: "driver-row", name: "Driver cabins", kind: .shed,
            district: .clients, gx: 0.05, gy: 7.5, w: 1.15, d: 0.7, h: 1.0,
            verdict: .real,
            blurb: "iOS + Android. Delivery mutations + telemetry location posts.",
            cites: [
                Cite(file: "apps/driver-app-ios", line: 1),
                Cite(file: "apps/driver-app-android", line: 1),
                Cite(file: "apps/backend-go/orderroutes/routes.go", line: 44, note: "POST /v1/delivery/arrive")
            ],
            tags: ["DRIVER"]
        ),
        Building(
            id: "factory-row", name: "Factory loft", kind: .plant,
            district: .clients, gx: 1.35, gy: 7.45, w: 0.7, d: 0.75, h: 1.7,
            verdict: .partial,
            blurb: "Portal + iOS + Android. Factory truck plane is a different table from supplier trucks.",
            cites: [
                Cite(file: "apps/factory-portal", line: 1),
                Cite(file: "apps/factory-app-ios", line: 1),
                Cite(file: "apps/factory-app-android", line: 1),
                Cite(file: "apps/backend-go/schema/spanner.ddl", line: 884, note: "FactoryTruckManifests")
            ]
        ),
        Building(
            id: "payload-row", name: "Payload dock", kind: .dock,
            district: .clients, gx: 0.05, gy: 9.0, w: 1.9, d: 0.85, h: 1.15,
            verdict: .partial,
            blurb: "Terminal (Expo) + iOS + Android. JWT PAYLOAD. Room payload:{sub|supplier}.",
            cites: [
                Cite(file: "apps/payload-terminal", line: 1),
                Cite(file: "apps/payload-app-ios", line: 1),
                Cite(file: "apps/payload-app-android", line: 1),
                Cite(file: "apps/backend-go/ws/handler.go", line: 169)
            ]
        ),
        Building(
            id: "auth-gate", name: "JWT gate", kind: .kiosk,
            district: .gate, gx: 2.6, gy: 4.6, w: 0.7, d: 0.7, h: 1.55,
            verdict: .real,
            blurb: "SessionAuth on the router. Scope from claims. Seeded SupplierId for every authenticated caller on this single-tenant commit.",
            cites: [
                Cite(file: "apps/backend-go/main.go", line: 111, note: "r.Use(auth.SessionAuth)"),
                Cite(file: "apps/backend-go/auth/claims.go", line: 5, note: "never trust body IDs")
            ]
        ),
        Building(
            id: "chi-colonnade", name: "Chi colonnade", kind: .tower,
            district: .spine, gx: 3.7, gy: 4.4, w: 1.05, d: 1.15, h: 4.1,
            verdict: .real,
            blurb: "main.go is lifecycle only: config → NewApp → RegisterRoutes → ListenAndServe. Domain logic is not here.",
            cites: [
                Cite(file: "apps/backend-go/main.go", line: 1),
                Cite(file: "apps/backend-go/main.go", line: 135, note: "infraroutes … catalogroutes + ws")
            ],
            tags: ["composition"]
        ),
        Building(
            id: "bootstrap", name: "Composition root", kind: .stack,
            district: .spine, gx: 3.7, gy: 6.1, w: 1.05, d: 0.85, h: 1.8,
            verdict: .real,
            blurb: "NewApp owns Spanner, Redis, Kafka, seven hubs, outbox relay. RequireInfraAdapters default true — no silent memory path in prod.",
            cites: [
                Cite(file: "apps/backend-go/bootstrap/bootstrap.go", line: 326, note: "func NewApp"),
                Cite(file: "apps/backend-go/bootstrap/bootstrap.go", line: 287, note: "RequireInfraAdapters default true"),
                Cite(file: "apps/backend-go/bootstrap/bootstrap.go", line: 321)
            ]
        ),
        Building(
            id: "idempotency", name: "Idempotency lodge", kind: .kiosk,
            district: .spine, gx: 3.75, gy: 3.2, w: 0.65, d: 0.65, h: 1.2,
            verdict: .real,
            blurb: "chi middleware when app.Idempotency != nil. Mutating handlers also guard locally (order create).",
            cites: [
                Cite(file: "apps/backend-go/main.go", line: 126),
                Cite(file: "apps/backend-go/order/service.go", line: 2326, note: "guardIdempotency")
            ]
        ),
        Building(
            id: "health", name: "Health booth", kind: .kiosk,
            district: .spine, gx: 3.8, gy: 7.5, w: 0.6, d: 0.55, h: 0.9,
            verdict: .real,
            blurb: "/healthz /ready /metrics via infraroutes.",
            cites: [Cite(file: "apps/backend-go/infraroutes/routes.go", line: 34)]
        ),
        Building(
            id: "supplier-svc", name: "Supplier hall", kind: .tower,
            district: .domain, gx: 5.2, gy: 1.8, w: 0.85, d: 0.85, h: 2.6,
            verdict: .partial,
            blurb: "Register + login mint RoleAdmin. Billing setup stores bank fields — no payout rail.",
            cites: [
                Cite(file: "apps/backend-go/supplierroutes/routes.go", line: 65),
                Cite(file: "apps/backend-go/supplier/service.go", line: 931, note: "writeSessionCookie RoleAdmin")
            ]
        ),
        Building(
            id: "catalog", name: "Catalog shed", kind: .warehouse,
            district: .domain, gx: 6.3, gy: 1.7, w: 1.2, d: 0.85, h: 1.2,
            verdict: .real,
            blurb: "catalogroutes. Stock reservation participates in order create.",
            cites: [Cite(file: "apps/backend-go/catalogroutes/routes.go", line: 22)]
        ),
        Building(
            id: "order-hall", name: "Order hall", kind: .tower,
            district: .domain, gx: 5.25, gy: 3.5, w: 1.15, d: 1.1, h: 3.2,
            verdict: .real,
            blurb: "POST /v1/order/create (RETAILER) and cart POST /v1/checkout/unified both land in Create. Same-txn OutboxEvents ORDER_CREATED. Create does not Broadcast WS itself.",
            cites: [
                Cite(file: "apps/backend-go/orderroutes/routes.go", line: 40, note: "POST /v1/order/create"),
                Cite(file: "apps/backend-go/paymentroutes/routes.go", line: 29, note: "POST /v1/checkout/unified — desktop cart"),
                Cite(file: "apps/backend-go/order/service.go", line: 2311, note: "HandleCreate"),
                Cite(file: "apps/backend-go/order/service.go", line: 1298, note: "outbox.EmitJSON EventOrderCreated"),
                Cite(file: "apps/backend-go/orderroutes/routes.go", line: 54, note: "POST /v1/order/{id}/fiscal/retry")
            ],
            tags: ["Class A"]
        ),
        Building(
            id: "proximity", name: "H3 cupola", kind: .silo,
            district: .domain, gx: 6.7, gy: 3.35, w: 0.7, d: 0.7, h: 2.1,
            verdict: .real,
            blurb: "Closest on-shift warehouse: H3 res 9 disk then haversine vs CoverageRadiusKm. Zero lat/lng → empty id (zone miss). ResolveServingWarehouse is not on this commit.",
            cites: [
                Cite(file: "apps/backend-go/order/warehouse_resolver_spanner.go", line: 29),
                Cite(file: "apps/backend-go/order/warehouse_resolver_spanner.go", line: 47, note: "CellsInRadius res 9 + H3Cell IN UNNEST"),
                Cite(file: "apps/backend-go/order/service.go", line: 1160)
            ]
        ),
        Building(
            id: "warehouse-ops", name: "Warehouse ops", kind: .warehouse,
            district: .domain, gx: 5.2, gy: 5.2, w: 1.35, d: 0.9, h: 1.45,
            verdict: .partial,
            blurb: "warehouseroutes + auto-dispatch worker + plan warmer. SupplierTruckManifests live here.",
            cites: [
                Cite(file: "apps/backend-go/warehouseroutes/routes.go", line: 1),
                Cite(file: "apps/backend-go/runtime_workers.go", line: 41, note: "StartAutoDispatchWorker"),
                Cite(file: "apps/backend-go/schema/spanner.ddl", line: 798, note: "SupplierTruckManifests")
            ]
        ),
        Building(
            id: "retailer-svc", name: "Retailer desk", kind: .shed,
            district: .domain, gx: 6.75, gy: 5.25, w: 0.85, d: 0.85, h: 1.35,
            verdict: .partial,
            blurb: "retailerroutes. Card/cash checkout prefer PaymentService when wired.",
            cites: [
                Cite(file: "apps/backend-go/retailerroutes/routes.go", line: 32),
                Cite(file: "apps/backend-go/retailerroutes/routes.go", line: 226, note: "cash/card checkout")
            ]
        ),
        Building(
            id: "payment-court", name: "Payment court", kind: .tower,
            district: .domain, gx: 5.3, gy: 6.7, w: 0.95, d: 0.95, h: 2.5,
            verdict: .partial,
            blurb: "Checkout preview + unified cart. Pre-delivery /v1/checkout/b2b is 410. Stripe/Adyen are static redirects.",
            cites: [
                Cite(file: "apps/backend-go/paymentroutes/routes.go", line: 2, note: "webhooks are not here"),
                Cite(file: "apps/backend-go/paymentroutes/routes.go", line: 28),
                Cite(file: "apps/backend-go/payment/execution.go", line: 139)
            ]
        ),
        Building(
            id: "factory-ops", name: "Factory plant", kind: .plant,
            district: .domain, gx: 6.6, gy: 6.7, w: 0.95, d: 0.95, h: 2.2,
            verdict: .partial,
            blurb: "factoryroutes. Dual plane: FactoryTruckManifests must not merge with supplier trucks.",
            cites: [
                Cite(file: "apps/backend-go/factoryroutes/routes.go", line: 20),
                Cite(file: "apps/backend-go/schema/spanner.ddl", line: 884)
            ]
        ),
        Building(
            id: "payload-svc", name: "Payload bay", kind: .dock,
            district: .domain, gx: 5.25, gy: 8.2, w: 1.15, d: 0.75, h: 1.1,
            verdict: .partial,
            blurb: "payloaderroutes. Terminal-scoped.",
            cites: [Cite(file: "apps/backend-go/payloaderroutes/routes.go", line: 24)]
        ),
        Building(
            id: "driver-svc", name: "Driver desk", kind: .shed,
            district: .domain, gx: 6.6, gy: 8.2, w: 0.95, d: 0.75, h: 1.05,
            verdict: .real,
            blurb: "driverroutes + delivery mutations on orderroutes.",
            cites: [Cite(file: "apps/backend-go/driverroutes/routes.go", line: 43)]
        ),
        Building(
            id: "credit", name: "Credit & AR", kind: .vault,
            district: .domain, gx: 7.8, gy: 6.75, w: 0.7, d: 0.85, h: 1.3,
            verdict: .partial,
            blurb: "creditroutes + credit notes + cash recon. Ledger, not a payout rail.",
            cites: [
                Cite(file: "apps/backend-go/creditroutes/routes.go", line: 21),
                Cite(file: "apps/backend-go/cashreconroutes/routes.go", line: 17)
            ]
        ),
        Building(
            id: "returns", name: "Returns shed", kind: .shed,
            district: .domain, gx: 7.85, gy: 8.2, w: 0.7, d: 0.75, h: 1.0,
            verdict: .partial,
            blurb: "returnsroutes + driver + supplier history mounts.",
            cites: [Cite(file: "apps/backend-go/returnsroutes/routes.go", line: 20)]
        ),
        Building(
            id: "outbox", name: "Outbox lock", kind: .lock,
            district: .canal, gx: 4.9, gy: 0.15, w: 1.15, d: 0.85, h: 1.35,
            verdict: .real,
            blurb: "Same-txn OutboxEvents. Relay tick 250ms, batch 100, Required path via Kafka publisher. Started only if RunsWorkers.",
            cites: [
                Cite(file: "apps/backend-go/outbox/relay.go", line: 48, note: "Start drains OutboxEvents"),
                Cite(file: "apps/backend-go/outbox/relay.go", line: 15, note: "250ms / batch 100"),
                Cite(file: "apps/backend-go/runtime_workers.go", line: 16, note: "OutboxRelay.Start")
            ]
        ),
        Building(
            id: "kafka", name: "Kafka barges", kind: .lock,
            district: .canal, gx: 6.3, gy: 0.1, w: 1.4, d: 0.9, h: 1.5,
            verdict: .partial,
            blurb: "TopicMain default pegasusx-main. TF does not provision a cluster — only Secret Manager strings. Staging overlay names GCP Managed Kafka.",
            cites: [
                Cite(file: "apps/backend-go/events/events.go", line: 12, note: "DefaultTopicMain"),
                Cite(file: "apps/backend-go/bootstrap/bootstrap.go", line: 256, note: "KAFKA_TOPIC_MAIN"),
                Cite(file: "infra/terraform/main.tf", line: 84, note: "Kafka remains provider-agnostic")
            ]
        ),
        Building(
            id: "dispatcher", name: "Notification mill", kind: .stack,
            district: .canal, gx: 7.95, gy: 0.15, w: 0.9, d: 0.85, h: 1.7,
            verdict: .real,
            blurb: "NotificationDispatcher.HandleEvent. ORDER_CREATED → handleOrderEvent. Dedup by consumer group + offset.",
            cites: [
                Cite(file: "apps/backend-go/kafka/notification_dispatcher.go", line: 65),
                Cite(file: "apps/backend-go/kafka/notification_dispatcher.go", line: 116, note: "EventOrderCreated")
            ]
        ),
        Building(
            id: "consumers", name: "Consumer sheds", kind: .shed,
            district: .canal, gx: 9.1, gy: 0.2, w: 1.15, d: 0.8, h: 1.15,
            verdict: .partial,
            blurb: "Notification + order + warehouse + returns consumers started from runtime_workers. PAYMENT_CLEARED settles.",
            cites: [
                Cite(file: "apps/backend-go/runtime_workers.go", line: 25),
                Cite(file: "apps/backend-go/bootstrap/bootstrap.go", line: 1424)
            ]
        ),
        Building(
            id: "ws-masts", name: "WS radio masts", kind: .mast,
            district: .canal, gx: 8.95, gy: 1.55, w: 0.55, d: 0.55, h: 3.6,
            verdict: .real,
            blurb: "One upgrade: GET /v1/ws. Seven hubs. Broadcast = local fan-out + Redis ws:<hub>:fanout. Fail-open.",
            cites: [
                Cite(file: "apps/backend-go/ws/handler.go", line: 84, note: "GET /v1/ws"),
                Cite(file: "apps/backend-go/ws/hub.go", line: 3, note: "local + Publish"),
                Cite(file: "apps/backend-go/main.go", line: 304)
            ]
        ),
        Building(
            id: "spanner", name: "Spanner silos", kind: .silo,
            district: .vaults, gx: 5.3, gy: 9.85, w: 1.1, d: 1.0, h: 2.8,
            verdict: .real,
            blurb: "Schema of record. Dual truck tables. OutboxEvents. TF: regional instance, 100 PU. Not a claim that apply ran.",
            cites: [
                Cite(file: "apps/backend-go/schema/spanner.ddl", line: 11, note: "CREATE TABLE Suppliers"),
                Cite(file: "apps/backend-go/schema/spanner.ddl", line: 615, note: "OutboxEvents"),
                Cite(file: "infra/terraform/main.tf", line: 71)
            ]
        ),
        Building(
            id: "redis", name: "Redis vault", kind: .vault,
            district: .vaults, gx: 6.7, gy: 9.95, w: 0.95, d: 0.85, h: 1.45,
            verdict: .partial,
            blurb: "Cache invalidate-after-commit + WS relay + idempotency. TF Memorystore tier BASIC. Prod ConfigMap points at in-cluster redis that the overlay does not install.",
            cites: [
                Cite(file: "apps/backend-go/cache/cache.go", line: 118, note: "Invalidate + PUBLISH"),
                Cite(file: "infra/terraform/main.tf", line: 57, note: "tier = BASIC"),
                Cite(file: "infra/k8s/backend-go/configmap.yaml", line: 12)
            ]
        ),
        Building(
            id: "gcs", name: "Updates loft", kind: .shed,
            district: .vaults, gx: 7.9, gy: 10.0, w: 0.85, d: 0.75, h: 1.05,
            verdict: .partial,
            blurb: "updateroutes + media upload ticket. GCS init is best-effort in NewApp.",
            cites: [
                Cite(file: "apps/backend-go/updateroutes", line: 1),
                Cite(file: "apps/backend-go/platformroutes/routes.go", line: 32, note: "GET /v1/media/upload-ticket")
            ]
        ),
        Building(
            id: "webhooks", name: "Webhook gate", kind: .kiosk,
            district: .money, gx: 9.15, gy: 3.3, w: 0.85, d: 0.8, h: 1.6,
            verdict: .real,
            blurb: "Ingress only. GlobalPay / Adyen / Stripe / Payme / Click. Persist + outbox. Ingress ≠ live checkout-init.",
            cites: [
                Cite(file: "apps/backend-go/webhookroutes/routes.go", line: 23),
                Cite(file: "apps/backend-go/main.go", line: 230)
            ]
        ),
        Building(
            id: "globalpay", name: "GLOBAL_PAY booth", kind: .kiosk,
            district: .money, gx: 9.2, gy: 4.6, w: 0.75, d: 0.7, h: 1.35,
            verdict: .partial,
            blurb: "Real HTTP executor. Empty creds stub a test URL. Non-prod mounts /sim/globalpay. Prod ConfigMap GLOBAL_PAY_ENV=production.",
            cites: [
                Cite(file: "apps/backend-go/payment/execution.go", line: 140),
                Cite(file: "apps/backend-go/main.go", line: 349, note: "/sim/globalpay"),
                Cite(file: "infra/k8s/backend-go/configmap.yaml", line: 29)
            ]
        ),
        Building(
            id: "stripe-adyen", name: "Stripe / Adyen façade", kind: .lot,
            district: .money, gx: 9.15, gy: 5.8, w: 0.85, d: 0.75, h: 0.55,
            verdict: .theatre,
            blurb: "staticProviderExecutor writes RedirectURL /v1/payment/redirect/{stripe|adyen}/{id}. That path is not mounted. No Charge/Intent create.",
            cites: [
                Cite(file: "apps/backend-go/payment/execution.go", line: 145, note: "ADYEN static"),
                Cite(file: "apps/backend-go/payment/execution.go", line: 153, note: "STRIPE static"),
                Cite(file: "apps/backend-go/payment/execution.go", line: 345, note: "RedirectURL = prefix + sessionID")
            ]
        ),
        Building(
            id: "fiscal", name: "Fiscal kiosk", kind: .kiosk,
            district: .money, gx: 9.2, gy: 7.1, w: 0.75, d: 0.7, h: 1.4,
            verdict: .partial,
            blurb: "FISCAL_PROVIDER unset → PEGASUS in-process PX-RCPT (tax_ofd false). MY_SOLIQ has no EDSSigner. Staging overlay sets PEGASUS.",
            cites: [
                Cite(file: "apps/backend-go/order/fiscal.go", line: 34),
                Cite(file: "apps/backend-go/order/fiscal_provider.go", line: 16),
                Cite(file: "infra/k8s/overlays/staging/kustomization.yaml", line: 86, note: "FISCAL_PROVIDER=PEGASUS")
            ]
        ),
        Building(
            id: "payout-lot", name: "Payout lot", kind: .lot,
            district: .money, gx: 9.15, gy: 8.4, w: 0.85, d: 0.7, h: 0.35,
            verdict: .gone,
            blurb: "No payout executor on this commit. Earnings = SUM(PaymentLedgerEntries).",
            cites: [
                Cite(file: "apps/backend-go/supplier/service.go", line: 596, note: "billing setup stores fields only")
            ]
        ),
        Building(
            id: "ai-worker", name: "AI worker", kind: .stack,
            district: .yard, gx: 11.0, gy: 4.4, w: 0.85, d: 0.85, h: 1.9,
            verdict: .partial,
            blurb: "In prod overlay. Kafka consumers + /healthz. Replicas 1.",
            cites: [
                Cite(file: "apps/ai-worker/main.go", line: 1),
                Cite(file: "infra/k8s/overlays/prod/kustomization.yaml", line: 9)
            ]
        ),
        Building(
            id: "optimizer", name: "OR-Tools sidecar", kind: .plant,
            district: .yard, gx: 12.05, gy: 4.4, w: 0.8, d: 0.85, h: 1.7,
            verdict: .partial,
            blurb: "optimizer-core replicas 1, port 8082, image :local in the manifest. Compose SSMR builds it.",
            cites: [
                Cite(file: "infra/k8s/optimizer-core/deployment.yaml", line: 9),
                Cite(file: "infra/docker-compose.ssmr.yml", line: 1)
            ]
        ),
        Building(
            id: "osrm", name: "OSRM shed", kind: .shed,
            district: .yard, gx: 11.05, gy: 6.0, w: 0.8, d: 0.7, h: 1.05,
            verdict: .gated,
            blurb: "Deployment exists. Not in prod overlay resources. Staging patches an osrm Deployment it does not include.",
            cites: [Cite(file: "infra/k8s/osrm/deployment.yaml", line: 1)]
        ),
        Building(
            id: "handoff", name: "Handoff hut", kind: .kiosk,
            district: .yard, gx: 12.1, gy: 6.0, w: 0.7, d: 0.65, h: 1.1,
            verdict: .gated,
            blurb: "Compose profile only. No k8s. Backend also embeds packages/handoff in-process.",
            cites: [Cite(file: "apps/handoff-service/main.go", line: 1)]
        ),
        Building(
            id: "terraform", name: "Terraform foundry", kind: .plant,
            district: .yard, gx: 11.05, gy: 7.5, w: 0.95, d: 0.9, h: 1.85,
            verdict: .gated,
            blurb: "Declares VPC, Spanner 100 PU, Redis BASIC, GSM. GKE Autopilot behind enable_gke default false. Does not apply itself.",
            cites: [
                Cite(file: "infra/terraform/main.tf", line: 55),
                Cite(file: "infra/terraform/gke.tf", line: 1, note: "enable_gke default false"),
                Cite(file: "infra/terraform/gke.tf", line: 44, note: "enable_autopilot = true")
            ]
        ),
        Building(
            id: "gke", name: "GKE Autopilot (off)", kind: .lot,
            district: .yard, gx: 12.2, gy: 7.55, w: 0.75, d: 0.85, h: 0.45,
            verdict: .gated,
            blurb: "Cluster resource is count = var.enable_gke ? 1 : 0. Default off.",
            cites: [Cite(file: "infra/terraform/gke.tf", line: 40)]
        ),
        Building(
            id: "cells-lot", name: "Cells — empty lot", kind: .lot,
            district: .yard, gx: 11.1, gy: 9.3, w: 1.7, d: 0.95, h: 0.3,
            verdict: .absent,
            blurb: "No auth/cell_directory.go, no infra/terraform/cells, no overlays/cells on this SHA. Multi-cell is later work.",
            cites: [
                Cite(file: "apps/backend-go/auth/claims.go", line: 1, note: "auth/ on main is JWT + scopes only")
            ]
        ),
        Building(
            id: "pack-lot", name: "MarketPack — empty lot", kind: .lot,
            district: .yard, gx: 11.1, gy: 2.7, w: 1.7, d: 0.85, h: 0.3,
            verdict: .absent,
            blurb: "No market_pack.go, no GET /v1/auth/session, no tenantreg. Checkout does not read a pack. Policy default CASH+GLOBAL_PAY.",
            cites: [
                Cite(file: "apps/backend-go/platformroutes/routes.go", line: 20, note: "no session / packs / cells / register"),
                Cite(file: "apps/backend-go/payment/policy.go", line: 12, note: "default CASH + GLOBAL_PAY")
            ]
        )
    ]

    static let traces: [Trace] = [
        Trace(
            id: "order-create",
            title: "Order create",
            lane: .data,
            verdict: .real,
            payload: "ORDER_CREATED  events.OrderEvent  TopicMain / pegasusx-main",
            hops: [
                "Retailer desktop cart → POST /v1/checkout/unified",
                "or POST /v1/order/create (no api-client wrapper)",
                "SessionAuth + RequireRole(RETAILER)",
                "HandleCreate / UnifiedCheckout → claims.Subject",
                "ResolveNearestWarehouseID (H3 res 9)",
                "Spanner Orders + OutboxEvents same txn",
                "OutboxRelay → pegasusx-main",
                "void-notification-dispatcher handleOrderEvent",
                "Hub.Broadcast rooms (only if workers+Kafka run)"
            ],
            waypoints: [
                (1.0, 4.85), (2.9, 4.9), (4.2, 4.95), (5.8, 4.05),
                (5.8, 3.7), (5.45, 0.55), (7.0, 0.55), (8.4, 0.55),
                (9.2, 1.75), (5.85, 10.25)
            ],
            cites: [
                Cite(file: "apps/backend-go/orderroutes/routes.go", line: 40),
                Cite(file: "apps/backend-go/paymentroutes/routes.go", line: 29),
                Cite(file: "apps/backend-go/order/service.go", line: 2311),
                Cite(file: "apps/backend-go/order/service.go", line: 1160),
                Cite(file: "apps/backend-go/order/service.go", line: 1298),
                Cite(file: "apps/backend-go/events/events.go", line: 119),
                Cite(file: "apps/backend-go/outbox/relay.go", line: 48),
                Cite(file: "apps/backend-go/kafka/notification_dispatcher.go", line: 110)
            ],
            explainer: """
            Durable write is REAL. Desktop cart calls POST /v1/checkout/unified, which \
            binds to the same Create. JWT subject is the retailer id. OutboxEvents \
            ORDER_CREATED is in the same Spanner commit. Relay publishes pegasusx-main \
            when workers run. Create itself does not Broadcast — the stale comment on \
            Create is theatre. WS/FCM fire only after void-notification-dispatcher. \
            The order mutator ignores ORDER_CREATED (it handles PAYMENT_CLEARED / \
            FISCAL_RECEIPT_REQUESTED). Warehouse pick is H3 res 9 + coverage radius; \
            ResolveServingWarehouse and ParentOrders are not on this commit.
            """
        ),
        Trace(
            id: "delivery",
            title: "Driver delivery",
            lane: .control,
            verdict: .real,
            payload: "arrive / shop-closed / deliver / collect-cash / fiscal/retry",
            hops: [
                "Driver app",
                "RequireRole(DRIVER)",
                "orderroutes delivery POSTs",
                "Order row + outbox",
                "Fiscal retry if OFD failed"
            ],
            waypoints: [
                (0.6, 7.8), (2.9, 5.2), (4.2, 5.2), (5.8, 4.1), (9.5, 7.4)
            ],
            cites: [
                Cite(file: "apps/backend-go/orderroutes/routes.go", line: 44),
                Cite(file: "apps/backend-go/orderroutes/routes.go", line: 52),
                Cite(file: "apps/backend-go/driverroutes/routes.go", line: 43)
            ],
            explainer: """
            Last-mile mutations sit on orderroutes, not a separate delivery service. \
            Fiscal retry is DRIVER / ADMIN / WAREHOUSE_ADMIN. Pay-at-delivery cash \
            collection is POST /v1/order/collect-cash, not the retailer cash-checkout \
            JSON ack.
            """
        ),
        Trace(
            id: "pay-at-delivery",
            title: "Pay-at-delivery",
            lane: .money,
            verdict: .partial,
            payload: "PAYMENT_REQUIRED  session + attempt  FinanceEvent",
            hops: [
                "Retailer",
                "POST /v1/order/card-checkout | cash-checkout",
                "payment.Service initCheckoutSession",
                "ProviderExecutionRouter",
                "GLOBAL_PAY HTTP or CASH MANUAL",
                "Webhook later → PAYMENT_CLEARED"
            ],
            waypoints: [
                (0.9, 4.8), (4.2, 5.4), (6.9, 5.6), (5.8, 7.1), (9.5, 4.9), (9.5, 3.7)
            ],
            cites: [
                Cite(file: "apps/backend-go/retailerroutes/routes.go", line: 226),
                Cite(file: "apps/backend-go/payment/execution.go", line: 139),
                Cite(file: "apps/backend-go/paymentroutes/routes.go", line: 28)
            ],
            explainer: """
            Pre-delivery POST /v1/checkout/b2b is 410. Unified checkout with items[] \
            goes to the cart/order path; an order_id body is also 410. Card checkout \
            runs ExecutionActionCheckoutInit. CASH retailer checkout returns PENDING \
            without calling the executor — real cash is collect-cash on the order.
            """
        ),
        Trace(
            id: "stripe-theatre",
            title: "Stripe / Adyen façade",
            lane: .theatre,
            verdict: .theatre,
            payload: "RedirectURL = /v1/payment/redirect/{stripe|adyen}/{session}",
            hops: [
                "CheckoutInit STRIPE|ADYEN",
                "staticProviderExecutor",
                "Synthetic redirect string",
                "No route mount",
                "No Charge / PaymentIntent"
            ],
            waypoints: [
                (5.8, 7.2), (9.5, 6.1)
            ],
            cites: [
                Cite(file: "apps/backend-go/payment/execution.go", line: 153),
                Cite(file: "apps/backend-go/payment/execution.go", line: 332),
                Cite(file: "apps/backend-go/payment/execution.go", line: 345)
            ],
            explainer: """
            On this commit the Stripe and Adyen executors are static. They do not \
            talk to Stripe or Adyen. They concatenate a prefix that paymentroutes \
            never registers. Webhook handlers for those brands exist and can persist \
            if someone posts a signed payload — that is ingress, not checkout-init.
            """
        ),
        Trace(
            id: "webhook-ingress",
            title: "PSP webhook ingress",
            lane: .money,
            verdict: .real,
            payload: "FinanceEvent PAYMENT_CLEARED | PAYMENT_FAILED | PAYMENT_REQUIRED",
            hops: [
                "PSP",
                "POST /v1/webhooks/{global-pay|adyen|stripe|payme|click}",
                "Verify + WebhookInbox",
                "outbox FinanceEvent",
                "order consumer SettleExternalPayment"
            ],
            waypoints: [
                (10.4, 3.6), (9.5, 3.7), (5.8, 7.1), (5.5, 0.5), (7.0, 0.5), (9.6, 0.6)
            ],
            cites: [
                Cite(file: "apps/backend-go/webhookroutes/routes.go", line: 23),
                Cite(file: "apps/backend-go/main.go", line: 230)
            ],
            explainer: """
            Five ingress doors. Each verifies (Basic, HMAC, Stripe-Signature, MD5) \
            and writes. PAYME and CLICK have webhooks and no checkout executor on \
            this SHA. Inbox reconciler runs with the other workers.
            """
        ),
        Trace(
            id: "fiscal",
            title: "Fiscal hard-gate",
            lane: .money,
            verdict: .partial,
            payload: "FISCAL_RECEIPT_REQUESTED → PX-RCPT-* (tax_ofd: false)",
            hops: [
                "Capture / collect-cash",
                "FISCAL_RECEIPT_REQUESTED outbox",
                "Consumer ApplyFiscalWorkerResult",
                "PegasusReceiptProvider",
                "GET /v1/platform/receipts/{id}"
            ],
            waypoints: [
                (5.8, 4.1), (9.5, 7.4), (7.0, 0.5), (9.6, 0.6), (4.2, 5.0)
            ],
            cites: [
                Cite(file: "apps/backend-go/order/fiscal.go", line: 34),
                Cite(file: "apps/backend-go/order/fiscal_provider.go", line: 16),
                Cite(file: "apps/backend-go/orderroutes/routes.go", line: 32, note: "platform receipt GET")
            ],
            explainer: """
            Default provider is PEGASUS: an in-process commercial receipt, not Soliq \
            OFD. MY_SOLIQ constructs without an EDSSigner so live OFD cannot succeed. \
            Staging sets FISCAL_PROVIDER=PEGASUS. Prod overlay leaves the env unset \
            (same default).
            """
        ),
        Trace(
            id: "telemetry",
            title: "Driver telemetry",
            lane: .live,
            verdict: .real,
            payload: "location envelope + optional DELIVERY_ARRIVING",
            hops: [
                "Driver",
                "POST /v1/telemetry/location",
                "LastLocations.SaveDriverLocation",
                "TelemetryHub.Broadcast",
                "Approach → retailer room"
            ],
            waypoints: [
                (0.6, 7.8), (4.2, 5.6), (9.2, 1.8), (0.9, 4.8)
            ],
            cites: [
                Cite(file: "apps/backend-go/telemetryroutes/routes.go", line: 92),
                Cite(file: "apps/backend-go/telemetryroutes/routes.go", line: 107, note: "handleLocation"),
                Cite(file: "apps/backend-go/ws/hub.go", line: 3)
            ],
            explainer: """
            Location is claims-derived. Saved to the last-location store, then \
            broadcast on telemetry rooms. If the next stop is inside the approach \
            radius, a DELIVERY_ARRIVING payload is also sent to retailer:{id}.
            """
        ),
        Trace(
            id: "ws-fanout",
            title: "Cross-pod WS relay",
            lane: .live,
            verdict: .real,
            payload: "{source, room, payload} on ws:<hub>:fanout",
            hops: [
                "Hub.Broadcast",
                "fanoutLocal",
                "Redis Publish fail-open",
                "peer StartRelaySubscriber",
                "role rooms"
            ],
            waypoints: [
                (9.2, 1.8), (7.1, 10.3), (4.2, 5.0), (1.5, 2.4)
            ],
            cites: [
                Cite(file: "apps/backend-go/ws/hub.go", line: 3),
                Cite(file: "apps/backend-go/ws/handler.go", line: 84),
                Cite(file: "apps/backend-go/bootstrap/bootstrap.go", line: 615)
            ],
            explainer: """
            Kafka is not the websocket bus. Consumers may call hubs; hubs talk \
            Redis. A Publish failure must not fail the HTTP handler. Unauthenticated \
            upgrade is 401 before gorilla Upgrade.
            """
        ),
        Trace(
            id: "cache-kill",
            title: "Cache kill-switch",
            lane: .data,
            verdict: .real,
            payload: "DEL keys + PUBLISH invalidation channel",
            hops: [
                "Mutation commits",
                "cache.Invalidate",
                "local DEL",
                "Redis publish",
                "peer StartInvalidationSubscriber"
            ],
            waypoints: [
                (5.8, 4.1), (7.1, 10.3), (3.9, 3.5)
            ],
            cites: [
                Cite(file: "apps/backend-go/cache/cache.go", line: 2),
                Cite(file: "apps/backend-go/cache/cache.go", line: 118),
                Cite(file: "apps/backend-go/runtime_workers.go", line: 21)
            ],
            explainer: """
            Doctrine: invalidate after commit, never before. Peers subscribe so a \
            write on pod A drops the key on pod B. Memorystore is BASIC in Terraform \
            — no HA claim.
            """
        ),
        Trace(
            id: "dual-trucks",
            title: "Dual truck planes",
            lane: .control,
            verdict: .real,
            payload: "SupplierTruckManifests ≠ FactoryTruckManifests",
            hops: [
                "Supplier dispatch",
                "SupplierTruckManifests",
                "Factory plant",
                "FactoryTruckManifests"
            ],
            waypoints: [
                (5.8, 5.6), (5.8, 10.3), (7.1, 7.1), (5.8, 10.3)
            ],
            cites: [
                Cite(file: "apps/backend-go/schema/spanner.ddl", line: 798),
                Cite(file: "apps/backend-go/schema/spanner.ddl", line: 884)
            ],
            explainer: """
            Two parent tables, two indexes, two product planes. Merging them in a \
            dashboard is a doctrine violation even if the UI would look simpler.
            """
        )
    ]

    static func building(_ id: String) -> Building? {
        buildings.first { $0.id == id }
    }

    static func trace(_ id: String) -> Trace? {
        traces.first { $0.id == id }
    }

    static func mark(_ id: String) -> String {
        codes[id] ?? "—"
    }

    static let codes: [String: String] = [
        "supplier-portal": "HQ",
        "supplier-native": "SN",
        "admin-portal": "AD",
        "retailer-row": "RT",
        "warehouse-row": "WH",
        "driver-row": "DR",
        "factory-row": "FA",
        "payload-row": "PL",
        "auth-gate": "AU",
        "chi-colonnade": "CH",
        "bootstrap": "BS",
        "idempotency": "ID",
        "health": "HZ",
        "supplier-svc": "SS",
        "catalog": "CA",
        "order-hall": "OR",
        "proximity": "H3",
        "warehouse-ops": "WO",
        "retailer-svc": "RS",
        "payment-court": "PY",
        "factory-ops": "FO",
        "payload-svc": "PS",
        "driver-svc": "DS",
        "credit": "CR",
        "returns": "RV",
        "outbox": "OB",
        "kafka": "KF",
        "dispatcher": "ND",
        "consumers": "CN",
        "ws-masts": "WS",
        "spanner": "SP",
        "redis": "RD",
        "gcs": "GS",
        "webhooks": "HK",
        "globalpay": "GP",
        "stripe-adyen": "ST",
        "fiscal": "FX",
        "payout-lot": "PO",
        "ai-worker": "AI",
        "optimizer": "OP",
        "osrm": "OS",
        "handoff": "HF",
        "terraform": "TF",
        "gke": "GK",
        "cells-lot": "CL",
        "pack-lot": "PK"
    ]
}
