import json
import logging
import uuid
import time
from confluent_kafka import Consumer, Producer
from ortools.constraint_solver import routing_enums_pb2
from ortools.constraint_solver import pywrapcp

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def solve_vrp(payload):
    # Dummy implementation of OR-Tools VRP
    # In reality, this would use distance matrices between orders and depot
    
    fleet = payload.get("fleet", [])
    orders = payload.get("orders", [])
    
    if not fleet or not orders:
        return {"routes": []}

    # Simplistic chunking to simulate routing
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
                "order_ids": [o.get("order_id") for o in route_orders]
            })

    return {"routes": routes}

def start_consumer(brokers="localhost:9092", topic="pegasusx.main"):
    conf = {
        'bootstrap.servers': brokers,
        'group.id': 'dispatch-optimizer-group',
        'auto.offset.reset': 'earliest'
    }

    consumer = Consumer(conf)
    consumer.subscribe([topic])

    producer_conf = {'bootstrap.servers': brokers}
    producer = Producer(producer_conf)

    logger.info(f"Starting optimizer consumer on {brokers}, topic {topic}")

    try:
        while True:
            msg = consumer.poll(timeout=1.0)
            if msg is None:
                continue
            if msg.error():
                logger.error(f"Consumer error: {msg.error()}")
                continue
            
            try:
                val = json.loads(msg.value().decode('utf-8'))
                if val.get("type") == "DISPATCH_REQUESTED":
                    logger.info(f"Received DISPATCH_REQUESTED for supplier {val.get('supplier_id')}")
                    
                    # Run optimizer
                    solution = solve_vrp(val)
                    
                    # Emit DISPATCH_PLANNED
                    planned_event = {
                        "event_id": str(uuid.uuid4()),
                        "type": "DISPATCH_PLANNED",
                        "supplier_id": val.get("supplier_id"),
                        "warehouse_id": val.get("warehouse_id"),
                        "routes": solution.get("routes"),
                        "timestamp": int(time.time() * 1000)
                    }
                    
                    producer.produce(
                        topic,
                        key=val.get("supplier_id", "").encode('utf-8'),
                        value=json.dumps(planned_event).encode('utf-8')
                    )
                    producer.flush()
                    logger.info(f"Emitted DISPATCH_PLANNED for supplier {val.get('supplier_id')}")
            except Exception as e:
                logger.error(f"Error processing message: {e}")
                
    except KeyboardInterrupt:
        pass
    finally:
        consumer.close()

if __name__ == "__main__":
    start_consumer()
