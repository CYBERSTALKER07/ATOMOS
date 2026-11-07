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


DLQ_TOPIC = "pegasusx-dead-letter"


def _extract_header(msg, key: str) -> str:
    headers = msg.headers() if hasattr(msg, "headers") and callable(msg.headers) else None
    if not headers:
        return ""
    for k, v in headers:
        if k and k.lower() == key.lower() and v is not None:
            return v.decode("utf-8", errors="replace") if isinstance(v, (bytes, bytearray)) else str(v)
    return ""


def _route_to_dlq(producer, msg, reason: str, trace_id: str = ""):
    if producer is None:
        return
    dlq_headers = [
        ("original_topic", (msg.topic() or "").encode("utf-8")),
        ("dlq_original_topic", (msg.topic() or "").encode("utf-8")),
        ("original_partition", str(msg.partition()).encode("utf-8")),
        ("original_offset", str(msg.offset()).encode("utf-8")),
        ("dlq_reason", str(reason).encode("utf-8")),
        ("dlq_timestamp", str(int(time.time() * 1000)).encode("utf-8")),
    ]
    if trace_id:
        dlq_headers.append(("trace_id", trace_id.encode("utf-8")))
    try:
        producer.produce(
            DLQ_TOPIC,
            key=msg.key(),
            value=msg.value(),
            headers=dlq_headers,
        )
        producer.flush(timeout=5.0)
        logger.warning(f"Routed failed message at {msg.topic()}[{msg.partition()}]@{msg.offset()} to {DLQ_TOPIC}: {reason}")
    except Exception as dlq_err:
        logger.error(f"Failed to route message to DLQ {DLQ_TOPIC}: {dlq_err}")
        raise


def start_consumer(brokers="localhost:9092", topics=None, max_retries=3):
    if topics is None:
        topics = ["pegasusx.main", "orders.v1", "dispatch.v1"]
    if isinstance(topics, str):
        topics = [topics]

    conf = {
        "bootstrap.servers": brokers,
        "group.id": "dispatch-optimizer-group",
        "auto.offset.reset": "earliest",
        "enable.auto.commit": False,
        "session.timeout.ms": 45000,
        "max.poll.interval.ms": 300000,
    }

    consumer = Consumer(conf)
    consumer.subscribe(topics)

    producer_conf = {
        "bootstrap.servers": brokers,
        "acks": "all",
        "enable.idempotence": True,
        "retries": 2147483647,
        "max.in.flight.requests.per.connection": 5,
        "compression.type": "snappy",
        "linger.ms": 10,
    }
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

            trace_id = _extract_header(msg, "trace_id")
            traceparent = _extract_header(msg, "traceparent")

            try:
                val = json.loads(msg.value().decode("utf-8"))
            except Exception as decode_err:
                logger.error(f"Poison pill JSON decode error: {decode_err}")
                try:
                    _route_to_dlq(producer, msg, f"decode_error: {decode_err}", trace_id)
                    consumer.commit(message=msg, asynchronous=False)
                except Exception:
                    pass
                continue

            if not trace_id:
                trace_id = str(val.get("trace_id") or "")
            if not traceparent:
                traceparent = str(val.get("traceparent") or "")

            event_type = val.get("type")
            last_err = None

            for attempt in range(1, max(1, max_retries) + 1):
                try:
                    out_headers = []
                    if trace_id:
                        out_headers.append(("trace_id", trace_id.encode("utf-8")))
                    if traceparent:
                        out_headers.append(("traceparent", traceparent.encode("utf-8")))

                    if event_type == "DISPATCH_REQUESTED":
                        logger.info(f"Received DISPATCH_REQUESTED for supplier {val.get('supplier_id')} (trace_id={trace_id})")
                        solution = solve_vrp(val)
                        planned_event = {
                            "event_id": str(uuid.uuid4()),
                            "type": "DISPATCH_PLANNED",
                            "supplier_id": val.get("supplier_id"),
                            "warehouse_id": val.get("warehouse_id"),
                            "routes": solution.get("routes"),
                            "timestamp": int(time.time() * 1000),
                            "trace_id": trace_id,
                        }
                        producer.produce(
                            msg.topic(),
                            key=(val.get("supplier_id") or "").encode("utf-8"),
                            value=json.dumps(planned_event).encode("utf-8"),
                            headers=out_headers + [("type", b"DISPATCH_PLANNED")],
                        )
                        producer.flush(timeout=10.0)
                        logger.info(f"Emitted DISPATCH_PLANNED for supplier {val.get('supplier_id')}")

                    elif event_type in SUPPORTED_REPLAN_EVENTS:
                        logger.info(f"Received disruption event {event_type} for manifest {val.get('manifest_id')} (trace_id={trace_id})")
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
                                "trace_id": trace_id,
                            }
                            producer.produce(
                                msg.topic(),
                                key=(val.get("supplier_id") or "").encode("utf-8"),
                                value=json.dumps(replanned_event).encode("utf-8"),
                                headers=out_headers + [("type", b"ROUTE_REPLANNED")],
                            )
                            producer.flush(timeout=10.0)
                            logger.info(f"Emitted ROUTE_REPLANNED for manifest {val.get('manifest_id')}")

                    last_err = None
                    break
                except Exception as e:
                    last_err = e
                    logger.warning(f"Attempt {attempt}/{max_retries} failed processing {event_type}: {e}")
                    if attempt < max_retries:
                        time.sleep(0.1 * (2 ** (attempt - 1)))

            if last_err is not None:
                try:
                    _route_to_dlq(producer, msg, f"processing_exhausted: {last_err}", trace_id)
                    consumer.commit(message=msg, asynchronous=False)
                except Exception as dlq_commit_err:
                    logger.error(f"Leaving offset uncommitted after DLQ failure: {dlq_commit_err}")
                continue

            try:
                consumer.commit(message=msg, asynchronous=False)
            except Exception as commit_err:
                logger.error(f"Manual offset commit failed: {commit_err}")

    except KeyboardInterrupt:
        pass
    finally:
        consumer.close()


if __name__ == "__main__":
    start_consumer()
