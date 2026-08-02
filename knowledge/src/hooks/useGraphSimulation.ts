"use client";

import {type RefObject, useCallback, useEffect, useRef, useState} from "react";
import type {AppRouterInstance} from "next/dist/shared/lib/app-router-context.shared-runtime";
import {
    type Camera,
    type ColorPalette,
    DARK_PALETTE,
    fitCameraToNodes,
    getColorPalette,
    type GraphData,
    type GraphEdge,
    type GraphNode,
    nodeRadius,
    renderGraph,
    screenToWorld,
    stepPhysics,
    VELOCITY_THRESHOLD,
} from "@/lib/graph-render";
import {GRAPH_CANVAS_HEIGHT, GRAPH_MAX_SCALE, GRAPH_MIN_SCALE} from "@/lib/constants";

export interface UseGraphSimulationReturn {
  loading: boolean;
  empty: boolean;
  fetchError: string | null;
  canvasBg: string;
  onPointerDown: (e: React.PointerEvent<HTMLCanvasElement>) => void;
  onPointerMove: (e: React.PointerEvent<HTMLCanvasElement>) => void;
  onPointerUp: (e: React.PointerEvent<HTMLCanvasElement>) => void;
  onPointerLeave: () => void;
  fitToBounds: () => void;
}

interface CachedNodePosition {
  id: string;
  x: number;
  y: number;
}

function isCachedNodePositions(value: unknown): value is CachedNodePosition[] {
  if (!Array.isArray(value)) return false;
  return value.every(
    (entry) =>
      typeof entry === "object" &&
      entry !== null &&
      typeof (entry as CachedNodePosition).id === "string" &&
      typeof (entry as CachedNodePosition).x === "number" &&
      typeof (entry as CachedNodePosition).y === "number",
  );
}

