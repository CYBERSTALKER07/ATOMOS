"use client";

import { useMemo } from "react";
import {
  extractObjectPrimaryLabel,
  type OntologyGraphEdge,
  type OntologyNeighborResponse,
  type OntologyObject,
} from "@/lib/ontology-api";

interface PositionedNode {
  key: string;
  object: OntologyObject;
  hop: number;
  x: number;
  y: number;
  viaLinkType?: string;
  direction?: string;
}

function nodeKey(objectType: string, objectId: string): string {
  return `${objectType}:${objectId}`;
}

export function OntologyTopologyCanvas({
  neighbors,
  inspectedObject,
  onSelectObject,
  onTraverseObject,
}: {
  neighbors: OntologyNeighborResponse;
  inspectedObject: OntologyObject | null;
  onSelectObject: (obj: OntologyObject) => void;
  onTraverseObject: (obj: OntologyObject) => void;
}) {
  const layout = useMemo(() => {
    const byHop = new Map<number, PositionedNode[]>();
    const root = neighbors.root;
    const nodeWidth = 176;
    const nodeHeight = 60;
    const colGap = 230;
    const rowGap = 82;
    const padX = 24;
    const padY = 36;

    const rootNode: PositionedNode = {
      key: nodeKey(root.object_type, root.object_id),
      object: root,
      hop: 0,
      x: padX,
      y: padY,
    };
    byHop.set(0, [rootNode]);

    const seen = new Set<string>([rootNode.key]);
    for (const item of neighbors.nodes) {
      const k = nodeKey(item.object.object_type, item.object.object_id);
      if (seen.has(k)) continue;
      seen.add(k);
      const hop = Math.max(1, item.hop);
      const list = byHop.get(hop) ?? [];
      list.push({
        key: k,
        object: item.object,
        hop,
        x: padX + hop * colGap,
        y: padY + list.length * rowGap,
        viaLinkType: item.via_link_type,
        direction: item.direction,
      });
      byHop.set(hop, list);
    }

    const hops = [...byHop.keys()].sort((a, b) => a - b);
    let maxRows = 1;
    for (const hop of hops) {
      maxRows = Math.max(maxRows, byHop.get(hop)?.length ?? 1);
    }

    const canvasHeight = Math.max(240, padY * 2 + (maxRows - 1) * rowGap + nodeHeight);
    const maxHop = hops[hops.length - 1] ?? 0;
    const canvasWidth = Math.max(520, padX * 2 + maxHop * colGap + nodeWidth);

    const nodes: PositionedNode[] = [];
    const nodeMap = new Map<string, PositionedNode>();
    for (const hop of hops) {
      const group = byHop.get(hop) ?? [];
      const totalGroupHeight = (group.length - 1) * rowGap + nodeHeight;
      const startY = Math.max(padY, (canvasHeight - totalGroupHeight) / 2);
      group.forEach((n, idx) => {
        const positioned: PositionedNode = {
          ...n,
          y: startY + idx * rowGap,
        };
        nodes.push(positioned);
        nodeMap.set(positioned.key, positioned);
      });
    }

    const edges = (neighbors.edges ?? [])
      .map((edge, idx) => {
        const src = nodeMap.get(nodeKey(edge.source_object_type, edge.source_object_id));
        const dst = nodeMap.get(nodeKey(edge.target_object_type, edge.target_object_id));
        if (!src || !dst) return null;
        return {
          id: `${edge.source_object_type}:${edge.source_object_id}-${edge.link_type}-${edge.target_object_type}:${edge.target_object_id}-${idx}`,
          edge,
          src,
          dst,
        };
      })
      .filter(
        (
          item,
        ): item is {
          id: string;
          edge: OntologyGraphEdge;
          src: PositionedNode;
          dst: PositionedNode;
        } => item !== null,
      );

    return {
      nodes,
      edges,
      hops,
      nodeWidth,
      nodeHeight,
      canvasWidth,
      canvasHeight,
    };
  }, [neighbors]);

  const inspectedKey = inspectedObject
    ? nodeKey(inspectedObject.object_type, inspectedObject.object_id)
    : "";

  return (
    <div
      className="overflow-x-auto rounded-xl border p-3"
      style={{ borderColor: "var(--desk-border)", background: "var(--desk-canvas)" }}
    >
      <svg
        role="img"
        aria-label={`Ontology neighborhood topology for ${neighbors.root.object_type} ${neighbors.root.object_id}`}
        viewBox={`0 0 ${layout.canvasWidth} ${layout.canvasHeight}`}
        style={{ width: "100%", minWidth: layout.canvasWidth, height: layout.canvasHeight }}
      >
        <defs>
          <marker
            id="ontology-arrow"
            markerWidth="8"
            markerHeight="8"
            refX="7"
            refY="4"
            orient="auto"
          >
            <path d="M0,1 L7,4 L0,7 z" fill="currentColor" opacity="0.55" />
          </marker>
        </defs>

        {layout.hops.map((hop) => (
          <text
            key={`hop-col-${hop}`}
            x={24 + hop * 230}
            y={20}
            fontSize="11"
            fontWeight="600"
            fill="var(--desk-text-tertiary)"
          >
            {hop === 0 ? "HOP 0 · FOCUS" : `HOP ${hop}`}
          </text>
        ))}

        {layout.edges.map(({ id, edge, src, dst }) => {
          const sx = src.x + layout.nodeWidth / 2;
          const sy = src.y + layout.nodeHeight / 2;
          const tx = dst.x + layout.nodeWidth / 2;
          const ty = dst.y + layout.nodeHeight / 2;
          const mx = (sx + tx) / 2;
          const my = (sy + ty) / 2 - 12;
          const d = `M ${sx} ${sy} Q ${mx} ${my} ${tx} ${ty}`;
          return (
            <g key={id} style={{ color: "var(--desk-text-secondary)" }}>
              <path
                d={d}
                fill="none"
                stroke="var(--desk-border-strong, var(--desk-border))"
                strokeWidth="1.5"
                markerEnd="url(#ontology-arrow)"
              />
              <text
                x={mx}
                y={my}
                textAnchor="middle"
                fontSize="10"
                fontFamily="monospace"
                fill="var(--desk-text-secondary)"
              >
                {edge.link_type}
              </text>
            </g>
          );
        })}

        {layout.nodes.map((node) => {
          const isRoot = node.hop === 0;
          const isInspected = node.key === inspectedKey;
          const hasMarking = (node.object.security_markings?.length ?? 0) > 0;
          const primaryLabel = extractObjectPrimaryLabel(node.object);
          return (
            <g
              key={node.key}
              transform={`translate(${node.x}, ${node.y})`}
              onClick={() => onSelectObject(node.object)}
              onDoubleClick={() => onTraverseObject(node.object)}
              style={{ cursor: "pointer" }}
            >
              <rect
                width={layout.nodeWidth}
                height={layout.nodeHeight}
                rx={10}
                fill={
                  isRoot
                    ? "color-mix(in srgb, var(--desk-accent) 14%, var(--desk-surface))"
                    : "var(--desk-surface)"
                }
                stroke={
                  isInspected || isRoot
                    ? "var(--desk-accent)"
                    : hasMarking
                      ? "var(--desk-warning, #d97706)"
                      : "var(--desk-border)"
                }
                strokeWidth={isInspected || isRoot ? 2 : 1}
              />
              <text
                x={10}
                y={18}
                fontSize="10"
                fontFamily="monospace"
                fontWeight="700"
                fill="var(--desk-text-secondary)"
              >
                {node.object.object_type}
                {hasMarking ? " · !" : ""}
              </text>
              <text
                x={10}
                y={35}
                fontSize="12"
                fontWeight="600"
                fill="var(--desk-text-primary)"
              >
                {primaryLabel.length > 21 ? `${primaryLabel.slice(0, 20)}…` : primaryLabel}
              </text>
              <text
                x={10}
                y={50}
                fontSize="10"
                fontFamily="monospace"
                fill="var(--desk-text-tertiary)"
              >
                {node.object.object_id.length > 22
                  ? `${node.object.object_id.slice(0, 21)}…`
                  : node.object.object_id}
              </text>
            </g>
          );
        })}
      </svg>
    </div>
  );
}
