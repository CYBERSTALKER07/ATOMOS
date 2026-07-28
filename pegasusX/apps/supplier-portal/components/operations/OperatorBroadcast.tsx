import React from 'react';
import Link from 'next/link';
import { PageSection } from '@/components/PageSection';
import { SUPPLIER_BROADCAST_TEMPLATES } from '@pegasusx/types';

export const broadcastRoles = ["ALL", "DRIVER", "RETAILER", "PAYLOAD"] as const;

interface OperatorBroadcastProps {
  title: string;
  body: string;
  broadcastRole: (typeof broadcastRoles)[number];
  templateDate: string;
  broadcasting: boolean;
  onTitleChange: (v: string) => void;
  onBodyChange: (v: string) => void;
  onBroadcastRoleChange: (v: (typeof broadcastRoles)[number]) => void;
  onTemplateDateChange: (v: string) => void;
  onBroadcast: () => void;
}

export function OperatorBroadcast({
  title,
  body,
  broadcastRole,
  templateDate,
  broadcasting,
  onTitleChange,
  onBodyChange,
  onBroadcastRoleChange,
  onTemplateDateChange,
  onBroadcast,
}: OperatorBroadcastProps) {
  return (
    <PageSection title="Operator broadcast" description="Fan out a message to supplier WS rooms by role.">
      <p className="md-typescale-body-small mb-3 text-[var(--color-md-outline)]">
        Signal ingest health and planning projections live on{" "}
        <Link href={"/settings/planning" as any} className="underline text-[var(--color-md-primary)]">
          Planning settings
        </Link>
        .
      </p>
      <div className="flex flex-wrap gap-2 mb-4">
        {SUPPLIER_BROADCAST_TEMPLATES.map((template) => (
          <button
            key={template.id}
            type="button"
            className="md-btn md-btn-tonal text-xs px-3 py-1.5"
            onClick={() => {
              onTitleChange(template.title);
              onBodyChange(
                template.body.replace(
                  "{date}",
                  templateDate.trim() || "the selected date",
                ),
              );
              onBroadcastRoleChange(
                broadcastRoles.includes(template.default_role as (typeof broadcastRoles)[number])
                  ? (template.default_role as (typeof broadcastRoles)[number])
                  : "ALL",
              );
            }}
          >
            {template.title}
          </button>
        ))}
      </div>
      <label className="block space-y-1 mb-3 max-w-xs">
        <span className="md-typescale-label-medium" style={{ color: "var(--desk-text-secondary)" }}>
          Template date (optional)
        </span>
        <input
          type="date"
          className="md-input-outlined w-full"
          value={templateDate}
          onChange={(e) => onTemplateDateChange(e.target.value)}
        />
      </label>
      <div className="space-y-3">
        <label className="block space-y-1">
          <span className="md-typescale-label-medium" style={{ color: "var(--desk-text-secondary)" }}>
            Title
          </span>
          <input
            className="md-input-outlined w-full"
            placeholder="Title"
            value={title}
            onChange={(e) => onTitleChange(e.target.value)}
          />
        </label>
        <label className="block space-y-1">
          <span className="md-typescale-label-medium" style={{ color: "var(--desk-text-secondary)" }}>
            Message
          </span>
          <textarea
            className="md-input-outlined w-full min-h-24"
            placeholder="Message body"
            value={body}
            onChange={(e) => onBodyChange(e.target.value)}
          />
        </label>
        <label className="block space-y-1">
          <span className="md-typescale-label-medium" style={{ color: "var(--desk-text-secondary)" }}>
            Target role
          </span>
          <select
            className="md-input-outlined w-full"
            value={broadcastRole}
            onChange={(e) => onBroadcastRoleChange(e.target.value as (typeof broadcastRoles)[number])}
          >
            {broadcastRoles.map((role) => (
              <option key={role} value={role}>
                {role}
              </option>
            ))}
          </select>
        </label>
        <button
          type="button"
          className="md-btn md-btn-filled"
          onClick={onBroadcast}
          disabled={broadcasting || !title.trim() || !body.trim()}
        >
          {broadcasting ? "Sending…" : "Send broadcast"}
        </button>
      </div>
    </PageSection>
  );
}
