"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { createSupplierApi } from "@/lib/api";
import type { SupplierSupplyLaneRow } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import { SupplyLanesList } from "@/components/supply-lanes";

const api = createSupplierApi();

export default function SupplyLanesPage() {
  const [lanes, setLanes] = useState<SupplierSupplyLaneRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .getSupplierSupplyLanes()
      .then((resp) => setLanes(resp.lanes))
      .catch((err) => setError(err instanceof Error ? err.message : "load_supply_lanes_failed"))
      .finally(() => setLoading(false));
  }, []);

  return (
    <PageChrome
      icon="fleet"
      title="Supply lanes"
      description="Warehouse lanes derived from topology and live order volume. Lanes are read-only — edit warehouses and coverage on Topology."
      loading={loading}
      error={error}
      empty={!loading && lanes.length === 0}
      emptyMessage="No active warehouse lanes. Configure nodes on topology."
    >
      <SupplyLanesList lanes={lanes} />
    </PageChrome>
  );
}
