"use client";

import { usePortalT } from "@/lib/i18n";
import { type ReactNode } from 'react';
import type { BroadcastTemplate } from '@pegasusx/types';
import { PageSection } from '@/components/PageSection';
import { PortalField, PortalInput, PortalSection } from '@/components/portal';

const broadcastRoles = ['DRIVER', 'RETAILER', 'ALL'] as const;

interface OperationsBroadcastFormProps {
  templates: BroadcastTemplate[];
  loading: boolean;
  title: string;
  setTitle: (val: string) => void;
  body: string;
  setBody: (val: string) => void;
  broadcastRole: (typeof broadcastRoles)[number];
  setBroadcastRole: (val: (typeof broadcastRoles)[number]) => void;
  templateDate: string;
  setTemplateDate: (val: string) => void;
  customReason: string;
  setCustomReason: (val: string) => void;
  saveAsTemplate: boolean;
  setSaveAsTemplate: (val: boolean) => void;
  broadcasting: boolean;
  savingTemplate: boolean;
  onSelectTemplate: (t: BroadcastTemplate) => void;
  onDeleteTemplate: (t: BroadcastTemplate) => void;
  onBroadcast: () => void;
}

export function OperationsBroadcastForm({
  templates,
  loading,
  title,
  setTitle,
  body,
  setBody,
  broadcastRole,
  setBroadcastRole,
  templateDate,
  setTemplateDate,
  customReason,
  setCustomReason,
  saveAsTemplate,
  setSaveAsTemplate,
  broadcasting,
  savingTemplate,
  onSelectTemplate,
  onDeleteTemplate,
  onBroadcast,
}: OperationsBroadcastFormProps) {
  const t = usePortalT();
  return (
    <>
      <PageSection title={t("warehouse_portal.operations.operations_broadcast_form.text.broadcast_templates")} description={t("warehouse_portal.residual.text.built_in_depot_starters_plus_your_saved_custom_messages")}>
        {loading ? (
          <p className="text-sm text-muted">{t("warehouse_portal.operations.operations_broadcast_form.text.loading_templates")}</p>
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
                    onClick={() => onDeleteTemplate(template)}
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

      <PortalSection title={t("warehouse_portal.operations.operations_broadcast_form.text.send_depot_broadcast")}>
        <div className="grid gap-4 md:grid-cols-2">
          <PortalField id="templateDate" label={t("warehouse_portal.residual.text.effective_date_optional")}>
            <PortalInput value={templateDate} onChange={(e) => setTemplateDate(e.target.value)} placeholder="2026-07-01" />
          </PortalField>
          <PortalField id="customReason" label={t("warehouse_portal.residual.text.custom_reason_optional")}>
            <PortalInput value={customReason} onChange={(e) => setCustomReason(e.target.value)} placeholder={t("warehouse_portal.operations.operations_broadcast_form.text.bay_2_closed")} />
          </PortalField>
          <PortalField id="broadcastTitle" label={t("supplier_portal.admin.control_center.field.title")}>
            <PortalInput value={title} onChange={(e) => setTitle(e.target.value)} />
          </PortalField>
          <PortalField id="broadcastRole" label={t("warehouse_portal.residual.text.target_role")}>
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
        <PortalField id="broadcastBody" label={t("warehouse_portal.residual.text.message")}>
          <textarea
            id="broadcastBody"
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
          onClick={() => onBroadcast()}
        >
          {broadcasting || savingTemplate ? 'Sending…' : 'Send broadcast'}
        </button>
      </PortalSection>
    </>
  );
}
