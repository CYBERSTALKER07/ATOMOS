'use client';

import React, { useState } from 'react';
import { useLanguage } from '../context/LanguageContext';

type WorkflowTab = 'stack' | 'supplier' | 'warehouse' | 'retailer' | 'fleet';

interface ToolNode {
  id: string;
  name: string;
  sub: string;
  bgColor?: string;
  textColor?: string;
  icon: React.ReactNode;
}

interface TreeBranch {
  category: string;
  nodes: ToolNode[];
}

interface TreeData {
  centerLabel: string;
  subtitle: string;
  topLeft: TreeBranch;
  bottomLeft: TreeBranch;
  topRight: TreeBranch;
  bottomRight: TreeBranch;
}

// ── Icons for Stack & Logistics Nodes ──
function OpenAIIcon() {
  return (
    <svg viewBox="0 0 24 24" className="w-5 h-5" fill="currentColor">
      <path d="M22.282 9.821a5.985 5.985 0 0 0-.516-4.91 6.046 6.046 0 0 0-6.51-2.9A6.065 6.065 0 0 0 4.981 4.18a5.985 5.985 0 0 0-3.998 2.9 6.046 6.046 0 0 0 .743 7.097 5.98 5.98 0 0 0 .51 4.911 6.051 6.051 0 0 0 6.515 2.9A5.985 5.985 0 0 0 13.26 24a6.056 6.056 0 0 0 5.772-4.206 5.99 5.99 0 0 0 3.997-2.9 6.056 6.056 0 0 0-.747-7.073zM13.26 22.43a4.476 4.476 0 0 1-2.876-1.04l.141-.081 4.779-2.758a.795.795 0 0 0 .392-.681v-6.737l2.02 1.168a.071.071 0 0 1 .038.052v5.583a4.504 4.504 0 0 1-4.494 4.494zM3.6 18.304a4.47 4.47 0 0 1-.535-3.014l.142.085 4.783 2.759a.771.771 0 0 0 .78 0l5.843-3.369v2.332a.08.08 0 0 1-.033.062L9.74 19.95a4.5 4.5 0 0 1-6.14-1.646zM2.34 8.487a4.485 4.485 0 0 1 2.365-1.98v5.673a.792.792 0 0 0 .393.685l5.82 3.372-2.02 1.166a.08.08 0 0 1-.073.006l-4.84-2.794a4.499 4.499 0 0 1-1.645-6.128zm14.168 4.25-5.84-3.372 2.02-1.166a.08.08 0 0 1 .073-.006l4.84 2.793a4.502 4.502 0 0 1-.685 8.12v-5.688a.79.79 0 0 0-.408-.681zm2.757-4.148-4.78-2.763a.777.777 0 0 0-.784 0l-5.84 3.37v-2.332a.08.08 0 0 1 .033-.062L12.72 4.01a4.499 4.499 0 0 1 6.545 4.589zm-8.815-5.328a4.485 4.485 0 0 1 2.876 1.04l-.141.081-4.779 2.758a.795.795 0 0 0-.392.681v6.737l-2.02-1.168a.071.071 0 0 1-.038-.052V8.905a4.504 4.504 0 0 1 4.494-4.494zm1.085 7.76-2.614-1.509 2.614-1.509 2.614 1.509-2.614 1.509z" />
    </svg>
  );
}

function ClaudeIcon() {
  return (
    <span className="font-serif font-black text-sm tracking-tighter text-[#D97757]">
      A\
    </span>
  );
}

function GeminiIcon() {
  return (
    <svg viewBox="0 0 24 24" className="w-5 h-5 text-black" fill="currentColor">
      <path d="M12 0C12 6.627 6.627 12 0 12c6.627 0 12 5.373 12 12 0-6.627 5.373-12 12-12-6.627 0-12-5.373-12-12z" />
    </svg>
  );
}

function MistralIcon() {
  return (
    <svg viewBox="0 0 24 24" className="w-4 h-4 text-white" fill="currentColor">
      <path d="M3 3h4v4H3V3zm14 0h4v4h-4V3zM3 17h4v4H3v-4zm14 0h4v4h-4v-4zM7 7h10v4H7V7zm3 4h4v6h-4v-6z" />
    </svg>
  );
}

function SupabaseIcon() {
  return (
    <svg viewBox="0 0 24 24" className="w-4 h-4 text-white" fill="currentColor">
      <path d="M21.362 9.354H12V.396a.396.396 0 0 0-.716-.233L.392 13.914a.396.396 0 0 0 .31.632H12v8.958a.396.396 0 0 0 .716.233l10.892-13.751a.396.396 0 0 0-.246-.632z" />
    </svg>
  );
}

function LangChainIcon() {
  return (
    <svg viewBox="0 0 24 24" className="w-4 h-4 text-[#38bdf8]" fill="none" stroke="currentColor" strokeWidth="2.5">
      <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71" />
      <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" />
    </svg>
  );
}

function ReactIcon() {
  return (
    <svg viewBox="-11.5 -10.232 23 20.463" className="w-5 h-5 text-[#087ea4]" fill="currentColor">
      <circle cx="0" cy="0" r="2.05" />
      <g stroke="currentColor" strokeWidth="1" fill="none">
        <ellipse rx="11" ry="4.2" />
        <ellipse rx="11" ry="4.2" transform="rotate(60)" />
        <ellipse rx="11" ry="4.2" transform="rotate(120)" />
      </g>
    </svg>
  );
}

