'use client';

import { useCallback, useEffect, useState } from 'react';
import {
  warehouseBroadcastKey,
  warehouseBroadcastTemplateCreateKey,
  warehouseBroadcastTemplateDeleteKey,
} from '@pegasusx/api-client';
import type { BroadcastTemplate, RetailerOverridePreview } from '@pegasusx/types';
import { PageChrome } from '@/components/PageChrome';
import { PageSection } from '@/components/PageSection';
import { PortalField, PortalInput, PortalSection } from '@/components/portal';
import { warehouseApi } from '@/lib/warehouse-api';
import { warehouseHomeNodeId, warehouseScopeQuery } from '@/lib/warehouse-scope';

const broadcastRoles = ['DRIVER', 'RETAILER', 'ALL'] as const;

function applyTemplate(
  template: BroadcastTemplate,
  templateDate: string,
  customReason: string,
): { title: string; body: string; role: string } {
  let body = template.body;
  if (body.includes('{date}')) {
    const date = templateDate.trim() || 'the selected date';
    body = body.replaceAll('{date}', date);
  }
  if (body.includes('{reason}')) {
    const reason = customReason.trim() || 'operational delay';
    body = body.replaceAll('{reason}', reason);
  }
  return { title: template.title, body, role: template.default_role };
}

export default function WarehouseOperationsPage() {
  const warehouseId = warehouseHomeNodeId();
  const scope = warehouseScopeQuery();

  const [templates, setTemplates] = useState<BroadcastTemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);

  const [title, setTitle] = useState('');
  const [body, setBody] = useState('');
  const [broadcastRole, setBroadcastRole] = useState<(typeof broadcastRoles)[number]>('DRIVER');
  const [templateDate, setTemplateDate] = useState('');
  const [customReason, setCustomReason] = useState('');
  const [broadcasting, setBroadcasting] = useState(false);

  const [saveAsTemplate, setSaveAsTemplate] = useState(false);
  const [savingTemplate, setSavingTemplate] = useState(false);

  const [productId, setProductId] = useState('');
  const [retailerId, setRetailerId] = useState('');
  const [proposedPrice, setProposedPrice] = useState('');
  const [preview, setPreview] = useState<RetailerOverridePreview | null>(null);
  const [previewLoading, setPreviewLoading] = useState(false);

  const loadTemplates = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const resp = await warehouseApi.getWarehouseBroadcastTemplates(scope);
      setTemplates(resp.templates ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load templates');
    } finally {
      setLoading(false);
    }
  }, [scope]);

  useEffect(() => {
    void loadTemplates();
  }, [loadTemplates]);

  useEffect(() => {
    const product = productId.trim();
    const price = Number(proposedPrice);
    if (!product || !Number.isFinite(price) || price <= 0) {
      setPreview(null);
      return;
    }
    const handle = window.setTimeout(async () => {
      setPreviewLoading(true);
      try {
        const result = await warehouseApi.previewWarehouseRetailerPriceOverride(
          {
            product_id: product,
            retailer_id: retailerId.trim() || undefined,
            proposed_price: price,
          },
          scope,
        );
        setPreview(result);
      } catch {
        setPreview(null);
      } finally {
        setPreviewLoading(false);
      }
    }, 400);
    return () => window.clearTimeout(handle);
  }, [productId, retailerId, proposedPrice, scope]);

  const onSelectTemplate = (template: BroadcastTemplate) => {
    const applied = applyTemplate(template, templateDate, customReason);
    setTitle(applied.title);
    setBody(applied.body);
    if (broadcastRoles.includes(applied.role as (typeof broadcastRoles)[number])) {
      setBroadcastRole(applied.role as (typeof broadcastRoles)[number]);
    }
  };

  const onBroadcast = async () => {
    const trimmedTitle = title.trim();
    const trimmedBody = body.trim();
    if (!trimmedTitle || !trimmedBody) {
      setError('Title and message are required');
      return;
    }
    setBroadcasting(true);
    setError(null);
    setMessage(null);
    try {
      const wh = warehouseId || 'warehouse';
      if (saveAsTemplate) {
        setSavingTemplate(true);
        await warehouseApi.createWarehouseBroadcastTemplate(
          {
            title: trimmedTitle,
            body: trimmedBody,
            default_role: broadcastRole,
            category: 'custom',
          },
          scope,
          warehouseBroadcastTemplateCreateKey(wh, trimmedTitle, trimmedBody),
        );
      }
      const resp = await warehouseApi.postWarehouseBroadcast(
        { title: trimmedTitle, body: trimmedBody, role: broadcastRole },
        scope,
        warehouseBroadcastKey(wh, broadcastRole, trimmedTitle, trimmedBody),
      );
      setMessage(`Broadcast sent from depot ${resp.warehouse_id}.`);
      setTitle('');
      setBody('');
      setSaveAsTemplate(false);
      await loadTemplates();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Broadcast failed');
    } finally {
      setBroadcasting(false);
      setSavingTemplate(false);
    }
  };

  const onDeleteTemplate = async (template: BroadcastTemplate) => {
    if (template.source !== 'custom' || !template.id) return;
    const wh = warehouseId || 'warehouse';
    try {
      await warehouseApi.deleteWarehouseBroadcastTemplate(
        template.id,
        scope,
        warehouseBroadcastTemplateDeleteKey(wh, template.id),
      );
      await loadTemplates();
      setMessage('Custom template removed.');
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Delete failed');
    }
  };

  return (
    <PageChrome
      icon="send"
      title="Depot operations"
      description="Depot-scoped broadcasts and read-only pricing impact preview."
    >
      {error ? <p className="text-sm text-red-600">{error}</p> : null}
      {message ? <p className="text-sm text-emerald-700">{message}</p> : null}

      <PageSection title="Broadcast templates" subtitle="Built-in depot starters plus your saved custom messages.">
        {loading ? (
          <p className="text-sm text-muted">Loading templates…</p>
        ) : (
          <div className="flex flex-wrap gap-2">
            {templates.map((template) => (
              <div key={template.id} className="flex items-center gap-1">
                <button
                  type="button"
                  className="rounded-full border px-3 py-1 text-sm hover:bg-surface-elevated"
                  onClick={() => onSelectTemplate(template)}
                >
                  {template.title}
                  {template.source === 'custom' ? ' · saved' : ''}
                </button>
                {template.source === 'custom' ? (
                  <button
                    type="button"
                    className="text-xs text-red-600"
                    onClick={() => void onDeleteTemplate(template)}
                    aria-label={`Delete ${template.title}`}
                  >
                    ×
                  </button>
                ) : null}
              </div>
            ))}
          </div>
        )}
      </PageSection>

      <PortalSection title="Send depot broadcast">
        <div className="grid gap-4 md:grid-cols-2">
          <PortalField label="Effective date (optional)">
            <PortalInput value={templateDate} onChange={(e) => setTemplateDate(e.target.value)} placeholder="2026-07-01" />
          </PortalField>
          <PortalField label="Custom reason (optional)">
            <PortalInput value={customReason} onChange={(e) => setCustomReason(e.target.value)} placeholder="Bay 2 closed" />
          </PortalField>
          <PortalField label="Title">
            <PortalInput value={title} onChange={(e) => setTitle(e.target.value)} />
          </PortalField>
          <PortalField label="Target role">
            <select
              className="w-full rounded-md border bg-background px-3 py-2 text-sm"
              value={broadcastRole}
              onChange={(e) => setBroadcastRole(e.target.value as (typeof broadcastRoles)[number])}
            >
              {broadcastRoles.map((role) => (
                <option key={role} value={role}>
                  {role}
                </option>
              ))}
            </select>
          </PortalField>
        </div>
        <PortalField label="Message">
          <textarea
            className="min-h-[120px] w-full rounded-md border bg-background px-3 py-2 text-sm"
            value={body}
            onChange={(e) => setBody(e.target.value)}
          />
        </PortalField>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={saveAsTemplate} onChange={(e) => setSaveAsTemplate(e.target.checked)} />
          Save as custom template for this depot
        </label>
        <button
          type="button"
          className="rounded-md bg-foreground px-4 py-2 text-sm text-background disabled:opacity-50"
          disabled={broadcasting || savingTemplate}
          onClick={() => void onBroadcast()}
        >
          {broadcasting || savingTemplate ? 'Sending…' : 'Send broadcast'}
        </button>
      </PortalSection>

      <PortalSection title="Pricing impact preview (read-only)">
        <p className="mb-3 text-sm text-muted">
          Preview how a proposed retailer price would compare to catalog list price for SKUs touching this depot. Does not create overrides.
        </p>
        <div className="grid gap-4 md:grid-cols-3">
          <PortalField label="Product / SKU ID">
            <PortalInput value={productId} onChange={(e) => setProductId(e.target.value)} />
          </PortalField>
          <PortalField label="Retailer ID (optional)">
            <PortalInput value={retailerId} onChange={(e) => setRetailerId(e.target.value)} />
          </PortalField>
          <PortalField label="Proposed price (minor units)">
            <PortalInput value={proposedPrice} onChange={(e) => setProposedPrice(e.target.value)} inputMode="numeric" />
          </PortalField>
        </div>
        {previewLoading ? <p className="text-sm text-muted">Loading preview…</p> : null}
        {preview ? (
          <div className="mt-3 grid gap-2 rounded-md border p-4 text-sm md:grid-cols-2">
            <div>Retailers on SKU: {preview.retailers_on_sku_count}</div>
            <div>Active overrides: {preview.active_override_count}</div>
            <div>Catalog list price: {preview.catalog_list_price}</div>
            <div>Margin delta / unit: {preview.margin_delta_per_unit}</div>
            <div className="md:col-span-2 text-muted">{preview.margin_estimate_label}</div>
            {preview.read_only ? <div className="md:col-span-2 font-medium">Read-only — contact supplier to apply overrides.</div> : null}
          </div>
        ) : null}
      </PortalSection>
    </PageChrome>
  );
}
