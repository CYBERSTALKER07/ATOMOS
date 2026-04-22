'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { Button } from '@heroui/react';
import Icon from '../../../components/Icon';
import { apiFetch } from '../../../lib/auth';

interface SupplierProfile {
  supplier_id: string;
  name: string;
  phone: string;
  email: string;
  category: string;
  tax_id: string;
  contact_person: string;
  company_reg_number: string;
  billing_address: string;
  is_configured: boolean;
  operating_categories: string[];
  warehouse_location: string;
  warehouse_lat: number;
  warehouse_lng: number;
  bank_name: string;
  account_number: string;
  card_number: string;
  payment_gateway: string;
  manual_off_shift: boolean;
  operating_schedule: Record<string, { open: string; close: string }>;
  is_active: boolean;
}

function Field({ label, value, editing, onChange, type = 'text' }: {
  label: string; value: string; editing: boolean;
  onChange?: (v: string) => void; type?: string;
}) {
  return (
    <div className="w-full relative group">
      <label className="absolute -top-3 left-4 bg-[var(--background)] px-2 text-xs font-bold text-[var(--accent)] tracking-wide uppercase z-10">
        {label}
      </label>
      {editing ? (
        <input
          type={type}
          value={value}
          onChange={(e) => onChange?.(e.target.value)}
          className="w-full min-h-[56px] px-6 rounded-2xl bg-transparent border-2 border-[var(--border)] text-base text-[var(--foreground)] font-medium focus:border-[var(--accent)] focus:ring-0 transition-colors outline-none"
        />
      ) : (
        <div className="w-full min-h-[56px] px-6 rounded-2xl border-2 border-[var(--border)] flex items-center text-base text-[var(--foreground)] font-medium">
          {value || <span className="text-[var(--muted)] italic">Not set</span>}
        </div>
      )}
    </div>
  );
}