function ViteIcon() {
  return (
    <span className="font-mono font-bold text-xs text-white">
      V
    </span>
  );
}

function SvelteIcon() {
  return (
    <svg viewBox="0 0 24 24" className="w-4 h-4 text-white" fill="currentColor">
      <path d="M20.5 8.7c-.5-1.5-1.6-2.7-3.1-3.4L11.5 2C9.9 1.2 8 .9 6.2 1.4c-1.8.5-3.3 1.6-4.2 3.1-.9 1.5-1.2 3.3-.8 5 .4 1.7 1.5 3.2 3 4.1l2.3 1.4-1.5 2.6c-.7 1.2-.8 2.6-.3 3.9.5 1.3 1.5 2.2 2.7 2.7 1.3.5 2.7.4 3.9-.3l5.9-3.4c1.2-.7 2.1-1.8 2.6-3.1.5-1.3.4-2.7-.3-3.9l-2.3-1.4 1.5-2.6c.9-1.2 1.2-2.5 1.2-3.8z" />
    </svg>
  );
}

function ShadcnIcon() {
  return (
    <svg viewBox="0 0 24 24" className="w-4 h-4 text-white" fill="currentColor">
      <circle cx="12" cy="12" r="8" fill="none" stroke="currentColor" strokeWidth="2.5" strokeDasharray="16 6" />
    </svg>
  );
}

function TailwindIcon() {
  return (
    <span className="font-mono font-bold text-xs text-white">
      //
    </span>
  );
}

function MuiIcon() {
  return (
    <span className="font-sans font-black text-xs text-white">
      M
    </span>
  );
}

function DaisyIcon() {
  return (
    <span className="font-sans font-black text-xs text-white">
      D
    </span>
  );
}

function GenericLogisticsIcon({ label }: { label: string }) {
  return (
    <span className="font-mono text-[10px] font-bold text-white tracking-tighter">
      {label}
    </span>
  );
}

