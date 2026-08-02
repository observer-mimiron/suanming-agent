// Graph rendering helpers — pure data shapes, palettes, sizing, and canvas
// drawing primitives extracted from the wiki graph page so the React component
// can stay focused on hooks/event handlers and so this logic is testable in
// isolation. This module is environment-neutral: functions that touch
// `window`/`document` (like getColorPalette) only do so when invoked, so the
// module itself can be imported from either client or server code.

// --- Data shapes ---

export interface GraphNode {
  id: string;
  label: string;
  /** Canonical tenant for navigating to `/u/<tenant>/<slug>` on click. */
  tenant: string;
  linkCount: number;
  tags: string[];
  cluster: number;
  x: number;
  y: number;
  vx: number;
  vy: number;
}

export interface GraphEdge {
  source: string;
  target: string;
}

export interface GraphData {
  nodes: GraphNode[];
  edges: GraphEdge[];
}

// --- Camera / viewport ---

export interface Camera {
  scale: number;
  offsetX: number;
  offsetY: number;
}

export function worldToScreen(wx: number, wy: number, c: Camera) {
  return { sx: wx * c.scale + c.offsetX, sy: wy * c.scale + c.offsetY };
}

export function screenToWorld(sx: number, sy: number, c: Camera) {
  return { wx: (sx - c.offsetX) / c.scale, wy: (sy - c.offsetY) / c.scale };
}

/** Fit camera so all nodes are visible with padding. */
export function fitCameraToNodes(
  nodes: GraphNode[],
  canvasW: number,
  canvasH: number,
  camera: Camera,
  padding = 48,
  maxScale = 2.2,
) {
  if (nodes.length === 0) return;

  let minX = Infinity, maxX = -Infinity, minY = Infinity, maxY = -Infinity;
  for (const n of nodes) {
    if (n.x < minX) minX = n.x;
    if (n.x > maxX) maxX = n.x;
    if (n.y < minY) minY = n.y;
    if (n.y > maxY) maxY = n.y;
  }

  const worldW = maxX - minX || 1;
  const worldH = maxY - minY || 1;
  const availW = Math.max(100, canvasW - padding * 2);
  const availH = Math.max(100, canvasH - padding * 2);

  const fitScale = Math.min(availW / worldW, availH / worldH, maxScale);
  camera.scale = Math.max(0.35, fitScale);
  camera.offsetX = (canvasW - (minX + maxX) * camera.scale) / 2;
  camera.offsetY = (canvasH - (minY + maxY) * camera.scale) / 2;
}

// --- Color palettes ---

export interface ColorPalette {
  bg: string;
  edge: string;
  node: string;
  nodeStroke: string;
  label: string;
  tooltip: string;
  tooltipBg: string;
}

export const DARK_PALETTE: ColorPalette = {
  bg: "#0a0a0a",
  edge: "#4a5568",
  node: "#60a5fa",
  nodeStroke: "#93c5fd",
  label: "#e2e8f0",
  tooltip: "#f1f5f9",
  tooltipBg: "rgba(30, 41, 59, 0.92)",
};

export const LIGHT_PALETTE: ColorPalette = {
  bg: "#ffffff",
  edge: "#cbd5e1",
  node: "#3b82f6",
  nodeStroke: "#2563eb",
  label: "#1e293b",
  tooltip: "#1e293b",
  tooltipBg: "rgba(248, 250, 252, 0.92)",
};

export function getColorPalette(): ColorPalette {
  if (typeof window === "undefined") return DARK_PALETTE;
  if (typeof document !== "undefined") {
    const html = document.documentElement;
    if (html.classList.contains("dark")) return DARK_PALETTE;
    if (html.classList.contains("light")) return LIGHT_PALETTE;
  }
  const isDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
  return isDark ? DARK_PALETTE : LIGHT_PALETTE;
}

// Cluster colors: distinct hues for book-group coloring
export const CLUSTER_COLORS_DARK = [
  "#60a5fa", // blue
  "#34d399", // emerald
  "#fbbf24", // amber
  "#f87171", // rose
  "#a78bfa", // violet
  "#22d3ee", // cyan
  "#fb923c", // orange
  "#a3e635", // lime
  "#f472b6", // pink
  "#2dd4bf", // teal
];

export const CLUSTER_COLORS_LIGHT = [
  "#3b82f6", // blue
  "#10b981", // emerald
  "#f59e0b", // amber
  "#ef4444", // rose
  "#8b5cf6", // violet
  "#06b6d4", // cyan
  "#f97316", // orange
  "#84cc16", // lime
  "#ec4899", // pink
  "#14b8a6", // teal
];

