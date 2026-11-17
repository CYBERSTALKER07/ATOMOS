import unittest
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import consumer

class TestDispatchOptimizerConsumer(unittest.TestCase):
    def test_solve_replan_anchors_arrived_stops(self):
        payload = {
            "manifest_id": "man-123",
            "supplier_id": "sup-456",
            "anchored_order_ids": ["ord-arrived-1"],
            "orders": [
                {"order_id": "ord-arrived-1", "lat": 41.31, "lng": 69.24},
                {"order_id": "ord-cand-2", "lat": 41.32, "lng": 69.25},
                {"order_id": "ord-cand-3", "lat": 41.30, "lng": 69.23},
            ]
        }
        res = consumer.solve_replan("MANIFEST_ORDER_INJECTED", payload)
        self.assertEqual(res["manifest_id"], "man-123")
        self.assertEqual(res["supplier_id"], "sup-456")
        new_ids = res["new_order_ids"]
        self.assertEqual(len(new_ids), 3)
        self.assertEqual(new_ids[0], "ord-arrived-1", "Anchored stop must remain first")
        self.assertIn("ord-cand-2", new_ids)
        self.assertIn("ord-cand-3", new_ids)

    def test_solve_replan_shop_closed_excludes_closed_order(self):
        payload = {
            "manifest_id": "man-123",
            "supplier_id": "sup-456",
            "order_id": "ord-closed",
            "orders": [
                {"order_id": "ord-1", "lat": 41.31, "lng": 69.24},
                {"order_id": "ord-closed", "lat": 41.32, "lng": 69.25},
                {"order_id": "ord-2", "lat": 41.30, "lng": 69.23},
            ]
        }
        res = consumer.solve_replan("DRIVER_ORDER_SHOP_CLOSED_ESCALATED", payload)
        new_ids = res["new_order_ids"]
        self.assertEqual(len(new_ids), 2)
        self.assertNotIn("ord-closed", new_ids)
        self.assertIn("ord-1", new_ids)
        self.assertIn("ord-2", new_ids)

    def test_solve_vrp_chunking_fallback(self):
        payload = {
            "fleet": [{"driver_id": "d1", "vehicle_id": "v1"}, {"driver_id": "d2", "vehicle_id": "v2"}],
            "orders": [{"order_id": f"o{i}"} for i in range(4)]
        }
        res = consumer.solve_vrp(payload)
        self.assertEqual(len(res["routes"]), 2)
        total_orders = sum(len(r["order_ids"]) for r in res["routes"])
        self.assertEqual(total_orders, 4)
        for r in res["routes"]:
            self.assertIn("wave_index", r)
            self.assertEqual(r["wave_index"], 1)

    def test_solve_vrp_propagates_wave_index_from_fleet_route(self):
        payload = {
            "depot": {"lat": 41.2995, "lng": 69.2401},
            "fleet": [{"vehicle_id": "v1", "capacity_vu": 100}],
            "orders": [
                {"order_id": "o1", "lat": 41.31, "lng": 69.27, "volume_vu": 30},
                {"order_id": "o2", "lat": 41.32, "lng": 69.28, "volume_vu": 30},
            ]
        }
        res = consumer.solve_vrp(payload)
        self.assertIn("routes", res)
        self.assertEqual(len(res["routes"]), 1)
        self.assertIn("wave_index", res["routes"][0])
        self.assertEqual(res["routes"][0]["wave_index"], 1)

    def test_extract_header_and_dlq_routing(self):
        class FakeMsg:
            def headers(self):
                return [("trace_id", b"tr-123"), ("traceparent", b"00-abc-def-01")]
            def topic(self):
                return "orders.v1"
            def partition(self):
                return 2
            def offset(self):
                return 42
            def key(self):
                return b"sup-456"
            def value(self):
                return b"invalid-json"

        class FakeProducer:
            def __init__(self):
                self.produced = []
            def produce(self, topic, key=None, value=None, headers=None):
                self.produced.append((topic, key, value, dict(headers or [])))
            def flush(self, timeout=None):
                return 0

        msg = FakeMsg()
        self.assertEqual(consumer._extract_header(msg, "trace_id"), "tr-123")
        self.assertEqual(consumer._extract_header(msg, "traceparent"), "00-abc-def-01")
        prod = FakeProducer()
        consumer._route_to_dlq(prod, msg, "decode_error", trace_id="tr-123")
        self.assertEqual(len(prod.produced), 1)
        topic, key, val, hdrs = prod.produced[0]
        self.assertEqual(topic, consumer.DLQ_TOPIC)
        self.assertEqual(key, b"sup-456")
        self.assertEqual(val, b"invalid-json")
        self.assertEqual(hdrs[b"original_topic" if b"original_topic" in hdrs else "original_topic"], b"orders.v1")
        self.assertEqual(hdrs[b"trace_id" if b"trace_id" in hdrs else "trace_id"], b"tr-123")

    def test_solve_vrp_fallback_bin_packing_respects_capacity(self):
        old_main = consumer.main
        consumer.main = None
        try:
            payload = {
                "fleet": [{"driver_id": "d1", "vehicle_id": "v1", "capacity_vu": 50}],
                "orders": [
                    {"order_id": "o1", "volume_vu": 40},
                    {"order_id": "o2", "volume_vu": 40},
                ],
            }
            res = consumer.solve_vrp(payload)
            self.assertEqual(len(res["routes"]), 2)
            self.assertEqual([r["wave_index"] for r in res["routes"]], [1, 2])
        finally:
            consumer.main = old_main

    def test_solve_replan_respects_remaining_capacity(self):
        res = consumer.solve_replan("MANIFEST_ORDER_INJECTED", {
            "manifest_id": "m1",
            "supplier_id": "s1",
            "anchored_order_ids": ["a1"],
            "remaining_capacity_vu": 30,
            "orders": [
                {"order_id": "a1", "lat": 41.31, "lng": 69.24, "volume_vu": 50},
                {"order_id": "c1", "lat": 41.311, "lng": 69.241, "volume_vu": 25},
                {"order_id": "c2", "lat": 41.312, "lng": 69.242, "volume_vu": 25},
            ],
        })
        self.assertEqual(len(res["new_order_ids"]), 2)
        self.assertEqual(res["new_order_ids"][0], "a1")

if __name__ == "__main__":
    unittest.main()