// ── DATA FOR ALL TABS ──
const ALL_TREES: Record<WorkflowTab, TreeData> = {
  stack: {
    centerLabel: 'WORKS WITH ANY STACK',
    subtitle: 'Plug into any client, backend, vector DB, or UI design library',
    topLeft: {
      category: 'LLM',
      nodes: [
        { id: 'openai', name: 'OpenAI', sub: 'GPT-4o / o1', bgColor: '#FFFFFF', textColor: '#000000', icon: <OpenAIIcon /> },
        { id: 'claude', name: 'Anthropic', sub: 'Claude 3.7', bgColor: '#F4ECE4', textColor: '#D97757', icon: <ClaudeIcon /> },
        { id: 'gemini', name: 'Google', sub: 'Gemini 2.5', bgColor: '#FFFFFF', textColor: '#000000', icon: <GeminiIcon /> },
        { id: 'mistral', name: 'Mistral', sub: 'Large 2', bgColor: '#FF7A1A', textColor: '#FFFFFF', icon: <MistralIcon /> },
      ],
    },
    bottomLeft: {
      category: 'Backend',
      nodes: [
        { id: 'supabase', name: 'Supabase', sub: 'Postgres pgvector', bgColor: '#1E293B', textColor: '#FFFFFF', icon: <SupabaseIcon /> },
        { id: 'langchain', name: 'LangChain', sub: 'Orchestration', bgColor: '#0F172A', textColor: '#38bdf8', icon: <LangChainIcon /> },
        { id: 'copilot', name: 'CopilotKit', sub: 'AI Co-pilot SDK', bgColor: '#EF4444', textColor: '#FFFFFF', icon: <span className="font-bold text-xs">C</span> },
        { id: 'fastapi', name: 'Python / Go', sub: 'gRPC & REST', bgColor: '#FFFFFF', textColor: '#000000', icon: <OpenAIIcon /> },
      ],
    },
    topRight: {
      category: 'Client',
      nodes: [
        { id: 'react', name: 'React', sub: 'Next.js 15', bgColor: '#FFFFFF', textColor: '#000000', icon: <ReactIcon /> },
        { id: 'vite', name: 'Vite', sub: 'SPA & PWA', bgColor: '#18181B', textColor: '#FFFFFF', icon: <ViteIcon /> },
        { id: 'svelte', name: 'Svelte', sub: 'SvelteKit 2', bgColor: '#18181B', textColor: '#FF3E00', icon: <SvelteIcon /> },
        { id: 'rn', name: 'React Native', sub: 'iOS / Android', bgColor: '#FFFFFF', textColor: '#000000', icon: <ReactIcon /> },
      ],
    },
    bottomRight: {
      category: 'Design library',
      nodes: [
        { id: 'shadcn', name: 'shadcn/ui', sub: 'Radix Primitives', bgColor: '#FFFFFF', textColor: '#000000', icon: <ShadcnIcon /> },
        { id: 'tailwind', name: 'Tailwind CSS', sub: 'v4 Tokens', bgColor: '#18181B', textColor: '#38BDF8', icon: <TailwindIcon /> },
        { id: 'mui', name: 'Material UI', sub: 'MUI Core v6', bgColor: '#818CF8', textColor: '#FFFFFF', icon: <MuiIcon /> },
        { id: 'daisy', name: 'DaisyUI', sub: 'Semantic CSS', bgColor: '#EC4899', textColor: '#FFFFFF', icon: <DaisyIcon /> },
      ],
    },
  },

  supplier: {
    centerLabel: 'SUPPLIER WORKFLOW CORE',
    subtitle: 'Autonomous B2B order orchestration & inventory allocation',
    topLeft: {
      category: 'Inbound Orders',
      nodes: [
        { id: 's_edi', name: 'EDI 850', sub: 'B2B Purchase Order', bgColor: '#FFFFFF', textColor: '#000', icon: <GenericLogisticsIcon label="EDI" /> },
        { id: 's_api', name: 'REST API', sub: 'ERP Ingestion', bgColor: '#1E293B', textColor: '#CEFF00', icon: <GenericLogisticsIcon label="API" /> },
        { id: 's_tg', name: 'Telegram Bot', sub: 'Direct Retailer PO', bgColor: '#0284C7', textColor: '#FFF', icon: <GenericLogisticsIcon label="TG" /> },
        { id: 's_hook', name: 'Webhook', sub: 'Real-time Event', bgColor: '#FF7A1A', textColor: '#FFF', icon: <GenericLogisticsIcon label="HOOK" /> },
      ],
    },
    bottomLeft: {
      category: 'AI Inventory Check',
      nodes: [
        { id: 's_stock', name: 'Stock Check', sub: 'Realtime Matrix', bgColor: '#18181B', textColor: '#FFF', icon: <GenericLogisticsIcon label="STK" /> },
        { id: 's_wms', name: 'WMS Sync', sub: 'Warehouse Bin Level', bgColor: '#047857', textColor: '#FFF', icon: <GenericLogisticsIcon label="WMS" /> },
        { id: 's_safe', name: 'Safety Margin', sub: 'Demand Buffer', bgColor: '#7C3AED', textColor: '#FFF', icon: <GenericLogisticsIcon label="BUF" /> },
        { id: 's_alloc', name: 'Auto-Allocate', sub: 'Batch Priority Lock', bgColor: '#FFFFFF', textColor: '#000', icon: <GenericLogisticsIcon label="LOCK" /> },
      ],
    },
    topRight: {
      category: 'Release to Floor',
      nodes: [
        { id: 's_floor', name: 'Floor Release', sub: 'Digital Pick Ticket', bgColor: '#FFFFFF', textColor: '#000', icon: <GenericLogisticsIcon label="REL" /> },
        { id: 's_bar', name: 'Barcode Pick', sub: 'PDA Handheld', bgColor: '#18181B', textColor: '#FFF', icon: <GenericLogisticsIcon label="PDA" /> },
        { id: 's_qa', name: 'Vision QA', sub: 'Visual Inspection AI', bgColor: '#18181B', textColor: '#CEFF00', icon: <GenericLogisticsIcon label="QA" /> },
        { id: 's_pal', name: 'Palletize', sub: 'Weight & Dimension', bgColor: '#FFFFFF', textColor: '#000', icon: <GenericLogisticsIcon label="PLT" /> },
      ],
    },
    bottomRight: {
      category: 'Dispatch Alert',
      nodes: [
        { id: 's_tms', name: 'TMS Routing', sub: 'Vehicle Assignment', bgColor: '#FFFFFF', textColor: '#000', icon: <GenericLogisticsIcon label="TMS" /> },
        { id: 's_eta', name: 'Carrier ETA', sub: 'GPS Fleet Link', bgColor: '#18181B', textColor: '#38BDF8', icon: <GenericLogisticsIcon label="ETA" /> },
        { id: 's_asn', name: 'Outbox Event', sub: 'Kafka Fanout', bgColor: '#818CF8', textColor: '#FFF', icon: <GenericLogisticsIcon label="EVT" /> },
        { id: 's_bill', name: 'Auto-Bill', sub: 'Tiyin Ledger Post', bgColor: '#EC4899', textColor: '#FFF', icon: <GenericLogisticsIcon label="BILL" /> },
      ],
    },
  },

  warehouse: {
    centerLabel: 'WAREHOUSE DOCK ENGINE',
    subtitle: 'Smart dock scheduling, conveyor IoT, and automated putaway',
    topLeft: {
      category: 'Inbound Alert',
      nodes: [
        { id: 'w_eta', name: 'Carrier ETA', sub: 'Dock Slot Reserved', bgColor: '#FFFFFF', textColor: '#000', icon: <GenericLogisticsIcon label="ETA" /> },
        { id: 'w_gate', name: 'Gate Check', sub: 'ANPR License Plate', bgColor: '#1E293B', textColor: '#CEFF00', icon: <GenericLogisticsIcon label="GATE" /> },
        { id: 'w_dock', name: 'Dock Assign', sub: 'AI Door Optimizer', bgColor: '#0284C7', textColor: '#FFF', icon: <GenericLogisticsIcon label="DOCK" /> },
        { id: 'w_seal', name: 'Seal Verify', sub: 'Tamper Audit', bgColor: '#FF7A1A', textColor: '#FFF', icon: <GenericLogisticsIcon label="SEAL" /> },
      ],
    },
    bottomLeft: {
      category: 'Scan & Sort',
      nodes: [
        { id: 'w_rfid', name: 'RFID Portal', sub: 'Bulk Pallet Ingest', bgColor: '#18181B', textColor: '#FFF', icon: <GenericLogisticsIcon label="RFID" /> },
        { id: 'w_conv', name: 'Conveyor Sort', sub: 'High-speed Diverter', bgColor: '#047857', textColor: '#FFF', icon: <GenericLogisticsIcon label="SORT" /> },
        { id: 'w_dim', name: 'Dimensioner', sub: 'Laser Cube Volumetric', bgColor: '#7C3AED', textColor: '#FFF', icon: <GenericLogisticsIcon label="DIM" /> },
        { id: 'w_wgt', name: 'Scale Check', sub: 'Gross Weight Match', bgColor: '#FFFFFF', textColor: '#000', icon: <GenericLogisticsIcon label="WGT" /> },
      ],
    },
    topRight: {
      category: 'QC Inspection',
      nodes: [
        { id: 'w_ai_qc', name: 'Visual AI', sub: 'Box Damage Detector', bgColor: '#FFFFFF', textColor: '#000', icon: <GenericLogisticsIcon label="CAM" /> },
        { id: 'w_temp', name: 'Cold Chain', sub: 'IoT Temp Logger', bgColor: '#18181B', textColor: '#38BDF8', icon: <GenericLogisticsIcon label="COLD" /> },
        { id: 'w_batch', name: 'Batch Audit', sub: 'Expiry & Lot Match', bgColor: '#18181B', textColor: '#FFF', icon: <GenericLogisticsIcon label="LOT" /> },
        { id: 'w_quar', name: 'Quarantine', sub: 'Auto-Hold Logic', bgColor: '#EF4444', textColor: '#FFF', icon: <GenericLogisticsIcon label="HOLD" /> },
      ],
    },
    bottomRight: {
      category: 'Putaway & Sync',
      nodes: [
        { id: 'w_fork', name: 'Forklift Nav', sub: 'Directed Putaway', bgColor: '#FFFFFF', textColor: '#000', icon: <GenericLogisticsIcon label="FORK" /> },
        { id: 'w_bin', name: 'Bin Assign', sub: 'Aisle/Bay/Tier', bgColor: '#18181B', textColor: '#CEFF00', icon: <GenericLogisticsIcon label="BIN" /> },
        { id: 'w_erp', name: 'ERP Sync', sub: 'Double-entry Credit', bgColor: '#818CF8', textColor: '#FFF', icon: <GenericLogisticsIcon label="ERP" /> },
        { id: 'w_asn_out', name: 'ASN Ack', sub: 'Supplier Webhook', bgColor: '#EC4899', textColor: '#FFF', icon: <GenericLogisticsIcon label="ACK" /> },
      ],
    },
  },

  retailer: {
    centerLabel: 'RETAIL INVENTORY HUB',
    subtitle: 'Demand forecasting, shelf replenishment, and multi-supplier orders',
    topLeft: {
      category: 'Demand Signal',
      nodes: [
        { id: 'r_pos', name: 'POS Velocity', sub: 'Hourly Sales Rate', bgColor: '#FFFFFF', textColor: '#000', icon: <GenericLogisticsIcon label="POS" /> },
        { id: 'r_alert', name: 'Low Stock Alert', sub: 'Buffer Breach', bgColor: '#EF4444', textColor: '#FFF', icon: <GenericLogisticsIcon label="WARN" /> },
        { id: 'r_pred', name: 'AI Forecast', sub: 'Weekend Spikes', bgColor: '#0284C7', textColor: '#FFF', icon: <GenericLogisticsIcon label="PRED" /> },
        { id: 'r_reord', name: 'Reorder Point', sub: 'Dynamic Safety Days', bgColor: '#FF7A1A', textColor: '#FFF', icon: <GenericLogisticsIcon label="ROP" /> },
      ],
    },
    bottomLeft: {
      category: 'Auto-Procure',
      nodes: [
        { id: 'r_po', name: 'Create PO', sub: 'Automated Draft', bgColor: '#18181B', textColor: '#FFF', icon: <GenericLogisticsIcon label="PO" /> },
        { id: 'r_edi855', name: 'EDI 855', sub: 'Supplier Confirm', bgColor: '#047857', textColor: '#FFF', icon: <GenericLogisticsIcon label="CONF" /> },
        { id: 'r_multi', name: 'Multi-Vendor', sub: 'Lowest Unit Landed', bgColor: '#7C3AED', textColor: '#FFF', icon: <GenericLogisticsIcon label="VEND" /> },
        { id: 'r_escrow', name: 'Pack Escrow', sub: 'Hold Minor Units', bgColor: '#FFFFFF', textColor: '#000', icon: <GenericLogisticsIcon label="ESC" /> },
      ],
    },
    topRight: {
      category: 'Inbound Prep',
      nodes: [
        { id: 'r_asn', name: 'ASN Received', sub: 'Manifest Match', bgColor: '#FFFFFF', textColor: '#000', icon: <GenericLogisticsIcon label="ASN" /> },
        { id: 'r_dock', name: 'Staging Bay', sub: 'Curbside Unload', bgColor: '#18181B', textColor: '#FFF', icon: <GenericLogisticsIcon label="BAY" /> },
        { id: 'r_scan', name: 'Tally Scan', sub: 'Count Verification', bgColor: '#18181B', textColor: '#CEFF00', icon: <GenericLogisticsIcon label="SCAN" /> },
        { id: 'r_cross', name: 'Cross-Dock', sub: 'Direct to Display', bgColor: '#FFFFFF', textColor: '#000', icon: <GenericLogisticsIcon label="XDK" /> },
      ],
    },
    bottomRight: {
      category: 'Shelf Restock',
      nodes: [
        { id: 'r_shelf', name: 'Shelf Restock', sub: 'Facing Planogram', bgColor: '#FFFFFF', textColor: '#000', icon: <GenericLogisticsIcon label="SHLF" /> },
        { id: 'r_endcap', name: 'Endcap Promo', sub: 'Dynamic Tag Sync', bgColor: '#18181B', textColor: '#38BDF8', icon: <GenericLogisticsIcon label="TAG" /> },
        { id: 'r_live', name: 'Store Ledger', sub: 'On-Hand Available', bgColor: '#818CF8', textColor: '#FFF', icon: <GenericLogisticsIcon label="QTY" /> },
        { id: 'r_close', name: 'Close PO', sub: 'Release Escrow Pay', bgColor: '#EC4899', textColor: '#FFF', icon: <GenericLogisticsIcon label="PAID" /> },
      ],
    },
  },

  fleet: {
    centerLabel: 'FLEET DISPATCH MATRIX',
    subtitle: 'Dynamic route optimization, mid-shift re-routing, and DVIR telemetry',
    topLeft: {
      category: 'Routing & TMS',
      nodes: [
        { id: 'f_cvrp', name: 'CVRP Engine', sub: 'Google OR-Tools', bgColor: '#FFFFFF', textColor: '#000', icon: <GenericLogisticsIcon label="CVRP" /> },
        { id: 'f_truck', name: 'Vehicle Match', sub: 'Weight & Volume', bgColor: '#1E293B', textColor: '#CEFF00', icon: <GenericLogisticsIcon label="TRK" /> },
        { id: 'f_driver', name: 'Driver Shift', sub: 'HOS Compliance', bgColor: '#0284C7', textColor: '#FFF', icon: <GenericLogisticsIcon label="DRV" /> },
        { id: 'f_slot', name: 'Time Windows', sub: 'Delivery Windows', bgColor: '#FF7A1A', textColor: '#FFF', icon: <GenericLogisticsIcon label="WIN" /> },
      ],
    },
    bottomLeft: {
      category: 'Weather & Traffic',
      nodes: [
        { id: 'f_traf', name: 'Live Traffic', sub: 'Congestion Mesh', bgColor: '#18181B', textColor: '#FFF', icon: <GenericLogisticsIcon label="JAM" /> },
        { id: 'f_wx', name: 'Weather Radar', sub: 'Storm & Snow Warn', bgColor: '#047857', textColor: '#FFF', icon: <GenericLogisticsIcon label="RADR" /> },
        { id: 'f_reroute', name: 'Dynamic Reroute', sub: 'Mid-shift Hot Swap', bgColor: '#7C3AED', textColor: '#FFF', icon: <GenericLogisticsIcon label="SWP" /> },
        { id: 'f_geofence', name: 'Geofence Ping', sub: 'Arrival Trigger', bgColor: '#FFFFFF', textColor: '#000', icon: <GenericLogisticsIcon label="GEO" /> },
      ],
    },
    topRight: {
      category: 'Driver Telemetry',
      nodes: [
        { id: 'f_brief', name: 'Driver App', sub: 'Native Turn-by-Turn', bgColor: '#FFFFFF', textColor: '#000', icon: <GenericLogisticsIcon label="APP" /> },
        { id: 'f_dvir', name: 'Pre-trip DVIR', sub: 'Safety Checklist', bgColor: '#18181B', textColor: '#FFF', icon: <GenericLogisticsIcon label="DVIR" /> },
        { id: 'f_gps', name: 'Breadcrumbs', sub: '1Hz Telemetry Ping', bgColor: '#18181B', textColor: '#CEFF00', icon: <GenericLogisticsIcon label="GPS" /> },
        { id: 'f_speed', name: 'Speed Guard', sub: 'Safety Telematics', bgColor: '#FFFFFF', textColor: '#000', icon: <GenericLogisticsIcon label="SPD" /> },
      ],
    },
    bottomRight: {
      category: 'Fulfillment',
      nodes: [
        { id: 'f_epod', name: 'ePOD Delivery', sub: 'Customer Signature', bgColor: '#FFFFFF', textColor: '#000', icon: <GenericLogisticsIcon label="EPOD" /> },
        { id: 'f_photo', name: 'Proof Photo', sub: 'Curbside Drop Off', bgColor: '#18181B', textColor: '#38BDF8', icon: <GenericLogisticsIcon label="CAM" /> },
        { id: 'f_inv', name: 'Auto-Invoice', sub: 'Immediate Billing', bgColor: '#818CF8', textColor: '#FFF', icon: <GenericLogisticsIcon label="INV" /> },
        { id: 'f_done', name: 'Shift Close', sub: 'Vehicle Return Audit', bgColor: '#EC4899', textColor: '#FFF', icon: <GenericLogisticsIcon label="END" /> },
      ],
    },
  },
};