export const CLUSTER_STROKES_DARK = [
  "#93c5fd", "#6ee7b7", "#fcd34d", "#fca5a5", "#c4b5fd",
  "#67e8f9", "#fdba74", "#bef264", "#f9a8d4", "#5eead4",
];

export const CLUSTER_STROKES_LIGHT = [
  "#2563eb", "#059669", "#d97706", "#dc2626", "#7c3aed",
  "#0891b2", "#ea580c", "#65a30d", "#db2777", "#0d9488",
];

// --- Physics constants ---

export const REPULSION = 2200;
export const ATTRACTION = 0.008;
export const CENTER_GRAVITY = 0.004;
export const DAMPING = 0.86;
export const VELOCITY_THRESHOLD = 1.0;
const CELL_SIZE = 180;
const MAX_REPULSION_DIST = 260;
const MAX_REPULSION_DIST_SQ = MAX_REPULSION_DIST * MAX_REPULSION_DIST;
const MAX_SPEED = 7;
const FAR_REPULSION_FACTOR = 0.35;

// --- Node sizing constants ---

export const BASE_RADIUS = 6;
export const RADIUS_SCALE = 4;
export const MIN_RADIUS = 6;
export const MAX_RADIUS = 24;

export function nodeRadius(linkCount: number): number {
  const r = BASE_RADIUS + Math.sqrt(linkCount) * RADIUS_SCALE;
  return Math.max(MIN_RADIUS, Math.min(MAX_RADIUS, r));
}

// --- Physics simulation ---

export interface PhysicsResult {
  totalVelocity: number;
}

/**
 * Run one step of the force-directed physics simulation.
 * Mutates node positions (x, y) and velocities (vx, vy) in-place.
 */
export function stepPhysics(
  nodes: GraphNode[],
  edges: GraphEdge[],
  nodeMap: Map<string, GraphNode>,
  cx: number,
  cy: number,
): PhysicsResult {
  type Bucket = { nodes: GraphNode[]; sumX: number; sumY: number; count: number };
  const grid = new Map<string, Bucket>();
  for (const node of nodes) {
    const gx = Math.floor(node.x / CELL_SIZE);
    const gy = Math.floor(node.y / CELL_SIZE);
    const key = `${gx},${gy}`;
    let bucket = grid.get(key);
    if (!bucket) { bucket = { nodes: [], sumX: 0, sumY: 0, count: 0 }; grid.set(key, bucket); }
    bucket.nodes.push(node);
    bucket.sumX += node.x;
    bucket.sumY += node.y;
    bucket.count += 1;
  }

  for (const a of nodes) {
    const gx = Math.floor(a.x / CELL_SIZE);
    const gy = Math.floor(a.y / CELL_SIZE);

    // Near-field: precise repulsion from neighbor cells
    for (let ox = -1; ox <= 1; ox++) {
      for (let oy = -1; oy <= 1; oy++) {
        const bucket = grid.get(`${gx + ox},${gy + oy}`);
        if (!bucket) continue;
        for (const b of bucket.nodes) {
          if (a === b) continue;
          const dx = a.x - b.x;
          const dy = a.y - b.y;
          const distSq = dx * dx + dy * dy || 1;
          if (distSq > MAX_REPULSION_DIST_SQ) continue;
          const dist = Math.sqrt(distSq);
          const force = REPULSION / distSq;
          a.vx += (dx / dist) * force;
          a.vy += (dy / dist) * force;
        }
      }
    }

    // Far-field: aggregate repulsion from distant bucket centroids
    for (const [key, bucket] of grid) {
      const [bx, by] = key.split(",").map(Number);
      if (Math.abs(bx - gx) <= 1 && Math.abs(by - gy) <= 1) continue;
      const centerX = bucket.sumX / bucket.count;
      const centerY = bucket.sumY / bucket.count;
      const dx = a.x - centerX;
      const dy = a.y - centerY;
      const distSq = dx * dx + dy * dy || 1;
      const dist = Math.sqrt(distSq);
      const force = (REPULSION * FAR_REPULSION_FACTOR * bucket.count) / distSq;
      a.vx += (dx / dist) * force;
      a.vy += (dy / dist) * force;
    }
  }

  // Attraction along edges
  for (const edge of edges) {
    const a = nodeMap.get(edge.source);
    const b = nodeMap.get(edge.target);
    if (!a || !b) continue;
    const dx = b.x - a.x;
    const dy = b.y - a.y;
    const force = ATTRACTION * Math.sqrt(dx * dx + dy * dy);
    const dist = Math.sqrt(dx * dx + dy * dy) || 1;
    a.vx += (dx / dist) * force;
    a.vy += (dy / dist) * force;
    b.vx -= (dx / dist) * force;
    b.vy -= (dy / dist) * force;
  }

  // Center gravity + damping + apply
  let totalVelocity = 0;
  for (const n of nodes) {
    n.vx += (cx - n.x) * CENTER_GRAVITY;
    n.vy += (cy - n.y) * CENTER_GRAVITY;
    n.vx *= DAMPING;
    n.vy *= DAMPING;
    const speed = Math.hypot(n.vx, n.vy);
    if (speed > MAX_SPEED) {
      n.vx = (n.vx / speed) * MAX_SPEED;
      n.vy = (n.vy / speed) * MAX_SPEED;
    }
    n.x += n.vx;
    n.y += n.vy;
    totalVelocity += Math.abs(n.vx) + Math.abs(n.vy);
  }

  return { totalVelocity };
}

