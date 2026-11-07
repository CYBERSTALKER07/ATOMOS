'use client';

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { useSearchParams } from 'next/navigation';
import { ExplainStatusBanner, explainFromApiError } from '@pegasusx/explain-ui';
import { FACTORY_TRANSFER_STATES, type StatusExplain } from '@pegasusx/types';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
import { downloadCsv, exportCsv } from '@/lib/csv';
import { usePagination } from '@/lib/use-pagination';
import { ListToolbar } from '@/components/ListToolbar';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';
import { PageSection } from '@/components/PageSection';
import EmptyState from '@/components/EmptyState';
import { TransferFilters } from '@/components/transfers/TransferFilters';
import { TransferList, type Transfer } from '@/components/transfers/TransferList';

const STATE_FILTERS = ['ALL', ...FACTORY_TRANSFER_STATES];


export default function TransfersPage() {
  const t = usePortalT();
  const searchParams = useSearchParams();
  const [transfers, setTransfers] = useState<Transfer[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [fetchExplain, setFetchExplain] = useState<StatusExplain | null>(null);
  const initialState = (searchParams.get('state') ?? 'ALL').trim().toUpperCase();
  const [stateFilter, setStateFilter] = useState(
    initialState && STATE_FILTERS.includes(initialState) ? initialState : 'ALL',
  );

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    setFetchExplain(null);
    try {
      const query = stateFilter !== 'ALL' ? `?state=${stateFilter}` : '';
      const res = await apiFetch(`/v1/factory/transfers${query}`);
      if (res.ok) {
        const data = await res.json();
        setTransfers(data.transfers || []);
      } else {
        const body = await res.json().catch(() => ({}));
        setFetchExplain(explainFromApiError(body));
        setError(body.error || `Unable to load transfers (${res.status}).`);
      }
    } catch {
      setError('Unable to load transfers right now.');
    } finally {
      setLoading(false);
    }
  }, [stateFilter]);

  useEffect(() => { load(); }, [load]);

  useFactorySessionReconcile(() => {
    void load();
  });

  const { page, pageCount, pageItems, next, prev, reset } = usePagination(transfers, 25);

  useEffect(() => {
    reset();
  }, [stateFilter, reset]);

  const exportCsvHandler = async () => {
    const result = await exportCsv(
      `factory-transfers${stateFilter !== 'ALL' ? `-${stateFilter.toLowerCase()}` : ''}.csv`,
      ['id', 'warehouse_name', 'state', 'priority', 'total_items', 'total_volume_m3', 'created_at'],
      transfers.map((transfer) => [
        transfer.id,
        transfer.warehouse_name || transfer.destination_warehouse_id,
        transfer.state,
        transfer.priority,
        String(transfer.total_items),
        String(transfer.total_volume_m3),
        transfer.created_at,
      ]),
    );
    if (!result.saved && !result.cancelled && result.reason) {
      setError(`Export failed: ${result.reason}`);
    }
  };

  useEffect(() => {
    const unsubscribe = subscribeFactoryWS({
      onMessage: payload => {
        const event = parseFactoryLiveEvent(payload);
        if (!event) {
          return;
        }
        if (!event.type.startsWith('TRANSFER_') && !event.type.startsWith('MANIFEST_') && !event.type.startsWith('WAREHOUSE_TRANSFER_') && !event.type.startsWith('FACTORY_SUPPLY_')) { return; }
        void load();
      },
    });

    return () => {
      unsubscribe();
    };
  }, [load]);

  const totalVolume = useMemo(
    () => transfers.reduce((sum, transfer) => sum + transfer.total_volume_m3, 0),
    [transfers],
  );
  const approvedCount = useMemo(
    () => transfers.filter((transfer) => transfer.state === 'APPROVED').length,
    [transfers],
  );
  const inFlightCount = useMemo(
    () => transfers.filter((transfer) => ['LOADING', 'DISPATCHED', 'IN_TRANSIT'].includes(transfer.state)).length,
    [transfers],
  );
  const highPriorityCount = useMemo(
    () => transfers.filter((transfer) => transfer.priority === 'HIGH').length,
    [transfers],
  );

  return (
    <PageTransition>
      <PageChrome
        icon="transfers"
        title={t("portal.nav.transfers")}
        description={t("factory_portal.residual.text.review_warehouse_destination_movements_filter_by_state_and_open_")}
        loading={loading}
        skeletonVariant="table"
        error={error && transfers.length === 0 ? error : null}
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <Link
              href="/transfers/create"
              className="portal-btn portal-btn--primary inline-flex h-10 items-center gap-2"
            >
              <Icon name="add" size={16} /> Create transfer
            </Link>
            <button
              type="button"
              onClick={() => void load()}
              className="portal-btn portal-btn--ghost inline-flex h-10 items-center gap-2"
            >
              <Icon name="refresh" size={16} /> Refresh
            </button>
          </div>
        }
      >
        {fetchExplain ? <ExplainStatusBanner explain={fetchExplain} className="mb-4" /> : null}
        <KpiStatGrid columns={4}>
          <KpiStatCard
            label={t("factory_portal.residual.text.visible_transfers")}
            value={transfers.length}
            sub={stateFilter === 'ALL' ? 'Across the full pipeline' : `Filtered to ${stateFilter}`}
          />
          <KpiStatCard label={t("factory_portal.residual.text.approved")} value={approvedCount} sub="Waiting to enter loading" />
          <KpiStatCard label={t("factory_portal.residual.text.in_flight")} value={inFlightCount} sub="Loading, dispatched, or in transit" />
          <KpiStatCard label={t("factory_portal.residual.text.high_priority")} value={highPriorityCount} sub="Require close operator attention" />
        </KpiStatGrid>

        <PageSection
          title={t("factory_portal.transfers.text.pipeline_view")}
          description={t("factory_portal.residual.text.filter_by_transfer_state_total_volume_reflects_the_current_view")}
          className="mt-6"
          actions={
            <span className="rounded-full px-4 py-2 text-sm" style={{ background: 'var(--desk-surface-subtle)', color: 'var(--desk-text-secondary)' }}>
              Total volume: <span className="font-semibold" style={{ color: 'var(--desk-text-primary)' }}>{totalVolume.toFixed(1)} m³</span>
            </span>
          }
        >
          <TransferFilters
            stateFilter={stateFilter}
            setStateFilter={setStateFilter}
            stateFilters={STATE_FILTERS}
          />
        </PageSection>

        {transfers.length === 0 ? (
          <EmptyState
            variant={stateFilter === 'ALL' ? 'no-data' : 'no-results'}
            imageUrl="/images/empty-production-line.png"
            headline={stateFilter === 'ALL' ? 'No transfers found' : 'No transfers match this filter'}
            body={
              stateFilter === 'ALL'
                ? 'Wait for the next warehouse request cycle to create new transfer work.'
                : `There are no ${stateFilter.toLowerCase()} transfers in the current pipeline view.`
            }
            action={stateFilter === 'ALL' ? undefined : 'Clear filter'}
            onAction={stateFilter === 'ALL' ? undefined : () => setStateFilter('ALL')}
          />
        ) : (
          <>
            <ListToolbar
              page={page}
              pageCount={pageCount}
              totalLabel={`${transfers.length} transfers`}
              onPrev={prev}
              onNext={next}
              onExport={() => void exportCsvHandler()}
            />
            <TransferList transfers={pageItems} />
          </>
        )}
      </PageChrome>
    </PageTransition>
  );
}
