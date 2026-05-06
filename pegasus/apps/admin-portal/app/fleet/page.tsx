"use client";

import React, { useState, useEffect, useRef, useCallback, useMemo } from "react";
import dynamic from "next/dynamic";
import type { MarkerProps, SourceProps, LayerProps, NavigationControlProps } from "react-map-gl/maplibre";
import "maplibre-gl/dist/maplibre-gl.css";
import { apiFetch, getAdminToken, readTokenFromCookie } from "@/lib/auth";
import { useSyncHub } from "@/lib/useSyncHub";
import { extractDriverPositions, useTelemetry } from "@/hooks/useTelemetry";
import type { TelemetryMessage } from "@/hooks/useTelemetry";
import {
    isTauri,
    connectNativeTelemetry,
    disconnectNativeTelemetry,
    onTelemetryPing,
    onTelemetryStatus,
} from "@/lib/bridge";

const MapGL = dynamic(() => import("react-map-gl/maplibre").then(m => m.default), { ssr: false });
const Marker = dynamic(() => import("react-map-gl/maplibre").then(m => m.Marker), { ssr: false }) as React.ComponentType<MarkerProps>;
const Source = dynamic(() => import("react-map-gl/maplibre").then(m => m.Source), { ssr: false }) as React.ComponentType<SourceProps>;
const Layer = dynamic(() => import("react-map-gl/maplibre").then(m => m.Layer), { ssr: false }) as React.ComponentType<LayerProps>;
const NavigationControl = dynamic(() => import("react-map-gl/maplibre").then(m => m.NavigationControl), { ssr: false }) as React.ComponentType<NavigationControlProps>;

interface DriverPing {
    driver_id: string;
    latitude: number;
    longitude: number;
    timestamp?: number;
}

interface DriverState {
    driver_id: string;
    latitude: number;
    longitude: number;
    last_seen: string;
    last_seen_epoch: number;
    ping_count: number;
}

interface ActiveMission {
    order_id: string;
    state: string;
    target_lat: number;
    target_lng: number;
    amount: number;
    gateway: string;
    route_id: string;
    estimated_arrival_at?: string;
}

interface FleetDriverInfo {
    driver_id: string;
    name: string;
    phone: string;
    vehicle_type: string;
    license_plate: string;
    truck_status: string;
    is_active: boolean;
    vehicle_id?: string;
    vehicle_class?: string;
    estimated_return_at?: string;
    return_duration_sec?: number;
}

const MAP_STYLE =
    "https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json";

// Stale thresholds
const AMBER_THRESHOLD = 30_000; // 30s
const RED_THRESHOLD = 60_000;   // 60s

function getStaleness(epochMs: number): "live" | "amber" | "stale" {
    const delta = Date.now() - epochMs;
    if (delta > RED_THRESHOLD) return "stale";
    if (delta > AMBER_THRESHOLD) return "amber";
    return "live";
}

function getMarkerColor(staleness: "live" | "amber" | "stale"): string {
    switch (staleness) {
        case "stale": return "var(--danger)";
        case "amber": return "var(--muted)";
        default: return "var(--accent)";
    }
}

function getTruckStatusStyle(status: string): { bg: string; fg: string; label: string } {
    switch (status) {
        case "IN_TRANSIT": return { bg: "var(--accent)", fg: "var(--accent-foreground)", label: "In Transit" };
        case "DISPATCHED": return { bg: "var(--accent)", fg: "var(--accent-foreground)", label: "Dispatched" };
        case "RETURNING": return { bg: "var(--warning)", fg: "var(--background)", label: "Returning" };
        case "LOADING": return { bg: "var(--muted)", fg: "var(--foreground)", label: "Loading" };
        case "READY": return { bg: "var(--muted)", fg: "var(--foreground)", label: "Ready" };
        case "AVAILABLE": return { bg: "var(--surface)", fg: "var(--foreground)", label: "Available" };
        default: return { bg: "var(--surface)", fg: "var(--muted)", label: status };
    }
}

