import json
import logging
import uuid
import time
try:
    from confluent_kafka import Consumer, Producer
except ImportError:
    Consumer = None
    Producer = None

try:
    import main
except ImportError:
    main = None

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

SUPPORTED_REPLAN_EVENTS = {
    "MANIFEST_ORDER_INJECTED",
    "DRIVER_ORDER_SHOP_CLOSED_ESCALATED",
    "DRIVER_RESCUE_RESPONDED",
}


def solve_vrp(payload):
    fleet = payload.get("fleet", [])
    orders = payload.get("orders", [])

    if not fleet or not orders:
        return {"routes": []}

    # If main and OR-Tools are available, construct FleetRouteRequest
    if main and hasattr(main, "get_fleet_route"):
        try:
            depot_data = payload.get("depot", {"id": "depot", "x": 41.31, "y": 69.24, "demand_vu": 0})
            depot = main.FleetLocation(
                id=depot_data.get("id", "depot"),
                x=float(depot_data.get("x", depot_data.get("lat", 41.31))),
                y=float(depot_data.get("y", depot_data.get("lng", 69.24))),
                demand_vu=int(depot_data.get("demand_vu", 0)),
            )
            locations = [
                main.FleetLocation(
                    id=o.get("order_id", o.get("id", f"ord-{i}")),
                    x=float(o.get("x", o.get("lat", 41.31))),
                    y=float(o.get("y", o.get("lng", 69.24))),
                    demand_vu=int(o.get("demand_vu", o.get("volume_vu", 10))),
                )
                for i, o in enumerate(orders)
            ]
            vehicles = [
                main.FleetVehicle(
                    id=d.get("vehicle_id", d.get("driver_id", f"veh-{i}")),
                    capacity_vu=int(d.get("capacity_vu", d.get("max_volume_vu", 100))),
                )
                for i, d in enumerate(fleet)
            ]
            req = main.FleetRouteRequest(depot=depot, locations=locations, vehicles=vehicles)
            resp = main.get_fleet_route(req)
            routes = []
            for r in resp.routes:
                routes.append({
                    "driver_id": r.vehicle_id,
                    "vehicle_id": r.vehicle_id,
                    "route_id": str(uuid.uuid4()),
                    "order_ids": r.location_ids,
                })
            return {"routes": routes}
        except Exception as e:
            logger.warning(f"OR-Tools solve_vrp failed, falling back to chunking: {e}")

    # Fallback chunking heuristic
    routes = []
    chunk_size = max(1, len(orders) // len(fleet))
    for i, driver in enumerate(fleet):
        start = i * chunk_size
        end = (i + 1) * chunk_size if i < len(fleet) - 1 else len(orders)
        route_orders = orders[start:end]
        if route_orders:
            routes.append({
                "driver_id": driver.get("driver_id"),
                "vehicle_id": driver.get("vehicle_id"),
                "route_id": str(uuid.uuid4()),
                "order_ids": [o.get("order_id") for o in route_orders],
            })

    return {"routes": routes}


def solve_replan(event_type: str, payload: dict) -> dict:
    manifest_id = payload.get("manifest_id")
    supplier_id = payload.get("supplier_id")
    raw_orders = payload.get("orders") or payload.get("remaining_orders") or []
    anchored_order_ids = payload.get("anchored_order_ids") or []

    if not raw_orders:
        return {"manifest_id": manifest_id, "supplier_id": supplier_id, "new_order_ids": []}

    # Exclude shop-closed orders if event is DRIVER_ORDER_SHOP_CLOSED_ESCALATED
    if event_type == "DRIVER_ORDER_SHOP_CLOSED_ESCALATED":
        closed_order_id = payload.get("order_id")
        raw_orders = [o for o in raw_orders if o.get("order_id") != closed_order_id]

    # Extract candidate orders (those not already anchored)
    candidates = [o for o in raw_orders if o.get("order_id") not in anchored_order_ids]

    optimized_candidate_ids = []
    if main and hasattr(main, "solve_tsp") and candidates:
        try:
            locs = []
            for o in candidates:
                x = float(o.get("lat", o.get("x", 41.31)))
                y = float(o.get("lng", o.get("y", 69.24)))
                locs.append(main.Location(id=o.get("order_id", o.get("id")), x=x, y=y))
            optimized_candidate_ids = main.solve_tsp(locs)
        except Exception as e:
            logger.warning(f"solve_tsp failed during replan: {e}")
            optimized_candidate_ids = [o.get("order_id") for o in candidates if o.get("order_id")]
    else:
        optimized_candidate_ids = [o.get("order_id") for o in candidates if o.get("order_id")]

    final_sequence = list(anchored_order_ids) + [
        oid for oid in optimized_candidate_ids if oid not in anchored_order_ids
    ]
    return {
        "manifest_id": manifest_id,
        "supplier_id": supplier_id,
        "new_order_ids": final_sequence,
        "timestamp": int(time.time() * 1000),
    }


def start_consumer(brokers="localhost:9092", topics=None):
    if topics is None:
        topics = ["pegasusx.main", "orders.v1"]
    if isinstance(topics, str):
        topics = [topics]

    conf = {
        "bootstrap.servers": brokers,
        "group.id": "dispatch-optimizer-group",
        "auto.offset.reset": "earliest",
    }

    consumer = Consumer(conf)
    consumer.subscribe(topics)

    producer_conf = {"bootstrap.servers": brokers}
    producer = Producer(producer_conf)

    logger.info(f"Starting optimizer consumer on {brokers}, topics {topics}")

    try:
        while True:
            msg = consumer.poll(timeout=1.0)
            if msg is None:
                continue
            if msg.error():
                logger.error(f"Consumer error: {msg.error()}")
                continue

            try:
                val = json.loads(msg.value().decode("utf-8"))
                event_type = val.get("type")

                if event_type == "DISPATCH_REQUESTED":
                    logger.info(f"Received DISPATCH_REQUESTED for supplier {val.get('supplier_id')}")
                    solution = solve_vrp(val)
                    planned_event = {
                        "event_id": str(uuid.uuid4()),
                        "type": "DISPATCH_PLANNED",
                        "supplier_id": val.get("supplier_id"),
                        "warehouse_id": val.get("warehouse_id"),
                        "routes": solution.get("routes"),
                        "timestamp": int(time.time() * 1000),
                    }
                    producer.produce(
                        msg.topic(),
                        key=val.get("supplier_id", "").encode("utf-8"),
                        value=json.dumps(planned_event).encode("utf-8"),
                    )
                    producer.flush()
                    logger.info(f"Emitted DISPATCH_PLANNED for supplier {val.get('supplier_id')}")

                elif event_type in SUPPORTED_REPLAN_EVENTS:
                    logger.info(f"Received disruption event {event_type} for manifest {val.get('manifest_id')}")
                    replan_solution = solve_replan(event_type, val)
                    if replan_solution.get("new_order_ids"):
                        replanned_event = {
                            "event_id": str(uuid.uuid4()),
                            "type": "ROUTE_REPLANNED",
                            "supplier_id": replan_solution.get("supplier_id"),
                            "manifest_id": replan_solution.get("manifest_id"),
                            "driver_id": val.get("driver_id"),
                            "new_order_ids": replan_solution.get("new_order_ids"),
                            "replan_reason": event_type.lower(),
                            "timestamp": replan_solution.get("timestamp"),
                        }
                        producer.produce(
                            msg.topic(),
                            key=val.get("supplier_id", "").encode("utf-8"),
                            value=json.dumps(replanned_event).encode("utf-8"),
                        )
                        producer.flush()
                        logger.info(f"Emitted ROUTE_REPLANNED for manifest {val.get('manifest_id')}")

            except Exception as e:
                logger.error(f"Error processing message: {e}")

    except KeyboardInterrupt:
        pass
    finally:
        consumer.close()


if __name__ == "__main__":
    start_consumer()
