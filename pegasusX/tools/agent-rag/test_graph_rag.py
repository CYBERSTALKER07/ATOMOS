"""
test_graph_rag.py — Comprehensive Verification Suite for PegasusX Graph RAG.

Validates:
1. Graph Exploration (Symbols, Callers, Outbox, Tables, Kafka)
2. Seam & Flow Tracing (Client API -> HTTP Route -> Service -> Spanner)
3. Multi-Hop Blast Radius Impact Analysis
4. Spanner Schema Dependencies & Tenant Isolation Audit
5. Safe Read-Only Cypher Execution (with safety rejection of mutations)
"""

import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from graph_rag import (
    graph_explore_symbol,
    graph_trace_flow,
    graph_blast_radius,
    graph_schema_dependencies,
    graph_cypher_query,
)


class TestGraphRAG(unittest.TestCase):

    def test_graph_explore_symbol(self):
        result = graph_explore_symbol("Suppliers", depth=1)
        self.assertIn("SpannerTable", result)
        self.assertIn("Suppliers", result)
        self.assertIn("Inbound", result)
        self.assertIn("RepositoryMethod", result)

    def test_graph_schema_dependencies_orders(self):
        result = graph_schema_dependencies("Orders")
        self.assertIn("Spanner Table: `Orders`", result)
        self.assertIn("**Multi-Tenancy Tenant Isolated:** YES", result)
        self.assertIn("**Has Supplier ID Column:** YES", result)
        self.assertIn("Mutations", result)
        self.assertIn("Reads", result)
        self.assertIn("rRepository.CreateOrder", result)

    def test_graph_blast_radius_warehouses(self):
        result = graph_blast_radius("Warehouses", hops=2)
        self.assertIn("Blast Radius Impact Analysis", result)
        self.assertIn("Spanner Database Tables", result)
        self.assertIn("Service & Repository Logic", result)

    def test_graph_trace_flow(self):
        result = graph_trace_flow("login", "route_POST_/v1/auth/warehouse/login")
        self.assertIn("ApiClientMethod", result)
        self.assertIn("RouteEndpoint", result)
        self.assertIn("CONSUMES_ROUTE", result)

    def test_graph_cypher_query_readonly(self):
        result = graph_cypher_query("MATCH (c:ClientApp) RETURN c.id AS id, c.platform AS platform LIMIT 5")
        self.assertIn("Results", result)
        self.assertIn("platform", result)

    def test_graph_cypher_query_blocks_mutations(self):
        result = graph_cypher_query("CREATE (n:TestNode {name: 'illegal'})")
        self.assertIn("[SECURITY ERROR]", result)
        self.assertIn("forbidden", result)

    def test_graph_cypher_query_blocks_delete(self):
        result = graph_cypher_query("MATCH (n:SpannerTable) DELETE n")
        self.assertIn("[SECURITY ERROR]", result)


if __name__ == "__main__":
    unittest.main()
