'use client';

import React, { useState } from 'react';
import Link from 'next/link';
import PageSection from './layout/PageSection';
import SectionHeader from './layout/SectionHeader';
import ChamferButton from './ChamferButton';
import { useLanguage } from '../context/LanguageContext';

type PlaybookId = 'smart-fit' | 'fleet-hotswap' | 'gate-seal' | 'zone-miss' | 'cod-settlement';

interface PlaybookData {
  id: PlaybookId;
  slug: string;
  category: string;
  title: string;
  tag: string;
  tagColor: string;
  summary: string;
  vehicle: {
    plate: string;
    model: string;
    driver: string;
    shift: string;
    route: string;
    capacityPercent: number;
    capacityDetail: string;
    sealId: string;
    dvirStatus: string;
    eta: string;
    speed: string;
  };
  events: {
    time: string;
    code: string;
    text: string;
    status: 'VERIFIED' | 'OPTIMIZED' | 'GUARDED' | 'ATOMIC';
  }[];
  stages: {
    name: string;
    state: 'done' | 'active' | 'pending';
    desc: string;
  }[];
  invariants: string[];
}

const PLAYBOOKS: Record<PlaybookId, PlaybookData> = {
  'smart-fit': {
    id: 'smart-fit',
    slug: 'truck-too-small',
    category: 'CAPACITY & DISPATCH',
    title: 'Smart Fit & Volumetric Overflow',
    tag: 'ALGORITHMIC PACKING',
    tagColor: '#3B82F6',
    summary: 'Volumetric Units (VU) load calculation with 5% Tetris buffer. Splits oversized morning batches across multi-truck plans without breaking retailer super-order integrity.',
    vehicle: {
      plate: '01 | 742 AFA',
      model: 'Volvo FH Electric (50 m³ Reefer)',
      driver: 'Rustam Kadyrov',
      shift: 'Shift #4029 · Class A',
      route: 'Tashkent DC-01 → Chorsu & Old City Hubs',
      capacityPercent: 94,
      capacityDetail: '47.2 / 50.0 VU (94.4% Packed)',
      sealId: 'SEAL-UZ-992014',
      dvirStatus: 'Passed (18/18 Items)',
      eta: '07:45 (+2 min buffer)',
      speed: '58 km/h',
    },
    events: [
      { time: '05:15:00', code: 'SMART_FIT_INIT', text: 'Consolidating 42 orders: 62.4 VU exceeds 50 VU limit', status: 'GUARDED' },
      { time: '05:15:02', code: 'GREEDY_FIRST_FIT', text: 'Packed 34 stores (47.2 VU) into Truck #01-742-AFA', status: 'OPTIMIZED' },
      { time: '05:15:03', code: 'OVERFLOW_SPLIT', text: '8 stores (15.2 VU) queued into Wave-2 Truck #01-381-BAA', status: 'ATOMIC' },
      { time: '05:18:20', code: 'FREEZE_LOCK', text: 'Manifest frozen. Prevents manual/AI race conditions', status: 'VERIFIED' },
    ],
    stages: [
      { name: '1. Ingestion', state: 'done', desc: 'Pre-order cut-off committed' },
      { name: '2. VU Calculation', state: 'done', desc: 'Tetris buffer & weight check' },
      { name: '3. Smart Fit Split', state: 'done', desc: 'Overflow route allocated' },
      { name: '4. Freeze-Lock', state: 'active', desc: 'Manifest locked to loading bay' },
      { name: '5. Staging Commit', state: 'pending', desc: 'Pallet loading barcode scan' },
    ],
    invariants: [
      'Same-cell order consolidation invariant',
      'Greedy first-fit capped at 95% volumetric ceiling',
      'Zero phantom manifest guarantee via Spanner RW',
    ],
  },
  'fleet-hotswap': {
    id: 'fleet-hotswap',
    slug: 'driver-reassignment',
    category: 'FLEET & DRIVER MGMT',
    title: 'Dynamic Fleet & Mid-Shift Hot-Swap',
    tag: 'DISPATCH RESILIENCE',
    tagColor: '#E2FD52',
    summary: 'Driver calls in sick after manifest loading: instant capacity-safe driver hot-swap and terminal credential re-pairing with zero duplicate side effects or ledger state corruption.',
    vehicle: {
      plate: '01 | 891 ZAA',
      model: 'Isuzu NPR 75L (24 m³ Box Truck)',
      driver: 'Timur Alimov (Replacing S. Nur)',
      shift: 'Shift #4031 · Standby Driver',
      route: 'Yunusabad Cluster → Mega Planet Corridor',
      capacityPercent: 88,
      capacityDetail: '21.1 / 24.0 VU (88% Packed)',
      sealId: 'SEAL-UZ-441829',
      dvirStatus: 'Re-certified (18/18 Items)',
      eta: '08:10 (+0 min deviation)',
      speed: '0 km/h (At Gate)',
    },
    events: [
      { time: '06:02:14', code: 'DRIVER_INCIDENT', text: 'Primary driver S. Nur logged sick during pre-departure', status: 'GUARDED' },
      { time: '06:02:45', code: 'CAPACITY_CHECK', text: 'Standby driver T. Alimov validated for Class C vehicle', status: 'VERIFIED' },
      { time: '06:03:10', code: 'TERMINAL_SWAP', text: 'Digital key revoked & issued to T. Alimov mobile app', status: 'ATOMIC' },
      { time: '06:04:00', code: 'REPLAY_AUDIT', text: 'Manifest re-signed; zero side-effect duplication verified', status: 'OPTIMIZED' },
    ],
    stages: [
      { name: '1. Incident Reported', state: 'done', desc: 'Driver unavailability logged' },
      { name: '2. Driver Match', state: 'done', desc: 'Standby driver assigned' },
      { name: '3. DVIR Transfer', state: 'done', desc: 'Digital key & checklist re-paired' },
      { name: '4. Replay Validation', state: 'active', desc: 'Outbox state replay sanitized' },
      { name: '5. Gate Release', state: 'pending', desc: 'Terminal barrier departure clearance' },
    ],
    invariants: [
      'Driver ↔ Truck ↔ Manifest triple-match validation',
      'Pre-trip DVIR 18-point mandatory digital verification',
      'Idempotent event replay with transactional deduplication',
    ],
  },
  'gate-seal': {
    id: 'gate-seal',
    slug: 'wrong-truck-sealed',
    category: 'YARD & GATE CONTROL',
    title: 'Payload Gate & RFID Seal Match',
    tag: 'YARD SECURITY',
    tagColor: '#FF7A1A',
    summary: 'Automated yard departure barrier opens only when physical RFID seal, driver credentials, and scanned manifest triple-match. Halts wrong-truck departures before gate exit.',
    vehicle: {
      plate: '10 | 442 XBA',
      model: 'MAN TGX 18.510 (85 m³ Intermodal)',
      driver: 'Jamshid Karimov',
      shift: 'Shift #4018 · Long Haul',
      route: 'Tashkent Central Yard → Samarkand Cross-Dock',
      capacityPercent: 98,
      capacityDetail: '83.3 / 85.0 VU (98% Full)',
      sealId: 'SEAL-UZ-109284 [VERIFIED]',
      dvirStatus: 'Gate Approved (Security Pass #71)',
      eta: '11:30 (Highway Segment)',
      speed: '72 km/h',
    },
    events: [
      { time: '06:40:12', code: 'GATE_ARRIVE', text: 'Vehicle reached Departure Lane 2 RFID scanner', status: 'VERIFIED' },
      { time: '06:40:15', code: 'SEAL_READ', text: 'Optical & RFID seal code SEAL-UZ-109284 detected', status: 'VERIFIED' },
      { time: '06:40:16', code: 'MANIFEST_TRIPLE', text: 'Manifest #8819 matches Truck 10-442-XBA & Driver #4018', status: 'ATOMIC' },
      { time: '06:40:18', code: 'BARRIER_CLEAR', text: 'Departure authorized; Spanner outbox state set IN_TRANSIT', status: 'OPTIMIZED' },
    ],
    stages: [
      { name: '1. Yard Approach', state: 'done', desc: 'Optical plate recognition' },
      { name: '2. RFID Scan', state: 'done', desc: 'Seal hash validated on-wire' },
      { name: '3. Triple Match', state: 'done', desc: 'Driver + Truck + Manifest verified' },
      { name: '4. Gate Authorization', state: 'active', desc: 'Hydraulic barrier raised' },
      { name: '5. State Broadcast', state: 'pending', desc: 'Kafka outbox notifies 6 role apps' },
    ],
    invariants: [
      'Per-truck optical & RFID tamper-evident seal match',
      'Zero-tolerance wrong-truck departure lock',
      'Hardware PLC barrier interlocked with Cloud Spanner',
    ],
  },
  'zone-miss': {
    id: 'zone-miss',
    slug: 'zone-miss-handling',
    category: 'TOPOLOGY & ROUTING',
    title: 'Zone Miss & Spatial Boundary Guard',
    tag: 'PRE-DISPATCH INTEGRITY',
    tagColor: '#10B981',
    summary: 'Real-time PostGIS polygon boundary validation at instant checkout. Rejects out-of-zone retailer orders with clear geographic error boundaries before warehouse pick-lists generate.',
    vehicle: {
      plate: '01 | 553 CAA',
      model: 'Ford Transit Custom (12 m³ Van)',
      driver: 'Otabek Rasulov',
      shift: 'Shift #4035 · Express Delivery',
      route: 'Sergeli Hub → Chilanzar Sector B',
      capacityPercent: 78,
      capacityDetail: '9.4 / 12.0 VU (78% Packed)',
      sealId: 'SEAL-UZ-551982',
      dvirStatus: 'Active on Route',
      eta: '09:20 (+1 min buffer)',
      speed: '44 km/h',
    },
    events: [
      { time: '07:12:04', code: 'CHECKOUT_VALIDATE', text: 'Store #2891 submitted order; coords [41.284, 69.219]', status: 'VERIFIED' },
      { time: '07:12:05', code: 'POSTGIS_POLYGON', text: 'Coordinates inside Chilanzar Sector B Delivery Fence', status: 'OPTIMIZED' },
      { time: '07:12:06', code: 'ZONE_ATTACHED', text: 'Assigned to Route #RT-09; carrier SLA 90 min guaranteed', status: 'ATOMIC' },
      { time: '07:14:22', code: 'ANOMALY_BLOCKED', text: 'Shop #9102 outside boundary: blocked with typed zone_miss', status: 'GUARDED' },
    ],
    stages: [
      { name: '1. Geo-Coordinate Pin', state: 'done', desc: 'Retailer lat/long captured' },
      { name: '2. Polygon Query', state: 'done', desc: 'PostGIS spatial containment test' },
      { name: '3. Route Cluster', state: 'done', desc: 'Matched to nearest warehouse cell' },
      { name: '4. Checkout Lock', state: 'active', desc: 'Stock reserved in valid zone' },
      { name: '5. Dispatch Handshake', state: 'pending', desc: 'Added to driver stop sequence' },
    ],
    invariants: [
      'Sub-10ms PostGIS polygon boundary validation at checkout',
      'Typed sentinel errors: zero unhandled 400/500 errors',
      'Automated supplier notification to extend delivery polygon',
    ],
  },
  'cod-settlement': {
    id: 'cod-settlement',
    slug: 'cash-at-door-cod',
    category: 'TREASURY & FINANCE',
    title: 'Cash-at-Door (COD) Dual Settlement',
    tag: 'TREASURY INTEGRITY',
    tagColor: '#06B6D4',
    summary: 'Mobile cashier module reconciles physical cash collections at delivery door with digital fiscal receipts, offline crypto signature tolerance, and instant double-entry ledger settlement.',
    vehicle: {
      plate: '01 | 381 BAA',
      model: 'Isuzu NPR 75L (24 m³ Box Truck)',
      driver: 'Bakhtiyor Tursunov',
      shift: 'Shift #4028 · Urban Delivery',
      route: 'Mirzo Ulugbek Corridor → 18 Retail Stores',
      capacityPercent: 91,
      capacityDetail: '21.8 / 24.0 VU (91% Full)',
      sealId: 'SEAL-UZ-772910',
      dvirStatus: 'Active Deliveries',
      eta: '10:15 (Stop 7 of 18)',
      speed: '28 km/h',
    },
    events: [
      { time: '08:45:10', code: 'DOOR_ARRIVE', text: 'Driver arrived at Oasis Supermarket; Stop #6', status: 'VERIFIED' },
      { time: '08:46:22', code: 'CASH_COLLECTED', text: 'Collected $1,840.00 cash; driver counted & confirmed', status: 'VERIFIED' },
      { time: '08:46:25', code: 'OTP_HANDSHAKE', text: 'Retailer entered 6-digit OTP signature on driver mobile app', status: 'ATOMIC' },
      { time: '08:46:28', code: 'LEDGER_SETTLE', text: 'Double-entry ledger credited Supplier; Cash-in-Transit logged', status: 'OPTIMIZED' },
    ],
    stages: [
      { name: '1. Delivery Verification', state: 'done', desc: 'Barcode scan of all cartons' },
      { name: '2. Physical Cash Audit', state: 'done', desc: 'Denomination count logged' },
      { name: '3. OTP Signature', state: 'done', desc: 'Retailer cryptographic authorization' },
      { name: '4. Treasury Invariant', state: 'active', desc: 'Double-entry ledger credit write' },
      { name: '5. Return-to-Vault', state: 'pending', desc: 'Evening driver cash bag deposit' },
    ],
    invariants: [
      'Dual-key confirmation: Driver count + Retailer OTP handshake',
      'Atomic double-entry ledger write (Spanner/PG balance parity)',
      'Offline-tolerant crypto-signed receipts with sync queue',
    ],
  },
};

