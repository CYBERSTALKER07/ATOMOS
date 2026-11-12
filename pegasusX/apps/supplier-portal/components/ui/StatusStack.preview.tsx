"use client";

import { ORDER_STATUS_FUNNEL, emptyOrderStatusCounts } from "@pegasusx/types";
import { StatusStack } from "./StatusStack";

const liveCounts = {
  ...emptyOrderStatusCounts(),
  PENDING: 4,
  LOADED: 2,
  IN_TRANSIT: 3,
  COMPLETED: 1,
};

/** Story/preview: empty, zero, unavailable, full funnel. */
export function StatusStackPreview() {
  return (
    <div data-testid="gs-u-status-stack-preview" style={{ display: "grid", gap: 24 }}>
      <section>
        <h3>empty</h3>
        <StatusStack dictionary={ORDER_STATUS_FUNNEL} counts={null} />
      </section>
      <section>
        <h3>zero</h3>
        <StatusStack dictionary={ORDER_STATUS_FUNNEL} counts={emptyOrderStatusCounts()} />
      </section>
      <section>
        <h3>unavailable</h3>
        <StatusStack dictionary={ORDER_STATUS_FUNNEL} counts={liveCounts} available={false} />
      </section>
      <section>
        <h3>live 17</h3>
        <StatusStack dictionary={ORDER_STATUS_FUNNEL} counts={liveCounts} />
      </section>
    </div>
  );
}
