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

if __name__ == "__main__":
    unittest.main()