export default function SupplierProfilePage() {
  const [profile, setProfile] = useState<SupplierProfile | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');
  const [editing, setEditing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [draft, setDraft] = useState<Partial<SupplierProfile>>({});

  const fetchProfile = useCallback(async () => {
    try {
      const res = await apiFetch('/v1/supplier/profile');
      if (!res.ok) throw new Error(`${res.status}`);
      const data = await res.json();
      setProfile(data);
      setError('');
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to load profile');
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => { fetchProfile(); }, [fetchProfile]);

  const startEdit = () => {
    if (!profile) return;
    setDraft({
      name: profile.name,
      phone: profile.phone,
      email: profile.email,
      contact_person: profile.contact_person,
      tax_id: profile.tax_id,
      company_reg_number: profile.company_reg_number,
      billing_address: profile.billing_address,
      warehouse_location: profile.warehouse_location,
      bank_name: profile.bank_name,
      account_number: profile.account_number,
      card_number: profile.card_number,
      payment_gateway: profile.payment_gateway,
    });
    setEditing(true);
  };

  const cancelEdit = () => { setEditing(false); setDraft({}); };

  const saveProfile = async () => {
    if (!profile) return;
    setSaving(true);
    try {
      // compute diff — only send changed fields
      const payload: Record<string, unknown> = {};
      for (const [k, v] of Object.entries(draft)) {
        if (v !== (profile as unknown as Record<string, unknown>)[k]) {
          payload[k] = v;
        }
      }
      if (Object.keys(payload).length === 0) { setEditing(false); return; }

      const res = await apiFetch('/v1/supplier/profile', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      if (!res.ok) throw new Error(`${res.status}`);
      await fetchProfile();
      setEditing(false);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Save failed');
    } finally {
      setSaving(false);
    }
  };

  const d = (key: keyof SupplierProfile) => (editing ? (draft as Record<string, string>)[key] ?? '' : String(profile?.[key] ?? ''));
  const setD = (key: string) => (v: string) => setDraft(prev => ({ ...prev, [key]: v }));

  if (isLoading) {
    return (
      <div className="p-6 md:p-10 max-w-5xl mx-auto space-y-8">
        <div className="h-40 rounded-3xl skeleton" />
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {Array.from({ length: 6 }).map((_, i) => <div key={i} className="h-16 rounded-xl skeleton" />)}
        </div>
      </div>
    );
  }

  if (error && !profile) {
    return (
      <div className="p-10 text-center space-y-4">
        <Icon name="error" className="w-12 h-12 text-[var(--danger)] mx-auto" />
        <p className="md-typescale-title-medium text-[var(--danger)]">Failed to load profile</p>
        <Button variant="secondary" onPress={fetchProfile}>Retry</Button>
      </div>
    );
  }

  if (!profile) return null;

  return (
    <div className="p-6 md:p-10 max-w-5xl mx-auto space-y-10">

      {error && (
        <div className="px-4 py-3 rounded-xl bg-[var(--danger)] text-[var(--danger-foreground)] md-typescale-body-medium flex items-center gap-2">
          <Icon name="error" className="w-5 h-5" /> {error}
        </div>
      )}

      {/* HEADER */}
      <div className="flex flex-col md:flex-row items-center md:items-start gap-8 bg-[var(--surface)] p-8 rounded-3xl md-elevation-1">
        <div className="w-24 h-24 rounded-full bg-[var(--accent-soft)] flex items-center justify-center border-4 border-[var(--background)]">
          <Icon name="warehouse" className="w-12 h-12 text-[var(--accent-soft-foreground)]" />
        </div>

        <div className="flex-1 text-center md:text-left space-y-1">
          <h1 className="md-typescale-headline-medium text-[var(--foreground)]">{profile.name}</h1>
          <p className="md-typescale-title-small text-[var(--accent)] uppercase tracking-widest flex items-center justify-center md:justify-start gap-2">
            <Icon name="verified" className="w-4 h-4" />
            {profile.is_configured ? 'Configured Supplier' : 'Pending Configuration'}
          </p>
          <p className="md-typescale-body-medium text-[var(--muted)]">
            {profile.category} &middot; {profile.email} &middot; {profile.phone}
          </p>
        </div>

        <div className="flex gap-3 self-center md:self-start">
          {editing ? (
            <>
              <Button variant="outline" onPress={cancelEdit}>Cancel</Button>
              <Button variant="primary" onPress={saveProfile} isDisabled={saving}>
                {saving ? 'Saving…' : 'Save'}
              </Button>
            </>
          ) : (
            <Button variant="primary" onPress={startEdit}>
              <Icon name="edit" className="w-5 h-5 mr-2" /> Edit Profile
            </Button>
          )}
        </div>
      </div>

      {/* COMPANY DETAILS */}
      <section className="space-y-4">
        <h2 className="md-typescale-title-large text-[var(--foreground)] border-l-4 border-[var(--accent)] pl-3">Company Details</h2>
        <div className="md-card-elevated bg-[var(--background)] p-6 grid grid-cols-1 md:grid-cols-2 gap-6">
          <Field label="Company Name" value={d('name')} editing={editing} onChange={setD('name')} />
          <Field label="Contact Person" value={d('contact_person')} editing={editing} onChange={setD('contact_person')} />
          <Field label="Email" value={d('email')} editing={editing} onChange={setD('email')} type="email" />
          <Field label="Phone" value={d('phone')} editing={editing} onChange={setD('phone')} type="tel" />
          <Field label="Tax ID (STIR)" value={d('tax_id')} editing={editing} onChange={setD('tax_id')} />
          <Field label="Company Reg Number" value={d('company_reg_number')} editing={editing} onChange={setD('company_reg_number')} />
          <div className="md:col-span-2">
            <Field label="Billing Address" value={d('billing_address')} editing={editing} onChange={setD('billing_address')} />
          </div>
        </div>
      </section>

      {/* WAREHOUSE */}
      <section className="space-y-4">
        <h2 className="md-typescale-title-large text-[var(--foreground)] border-l-4 border-[var(--muted)] pl-3">Warehouse</h2>
        <div className="md-card-elevated bg-[var(--background)] p-6 grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="md:col-span-2">
            <Field label="Warehouse Address" value={d('warehouse_location')} editing={editing} onChange={setD('warehouse_location')} />
          </div>
          <Field label="Latitude" value={String(profile.warehouse_lat ?? '')} editing={false} />
          <Field label="Longitude" value={String(profile.warehouse_lng ?? '')} editing={false} />
        </div>
      </section>

      {/* BANKING */}
      <section className="space-y-4">
        <h2 className="md-typescale-title-large text-[var(--foreground)] border-l-4 border-[var(--muted)] pl-3">Banking & Payment</h2>
        <div className="md-card-elevated bg-[var(--background)] p-6 grid grid-cols-1 md:grid-cols-2 gap-6">
          <Field label="Bank Name" value={d('bank_name')} editing={editing} onChange={setD('bank_name')} />
          <Field label="Account Number" value={d('account_number')} editing={editing} onChange={setD('account_number')} />
          <Field label="Card Number" value={d('card_number')} editing={editing} onChange={setD('card_number')} />
          <Field label="Payment Gateway" value={d('payment_gateway')} editing={editing} onChange={setD('payment_gateway')} />
        </div>
      </section>

      {/* OPERATING CATEGORIES */}
      {profile.operating_categories?.length > 0 && (
        <section className="space-y-4">
          <h2 className="md-typescale-title-large text-[var(--foreground)] border-l-4 border-[var(--accent)] pl-3">Operating Categories</h2>
          <div className="flex flex-wrap gap-2">
            {profile.operating_categories.map(c => (
              <span key={c} className="md-chip px-4 py-1.5 md-typescale-label-large bg-[var(--accent-soft)] text-[var(--accent-soft-foreground)] rounded-full">
                {c}
              </span>
            ))}
          </div>
        </section>
      )}

      {/* STATUS */}
      <section className="space-y-4">
        <h2 className="md-typescale-title-large text-[var(--foreground)] border-l-4 border-[var(--border)] pl-3">Shift Status</h2>
        <div className="md-card-elevated bg-[var(--background)] p-6 flex items-center gap-4">
          <div className={`w-3 h-3 rounded-full ${profile.is_active ? 'bg-[var(--success)]' : 'bg-[var(--danger)]'}`} />
          <span className="md-typescale-body-large text-[var(--foreground)]">
            {profile.is_active ? 'Active — accepting orders' : 'Off shift — not accepting orders'}
          </span>
          {profile.manual_off_shift && (
            <span className="md-typescale-label-medium text-[var(--muted)]">(manual override)</span>
          )}
        </div>
      </section>
    </div>
  );
}