const WAR_STORIES = [
  {
    slug: 'zone-miss-handling',
    code: 'ERR_ZONE_OUT_OF_BOUNDS',
    title: 'Zone Miss Handling',
    problem: 'Retailer outside delivery zone places order; truck gets routed to an unreachable destination.',
    solution: 'Sub-10ms PostGIS boundary check at checkout halts invalid orders with actionable error guidance.',
    accent: '#10B981',
  },
  {
    slug: 'concurrent-stock-reject',
    code: 'ERR_STOCK_ATOMIC_RACE',
    title: 'Concurrent Stock Race',
    problem: 'Two retailers order the last inventory unit in flash sales, creating phantom manifests.',
    solution: 'Spanner ReadWriteTransaction atomic reservations reject concurrent race with instant stock recovery.',
    accent: '#EF4444',
  },
  {
    slug: 'truck-too-small',
    code: 'WARN_VU_CAPACITY_EXCEEDED',
    title: 'Truck Capacity Overflow',
    problem: 'Morning order batch volume exceeds physical truck load space, causing yard dispatch chaos.',
    solution: 'Smart Fit Tetris packing algorithm auto-splits orders across multi-truck plans with freeze-lock.',
    accent: '#3B82F6',
  },
  {
    slug: 'wrong-truck-sealed',
    code: 'ERR_SEAL_MISMATCH_LOCK',
    title: 'Wrong Truck Sealed',
    problem: 'Seal placed on Truck A while Driver B is assigned causes broken fiscal trails and lost shipments.',
    solution: 'Payload departure barrier enforces driver ↔ truck ↔ seal optical triple-match before gate unlocks.',
    accent: '#FF7A1A',
  },
  {
    slug: 'driver-reassignment',
    code: 'ACT_DRIVER_HOTSWAP_REPLAY',
    title: 'Mid-Shift Driver Hot-Swap',
    problem: 'Driver falls ill mid-manifest after partial loading; naive reassign duplicates downstream events.',
    solution: 'Capacity-safe driver hot-swap re-issues digital keys and replays manifest with zero event drift.',
    accent: '#E2FD52',
  },
  {
    slug: 'cash-at-door-cod',
    code: 'ACT_COD_DUAL_KEY_SETTLE',
    title: 'Cash at Door (COD) Disputes',
    problem: 'Cash collected by driver at delivery door leads to reconciliation discrepancies and audit gaps.',
    solution: 'Driver count + Retailer OTP dual-key signature commits atomic double-entry ledger balance instantly.',
    accent: '#06B6D4',
  },
];

