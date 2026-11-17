'use client';

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import {
  warehouseBroadcastKey,
  warehouseBroadcastTemplateCreateKey,
  warehouseBroadcastTemplateDeleteKey,
} from '@pegasusx/api-core';
import type { BroadcastTemplate, RetailerOverridePreview } from '@pegasusx/types';
import { PageChrome } from '@/components/PageChrome';
import { warehouseApi } from '@/lib/warehouse-api';
import { warehouseHomeNodeId, warehouseScopeQuery } from '@/lib/warehouse-scope';
import { OperationsBroadcastForm } from '@/components/operations/OperationsBroadcastForm';
import { OperationsPricingPreview } from '@/components/operations/OperationsPricingPreview';

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
  const t = usePortalT();
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
      title={t("warehouse_portal.operations.text.depot_operations")}
      description={t("warehouse_portal.residual.text.depot_scoped_broadcasts_and_read_only_pricing_impact_preview")}
    >
      {error ? <p className="text-sm text-red-600">{error}</p> : null}
      {message ? <p className="text-sm text-emerald-700">{message}</p> : null}

      <OperationsBroadcastForm
        templates={templates}
        loading={loading}
        title={title}
        setTitle={setTitle}
        body={body}
        setBody={setBody}
        broadcastRole={broadcastRole}
        setBroadcastRole={setBroadcastRole}
        templateDate={templateDate}
        setTemplateDate={setTemplateDate}
        customReason={customReason}
        setCustomReason={setCustomReason}
        saveAsTemplate={saveAsTemplate}
        setSaveAsTemplate={setSaveAsTemplate}
        broadcasting={broadcasting}
        savingTemplate={savingTemplate}
        onSelectTemplate={onSelectTemplate}
        onDeleteTemplate={onDeleteTemplate}
        onBroadcast={onBroadcast}
      />

      <OperationsPricingPreview
        productId={productId}
        setProductId={setProductId}
        retailerId={retailerId}
        setRetailerId={setRetailerId}
        proposedPrice={proposedPrice}
        setProposedPrice={setProposedPrice}
        previewLoading={previewLoading}
        preview={preview}
      />
    </PageChrome>
  );
}