const TAB_OPTIONS: { id: WorkflowTab; label: string }[] = [
  { id: 'stack', label: 'WORKS WITH ANY STACK' },
  { id: 'supplier', label: 'SUPPLIER' },
  { id: 'warehouse', label: 'WAREHOUSE' },
  { id: 'retailer', label: 'RETAILER' },
  { id: 'fleet', label: 'FLEET' },
];

export default function LogisticsWorkflow() {
  const { t } = useLanguage();
  const [activeTab, setActiveTab] = useState<WorkflowTab>('stack');
  const [hoveredNode, setHoveredNode] = useState<ToolNode | null>(null);

  const currentTree = ALL_TREES[activeTab];

  return (
    <div className="bg-black w-full relative overflow-hidden font-sans border-t border-white/10 py-24 sm:py-32 select-none">
      <div className="w-[94%] max-w-[1440px] mx-auto z-10 relative">
        {/* Header */}
        <div className="mb-10 text-center flex flex-col items-center">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-white/5 border border-white/10 text-white/50 mb-4">
            <span className="w-1.5 h-1.5 rounded-full bg-[#CEFF00] animate-pulse" />
            <span className="text-[10px] tracking-[0.2em] uppercase font-mono">
              {t('workflow_eyebrow', 'System Architecture')}
            </span>
          </div>

          <h2 className="text-4xl sm:text-5xl md:text-6xl font-title font-bold tracking-tight text-white mb-4">
            {activeTab === 'stack' ? 'Works with any stack' : 'Autonomous operational logic'}
          </h2>

          <p className="text-white/50 max-w-2xl text-sm sm:text-base md:text-lg leading-relaxed">
            {currentTree.subtitle}
          </p>

          {/* Segmented Role / Stack Controller */}
          <div className="mt-8 flex flex-wrap items-center justify-center gap-1.5 p-1.5 rounded-2xl bg-[#09090B] border border-white/15 shadow-2xl max-w-2xl">
            {TAB_OPTIONS.map((tab) => (
              <button
                key={tab.id}
                type="button"
                onClick={() => setActiveTab(tab.id)}
                className={`
                  px-4 py-2 rounded-xl text-xs font-mono tracking-wider transition-all duration-200 cursor-pointer
                  ${
                    activeTab === tab.id
                      ? 'bg-white text-black font-bold shadow-[0_0_20px_rgba(255,255,255,0.2)]'
                      : 'text-white/60 hover:text-white hover:bg-white/5'
                  }
                `}
              >
                {tab.label}
              </button>
            ))}
          </div>
        </div>

        {/* ── PURE SVG & HTML ARCHITECTURE TREE (ZERO SCROLL TRAPPING) ── */}
        <div className="relative w-full overflow-x-auto py-8">
          <div className="min-w-[980px] lg:min-w-[1100px] max-w-[1240px] mx-auto relative h-[420px]">
            {/* SVG Connecting Splines with Animated Flow Beams */}
            <svg
              viewBox="0 0 1200 420"
              className="absolute inset-0 w-full h-full pointer-events-none"
              fill="none"
              xmlns="http://www.w3.org/2000/svg"
            >
              <defs>
                <linearGradient id="beamGradient" x1="0%" y1="0%" x2="100%" y2="0%">
                  <stop offset="0%" stopColor="#3b82f6" stopOpacity="0" />
                  <stop offset="50%" stopColor="#3b82f6" stopOpacity="1" />
                  <stop offset="100%" stopColor="#60a5fa" stopOpacity="0" />
                </linearGradient>
                <style>{`
                  @keyframes pulseFlow {
                    0% { stroke-dashoffset: 300; }
                    100% { stroke-dashoffset: 0; }
                  }
                  .animate-flow-beam {
                    animation: pulseFlow 3.5s linear infinite;
                  }
                  .animate-flow-beam-rev {
                    animation: pulseFlow 3.5s linear infinite reverse;
                  }
                `}</style>
              </defs>

              {/* ── 1. Top-Left Spline + Horizontal Connector ── */}
              {/* Spline: Center -> LLM Pill */}
              <path
                d="M 480 210 C 440 210, 420 95, 395 95"
                stroke="#27272a"
                strokeWidth="1.5"
              />
              <path
                d="M 480 210 C 440 210, 420 95, 395 95"
                stroke="#3b82f6"
                strokeWidth="2.5"
                strokeDasharray="24 160"
                className="animate-flow-beam-rev"
              />
              {/* Horizontal line: LLM Pill -> Tool Nodes */}
              <line x1="330" y1="95" x2="60" y2="95" stroke="#27272a" strokeWidth="1.5" />
              <line
                x1="330"
                y1="95"
                x2="60"
                y2="95"
                stroke="#3b82f6"
                strokeWidth="2.5"
                strokeDasharray="20 120"
                className="animate-flow-beam-rev"
              />

              {/* ── 2. Bottom-Left Spline + Horizontal Connector ── */}
              {/* Spline: Center -> Backend Pill */}
              <path
                d="M 480 210 C 440 210, 420 325, 400 325"
                stroke="#27272a"
                strokeWidth="1.5"
              />
              <path
                d="M 480 210 C 440 210, 420 325, 400 325"
                stroke="#3b82f6"
                strokeWidth="2.5"
                strokeDasharray="24 160"
                className="animate-flow-beam-rev"
              />
              {/* Horizontal line: Backend Pill -> Tool Nodes */}
              <line x1="315" y1="325" x2="60" y2="325" stroke="#27272a" strokeWidth="1.5" />
              <line
                x1="315"
                y1="325"
                x2="60"
                y2="325"
                stroke="#3b82f6"
                strokeWidth="2.5"
                strokeDasharray="20 120"
                className="animate-flow-beam-rev"
              />

              {/* ── 3. Top-Right Spline + Horizontal Connector ── */}
              {/* Spline: Center -> Client Pill */}
              <path
                d="M 720 210 C 760 210, 780 95, 805 95"
                stroke="#27272a"
                strokeWidth="1.5"
              />
              <path
                d="M 720 210 C 760 210, 780 95, 805 95"
                stroke="#3b82f6"
                strokeWidth="2.5"
                strokeDasharray="24 160"
                className="animate-flow-beam"
              />
              {/* Horizontal line: Client Pill -> Tool Nodes */}
              <line x1="875" y1="95" x2="1140" y2="95" stroke="#27272a" strokeWidth="1.5" />
              <line
                x1="875"
                y1="95"
                x2="1140"
                y2="95"
                stroke="#3b82f6"
                strokeWidth="2.5"
                strokeDasharray="20 120"
                className="animate-flow-beam"
              />

              {/* ── 4. Bottom-Right Spline + Horizontal Connector ── */}
              {/* Spline: Center -> Design Library Pill */}
              <path
                d="M 720 210 C 760 210, 775 325, 780 325"
                stroke="#27272a"
                strokeWidth="1.5"
              />
              <path
                d="M 720 210 C 760 210, 775 325, 780 325"
                stroke="#3b82f6"
                strokeWidth="2.5"
                strokeDasharray="24 160"
                className="animate-flow-beam"
              />
              {/* Horizontal line: Design Library Pill -> Tool Nodes */}
              <line x1="895" y1="325" x2="1140" y2="325" stroke="#27272a" strokeWidth="1.5" />
              <line
                x1="895"
                y1="325"
                x2="1140"
                y2="325"
                stroke="#3b82f6"
                strokeWidth="2.5"
                strokeDasharray="20 120"
                className="animate-flow-beam"
              />
            </svg>

            {/* ── CENTER PILL (Double-Bordered Monospace Badge) ── */}
            <div className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 z-20">
              <div className="p-1 rounded-2xl bg-[#141416] border border-[#27272a] shadow-[0_12px_40px_rgba(0,0,0,0.9)]">
                <div className="px-6 py-3.5 rounded-xl bg-[#09090b] border border-white/10 flex items-center justify-center">
                  <span className="font-mono text-xs sm:text-sm font-semibold tracking-wider text-white whitespace-nowrap">
                    {currentTree.centerLabel}
                  </span>
                </div>
              </div>
            </div>

            {/* ── TOP-LEFT BRANCH (Category + Circular Nodes) ── */}
            <div className="absolute top-[75px] left-[330px] z-20">
              <div className="px-4 py-1.5 rounded-full bg-[#121216] border border-white/10 text-white font-mono text-xs font-semibold shadow-md whitespace-nowrap">
                {currentTree.topLeft.category}
              </div>
            </div>
            {/* Top-Left Circular Icons */}
            <div className="absolute top-[75px] left-[60px] flex items-center gap-7 z-20">
              {currentTree.topLeft.nodes.map((node) => (
                <div
                  key={node.id}
                  className="relative group/node"
                  onMouseEnter={() => setHoveredNode(node)}
                  onMouseLeave={() => setHoveredNode(null)}
                >
                  <div
                    style={{ backgroundColor: node.bgColor || '#FFFFFF', color: node.textColor || '#000000' }}
                    className="w-10 h-10 rounded-full flex items-center justify-center shadow-lg border border-white/20 transition-transform duration-200 group-hover/node:scale-115 cursor-pointer"
                  >
                    {node.icon}
                  </div>
                  {/* Tooltip */}
                  <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 px-2.5 py-1 rounded-md bg-[#18181B] border border-white/20 text-white text-[10px] font-mono whitespace-nowrap opacity-0 pointer-events-none group-hover/node:opacity-100 transition-opacity duration-150 z-30 shadow-xl">
                    <span className="font-bold">{node.name}</span>
                    <span className="text-white/50 ml-1.5">({node.sub})</span>
                  </div>
                </div>
              ))}
            </div>

            {/* ── BOTTOM-LEFT BRANCH (Category + Circular Nodes) ── */}
            <div className="absolute top-[305px] left-[320px] z-20">
              <div className="px-4 py-1.5 rounded-full bg-[#121216] border border-white/10 text-white font-mono text-xs font-semibold shadow-md whitespace-nowrap">
                {currentTree.bottomLeft.category}
              </div>
            </div>
            {/* Bottom-Left Circular Icons */}
            <div className="absolute top-[305px] left-[60px] flex items-center gap-7 z-20">
              {currentTree.bottomLeft.nodes.map((node) => (
                <div
                  key={node.id}
                  className="relative group/node"
                  onMouseEnter={() => setHoveredNode(node)}
                  onMouseLeave={() => setHoveredNode(null)}
                >
                  <div
                    style={{ backgroundColor: node.bgColor || '#18181B', color: node.textColor || '#FFFFFF' }}
                    className="w-10 h-10 rounded-full flex items-center justify-center shadow-lg border border-white/20 transition-transform duration-200 group-hover/node:scale-115 cursor-pointer"
                  >
                    {node.icon}
                  </div>
                  {/* Tooltip */}
                  <div className="absolute top-full left-1/2 -translate-x-1/2 mt-2 px-2.5 py-1 rounded-md bg-[#18181B] border border-white/20 text-white text-[10px] font-mono whitespace-nowrap opacity-0 pointer-events-none group-hover/node:opacity-100 transition-opacity duration-150 z-30 shadow-xl">
                    <span className="font-bold">{node.name}</span>
                    <span className="text-white/50 ml-1.5">({node.sub})</span>
                  </div>
                </div>
              ))}
            </div>

            {/* ── TOP-RIGHT BRANCH (Category + Circular Nodes) ── */}
            <div className="absolute top-[75px] left-[800px] z-20">
              <div className="px-4 py-1.5 rounded-full bg-[#121216] border border-white/10 text-white font-mono text-xs font-semibold shadow-md whitespace-nowrap">
                {currentTree.topRight.category}
              </div>
            </div>
            {/* Top-Right Circular Icons */}
            <div className="absolute top-[75px] left-[915px] flex items-center gap-7 z-20">
              {currentTree.topRight.nodes.map((node) => (
                <div
                  key={node.id}
                  className="relative group/node"
                  onMouseEnter={() => setHoveredNode(node)}
                  onMouseLeave={() => setHoveredNode(null)}
                >
                  <div
                    style={{ backgroundColor: node.bgColor || '#FFFFFF', color: node.textColor || '#000000' }}
                    className="w-10 h-10 rounded-full flex items-center justify-center shadow-lg border border-white/20 transition-transform duration-200 group-hover/node:scale-115 cursor-pointer"
                  >
                    {node.icon}
                  </div>
                  {/* Tooltip */}
                  <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 px-2.5 py-1 rounded-md bg-[#18181B] border border-white/20 text-white text-[10px] font-mono whitespace-nowrap opacity-0 pointer-events-none group-hover/node:opacity-100 transition-opacity duration-150 z-30 shadow-xl">
                    <span className="font-bold">{node.name}</span>
                    <span className="text-white/50 ml-1.5">({node.sub})</span>
                  </div>
                </div>
              ))}
            </div>

            {/* ── BOTTOM-RIGHT BRANCH (Category + Circular Nodes) ── */}
            <div className="absolute top-[305px] left-[780px] z-20">
              <div className="px-4 py-1.5 rounded-full bg-[#121216] border border-white/10 text-white font-mono text-xs font-semibold shadow-md whitespace-nowrap">
                {currentTree.bottomRight.category}
              </div>
            </div>
            {/* Bottom-Right Circular Icons */}
            <div className="absolute top-[305px] left-[940px] flex items-center gap-7 z-20">
              {currentTree.bottomRight.nodes.map((node) => (
                <div
                  key={node.id}
                  className="relative group/node"
                  onMouseEnter={() => setHoveredNode(node)}
                  onMouseLeave={() => setHoveredNode(null)}
                >
                  <div
                    style={{ backgroundColor: node.bgColor || '#FFFFFF', color: node.textColor || '#000000' }}
                    className="w-10 h-10 rounded-full flex items-center justify-center shadow-lg border border-white/20 transition-transform duration-200 group-hover/node:scale-115 cursor-pointer"
                  >
                    {node.icon}
                  </div>
                  {/* Tooltip */}
                  <div className="absolute top-full left-1/2 -translate-x-1/2 mt-2 px-2.5 py-1 rounded-md bg-[#18181B] border border-white/20 text-white text-[10px] font-mono whitespace-nowrap opacity-0 pointer-events-none group-hover/node:opacity-100 transition-opacity duration-150 z-30 shadow-xl">
                    <span className="font-bold">{node.name}</span>
                    <span className="text-white/50 ml-1.5">({node.sub})</span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Active Node Inspector Footer */}
        {hoveredNode ? (
          <div className="mt-4 flex items-center justify-center">
            <div className="px-4 py-2 rounded-xl bg-[#121216] border border-[#CEFF00]/40 text-[#CEFF00] font-mono text-xs flex items-center gap-3 animate-fade-in shadow-lg">
              <span className="w-2 h-2 rounded-full bg-[#CEFF00] animate-ping" />
              <span>ACTIVE NODE: <strong>{hoveredNode.name}</strong></span>
              <span className="text-white/50">|</span>
              <span className="text-white/80">{hoveredNode.sub}</span>
            </div>
          </div>
        ) : (
          <div className="mt-4 text-center">
            <span className="text-[11px] font-mono uppercase tracking-widest text-white/30">
              Hover nodes to inspect integration telemetry // Zero scroll interception
            </span>
          </div>
        )}
      </div>
    </div>
  );
}