export default function OperationsSection() {
  const { t } = useLanguage();
  const [activePlaybookId, setActivePlaybookId] = useState<PlaybookId>('smart-fit');
  const activePlaybook = PLAYBOOKS[activePlaybookId];

  return (
    <PageSection id="operations" className="border-t border-white/10 bg-[#09090B] text-white">
      {/* ── Section Header ── */}
      <div className="mb-12 flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
        <SectionHeader
          eyebrow={t('ops_control_eyebrow', '04 · OPERATIONS CONTROL TOWER')}
          title={t('ops_control_title', 'Guarded Execution Across Every Handoff')}
          description={t(
            'ops_control_desc',
            'Built for the war stories. From Smart Fit volumetric packing and dynamic fleet hot-swaps to gate seal verification and COD settlement — every field anomaly has an automated, guarded recovery path.'
          )}
          className="mb-0"
        />
        <div className="flex flex-wrap items-center gap-3 shrink-0">
          <ChamferButton href="/operations" variant="fill">
            {t('ops_cta_playbooks', 'Explore All 10 Playbooks')}
          </ChamferButton>
          <ChamferButton href="/demo/warehouse" variant="ghost">
            {t('ops_cta_demo', 'Warehouse Ops Demo')}
          </ChamferButton>
        </div>
      </div>

      {/* ── High-Density Tactical KPI Metrics Strip ── */}
      <div className="mb-12 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
        <div className="relative overflow-hidden rounded-xl border border-white/10 bg-[#121216] p-5">
          <div className="flex items-center justify-between mb-3">
            <span className="text-[10px] font-mono uppercase tracking-[0.2em] text-white/50">
              {t('ops_stat_continuity_label', 'Floor Continuity Guarantee')}
            </span>
            <span className="w-2 h-2 rounded-full bg-[#E2FD52] animate-pulse" />
          </div>
          <div className="text-3xl font-bold font-mono tabular-nums text-white tracking-tight mb-1">
            {t('ops_stat_continuity_val', '99.98%')}
          </div>
          <p className="text-xs text-white/60 font-sans">
            {t('ops_stat_continuity_sub', 'Zero dispatch stalls during peak morning waves')}
          </p>
          <div className="mt-3 h-1 w-full bg-white/5 rounded-full overflow-hidden">
            <div className="h-full bg-[#E2FD52]" style={{ width: '99.98%' }} />
          </div>
        </div>

        <div className="relative overflow-hidden rounded-xl border border-white/10 bg-[#121216] p-5">
          <div className="flex items-center justify-between mb-3">
            <span className="text-[10px] font-mono uppercase tracking-[0.2em] text-white/50">
              {t('ops_stat_latency_label', 'Distributed Sync Latency')}
            </span>
            <span className="w-2 h-2 rounded-full bg-[#3B82F6] animate-pulse" />
          </div>
          <div className="text-3xl font-bold font-mono tabular-nums text-white tracking-tight mb-1">
            {t('ops_stat_latency_val', '< 120ms')}
          </div>
          <p className="text-xs text-white/60 font-sans">
            {t('ops_stat_latency_sub', 'Spanner outbox to native driver terminals')}
          </p>
          <div className="mt-3 h-1 w-full bg-white/5 rounded-full overflow-hidden">
            <div className="h-full bg-[#3B82F6]" style={{ width: '85%' }} />
          </div>
        </div>

        <div className="relative overflow-hidden rounded-xl border border-white/10 bg-[#121216] p-5">
          <div className="flex items-center justify-between mb-3">
            <span className="text-[10px] font-mono uppercase tracking-[0.2em] text-white/50">
              {t('ops_stat_manifest_label', 'Phantom Manifest Rate')}
            </span>
            <span className="w-2 h-2 rounded-full bg-[#10B981]" />
          </div>
          <div className="text-3xl font-bold font-mono tabular-nums text-emerald-400 tracking-tight mb-1">
            {t('ops_stat_manifest_val', '0.00%')}
          </div>
          <p className="text-xs text-white/60 font-sans">
            {t('ops_stat_manifest_sub', 'Dual-key RFID gate seals & atomic ledger integrity')}
          </p>
          <div className="mt-3 h-1 w-full bg-white/5 rounded-full overflow-hidden">
            <div className="h-full bg-[#10B981]" style={{ width: '100%' }} />
          </div>
        </div>

        <div className="relative overflow-hidden rounded-xl border border-white/10 bg-[#121216] p-5">
          <div className="flex items-center justify-between mb-3">
            <span className="text-[10px] font-mono uppercase tracking-[0.2em] text-white/50">
              {t('ops_stat_playbooks_label', 'Guarded War Story Playbooks')}
            </span>
            <span className="w-2 h-2 rounded-full bg-[#FF7A1A]" />
          </div>
          <div className="text-3xl font-bold font-mono tabular-nums text-white tracking-tight mb-1">
            {t('ops_stat_playbooks_val', '10 / 10')}
          </div>
          <p className="text-xs text-white/60 font-sans">
            {t('ops_stat_playbooks_sub', 'Deterministic recovery for every field anomaly')}
          </p>
          <div className="mt-3 h-1 w-full bg-white/5 rounded-full overflow-hidden">
            <div className="h-full bg-[#FF7A1A]" style={{ width: '100%' }} />
          </div>
        </div>
      </div>

      {/* ── Interactive 3-Column Operations Command Console ── */}
      <div className="mb-14 rounded-2xl border border-white/15 bg-[#0D0D11] overflow-hidden shadow-2xl">
        {/* Console Header Bar */}
        <div className="flex flex-wrap items-center justify-between border-b border-white/10 bg-[#121216] px-5 py-3.5">
          <div className="flex items-center space-x-3">
            <span className="flex h-2.5 w-2.5 rounded-full bg-emerald-500 animate-pulse" />
            <span className="font-mono text-xs font-bold uppercase tracking-widest text-white">
              PEGASUS CONTROL TOWER · LIVE TELEMETRY
            </span>
            <span className="hidden sm:inline-block font-mono text-[11px] text-white/40 border-l border-white/10 pl-3">
              NODE: UZ-TAS-DC01
            </span>
          </div>
          <div className="flex items-center space-x-4 font-mono text-[11px] text-white/60">
            <span className="hidden md:inline-flex items-center space-x-1.5">
              <span className="w-1.5 h-1.5 rounded-full bg-blue-400" />
              <span>SPANNER TX: ATOMIC</span>
            </span>
            <span className="hidden md:inline-flex items-center space-x-1.5">
              <span className="w-1.5 h-1.5 rounded-full bg-[#E2FD52]" />
              <span>OUTBOX: 0 LAG</span>
            </span>
            <span className="px-2 py-0.5 rounded bg-white/10 text-white font-semibold">
              FIELD READY
            </span>
          </div>
        </div>

        {/* 3 Columns */}
        <div className="grid grid-cols-1 lg:grid-cols-12 divide-y lg:divide-y-0 lg:divide-x divide-white/10">
          {/* Left Rail: Playbook Selectors (4 cols) */}
          <div className="lg:col-span-4 p-4 sm:p-5 flex flex-col space-y-2 bg-[#09090B]">
            <div className="text-[10px] font-mono uppercase tracking-[0.2em] text-white/40 mb-2 px-2">
              SELECT OPERATIONAL PLAYBOOK
            </div>
            {(Object.keys(PLAYBOOKS) as PlaybookId[]).map((id) => {
              const pb = PLAYBOOKS[id];
              const isSelected = activePlaybookId === id;
              return (
                <button
                  key={id}
                  type="button"
                  onClick={() => setActivePlaybookId(id)}
                  className={`group relative text-left p-3.5 rounded-xl border transition-all duration-200 ${
                    isSelected
                      ? 'border-white/30 bg-[#181820] text-white shadow-lg'
                      : 'border-white/5 bg-[#121216]/60 text-white/70 hover:border-white/15 hover:bg-[#15151B] hover:text-white'
                  }`}
                >
                  <div className="flex items-center justify-between mb-1.5">
                    <span className="font-mono text-[10px] tracking-wider uppercase px-2 py-0.5 rounded bg-white/5 text-white/80 border border-white/10">
                      {pb.category}
                    </span>
                    {isSelected && (
                      <span
                        className="w-2 h-2 rounded-full"
                        style={{ backgroundColor: pb.tagColor }}
                      />
                    )}
                  </div>
                  <div className="font-sans text-sm font-semibold text-white group-hover:text-white mb-1">
                    {pb.title}
                  </div>
                  <div className="font-sans text-xs text-white/50 line-clamp-2 leading-relaxed">
                    {pb.summary}
                  </div>
                </button>
              );
            })}
          </div>

          {/* Center Stage: Vehicle Tracking & Live Event Feed (5 cols) */}
          <div className="lg:col-span-5 p-5 sm:p-6 flex flex-col space-y-6 bg-[#0E0E12]">
            {/* Active Vehicle Card */}
            <div className="rounded-xl border border-white/15 bg-[#14141A] p-5">
              <div className="flex items-center justify-between border-b border-white/10 pb-3 mb-4">
                <div>
                  <div className="font-mono text-xs font-bold text-white tracking-wide">
                    {activePlaybook.vehicle.plate}
                  </div>
                  <div className="text-[11px] text-white/50 font-sans">
                    {activePlaybook.vehicle.model}
                  </div>
                </div>
                <div className="text-right">
                  <span className="inline-flex items-center space-x-1.5 px-2.5 py-1 rounded-full text-[10px] font-mono uppercase bg-emerald-500/10 text-emerald-400 border border-emerald-500/30">
                    <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
                    <span>ON ROUTE</span>
                  </span>
                </div>
              </div>

              {/* Route & Driver */}
              <div className="grid grid-cols-2 gap-3 mb-4 text-xs font-sans">
                <div>
                  <span className="text-[10px] font-mono uppercase text-white/40 block mb-0.5">
                    ASSIGNED DRIVER
                  </span>
                  <div className="font-medium text-white">{activePlaybook.vehicle.driver}</div>
                  <div className="text-[11px] text-white/50 font-mono">{activePlaybook.vehicle.shift}</div>
                </div>
                <div>
                  <span className="text-[10px] font-mono uppercase text-white/40 block mb-0.5">
                    CORRIDOR
                  </span>
                  <div className="font-medium text-white line-clamp-1">{activePlaybook.vehicle.route}</div>
                  <div className="text-[11px] text-[#E2FD52] font-mono">ETA {activePlaybook.vehicle.eta}</div>
                </div>
              </div>

              {/* Volumetric Capacity Bar */}
              <div className="mb-4">
                <div className="flex justify-between text-[11px] font-mono mb-1.5">
                  <span className="text-white/60">VOLUMETRIC UTILIZATION</span>
                  <span className="text-white font-bold">{activePlaybook.vehicle.capacityDetail}</span>
                </div>
                <div className="h-2 w-full bg-white/10 rounded-full overflow-hidden">
                  <div
                    className="h-full transition-all duration-500 rounded-full"
                    style={{
                      width: `${activePlaybook.vehicle.capacityPercent}%`,
                      backgroundColor:
                        activePlaybook.vehicle.capacityPercent > 90 ? '#3B82F6' : '#10B981',
                    }}
                  />
                </div>
              </div>

              {/* Badges: Seal & DVIR */}
              <div className="flex flex-wrap gap-2 pt-2 border-t border-white/5 font-mono text-[11px]">
                <div className="px-2 py-1 rounded bg-white/5 border border-white/10 text-white/80">
                  SEAL: <span className="text-blue-300 font-semibold">{activePlaybook.vehicle.sealId}</span>
                </div>
                <div className="px-2 py-1 rounded bg-white/5 border border-white/10 text-white/80">
                  DVIR: <span className="text-emerald-400 font-semibold">{activePlaybook.vehicle.dvirStatus}</span>
                </div>
              </div>
            </div>

            {/* Live Operational Audit Feed */}
            <div className="flex-1">
              <div className="flex items-center justify-between text-[10px] font-mono uppercase tracking-[0.2em] text-white/40 mb-3">
                <span>EVENT AUDIT TRAIL</span>
                <span className="text-emerald-400">● LIVE BUFFER</span>
              </div>
              <div className="space-y-2.5">
                {activePlaybook.events.map((evt, idx) => (
                  <div
                    key={idx}
                    className="flex items-start space-x-3 p-2.5 rounded-lg bg-black/40 border border-white/5 font-mono text-xs"
                  >
                    <span className="text-white/40 shrink-0 text-[11px]">{evt.time}</span>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center space-x-2 mb-0.5">
                        <span className="text-white font-semibold text-[11px]">{evt.code}</span>
                        <span
                          className={`text-[9px] uppercase px-1.5 py-0.5 rounded font-bold ${
                            evt.status === 'GUARDED'
                              ? 'bg-amber-500/20 text-amber-300 border border-amber-500/30'
                              : evt.status === 'OPTIMIZED'
                              ? 'bg-blue-500/20 text-blue-300 border border-blue-500/30'
                              : evt.status === 'ATOMIC'
                              ? 'bg-purple-500/20 text-purple-300 border border-purple-500/30'
                              : 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/30'
                          }`}
                        >
                          {evt.status}
                        </span>
                      </div>
                      <p className="text-white/70 font-sans text-xs leading-relaxed">{evt.text}</p>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* Right Inspector: Playbook Stepper & Guardrail Invariants (3 cols) */}
          <div className="lg:col-span-3 p-5 sm:p-6 flex flex-col justify-between bg-[#121216]">
            <div>
              <div className="text-[10px] font-mono uppercase tracking-[0.2em] text-white/40 mb-3">
                LIFECYCLE RECOVERY STAGES
              </div>

              {/* Stepper */}
              <div className="space-y-3 mb-6">
                {activePlaybook.stages.map((stage, sIdx) => (
                  <div key={sIdx} className="flex items-start space-x-3">
                    <div className="mt-1">
                      {stage.state === 'done' ? (
                        <div className="w-4 h-4 rounded-full bg-emerald-500/20 border border-emerald-500 text-emerald-400 flex items-center justify-center text-[10px] font-bold">
                          ✓
                        </div>
                      ) : stage.state === 'active' ? (
                        <div className="w-4 h-4 rounded-full bg-blue-500/20 border border-blue-500 flex items-center justify-center">
                          <span className="w-1.5 h-1.5 rounded-full bg-blue-400 animate-ping" />
                        </div>
                      ) : (
                        <div className="w-4 h-4 rounded-full border border-white/20 bg-white/5" />
                      )}
                    </div>
                    <div>
                      <div
                        className={`text-xs font-semibold ${
                          stage.state === 'active'
                            ? 'text-blue-300'
                            : stage.state === 'done'
                            ? 'text-white'
                            : 'text-white/40'
                        }`}
                      >
                        {stage.name}
                      </div>
                      <div className="text-[11px] text-white/50 leading-tight">{stage.desc}</div>
                    </div>
                  </div>
                ))}
              </div>

              <div className="border-t border-white/10 pt-4 mb-6">
                <div className="text-[10px] font-mono uppercase tracking-[0.2em] text-white/40 mb-2">
                  ARCHITECTURAL INVARIANTS
                </div>
                <ul className="space-y-2 text-xs text-white/70 font-sans">
                  {activePlaybook.invariants.map((inv, iIdx) => (
                    <li key={iIdx} className="flex items-start space-x-2">
                      <span className="text-[#E2FD52] font-mono shrink-0">·</span>
                      <span className="leading-snug">{inv}</span>
                    </li>
                  ))}
                </ul>
              </div>
            </div>

            <div className="pt-4 border-t border-white/10">
              <Link
                href={`/operations/${activePlaybook.slug}`}
                className="group flex items-center justify-between w-full px-4 py-3 rounded-xl bg-white text-black font-semibold text-xs uppercase tracking-wider hover:bg-[#E2FD52] transition-colors"
              >
                <span>Read Full Spec</span>
                <span className="transition-transform duration-200 group-hover:translate-x-1 font-bold">
                  →
                </span>
              </Link>
            </div>
          </div>
        </div>
      </div>

      {/* ── War Stories Exception Playbooks Grid ── */}
      <div>
        <div className="flex flex-col sm:flex-row sm:items-end sm:justify-between mb-6">
          <div>
            <div className="text-[10px] font-mono uppercase tracking-[0.2em] text-white/40 mb-1">
              PROVEN PLAYBOOKS FOR THE WAR STORIES
            </div>
            <h3 className="text-xl sm:text-2xl font-light text-white">
              Every field exception has a guarded deterministic path
            </h3>
          </div>
          <Link
            href="/operations"
            className="mt-3 sm:mt-0 font-mono text-xs uppercase tracking-widest text-[#E2FD52] hover:underline"
          >
            Browse all 10 operations deep-dives →
          </Link>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {WAR_STORIES.map((story) => (
            <Link
              key={story.slug}
              href={`/operations/${story.slug}`}
              className="group relative flex flex-col justify-between p-6 rounded-xl border border-white/10 bg-[#121216] hover:border-white/25 hover:bg-[#16161D] transition-all duration-200"
            >
              <div>
                <div className="flex items-center justify-between mb-3">
                  <span className="font-mono text-[10px] tracking-wider text-white/50 bg-white/5 px-2 py-0.5 rounded border border-white/10">
                    {story.code}
                  </span>
                  <span
                    className="w-2 h-2 rounded-full group-hover:scale-125 transition-transform"
                    style={{ backgroundColor: story.accent }}
                  />
                </div>
                <h4 className="text-base font-semibold text-white mb-2 group-hover:text-white transition-colors">
                  {story.title}
                </h4>
                <div className="text-xs text-white/50 mb-3 leading-relaxed">
                  <strong className="text-white/70 font-medium">Anomaly: </strong>
                  {story.problem}
                </div>
                <div className="text-xs text-white/80 leading-relaxed bg-black/40 p-3 rounded-lg border border-white/5">
                  <strong className="text-emerald-400 font-medium">Resolution: </strong>
                  {story.solution}
                </div>
              </div>
              <div className="mt-4 pt-3 border-t border-white/5 flex items-center justify-between text-xs font-mono text-white/60 group-hover:text-white transition-colors">
                <span>VIEW PLAYBOOK FLOW</span>
                <span className="transform group-hover:translate-x-1 transition-transform">→</span>
              </div>
            </Link>
          ))}
        </div>
      </div>
    </PageSection>
  );
}
