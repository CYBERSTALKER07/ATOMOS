"use client";

import { useMemo, useState } from "react";

const COPY = {
  en: {
    title: "Enable trade credit — irreversible in-app",
    body: "Once enabled, credit cannot be turned off from this portal. Contact Pegaus support to disable. Temporary holds remain available for collections.",
    checkbox: "I understand this cannot be reversed in-app",
    typeLabel: "Type ENABLE to confirm",
    confirm: "Enable credit",
    cancel: "Cancel",
  },
  ru: {
    title: "Включить торговый кредит — необратимо в приложении",
    body: "После включения кредит нельзя отключить в портале. Для отключения обратитесь в поддержку Pegaus. Временные блокировки для взыскания остаются доступны.",
    checkbox: "Я понимаю, что это нельзя отменить в приложении",
    typeLabel: "Введите ENABLE для подтверждения",
    confirm: "Включить кредит",
    cancel: "Отмена",
  },
  uz: {
    title: "Savdo kreditini yoqish — ilovada qaytarib bo‘lmaydi",
    body: "Yoqilgandan keyin kreditni portalda o‘chirib bo‘lmaydi. O‘chirish uchun Pegaus qo‘llab-quvvatlashiga murojaat qiling. Undirish uchun vaqtinchalik bloklashlar mavjud.",
    checkbox: "Ilovada qaytarib bo‘lmasligini tushunaman",
    typeLabel: "Tasdiqlash uchun ENABLE yozing",
    confirm: "Kreditni yoqish",
    cancel: "Bekor qilish",
  },
} as const;

type Locale = keyof typeof COPY;

export function CreditEnableModal(props: {
  open: boolean;
  locale?: Locale;
  confirmToken?: string;
  onCancel: () => void;
  onConfirm: (ackAt: string) => void;
  busy?: boolean;
}) {
  const { open, onCancel, onConfirm, busy } = props;
  const locale = props.locale ?? "en";
  const token = (props.confirmToken || "ENABLE").toUpperCase();
  const t = COPY[locale] ?? COPY.en;
  const [checked, setChecked] = useState(false);
  const [typed, setTyped] = useState("");
  const ready = useMemo(
    () => checked && typed.trim().toUpperCase() === token,
    [checked, typed, token],
  );

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div
        role="dialog"
        aria-modal="true"
        className="w-full max-w-md rounded-xl border border-[var(--border)] bg-[var(--surface)] p-5 shadow-lg"
      >
        <h2 className="text-lg font-semibold text-[var(--foreground)]">{t.title}</h2>
        <p className="mt-2 text-sm text-[var(--muted)]">{t.body}</p>
        <label className="mt-4 flex items-start gap-2 text-sm">
          <input
            type="checkbox"
            checked={checked}
            onChange={(e) => setChecked(e.target.checked)}
          />
          <span>{t.checkbox}</span>
        </label>
        <label className="mt-3 block text-sm">
          <span className="text-[var(--muted)]">{t.typeLabel}</span>
          <input
            className="mt-1 w-full rounded-lg border border-[var(--border)] px-3 py-2"
            value={typed}
            onChange={(e) => setTyped(e.target.value)}
            autoComplete="off"
          />
        </label>
        <div className="mt-5 flex justify-end gap-2">
          <button
            type="button"
            className="rounded-lg px-3 py-1.5 text-sm border border-[var(--border)]"
            onClick={onCancel}
            disabled={busy}
          >
            {t.cancel}
          </button>
          <button
            type="button"
            className="rounded-lg px-3 py-1.5 text-sm bg-[var(--color-md-primary)] text-white disabled:opacity-40"
            disabled={!ready || busy}
            onClick={() => onConfirm(new Date().toISOString())}
          >
            {t.confirm}
          </button>
        </div>
      </div>
    </div>
  );
}
