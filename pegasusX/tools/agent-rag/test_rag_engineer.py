#!/usr/bin/env python3
import os
import sys
import unittest

os.environ["HF_HUB_DISABLE_SYMLINKS_WARNING"] = "1"
os.environ["TOKENIZERS_PARALLELISM"] = "false"
os.environ["TRANSFORMERS_VERBOSITY"] = "error"
os.environ["TQDM_DISABLE"] = "1"
os.environ["HF_HUB_OFFLINE"] = "1"

from hybrid_search import hybrid_search, search_dense, search_bm25, find_symbol, sanitize_fts_query
from indexer import chunk_code_ast, compute_file_hash

class TestRAGEngineer(unittest.TestCase):

    def test_sanitize_fts_query(self):
        q = sanitize_fts_query("defaultOrderID")
        self.assertIn("defaultOrderID", q)
        
        q_multi = sanitize_fts_query("order payment method: spanner")
        self.assertNotIn(":", q_multi)
        self.assertIn("order", q_multi)

    def test_go_ast_chunking(self):
        go_code = """package order

type Service struct {
    ID string
}

func (s *Service) UpdateStatus(ctx context.Context, orderID string) error {
    return nil
}

func defaultOrderID() string {
    return "test-id"
}
"""
        chunks = chunk_code_ast("apps/backend-go/order/test.go", go_code, "go", "backend")
        self.assertTrue(len(chunks) >= 2)
        symbols = [c["symbol"] for c in chunks]
        self.assertTrue(any("UpdateStatus" in s for s in symbols))
        self.assertTrue(any("defaultOrderID" in s for s in symbols))
        self.assertTrue(all("Lines:" in c["text"] for c in chunks))

    def test_typescript_ast_chunking(self):
        ts_code = """export interface OrderCardProps {
    orderId: string;
}

export function OrderCard(props: OrderCardProps) {
    return <div>{props.orderId}</div>;
}
"""
        chunks = chunk_code_ast("apps/retailer-portal/test.tsx", ts_code, "tsx", "client-web")
        self.assertTrue(len(chunks) >= 2)
        symbols = [c["symbol"] for c in chunks]
        self.assertIn("OrderCardProps", symbols)
        self.assertIn("OrderCard", symbols)

    def test_hybrid_search_exact_symbol(self):
        # Must return order/service.go as top result for defaultOrderID
        hits = hybrid_search("defaultOrderID", layer="backend", limit=3)
        self.assertTrue(len(hits) > 0)
        top_file = hits[0]["metadata"].get("file", "")
        self.assertIn("order/service.go", top_file)
        self.assertTrue(hits[0]["rrf_score"] > 0.01)

    def test_hybrid_search_semantic(self):
        hits = hybrid_search("preorder delivery window policy", layer="backend", limit=3)
        self.assertTrue(len(hits) > 0)
        files = [h["metadata"].get("file", "") for h in hits]
        self.assertTrue(any("preorder" in f or "order" in f for f in files))

if __name__ == "__main__":
    unittest.main()