// --- Canvas rendering ---

export interface RenderOptions {
  nodes: GraphNode[];
  edges: GraphEdge[];
  nodeMap: Map<string, GraphNode>;
  ctx: CanvasRenderingContext2D;
  width: number;
  height: number;
  palette: ColorPalette;
  hovered: GraphNode | null;
  mouse: { x: number; y: number };
  clusterCount: number;
  camera: Camera;
}

/**
 * Render the full graph scene through the camera: background, edges, nodes with
 * cluster colors, labels, hover tooltip, and cluster legend.
 *
 * Node positions are in world coordinates and transformed to screen coordinates
 * via the camera. The tooltip and legend remain in screen space.
 */
export function renderGraph(opts: RenderOptions): void {
  const {
    nodes, edges, nodeMap, ctx,
    width: W, height: H,
    palette, hovered, mouse, clusterCount,
    camera,
  } = opts;

  const { scale } = camera;
  const hoveredId = hovered?.id ?? null;
  const connectedToHovered = new Set<string>();

  if (hoveredId) {
    for (const edge of edges) {
      if (edge.source === hoveredId) connectedToHovered.add(edge.target);
      if (edge.target === hoveredId) connectedToHovered.add(edge.source);
    }
  }

  // Clear + fill background
  ctx.clearRect(0, 0, W, H);
  ctx.fillStyle = palette.bg;
  ctx.fillRect(0, 0, W, H);

  // Edges — transform both endpoints to screen space
  const scaledEdgeWidth = (lc: number) => Math.min(0.5 + lc * 0.15, 3) * Math.max(0.5, Math.min(scale, 2));
  for (const edge of edges) {
    const a = nodeMap.get(edge.source);
    const b = nodeMap.get(edge.target);
    if (!a || !b) continue;
    const combinedLinks = (a.linkCount + b.linkCount) / 2;
    const sa = worldToScreen(a.x, a.y, camera);
    const sb = worldToScreen(b.x, b.y, camera);
    const isHoveredEdge =
      hoveredId !== null && (edge.source === hoveredId || edge.target === hoveredId);
    const isNeighborEdge =
      hoveredId !== null && !isHoveredEdge
        && (connectedToHovered.has(edge.source) || connectedToHovered.has(edge.target));

    ctx.strokeStyle = palette.edge;
    ctx.lineWidth = isHoveredEdge
      ? scaledEdgeWidth(combinedLinks) * 1.35
      : isNeighborEdge
        ? scaledEdgeWidth(combinedLinks) * 0.95
        : scaledEdgeWidth(combinedLinks) * 0.75;
    ctx.globalAlpha = hoveredId === null
      ? 0.14
      : isHoveredEdge
        ? 0.72
        : isNeighborEdge
          ? 0.22
          : 0.05;
    ctx.beginPath();
    ctx.moveTo(sa.sx, sa.sy);
    ctx.lineTo(sb.sx, sb.sy);
    ctx.stroke();
  }
  ctx.globalAlpha = 1;

  // Nodes — transform positions and scale radii
  const isDark = palette === DARK_PALETTE;
  const clusterFills = isDark ? CLUSTER_COLORS_DARK : CLUSTER_COLORS_LIGHT;
  const clusterStrokes = isDark ? CLUSTER_STROKES_DARK : CLUSTER_STROKES_LIGHT;
  const scaledFontSize = Math.max(8, Math.min(16, 12 * Math.max(0.6, Math.min(scale, 2))));

  ctx.font = `${scaledFontSize}px sans-serif`;
  ctx.textAlign = "center";

  for (const n of nodes) {
    const r = nodeRadius(n.linkCount) * Math.max(0.5, scale);
    const { sx, sy } = worldToScreen(n.x, n.y, camera);
    const colorIdx = n.cluster % clusterFills.length;
    const isHoveredNode = hoveredId === n.id;
    const isNeighborNode = hoveredId !== null && connectedToHovered.has(n.id);
    const isMajorNode = n.linkCount >= 3;
    const showLabel = scale >= 1.1
      || isHoveredNode
      || (scale >= 0.7 && (isMajorNode || isNeighborNode))
      || (scale < 0.7 && isMajorNode);

    ctx.beginPath();
    ctx.arc(sx, sy, r, 0, Math.PI * 2);
    ctx.fillStyle = clusterFills[colorIdx];
    ctx.globalAlpha = hoveredId === null
      ? 0.96
      : isHoveredNode
        ? 1
        : isNeighborNode
          ? 0.98
          : 0.88;
    ctx.fill();
    ctx.strokeStyle = clusterStrokes[colorIdx];
    ctx.lineWidth = (isHoveredNode ? 2.5 : 1.5) * Math.max(0.6, Math.min(scale, 2));
    ctx.stroke();
    ctx.globalAlpha = 1;

    // Label
    if (showLabel) {
      ctx.fillStyle = palette.label;
      ctx.globalAlpha = isHoveredNode ? 1 : isNeighborNode ? 0.92 : isMajorNode ? 0.82 : 0.68;
      ctx.fillText(n.label, sx, sy - r - 4);
      ctx.globalAlpha = 1;
    }
  }

  // Tooltip for hovered node — screen space
  if (hovered) {
    const mx = mouse.x;
    const my = mouse.y;
    const connText = `${hovered.linkCount} connection${hovered.linkCount !== 1 ? "s" : ""}`;
    const tooltipText = `${hovered.label} — ${connText}`;
    ctx.font = "13px sans-serif";
    const metrics = ctx.measureText(tooltipText);
    const tipW = metrics.width + 16;
    const tipH = 28;
    let tipX = mx + 14;
    let tipY = my - tipH - 6;
    if (tipX + tipW > W) tipX = mx - tipW - 6;
    if (tipY < 0) tipY = my + 20;

    ctx.fillStyle = palette.tooltipBg;
    roundedRect(ctx, tipX, tipY, tipW, tipH, 5);
    ctx.fill();

    ctx.strokeStyle = palette.edge;
    ctx.lineWidth = 1;
    ctx.stroke();

    ctx.fillStyle = palette.tooltip;
    ctx.font = "13px sans-serif";
    ctx.textAlign = "left";
    ctx.fillText(tooltipText, tipX + 8, tipY + 18);
  }

  // Cluster legend — screen space, bottom-left corner
  if (clusterCount > 1) {
    const legendX = 12;
    const legendItemH = 18;
    const legendPad = 8;
    const displayCount = Math.min(clusterCount, clusterFills.length);
    const legendH = displayCount * legendItemH + legendPad * 2;
    const legendW = 110;
    const legendY = H - legendH - 12;

    const clusterSizes = new Map<number, number>();
    for (const n of nodes) {
      clusterSizes.set(n.cluster, (clusterSizes.get(n.cluster) ?? 0) + 1);
    }

    ctx.fillStyle = palette.tooltipBg;
    roundedRect(ctx, legendX, legendY, legendW, legendH, 5);
    ctx.fill();
    ctx.strokeStyle = palette.edge;
    ctx.lineWidth = 1;
    ctx.stroke();

    for (let i = 0; i < displayCount; i++) {
      const y = legendY + legendPad + i * legendItemH + 12;
      const colorIdx = i % clusterFills.length;
      const size = clusterSizes.get(i) ?? 0;

      ctx.beginPath();
      ctx.arc(legendX + 16, y - 4, 5, 0, Math.PI * 2);
      ctx.fillStyle = clusterFills[colorIdx];
      ctx.fill();

      ctx.fillStyle = palette.label;
      ctx.font = "11px sans-serif";
      ctx.textAlign = "left";
      ctx.fillText(`Cluster ${i + 1} (${size})`, legendX + 26, y);
    }
  }
}

// --- Canvas drawing primitives ---

export function roundedRect(
  ctx: CanvasRenderingContext2D,
  x: number, y: number, w: number, h: number, r: number,
) {
  if (typeof ctx.roundRect === "function") {
    ctx.beginPath();
    ctx.roundRect(x, y, w, h, r);
  } else {
    ctx.beginPath();
    ctx.moveTo(x + r, y);
    ctx.arcTo(x + w, y, x + w, y + h, r);
    ctx.arcTo(x + w, y + h, x, y + h, r);
    ctx.arcTo(x, y + h, x, y, r);
    ctx.arcTo(x, y, x + w, y, r);
    ctx.closePath();
  }
}