function formatETATime(isoString: string): string {
    try {
        const date = new Date(isoString);
        return date.toLocaleTimeString("en-US", { hour: "2-digit", minute: "2-digit", hour12: false });
    } catch {
        return "—";
    }
}

function formatDuration(seconds: number): string {
    if (seconds < 60) return `${seconds}s`;
    const mins = Math.round(seconds / 60);
    if (mins < 60) return `${mins}m`;
    const h = Math.floor(mins / 60);
    const m = mins % 60;
    return m > 0 ? `${h}h ${m}m` : `${h}h`;
}

export default function FleetPage() {
    const [drivers, setDrivers] = useState<Map<string, DriverState>>(new Map());
    const [missions, setMissions] = useState<ActiveMission[]>([]);
    const [fleetInfo, setFleetInfo] = useState<Map<string, FleetDriverInfo>>(new Map());
    const [wsStatus, setWsStatus] = useState<"CONNECTING" | "LIVE" | "OFFLINE">("CONNECTING");
    const [log, setLog] = useState<string[]>([]);
    const [selectedDriverId, setSelectedDriverId] = useState<string | null>(null);
    const fleetInfoRef = useRef<Map<string, FleetDriverInfo>>(new Map());
    const lastWebStatusRef = useRef<"CONNECTING" | "LIVE" | "OFFLINE" | null>(null);

    const appendLog = useCallback((msg: string) => {
        const ts = new Date().toLocaleTimeString("en-US", { hour12: false });
        setLog((prev) => [`[${ts}] ${msg}`, ...prev].slice(0, 50));
    }, []);

    // ── Fetch fleet metadata + active missions ─────────────────────────────
    const fetchFleetData = useCallback(async (signal?: AbortSignal) => {
        try {
            const [missionRes, driverRes] = await Promise.all([
                apiFetch('/v1/fleet/active', { signal }),
                apiFetch('/v1/supplier/fleet/drivers', { signal }),
            ]);

            if (missionRes.ok) {
                const data: ActiveMission[] = await missionRes.json();
                setMissions(data ?? []);
            }
            if (driverRes.ok) {
                const data: FleetDriverInfo[] = await driverRes.json();
                const map = new Map<string, FleetDriverInfo>();
                for (const d of data ?? []) map.set(d.driver_id, d);
                setFleetInfo(map);
                fleetInfoRef.current = map;
            }
        } catch (err) {
            if ((err as Error).name === 'AbortError') return;
            // Silent — WS is the primary feed, REST is supplemental
        }
    }, []);

    useSyncHub("POLL", "default", (signal) => fetchFleetData(signal), 10_000);

    // Ref for WS event handlers to trigger instant refetch
    const fetchFleetDataRef = useRef(fetchFleetData);
    fetchFleetDataRef.current = fetchFleetData;

    const handlePing = useCallback((ping: DriverPing) => {
        if (fleetInfoRef.current.size > 0 && !fleetInfoRef.current.has(ping.driver_id)) return;
        setDrivers((prev) => {
            const next = new Map(prev);
            const existing = next.get(ping.driver_id);
            next.set(ping.driver_id, {
                driver_id: ping.driver_id,
                latitude: ping.latitude,
                longitude: ping.longitude,
                last_seen: new Date().toLocaleTimeString("en-US", { hour12: false }),
                last_seen_epoch: Date.now(),
                ping_count: (existing?.ping_count ?? 0) + 1,
            });
            return next;
        });
        appendLog(`${ping.driver_id} → ${ping.latitude.toFixed(5)}, ${ping.longitude.toFixed(5)}`);
    }, [appendLog]);

    const webTelemetry = useTelemetry(
        useCallback((message: TelemetryMessage) => {
            if (
                message.type === "ORDER_STATE_CHANGED" ||
                message.type === "DRIVER_APPROACHING" ||
                message.type === "ETA_UPDATED"
            ) {
                fetchFleetDataRef.current();
                appendLog(`${message.type}: ${message.order_id ?? message.driver_id ?? message.state ?? ""}`);
                return;
            }

            const positions = extractDriverPositions(message);
            for (const position of positions) {
                handlePing({
                    driver_id: position.driver_id,
                    latitude: position.latitude,
                    longitude: position.longitude,
                    timestamp: position.timestamp,
                });
            }
        }, [appendLog, handlePing]),
        { enabled: !isTauri() },
    );

    useEffect(() => {
        if (isTauri()) return;

        setWsStatus(webTelemetry.status);
        if (lastWebStatusRef.current === webTelemetry.status) {
            return;
        }

        lastWebStatusRef.current = webTelemetry.status;
        if (webTelemetry.status === "LIVE") {
            appendLog("Telemetry pipe established.");
        } else if (webTelemetry.status === "CONNECTING") {
            appendLog("Telemetry pipe connecting.");
        } else {
            appendLog("Pipe offline.");
        }
    }, [appendLog, webTelemetry.status]);

    // ── WebSocket telemetry (desktop Rust bridge only; web uses shared hook) ──
    useEffect(() => {
        let isDisposed = false;
        const unlisteners: (() => void)[] = [];
        if (!isTauri()) {
            return;
        }

        const setup = async () => {
            const apiBase = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
            const cookieToken = readTokenFromCookie();
            let token = cookieToken;

            if (!token) {
                try {
                    token = await getAdminToken();
                } catch {
                    if (!isDisposed) {
                        setWsStatus("OFFLINE");
                        appendLog("Telemetry auth unavailable.");
                    }
                    return;
                }
            }

            if (isDisposed) return;

            // ── Desktop: use Rust-backed persistent WebSocket ──
            if (isTauri()) {
                appendLog("Using native Rust telemetry pipe.");

                const unPing = await onTelemetryPing(handlePing);
                unlisteners.push(unPing);

                const unStatus = await onTelemetryStatus((status: string) => {
                    if (isDisposed) return;
                    setWsStatus(status as "CONNECTING" | "LIVE" | "OFFLINE");
                    appendLog(`Rust pipe: ${status}`);
                });
                unlisteners.push(unStatus);

                await connectNativeTelemetry(apiBase, token);
                return;
            }
        };

        void setup();
        return () => {
            isDisposed = true;
            for (const fn of unlisteners) fn();
            if (isTauri()) {
                disconnectNativeTelemetry().catch(() => {});
            }
        };
    }, [appendLog, handlePing]);

    const driverList = useMemo(() => Array.from(drivers.values()), [drivers]);
    const missionsByDriver = useMemo(() => {
        const grouped = new Map<string, ActiveMission[]>();
        for (const mission of missions) {
            const existing = grouped.get(mission.route_id);
            if (existing) {
                existing.push(mission);
            } else {
                grouped.set(mission.route_id, [mission]);
            }
        }
        return grouped;
    }, [missions]);
    const selectedDriver = useMemo(() => (
        selectedDriverId ? drivers.get(selectedDriverId) : null
    ), [drivers, selectedDriverId]);
    const selectedInfo = useMemo(() => (
        selectedDriverId ? fleetInfo.get(selectedDriverId) : null
    ), [fleetInfo, selectedDriverId]);
    const selectedMissions = useMemo(() => (
        selectedDriverId ? (missionsByDriver.get(selectedDriverId) ?? []) : []
    ), [missionsByDriver, selectedDriverId]);

    // ── Build route GeoJSON ────────────────────────────────────────────────
    const routeGeoJSON: GeoJSON.FeatureCollection = useMemo(() => ({
        type: "FeatureCollection",
        features: driverList.flatMap((d) => {
            const driverMissions = (missionsByDriver.get(d.driver_id) ?? []).filter(m => m.target_lat !== 0);
            return driverMissions.map((m) => ({
                type: "Feature" as const,
                properties: { driver_id: d.driver_id, order_id: m.order_id, state: m.state },
                geometry: {
                    type: "LineString" as const,
                    coordinates: [
                        [d.longitude, d.latitude],
                        [m.target_lng, m.target_lat],
                    ],
                },
            }));
        }),
    }), [driverList, missionsByDriver]);

    const stopGeoJSON: GeoJSON.FeatureCollection = useMemo(() => ({
        type: "FeatureCollection",
        features: missions
            .filter(m => m.target_lat !== 0 && m.target_lng !== 0)
            .map((m) => ({
                type: "Feature" as const,
                properties: { order_id: m.order_id, state: m.state, route_id: m.route_id },
                geometry: {
                    type: "Point" as const,
                    coordinates: [m.target_lng, m.target_lat],
                },
            })),
    }), [missions]);

    // Helper: get enriched data for hover
    const getDriverSummary = useCallback((d: DriverState) => {
        const info = fleetInfo.get(d.driver_id);
        const driverMissions = missionsByDriver.get(d.driver_id) ?? [];
        const nextStop = driverMissions.find(m => m.state === "EN_ROUTE") ?? driverMissions[0];
        return { info, orderCount: driverMissions.length, nextStop };
    }, [fleetInfo, missionsByDriver]);

    return (
        <div className="min-h-full flex flex-col" style={{ background: 'var(--background)', color: 'var(--foreground)' }}>
            {/* Header */}
            <div className="px-6 md:px-8 py-5 flex justify-between items-center shrink-0" style={{ borderBottom: '1px solid var(--border)' }}>
                <div>
                    <h1 className="md-typescale-title-large">Fleet Management</h1>
                    <p className="md-typescale-label-small mt-1" style={{ color: 'var(--muted)' }}>
                        Real-time Asset Telemetry & Routing
                    </p>
                </div>
                <div className="flex items-center gap-3">
                    <div className="md-chip" style={{ cursor: 'default' }}>
                        <span className="md-typescale-label-small">{missions.length} active orders</span>
                    </div>
                    <div className="md-chip" style={{ cursor: 'default' }}>
                        <div
                            className={`w-2 h-2 rounded-full ${wsStatus === "LIVE" ? "animate-pulse" : ""}`}
                            style={{ background: wsStatus === "LIVE" ? 'var(--accent)' : 'var(--border)' }}
                        />
                        <span className="md-typescale-label-small">
                            {wsStatus === "LIVE" ? `Live · ${driverList.length} units` : wsStatus === "CONNECTING" ? "Syncing..." : "Offline"}
                        </span>
                    </div>
                </div>
            </div>

            {/* Main split */}
            <div className="flex flex-1 overflow-hidden">
                {/* MAP CANVAS */}
                <div className="flex-1 relative" style={{ borderRight: '1px solid var(--border)' }}>
                    <MapGL
                        initialViewState={{ longitude: 69.2401, latitude: 41.2995, zoom: 13 }}
                        style={{ width: "100%", height: "100%" }}
                        mapStyle={MAP_STYLE}
                    >
                        <NavigationControl position="top-right" />

                        {/* Route lines */}
                        <Source id="routes" type="geojson" data={routeGeoJSON}>
                            <Layer
                                id="route-lines"
                                type="line"
                                paint={{
                                    "line-color": "#3B82F6",
                                    "line-width": 2,
                                    "line-opacity": 0.6,
                                    "line-dasharray": [4, 3],
                                }}
                            />
                        </Source>

                        {/* Delivery stop markers */}
                        <Source id="stops" type="geojson" data={stopGeoJSON}>
                            <Layer
                                id="stop-circles"
                                type="circle"
                                paint={{
                                    "circle-radius": 5,
                                    "circle-color": "#DC2626",
                                    "circle-stroke-width": 1.5,
                                    "circle-stroke-color": "#FAFAFA",
                                    "circle-opacity": 0.8,
                                }}
                            />
                        </Source>

                        {/* Driver markers */}
                        {driverList.map((d) => {
                            const staleness = getStaleness(d.last_seen_epoch);
                            const { info, orderCount, nextStop } = getDriverSummary(d);
                            return (
                                <Marker key={d.driver_id} longitude={d.longitude} latitude={d.latitude} anchor="center">
                                    <div className="relative group cursor-pointer" onClick={() => setSelectedDriverId(d.driver_id)}>
                                        {/* Pin */}
                                        <div
                                            className={`w-5 h-5 rounded-full border-2 border-white shadow-lg flex items-center justify-center ${staleness === "live" ? "animate-pulse" : ""}`}
                                            style={{ background: getMarkerColor(staleness) }}
                                        >
                                            <div className="w-1.5 h-1.5 bg-white rounded-full" />
                                        </div>
                                        {/* Rich hover card */}
                                        <div className="absolute bottom-7 left-1/2 -translate-x-1/2 px-4 py-3 md-shape-xs opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none md-elevation-2 whitespace-nowrap space-y-1 z-20 min-w-[220px]" style={{ background: 'var(--foreground)', color: 'var(--background)' }}>
                                            <div className="md-typescale-label-medium font-medium">
                                                {info?.name ?? d.driver_id}
                                            </div>
                                            {info && (
                                                <div className="md-typescale-label-small" style={{ opacity: 0.85 }}>
                                                    {info.vehicle_type} · {info.license_plate}
                                                </div>
                                            )}
                                            <div className="md-typescale-label-small" style={{ opacity: 0.8 }}>
                                                Route: {d.driver_id} · {orderCount} order{orderCount !== 1 ? "s" : ""}
                                            </div>
                                            {nextStop && (
                                                <div className="md-typescale-label-small" style={{ opacity: 0.8 }}>
                                                    Next: {nextStop.order_id} ({nextStop.state})
                                                </div>
                                            )}
                                            <div className="md-typescale-label-small" style={{ opacity: 0.7 }}>
                                                Updated: {d.last_seen} · {d.ping_count} pings
                                            </div>
                                            {info?.truck_status === "RETURNING" && info.estimated_return_at && (
                                                <div className="md-typescale-label-small font-medium" style={{ color: 'var(--warning)' }}>
                                                    Returning · ETA {formatETATime(info.estimated_return_at)}
                                                </div>
                                            )}
                                            {staleness === "stale" && (
                                                <div className="md-typescale-label-small font-medium" style={{ color: 'var(--danger)' }}>
                                                    STALE ({Math.round((Date.now() - d.last_seen_epoch) / 1000)}s)
                                                </div>
                                            )}
                                            {staleness === "amber" && (
                                                <div className="md-typescale-label-small font-medium" style={{ color: 'var(--warning)' }}>
                                                    DELAYED ({Math.round((Date.now() - d.last_seen_epoch) / 1000)}s)
                                                </div>
                                            )}
                                        </div>
                                    </div>
                                </Marker>
                            );
                        })}
                    </MapGL>

                    {/* Empty state overlay */}
                    {driverList.length === 0 && (
                        <div className="absolute inset-0 flex items-center justify-center pointer-events-none backdrop-blur-[1px]" style={{ background: 'var(--backdrop)' }}>
                            <div className="px-8 py-5 md-card md-card-elevated flex flex-col items-center">
                                <div className="w-5 h-5 border-2 border-t-transparent rounded-full animate-spin" style={{ borderColor: 'var(--accent)', borderTopColor: 'transparent' }} />
                                <p className="md-typescale-label-small mt-4" style={{ color: 'var(--muted)' }}>
                                    Awaiting First Ping
                                </p>
                            </div>
                        </div>
                    )}
                </div>

                {/* RIGHT PANEL */}
                <div className="w-96 flex flex-col shrink-0" style={{ borderLeft: '1px solid var(--border)', background: 'var(--surface)' }}>
                    {/* Detail Drawer — when a driver is selected */}
                    {selectedDriver ? (
                        <div className="flex-1 overflow-y-auto">
                            <div className="px-5 py-4 flex justify-between items-center" style={{ borderBottom: '1px solid var(--border)' }}>
                                <p className="md-typescale-title-small">Driver Detail</p>
                                <button
                                    onClick={() => setSelectedDriverId(null)}
                                    className="md-typescale-label-small px-3 py-1 md-shape-xs"
                                    style={{ background: 'var(--surface)', color: 'var(--muted)' }}
                                >
                                    Close
                                </button>
                            </div>
                            <div className="p-5 space-y-4">
                                {/* Identity */}
                                <div className="md-card md-card-elevated p-4 space-y-2">
                                    <p className="md-typescale-title-medium font-medium">
                                        {selectedInfo?.name ?? selectedDriver.driver_id}
                                    </p>
                                    <div className="grid grid-cols-2 gap-2">
                                        <div>
                                            <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Driver ID</p>
                                            <p className="md-typescale-body-small font-mono">{selectedDriver.driver_id}</p>
                                        </div>
                                        <div>
                                            <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Status</p>
                                            {(() => {
                                                const st = getTruckStatusStyle(selectedInfo?.truck_status ?? "Unknown");
                                                return (
                                                    <span className="md-chip md-typescale-label-small inline-flex items-center" style={{
                                                        height: 22, padding: '0 8px', cursor: 'default',
                                                        background: st.bg, color: st.fg,
                                                    }}>
                                                        {st.label}
                                                    </span>
                                                );
                                            })()}
                                        </div>
                                        {selectedInfo && (
                                            <>
                                                <div>
                                                    <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Vehicle</p>
                                                    <p className="md-typescale-body-small">{selectedInfo.vehicle_type}</p>
                                                </div>
                                                <div>
                                                    <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Plate</p>
                                                    <p className="md-typescale-body-small">{selectedInfo.license_plate}</p>
                                                </div>
                                                <div>
                                                    <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Phone</p>
                                                    <p className="md-typescale-body-small">{selectedInfo.phone}</p>
                                                </div>
                                            </>
                                        )}
                                        <div>
                                            <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Position</p>
                                            <p className="md-typescale-body-small font-mono">{selectedDriver.latitude.toFixed(5)}, {selectedDriver.longitude.toFixed(5)}</p>
                                        </div>
                                    </div>
                                    <p className="md-typescale-label-small" style={{ color: 'var(--border)' }}>
                                        Last ping: {selectedDriver.last_seen} · {selectedDriver.ping_count} total
                                    </p>
                                    {/* Return ETA for RETURNING drivers */}
                                    {selectedInfo?.truck_status === "RETURNING" && selectedInfo.estimated_return_at && (
                                        <div className="mt-2 p-2 md-shape-xs" style={{ background: 'color-mix(in srgb, var(--warning) 12%, transparent)' }}>
                                            <p className="md-typescale-label-small" style={{ color: 'var(--warning)' }}>
                                                Returning to warehouse · ETA {formatETATime(selectedInfo.estimated_return_at)}
                                                {selectedInfo.return_duration_sec != null && ` (${formatDuration(selectedInfo.return_duration_sec)})`}
                                            </p>
                                        </div>
                                    )}
                                </div>

                                {/* Assigned Orders */}
                                <div>
                                    <p className="md-typescale-label-small mb-2" style={{ color: 'var(--muted)' }}>
                                        Assigned Orders ({selectedMissions.length})
                                    </p>
                                    {selectedMissions.length === 0 ? (
                                        <p className="md-typescale-body-small" style={{ color: 'var(--border)' }}>No active orders</p>
                                    ) : (
                                        <div className="space-y-2">
                                            {selectedMissions.map(m => (
                                                <div key={m.order_id} className="md-card md-card-outlined p-3">
                                                    <div className="flex justify-between items-center">
                                                        <span className="md-typescale-body-small font-mono">{m.order_id}</span>
                                                        <span className="md-chip md-typescale-label-small" style={{
                                                            height: 22, padding: '0 8px', cursor: 'default',
                                                            background: m.state === 'EN_ROUTE' ? 'var(--accent)' : 'var(--surface)',
                                                            color: m.state === 'EN_ROUTE' ? 'var(--accent-foreground)' : 'var(--foreground)',
                                                        }}>
                                                            {m.state}
                                                        </span>
                                                    </div>
                                                    <div className="flex justify-between mt-1">
                                                        <span className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
                                                            {m.amount.toLocaleString()}
                                                        </span>
                                                        <span className="md-typescale-label-small" style={{ color: 'var(--border)' }}>
                                                            {m.gateway}
                                                        </span>
                                                    </div>
                                                    {m.estimated_arrival_at && (
                                                        <p className="md-typescale-label-small mt-1" style={{ color: 'var(--accent)' }}>
                                                            ETA {formatETATime(m.estimated_arrival_at)}
                                                        </p>
                                                    )}
                                                </div>
                                            ))}
                                        </div>
                                    )}
                                </div>
                            </div>
                        </div>
                    ) : (
                        /* Default: Driver Roster */
                        <div className="flex-1 overflow-hidden flex flex-col">
                            <div className="px-5 py-4" style={{ borderBottom: '1px solid var(--border)' }}>
                                <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
                                    Active Inventory · Click a marker to inspect
                                </p>
                            </div>
                            {driverList.length === 0 ? (
                                <div className="px-4 py-8 text-center">
                                    <p className="md-typescale-body-small" style={{ color: 'var(--border)' }}>
                                        No assets in transit
                                    </p>
                                </div>
                            ) : (
                                <div className="flex-1 overflow-y-auto px-2 py-2">
                                    {driverList.map((d) => {
                                        const staleness = getStaleness(d.last_seen_epoch);
                                        const info = fleetInfo.get(d.driver_id);
                                        const orderCount = missions.filter(m => m.route_id === d.driver_id).length;
                                        return (
                                            <div
                                                key={d.driver_id}
                                                className="md-card md-card-elevated p-4 mb-2 cursor-pointer transition-colors"
                                                onClick={() => setSelectedDriverId(d.driver_id)}
                                                style={{ borderLeft: `3px solid ${getMarkerColor(staleness)}` }}
                                            >
                                                <div className="flex justify-between items-center mb-1">
                                                    <p className="md-typescale-body-medium font-medium">
                                                        {info?.name ?? d.driver_id}
                                                    </p>
                                                    <span className="md-chip md-typescale-label-small" style={{ height: 22, padding: '0 8px', cursor: 'default' }}>
                                                        {orderCount} order{orderCount !== 1 ? "s" : ""}
                                                    </span>
                                                </div>
                                                {info && (
                                                    <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
                                                        {info.vehicle_type} · {info.license_plate} · {(() => {
                                                            const st = getTruckStatusStyle(info.truck_status);
                                                            return <span style={{ color: info.truck_status === "RETURNING" ? "var(--warning)" : info.truck_status === "IN_TRANSIT" ? "var(--accent)" : undefined }}>{st.label}</span>;
                                                        })()}
                                                    </p>
                                                )}
                                                {info?.truck_status === "RETURNING" && info.estimated_return_at && (
                                                    <p className="md-typescale-label-small" style={{ color: 'var(--warning)' }}>
                                                        Return ETA {formatETATime(info.estimated_return_at)}
                                                        {info.return_duration_sec != null && ` (${formatDuration(info.return_duration_sec)})`}
                                                    </p>
                                                )}
                                                <p className="md-typescale-label-small mt-1" style={{ color: 'var(--border)' }}>
                                                    Last: {d.last_seen}
                                                    {staleness !== "live" && (
                                                        <span style={{ color: staleness === "stale" ? 'var(--danger)' : 'var(--warning)', fontWeight: 500 }}>
                                                            {" "}· {staleness === "stale" ? "STALE" : "DELAYED"}
                                                        </span>
                                                    )}
                                                </p>
                                            </div>
                                        );
                                    })}
                                </div>
                            )}
                        </div>
                    )}

                    {/* Raw Log */}
                    <div className="h-48 flex flex-col" style={{ borderTop: '1px solid var(--border)', background: 'var(--background)' }}>
                        <div className="px-5 py-3" style={{ borderBottom: '1px solid var(--border)' }}>
                            <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
                                Telemetry Log
                            </p>
                        </div>
                        <div className="flex-1 overflow-y-auto p-4 flex flex-col gap-1.5">
                            {log.map((entry, i) => (
                                <p key={i} className="md-typescale-label-small leading-4" style={{ color: 'var(--muted)' }}>
                                    <span className="mr-1.5" style={{ color: 'var(--border)' }}>·</span>
                                    {entry}
                                </p>
                            ))}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
}
