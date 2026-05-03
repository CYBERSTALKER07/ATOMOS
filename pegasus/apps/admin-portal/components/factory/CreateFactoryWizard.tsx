'use client';

import React, { useState, useCallback, useRef } from 'react';
import MapGL, { Marker, NavigationControl, type MapRef } from 'react-map-gl/maplibre';
import type { MapLayerMouseEvent } from 'react-map-gl/maplibre';
import 'maplibre-gl/dist/maplibre-gl.css';
import { apiFetch } from '@/lib/auth';
import { buildSupplierFactoryCreateIdempotencyKey } from '@/app/supplier/_shared/idempotency';
import Icon from '@/components/Icon';
import WarehouseAssignmentPanel from './WarehouseAssignmentPanel';

const MAP_STYLE = 'https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json';
const TASHKENT = { latitude: 41.2995, longitude: 69.2401 };
const STEPS = ['Location', 'Details', 'Warehouses'] as const;

interface GeocodedAddress {
  street: string;
  house_number: string;
  city: string;
  country: string;
  display_name: string;
}

interface CreateFactoryWizardProps {
  onCreated: () => void;
  onCancel: () => void;
}

export default function CreateFactoryWizard({ onCreated, onCancel }: CreateFactoryWizardProps) {
  const [step, setStep] = useState(0);

  // Step 1: Location
  const [marker, setMarker] = useState({ lat: TASHKENT.latitude, lng: TASHKENT.longitude });
  const [markerPlaced, setMarkerPlaced] = useState(false);
  const [geocoding, setGeocoding] = useState(false);
  const [geocoded, setGeocoded] = useState<GeocodedAddress | null>(null);
  const [addressOverride, setAddressOverride] = useState('');
  const [locating, setLocating] = useState(false);
  const mapRef = useRef<MapRef>(null);

  // Step 2: Details
  const [name, setName] = useState('');
  const [leadTimeDays, setLeadTimeDays] = useState('');
  const [capacityVU, setCapacityVU] = useState('');
  const [productTypes, setProductTypes] = useState('');
  const [regionCode, setRegionCode] = useState('');

  // Step 3: Warehouse assignment
  const [selectedWarehouses, setSelectedWarehouses] = useState<string[]>([]);

  // Submit
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  // ── Location handlers ───────────────────────────────────────────────────

  const reverseGeocode = useCallback(async (lat: number, lng: number) => {
    setGeocoding(true);
    try {
      const res = await apiFetch(`/v1/supplier/geocode/reverse?lat=${lat}&lng=${lng}`);
      if (!res.ok) throw new Error('Geocode failed');
      const data: GeocodedAddress = await res.json();
      setGeocoded(data);
      setAddressOverride('');
    } catch {
      setGeocoded(null);
    } finally {
      setGeocoding(false);
    }
  }, []);

  const handleMapClick = useCallback((e: MapLayerMouseEvent) => {
    const { lng, lat } = e.lngLat;
    setMarker({ lat, lng });
    setMarkerPlaced(true);
    reverseGeocode(lat, lng);
  }, [reverseGeocode]);

  const handleShareLocation = useCallback(() => {
    if (!navigator.geolocation) return;
    setLocating(true);
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        const { latitude: lat, longitude: lng } = pos.coords;
        setMarker({ lat, lng });
        setMarkerPlaced(true);
        reverseGeocode(lat, lng);
        mapRef.current?.flyTo({ center: [lng, lat], zoom: 16, duration: 800 });
        setLocating(false);
      },
      () => setLocating(false),
      { enableHighAccuracy: true, timeout: 10000 },
    );
  }, [reverseGeocode]);

  const resolvedAddress = addressOverride || geocoded?.display_name || '';
  const resolvedStreet = geocoded ? `${geocoded.street} ${geocoded.house_number}`.trim() : '';

  // ── Navigation ──────────────────────────────────────────────────────────

  const canProceedStep0 = markerPlaced;
  const canProceedStep1 = name.trim().length > 0;

  const handleNext = () => {
    if (step < 2) setStep(step + 1);
  };
  const handleBack = () => {
    if (step > 0) setStep(step - 1);
  };

  // ── Submit ──────────────────────────────────────────────────────────────

  const handleSubmit = async () => {
    setSaving(true);
    setError('');
    try {
      const body = {
        name: name.trim(),
        address: resolvedAddress,
        lat: marker.lat,
        lng: marker.lng,
        lead_time_days: leadTimeDays ? parseInt(leadTimeDays, 10) : 0,
        production_capacity_vu: capacityVU ? parseInt(capacityVU, 10) : 0,
        product_types: productTypes.split(',').map(s => s.trim()).filter(Boolean),
        region_code: regionCode.trim(),
        warehouse_ids: selectedWarehouses,
      };
      const res = await apiFetch('/v1/supplier/factories', {
        method: 'POST',
        headers: {
          'Idempotency-Key': buildSupplierFactoryCreateIdempotencyKey(body),
        },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || 'Failed to create factory');
      }
      onCreated();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Create failed');
    } finally {
      setSaving(false);
    }
  };

  // ── Render ──────────────────────────────────────────────────────────────

  return (
    <div className="flex flex-col h-full">
      {/* Stepper */}
      <div className="flex items-center gap-1 px-6 py-4" style={{ borderBottom: '1px solid var(--border)' }}>
        {STEPS.map((label, i) => (
          <React.Fragment key={label}>
            <button
              type="button"
              onClick={() => {
                if (i < step) setStep(i);
              }}
              className="flex items-center gap-2 px-3 py-1.5 rounded-lg transition-all"
              style={{
                background: i === step ? 'color-mix(in srgb, var(--accent) 15%, transparent)' : 'transparent',
                color: i === step ? 'var(--accent)' : i < step ? 'var(--foreground)' : 'var(--muted)',
                cursor: i < step ? 'pointer' : 'default',
              }}
            >
              <span
                className="w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold"
                style={{
                  background: i <= step ? 'var(--accent)' : 'var(--surface)',
                  color: i <= step ? 'var(--accent-foreground)' : 'var(--muted)',
                }}
              >
                {i < step ? (
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
                  </svg>
                ) : (
                  i + 1
                )}
              </span>
              <span className="md-typescale-label-medium hidden sm:inline">{label}</span>
            </button>
            {i < STEPS.length - 1 && (
              <div className="flex-1 h-px mx-1" style={{ background: i < step ? 'var(--accent)' : 'var(--border)' }} />
            )}
          </React.Fragment>
        ))}
      </div>

      {/* Step Content */}
      <div className="flex-1 overflow-y-auto p-6">
        {step === 0 && (
          <div className="space-y-4">
            <div>
              <h3 className="md-typescale-title-medium mb-1">Factory Location</h3>
              <p className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>
                Click on the map or use &quot;My Location&quot; to place your factory pin. The address will be auto-detected.
              </p>
            </div>

            {/* Map */}
            <div className="relative rounded-2xl overflow-hidden" style={{ height: 280, border: '2px solid var(--border)' }}>
              <MapGL
                ref={mapRef}
                initialViewState={{
                  latitude: marker.lat,
                  longitude: marker.lng,
                  zoom: 12,
                }}
                style={{ width: '100%', height: '100%' }}
                mapStyle={MAP_STYLE}
                onClick={handleMapClick}
              >
                <NavigationControl position="top-right" showCompass={false} />
                {markerPlaced && (
                  <Marker latitude={marker.lat} longitude={marker.lng} anchor="bottom">
                    <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
                      <svg width="36" height="36" viewBox="0 0 24 24" fill="var(--accent)">
                        <path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7zm0 9.5c-1.38 0-2.5-1.12-2.5-2.5s1.12-2.5 2.5-2.5 2.5 1.12 2.5 2.5-1.12 2.5-2.5 2.5z" />
                      </svg>
                    </div>
                  </Marker>
                )}
              </MapGL>

              <button
                type="button"
                onClick={handleShareLocation}
                disabled={locating}
                className="absolute bottom-3 right-3 flex items-center gap-2 px-3 py-2 rounded-full text-xs font-semibold transition-all"
                style={{
                  background: 'var(--background)',
                  color: 'var(--foreground)',
                  border: '1px solid var(--border)',
                  boxShadow: '0 2px 8px rgba(0,0,0,0.3)',
                }}
              >
                <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12 8c-2.21 0-4 1.79-4 4s1.79 4 4 4 4-1.79 4-4-1.79-4-4-4zm8.94 3c-.46-4.17-3.77-7.48-7.94-7.94V1h-2v2.06C6.83 3.52 3.52 6.83 3.06 11H1v2h2.06c.46 4.17 3.77 7.48 7.94 7.94V23h2v-2.06c4.17-.46 7.48-3.77 7.94-7.94H23v-2h-2.06zM12 19c-3.87 0-7-3.13-7-7s3.13-7 7-7 7 3.13 7 7-3.13 7-7 7z" />
                </svg>
                {locating ? 'Locating...' : 'My Location'}
              </button>
            </div>

            {/* Geocoded info */}
            {markerPlaced && (
              <div className="space-y-3">
                <div className="flex items-center gap-2 px-3 py-2 rounded-lg" style={{ background: 'var(--surface)' }}>
                  <Icon name="pin" size={14} style={{ color: 'var(--accent)' }} />
                  <span className="md-typescale-body-small tabular-nums" style={{ color: 'var(--muted)' }}>
                    {marker.lat.toFixed(6)}, {marker.lng.toFixed(6)}
                  </span>
                </div>

                {geocoding ? (
                  <div className="flex items-center gap-2 px-3 py-2">
                    <div className="w-4 h-4 rounded-full border-2 border-t-transparent animate-spin" style={{ borderColor: 'var(--accent)', borderTopColor: 'transparent' }} />
                    <span className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>Detecting address...</span>
                  </div>
                ) : geocoded ? (
                  <div className="rounded-xl p-4 space-y-2" style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}>
                    <div className="grid grid-cols-2 gap-3">
                      <div>
                        <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Street</p>
                        <p className="md-typescale-body-medium">{resolvedStreet || '—'}</p>
                      </div>
                      <div>
                        <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>City</p>
                        <p className="md-typescale-body-medium">{geocoded.city || '—'}</p>
                      </div>
                      <div className="col-span-2">
                        <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Country</p>
                        <p className="md-typescale-body-medium">{geocoded.country || '—'}</p>
                      </div>
                    </div>
                  </div>
                ) : null}

                {/* Address override */}
                <div>
                  <label className="md-typescale-label-small block mb-1" style={{ color: 'var(--muted)' }}>
                    Address (auto-detected, or type to override)
                  </label>
                  <input
                    type="text"
                    value={addressOverride || geocoded?.display_name || ''}
                    onChange={e => setAddressOverride(e.target.value)}
                    placeholder="Full address"
                    className="w-full px-4 py-3 rounded-xl md-typescale-body-medium outline-none transition-all"
                    style={{
                      background: 'var(--surface)',
                      color: 'var(--foreground)',
                      border: '2px solid var(--border)',
                    }}
                    onFocus={e => (e.target.style.borderColor = 'var(--accent)')}
                    onBlur={e => (e.target.style.borderColor = 'var(--border)')}
                  />
                </div>
              </div>
            )}
          </div>
        )}

        {step === 1 && (
          <div className="space-y-4">
            <div>
              <h3 className="md-typescale-title-medium mb-1">Factory Details</h3>
              <p className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>
                Configure production parameters for optimization and planning.
              </p>
            </div>

            <FieldInput label="Factory Name" value={name} onChange={setName} required placeholder="e.g. Central Production Unit" />
            <div className="grid grid-cols-2 gap-4">
              <FieldInput label="Lead Time (days)" value={leadTimeDays} onChange={setLeadTimeDays} type="number" placeholder="3" />
              <FieldInput label="Capacity (volume units)" value={capacityVU} onChange={setCapacityVU} type="number" placeholder="10000" />
            </div>
            <FieldInput label="Product Types" value={productTypes} onChange={setProductTypes} placeholder="beverages, dairy, snacks" hint="Comma-separated" />
            <FieldInput label="Region Code" value={regionCode} onChange={setRegionCode} placeholder="UZ-TAS" />
          </div>
        )}

        {step === 2 && (
          <div className="space-y-4">
            <div>
              <h3 className="md-typescale-title-medium mb-1">Warehouse Assignment</h3>
              <p className="md-typescale-body-small" style={{ color: 'var(--muted)' }}>
                Select which warehouses this factory will supply. Use &quot;Smart Assign&quot; for optimal routing.
              </p>
            </div>

            <WarehouseAssignmentPanel
              factoryLat={marker.lat}
              factoryLng={marker.lng}
              selected={selectedWarehouses}
              onChange={setSelectedWarehouses}
            />
          </div>
        )}
      </div>

      {/* Error */}
      {error && (
        <div className="mx-6 mb-2 flex items-center gap-2 p-3 rounded-lg" style={{ background: 'color-mix(in srgb, var(--danger) 10%, transparent)', color: 'var(--danger)' }}>
          <Icon name="error" size={14} />
          <span className="md-typescale-body-small">{error}</span>
        </div>
      )}

      {/* Footer navigation */}
      <div className="flex items-center justify-between gap-3 px-6 py-4" style={{ borderTop: '1px solid var(--border)' }}>
        <button
          type="button"
          onClick={step === 0 ? onCancel : handleBack}
          className="md-btn md-btn-outlined md-typescale-label-large px-5 py-2.5"
        >
          {step === 0 ? 'Cancel' : 'Back'}
        </button>

        <div className="flex items-center gap-2">
          {step === 2 && (
            <span className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
              {selectedWarehouses.length} warehouse{selectedWarehouses.length !== 1 ? 's' : ''} selected
            </span>
          )}
          {step < 2 ? (
            <button
              type="button"
              onClick={handleNext}
              disabled={step === 0 ? !canProceedStep0 : step === 1 ? !canProceedStep1 : false}
              className="md-btn md-btn-filled md-typescale-label-large px-5 py-2.5 flex items-center gap-2"
              style={{ opacity: (step === 0 && !canProceedStep0) || (step === 1 && !canProceedStep1) ? 0.5 : 1 }}
            >
              Next
              <Icon name="right" size={16} />
            </button>
          ) : (
            <button
              type="button"
              onClick={handleSubmit}
              disabled={saving}
              className="md-btn md-btn-filled md-typescale-label-large px-5 py-2.5 flex items-center gap-2"
            >
              {saving ? (
                <>
                  <div className="w-4 h-4 rounded-full border-2 border-t-transparent animate-spin" style={{ borderColor: 'var(--accent-foreground)', borderTopColor: 'transparent' }} />
                  Creating...
                </>
              ) : (
                <>
                  <Icon name="factory" size={16} />
                  Create Factory
                </>
              )}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

// ── Field Input helper ──────────────────────────────────────────────────────

function FieldInput({
  label,
  value,
  onChange,
  type = 'text',
  placeholder,
  hint,
  required,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  type?: string;
  placeholder?: string;
  hint?: string;
  required?: boolean;
}) {
  return (
    <div>
      <label className="md-typescale-label-small block mb-1.5" style={{ color: 'var(--muted)' }}>
        {label} {required && <span style={{ color: 'var(--danger)' }}>*</span>}
      </label>
      <input
        type={type}
        value={value}
        onChange={e => onChange(e.target.value)}
        placeholder={placeholder}
        required={required}
        step={type === 'number' ? 'any' : undefined}
        className="w-full px-4 py-3 rounded-xl md-typescale-body-medium outline-none transition-all"
        style={{
          background: 'var(--surface)',
          color: 'var(--foreground)',
          border: '2px solid var(--border)',
        }}
        onFocus={e => (e.target.style.borderColor = 'var(--accent)')}
        onBlur={e => (e.target.style.borderColor = 'var(--border)')}
      />
      {hint && <p className="md-typescale-label-small mt-1" style={{ color: 'var(--muted)' }}>{hint}</p>}
    </div>
  );
}