export function useGraphSimulation(
  canvasRef: RefObject<HTMLCanvasElement | null>,
  router: AppRouterInstance,
  /** Optional scope ("mine"/"owner:<h>") → fetches that silo's graph; undefined = commons. */
  scope?: string,
): UseGraphSimulationReturn {
  const MAX_TICKS = 300;
  const MAX_SIM_MS = 4000;
  const dataRef = useRef<GraphData | null>(null);
  const animRef = useRef<number>(0);
  const tickRef = useRef<number>(0);
  const startRef = useRef<number>(0);
  const paletteRef = useRef<ColorPalette>(DARK_PALETTE);
  const hoveredRef = useRef<GraphNode | null>(null);
  const mouseRef = useRef<{ x: number; y: number }>({ x: 0, y: 0 });
  const clusterCountRef = useRef<number>(0);
  const nodeMapRef = useRef<Map<string, GraphNode>>(new Map());

  // Camera / viewport
  const cameraRef = useRef<Camera>({ scale: 1, offsetX: 0, offsetY: 0 });
  const dragRef = useRef<{ active: boolean; moved: boolean; startX: number; startY: number; lastX: number; lastY: number }>(
    { active: false, moved: false, startX: 0, startY: 0, lastX: 0, lastY: 0 },
  );

  const [loading, setLoading] = useState(true);
  const [empty, setEmpty] = useState(false);
  const [fetchError, setFetchError] = useState<string | null>(null);
  const [canvasBg, setCanvasBg] = useState<string>(DARK_PALETTE.bg);

  const renderCurrent = useCallback(() => {
    const data = dataRef.current;
    const canvas = canvasRef.current;
    if (!data || !canvas) return;

    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    const dpr = window.devicePixelRatio || 1;
    renderGraph({
      nodes: data.nodes,
      edges: data.edges,
      nodeMap: nodeMapRef.current,
      ctx,
      width: canvas.width / dpr,
      height: canvas.height / dpr,
      palette: paletteRef.current,
      hovered: hoveredRef.current,
      mouse: mouseRef.current,
      clusterCount: clusterCountRef.current,
      camera: cameraRef.current,
    });
  }, [canvasRef]);

  // Simulation loop — mutates world coordinates until the layout settles.
  const simulate = useCallback(() => {
    const data = dataRef.current;
    const canvas = canvasRef.current;
    if (!data || !canvas) return;
    const dpr = window.devicePixelRatio || 1;
    const W = canvas.width / dpr;
    const H = canvas.height / dpr;
    const cx = W / 2;
    const cy = H / 2;
    const { nodes, edges } = data;
    const nodeMap = nodeMapRef.current;

    // --- Physics step ---
    const { totalVelocity } = stepPhysics(nodes, edges, nodeMap, cx, cy);

    renderCurrent();

    // Hard stop: velocity below threshold, max ticks, or max time
    tickRef.current++;
    const elapsed = performance.now() - startRef.current;
    const stopped = totalVelocity <= VELOCITY_THRESHOLD
      || tickRef.current >= MAX_TICKS
      || elapsed > MAX_SIM_MS;

    if (!stopped) {
      animRef.current = requestAnimationFrame(simulate);
    } else {
      // Cache converged positions
      try {
        const key = `graph-pos-v2-${scope ?? "commons"}`;
        const cached = data.nodes.map((n) => ({ id: n.id, x: n.x, y: n.y }));
        sessionStorage.setItem(key, JSON.stringify(cached));
      } catch { /* quota exceeded, ignore */ }

      fitCameraToNodes(data.nodes, W, H, cameraRef.current);
      renderCurrent();
    }
  }, [canvasRef, renderCurrent, scope]);

  // Fetch graph data
  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setEmpty(false);
    setFetchError(null);
    // Reset camera on new data
    cameraRef.current = { scale: 1, offsetX: 0, offsetY: 0 };

    fetch(`/api/wiki/graph${scope ? `?scope=${encodeURIComponent(scope)}` : ""}`)
      .then((r) => {
        if (!r.ok) throw new Error(`Graph API error: ${r.status}`);
        return r.json();
      })
      .then(
        (raw: {
          nodes: {
            id: string;
            label: string;
            tenant?: string;
            linkCount?: number;
            tags?: string[];
          }[];
          edges: GraphEdge[];
        }) => {
          if (cancelled) return;
          if (!raw.nodes || raw.nodes.length === 0) {
            setEmpty(true);
            setLoading(false);
            return;
          }
          // Try to restore cached positions from previous session
          const cacheKey = `graph-pos-v2-${scope ?? "commons"}`;
          let cachedPos: Map<string, { x: number; y: number }> | null = null;
          try {
            const rawCache = sessionStorage.getItem(cacheKey);
            if (rawCache) {
              const arr: unknown = JSON.parse(rawCache);
              if (isCachedNodePositions(arr) && arr.length === raw.nodes.length) {
                cachedPos = new Map(arr.map((p) => [p.id, { x: p.x, y: p.y }]));
              }
            }
          } catch { /* ignore */ }

          // Deterministic initial positions via djb2 hash
          const hashId = (s: string): number => {
            let h = 5381;
            for (let i = 0; i < s.length; i++) h = ((h << 5) + h + s.charCodeAt(i)) | 0;
            return h >>> 0;
          };
          const W = typeof window !== "undefined" ? window.innerWidth : 1200;
          const H = GRAPH_CANVAS_HEIGHT;
          const PAD_X = 80;
          const PAD_Y = 60;
          const usableW = Math.max(300, W - PAD_X * 2);
          const usableH = Math.max(240, H - PAD_Y * 2);
          const nodes: GraphNode[] = raw.nodes.map((n) => ({
            id: n.id,
            label: n.label,
            tenant: n.tenant ?? "knowledge",
            linkCount: n.linkCount ?? 0,
            tags: n.tags ?? [],
            cluster: 0,
            x: cachedPos?.get(n.id)?.x ?? ((hashId(n.id) % usableW) + PAD_X),
            y: cachedPos?.get(n.id)?.y ?? ((hashId(n.id + "y") % usableH) + PAD_Y),
            vx: 0,
            vy: 0,
          }));

          // Color by book group (slug prefix)
          const groupMap = new Map<string, number>();
          let nextGroup = 0;
          for (const node of nodes) {
            const m = node.id.match(/^(ref-bazi-\w+?)(?:-s\d+)?$/);
            const group = m ? m[1] : "other";
            if (!groupMap.has(group)) groupMap.set(group, nextGroup++);
            node.cluster = groupMap.get(group)!;
          }
          clusterCountRef.current = groupMap.size;

          dataRef.current = { nodes, edges: raw.edges ?? [] };
          nodeMapRef.current = new Map(nodes.map((n) => [n.id, n]));
          setLoading(false);
        },
      )
      .catch((err) => {
        if (cancelled) return;
        setFetchError(String(err));
        setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [scope]);

  // Detect color scheme and listen for changes
  useEffect(() => {
    const palette = getColorPalette();
    paletteRef.current = palette;
    setCanvasBg(palette.bg);

    const mql = window.matchMedia("(prefers-color-scheme: dark)");
    const handleChange = () => {
      const newPalette = getColorPalette();
      paletteRef.current = newPalette;
      setCanvasBg(newPalette.bg);
      renderCurrent();
    };
    mql.addEventListener("change", handleChange);
    return () => mql.removeEventListener("change", handleChange);
  }, [renderCurrent]);

  // Start simulation when data is ready
  useEffect(() => {
    if (!loading && !empty && dataRef.current) {
      const canvas = canvasRef.current;
      if (canvas) {
        const dpr = window.devicePixelRatio || 1;
        fitCameraToNodes(
          dataRef.current.nodes,
          canvas.width / dpr,
          canvas.height / dpr,
          cameraRef.current,
        );
      }

      tickRef.current = 0;
      startRef.current = performance.now();
      animRef.current = requestAnimationFrame(simulate);
    }
    return () => cancelAnimationFrame(animRef.current);
  }, [loading, empty, simulate, canvasRef]);

  // Handle canvas resizing (HiDPI-aware)
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const resizeCanvas = () => {
      const parent = canvas.parentElement;
      if (parent) {
        const dpr = window.devicePixelRatio || 1;
        const w = parent.clientWidth;
        const h = Math.max(GRAPH_CANVAS_HEIGHT, window.innerHeight * 0.75);
        canvas.width = w * dpr;
        canvas.height = h * dpr;
        canvas.style.width = `${w}px`;
        canvas.style.height = `${h}px`;
        const ctx = canvas.getContext("2d");
        if (ctx) {
          ctx.setTransform(1, 0, 0, 1, 0, 0);
          ctx.scale(dpr, dpr);
        }
        // Resize should preserve the settled layout and just refit/redraw it.
        if (dataRef.current) {
          fitCameraToNodes(dataRef.current.nodes, w, h, cameraRef.current);
          renderCurrent();
        }
      }
    };
    resizeCanvas();
    window.addEventListener("resize", resizeCanvas);
    return () => window.removeEventListener("resize", resizeCanvas);
  }, [loading, renderCurrent, canvasRef]);

  // Native wheel listener for zoom (must be non-passive to preventDefault)
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const onWheel = (e: WheelEvent) => {
      e.preventDefault();
      const data = dataRef.current;
      if (!data) return;
      const rect = canvas.getBoundingClientRect();
      const mx = e.clientX - rect.left;
      const my = e.clientY - rect.top;
      const camera = cameraRef.current;

      const zoomSpeed = 0.0012;
      const newScale = Math.max(GRAPH_MIN_SCALE, Math.min(GRAPH_MAX_SCALE, camera.scale * (1 - e.deltaY * zoomSpeed)));

      // Zoom toward cursor — keep the world point under cursor stationary
      const wx = (mx - camera.offsetX) / camera.scale;
      const wy = (my - camera.offsetY) / camera.scale;
      camera.offsetX = mx - wx * newScale;
      camera.offsetY = my - wy * newScale;
      camera.scale = newScale;
      renderCurrent();
    };

    canvas.addEventListener("wheel", onWheel, { passive: false });
    return () => canvas.removeEventListener("wheel", onWheel);
  }, [renderCurrent, canvasRef]);

  // --- Pointer event handlers (pan + hover + click) ---

  const onPointerDown = useCallback(
    (e: React.PointerEvent<HTMLCanvasElement>) => {
      const canvas = canvasRef.current;
      if (!canvas) return;
      canvas.setPointerCapture(e.pointerId);
      dragRef.current = {
        active: true,
        moved: false,
        startX: e.clientX,
        startY: e.clientY,
        lastX: e.clientX,
        lastY: e.clientY,
      };
    },
    [canvasRef],
  );

  const onPointerMove = useCallback(
    (e: React.PointerEvent<HTMLCanvasElement>) => {
      const data = dataRef.current;
      const canvas = canvasRef.current;
      if (!data || !canvas) return;

      const rect = canvas.getBoundingClientRect();
      const mx = e.clientX - rect.left;
      const my = e.clientY - rect.top;

      // Panning
      if (dragRef.current.active) {
        const dx = e.clientX - dragRef.current.lastX;
        const dy = e.clientY - dragRef.current.lastY;
        if (Math.abs(e.clientX - dragRef.current.startX) > 3 || Math.abs(e.clientY - dragRef.current.startY) > 3) {
          dragRef.current.moved = true;
        }
        cameraRef.current.offsetX += dx;
        cameraRef.current.offsetY += dy;
        dragRef.current.lastX = e.clientX;
        dragRef.current.lastY = e.clientY;
        canvas.style.cursor = "grabbing";
        renderCurrent();
        return; // Don't update hover while panning
      }

      // Hover detection — convert mouse to world coords for hit test
      const { wx, wy } = screenToWorld(mx, my, cameraRef.current);
      let found: GraphNode | null = null;
      for (const n of data.nodes) {
        const r = nodeRadius(n.linkCount);
        const dx = n.x - wx;
        const dy = n.y - wy;
        // hover padding in world space
        if (dx * dx + dy * dy <= (r + 4 / cameraRef.current.scale) ** 2) {
          found = n;
          break;
        }
      }

      const prev = hoveredRef.current;
      hoveredRef.current = found;
      mouseRef.current = { x: mx, y: my };
      canvas.style.cursor = found ? "pointer" : "default";

      if (prev?.id !== found?.id || found) {
        renderCurrent();
      }
    },
    [renderCurrent, canvasRef],
  );

  const onPointerUp = useCallback(
    (e: React.PointerEvent<HTMLCanvasElement>) => {
      const canvas = canvasRef.current;
      if (!canvas) return;
      canvas.releasePointerCapture(e.pointerId);

      const wasClick = dragRef.current.active && !dragRef.current.moved;
      dragRef.current.active = false;
      canvas.style.cursor = hoveredRef.current ? "pointer" : "default";

      if (wasClick) {
        // Navigate to clicked node
        const data = dataRef.current;
        if (!data) return;
        const rect = canvas.getBoundingClientRect();
        const mx = e.clientX - rect.left;
        const my = e.clientY - rect.top;
        const { wx, wy } = screenToWorld(mx, my, cameraRef.current);
        for (const n of data.nodes) {
          const r = nodeRadius(n.linkCount);
          const dx = n.x - wx;
          const dy = n.y - wy;
          if (dx * dx + dy * dy <= (r + 4) ** 2) {
            router.push(`/wiki/${n.id}`);
            return;
          }
        }
      }
    },
    [router, canvasRef],
  );

  const onPointerLeave = useCallback(() => {
    const canvas = canvasRef.current;
    dragRef.current.active = false;
    if (hoveredRef.current) {
      hoveredRef.current = null;
      if (canvas) canvas.style.cursor = "default";
      renderCurrent();
    }
  }, [renderCurrent, canvasRef]);

  // Fit camera to graph bounds then re-render
  const fitToBounds = useCallback(() => {
    const data = dataRef.current;
    const canvas = canvasRef.current;
    if (!data || !canvas) return;
    const dpr = window.devicePixelRatio || 1;
    const W = canvas.width / dpr;
    const H = canvas.height / dpr;
    fitCameraToNodes(data.nodes, W, H, cameraRef.current);
    renderCurrent();
  }, [renderCurrent, canvasRef]);

  return {
    loading,
    empty,
    fetchError,
    canvasBg,
    onPointerDown,
    onPointerMove,
    onPointerUp,
    onPointerLeave,
    fitToBounds,
  };
}
