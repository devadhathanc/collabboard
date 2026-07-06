import {
  useMemo,
  useRef,
  useEffect,
  useState,
  useCallback,
} from "react";
import type { Idea } from "../types";

// ─── Types ────────────────────────────────────────────────────────────────────

export interface IdeaLink {
  from: string; // idea ID
  to: string;   // idea ID
}

interface Props {
  ideas: Idea[];
  links: IdeaLink[];
}

// ─── Constants ────────────────────────────────────────────────────────────────

const CANVAS_W = 5000;
const CANVAS_H = 5000;
const CX = CANVAS_W / 2; // canvas center X
const CY = CANVAS_H / 2; // canvas center Y
const MIN_ZOOM = 0.15;
const MAX_ZOOM = 2.5;
const GAP = 18; // min gap between bubbles (px)

// ─── Colors ───────────────────────────────────────────────────────────────────

const BUBBLE_COLORS = [
  { bg: "rgba(99,102,241,0.18)", border: "rgba(99,102,241,0.55)", text: "#a5b4fc", line: "#818cf8" },
  { bg: "rgba(168,85,247,0.18)", border: "rgba(168,85,247,0.55)", text: "#c4b5fd", line: "#c084fc" },
  { bg: "rgba(236,72,153,0.18)", border: "rgba(236,72,153,0.55)", text: "#f9a8d4", line: "#f472b6" },
  { bg: "rgba(59,130,246,0.18)", border: "rgba(59,130,246,0.55)", text: "#93c5fd", line: "#60a5fa" },
  { bg: "rgba(6,182,212,0.18)", border: "rgba(6,182,212,0.55)", text: "#67e8f9", line: "#22d3ee" },
  { bg: "rgba(34,197,94,0.18)", border: "rgba(34,197,94,0.55)", text: "#86efac", line: "#4ade80" },
  { bg: "rgba(245,158,11,0.18)", border: "rgba(245,158,11,0.55)", text: "#fcd34d", line: "#fbbf24" },
  { bg: "rgba(239,68,68,0.18)", border: "rgba(239,68,68,0.55)", text: "#fca5a5", line: "#f87171" },
];

function hashColor(text: string) {
  let h = 0;
  for (let i = 0; i < text.length; i++) h = text.charCodeAt(i) + ((h << 5) - h);
  return BUBBLE_COLORS[Math.abs(h) % BUBBLE_COLORS.length];
}

// ─── Layout helpers ───────────────────────────────────────────────────────────

const MIN_FONT = 12;
const MAX_FONT = 22;
const MIN_PAD_V = 8;
const MAX_PAD_V = 14;
const MIN_PAD_H = 14;
const MAX_PAD_H = 24;

function bubbleDims(text: string, scale: number) {
  const font = MIN_FONT + (MAX_FONT - MIN_FONT) * scale;
  const chars = Math.min(text.length, 28);
  const w = chars * font * 0.54 + (MIN_PAD_H + (MAX_PAD_H - MIN_PAD_H) * scale) * 2;
  const h = font * 1.45 + (MIN_PAD_V + (MAX_PAD_V - MIN_PAD_V) * scale) * 2;
  return { w, h, font };
}

function overlaps(
  ax: number, ay: number, aw: number, ah: number,
  bx: number, by: number, bw: number, bh: number,
): boolean {
  return (
    Math.abs(ax - bx) < (aw + bw) / 2 + GAP &&
    Math.abs(ay - by) < (ah + bh) / 2 + GAP
  );
}

interface BubbleLayout {
  id: string;
  text: string;
  count: number;
  x: number;
  y: number;
  w: number;
  h: number;
  font: number;
  padV: number;
  padH: number;
  color: typeof BUBBLE_COLORS[0];
  isTop: boolean;
  isAI: boolean;
}

function computeLayout(
  ideas: Idea[],
): BubbleLayout[] {
  // Deduplicate
  const map = new Map<string, { idea: Idea; count: number }>();
  for (const idea of ideas) {
    const key = idea.text.trim().toLowerCase();
    if (!key) continue;
    const ex = map.get(key);
    if (ex) { ex.count = Math.max(ex.count, idea.count); }
    else { map.set(key, { idea, count: idea.count }); }
  }
  const groups = [...map.values()].sort((a, b) => b.count - a.count);
  if (groups.length === 0) return [];

  const maxCount = groups[0].count;
  const placed: { x: number; y: number; w: number; h: number }[] = [];
  const result: BubbleLayout[] = [];

  for (let i = 0; i < groups.length; i++) {
    const { idea, count } = groups[i];
    const scale = 0.35 + (count / maxCount) * 0.65;
    const { w, h, font } = bubbleDims(idea.text, scale);
    const padV = MIN_PAD_V + (MAX_PAD_V - MIN_PAD_V) * scale;
    const padH = MIN_PAD_H + (MAX_PAD_H - MIN_PAD_H) * scale;
    const color = hashColor(idea.text);
    const isTop = i === 0;
    const isAI = idea.created_by === "AI";

    if (i === 0) {
      placed.push({ x: CX, y: CY, w, h });
      result.push({ id: idea.id, text: idea.text, count, x: CX, y: CY, w, h, font, padV, padH, color, isTop, isAI });
      continue;
    }

    // Archimedean spiral from center
    const a = 6;
    let bx = CX, by = CY, found = false;
    for (let theta = 0; theta < 600; theta += 0.15) {
      const r = a * theta;
      const px = CX + r * Math.cos(theta);
      const py = CY + r * Math.sin(theta);

      let collision = false;
      for (const p of placed) {
        if (overlaps(px, py, w, h, p.x, p.y, p.w, p.h)) { collision = true; break; }
      }
      if (!collision) { bx = px; by = py; found = true; break; }
    }

    if (!found) {
      const prev = placed[placed.length - 1];
      bx = prev.x;
      by = prev.y + prev.h / 2 + h / 2 + GAP;
    }

    placed.push({ x: bx, y: by, w, h });
    result.push({ id: idea.id, text: idea.text, count, x: bx, y: by, w, h, font, padV, padH, color, isTop, isAI });
  }
  return result;
}

// ─── Relationship lines (SVG) ────────────────────────────────────────────────

function LinkLines({
  links,
  posMap,
  layoutMap,
}: {
  links: IdeaLink[];
  posMap: Map<string, { x: number; y: number }>;
  layoutMap: Map<string, BubbleLayout>;
}) {
  return (
    <>
      {links.map((link, i) => {
        const from = posMap.get(link.from);
        const to = posMap.get(link.to);
        if (!from || !to) return null;

        const fromLayout = layoutMap.get(link.from);
        const strokeColor = fromLayout?.color.line ?? "#818cf8";

        // Cubic bezier through a midpoint
        const midX = (from.x + to.x) / 2;
        const d = `M ${from.x} ${from.y} C ${midX} ${from.y}, ${midX} ${to.y}, ${to.x} ${to.y}`;

        return (
          <g key={i}>
            {/* Glow layer */}
            <path
              d={d}
              stroke={strokeColor}
              strokeWidth={6}
              fill="none"
              opacity={0.18}
              strokeLinecap="round"
            />
            {/* Main line */}
            <path
              d={d}
              stroke={strokeColor}
              strokeWidth={2}
              fill="none"
              opacity={0.9}
              strokeLinecap="round"
            />
            {/* Arrowhead dot at destination */}
            <circle
              cx={to.x}
              cy={to.y}
              r={4}
              fill={strokeColor}
              opacity={0.85}
            />
          </g>
        );
      })}
    </>
  );
}

// ─── Component ────────────────────────────────────────────────────────────────

export function IdeaCloud({ ideas, links }: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [view, setView] = useState({ x: 0, y: 0, scale: 0.85 });
  const [ready, setReady] = useState(false);
  const [isDragging, setIsDragging] = useState(false);
  const dragOrigin = useRef<{ mx: number; my: number; vx: number; vy: number } | null>(null);
  const lastTouch = useRef<{ dist: number; cx: number; cy: number } | null>(null);

  // Centre view on first render
  useEffect(() => {
    const el = containerRef.current;
    if (!el || ready) return;
    const scale = 0.85;
    setView({
      x: el.clientWidth / 2 - CX * scale,
      y: el.clientHeight / 2 - CY * scale,
      scale,
    });
    setReady(true);
  }, [ready]);

  // ── Zoom ──────────────────────────────────────────────────────────────────
  const zoomAt = useCallback((factor: number, px: number, py: number) => {
    setView((v) => {
      const newScale = Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, v.scale * factor));
      const cx = (px - v.x) / v.scale;
      const cy = (py - v.y) / v.scale;
      return { x: px - cx * newScale, y: py - cy * newScale, scale: newScale };
    });
  }, []);

  const handleWheel = useCallback((e: WheelEvent) => {
    e.preventDefault();
    const el = containerRef.current!;
    const rect = el.getBoundingClientRect();
    zoomAt(e.deltaY < 0 ? 1.1 : 0.91, e.clientX - rect.left, e.clientY - rect.top);
  }, [zoomAt]);

  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    el.addEventListener("wheel", handleWheel, { passive: false });
    return () => el.removeEventListener("wheel", handleWheel);
  }, [handleWheel]);

  // ── Mouse pan ─────────────────────────────────────────────────────────────
  const onMouseDown = useCallback((e: React.MouseEvent) => {
    if (e.button !== 0) return;
    setIsDragging(true);
    dragOrigin.current = { mx: e.clientX, my: e.clientY, vx: view.x, vy: view.y };
  }, [view]);

  const onMouseMove = useCallback((e: React.MouseEvent) => {
    if (!isDragging || !dragOrigin.current) return;
    const dx = e.clientX - dragOrigin.current.mx;
    const dy = e.clientY - dragOrigin.current.my;
    setView((v) => ({ ...v, x: dragOrigin.current!.vx + dx, y: dragOrigin.current!.vy + dy }));
  }, [isDragging]);

  const onMouseUp = useCallback(() => {
    setIsDragging(false);
    dragOrigin.current = null;
  }, []);

  // ── Touch pan / pinch zoom ────────────────────────────────────────────────
  const onTouchStart = useCallback((e: React.TouchEvent) => {
    if (e.touches.length === 1) {
      const t = e.touches[0];
      dragOrigin.current = { mx: t.clientX, my: t.clientY, vx: view.x, vy: view.y };
      lastTouch.current = null;
    } else if (e.touches.length === 2) {
      const t0 = e.touches[0], t1 = e.touches[1];
      const dist = Math.hypot(t1.clientX - t0.clientX, t1.clientY - t0.clientY);
      lastTouch.current = { dist, cx: (t0.clientX + t1.clientX) / 2, cy: (t0.clientY + t1.clientY) / 2 };
    }
  }, [view]);

  const onTouchMove = useCallback((e: React.TouchEvent) => {
    e.preventDefault();
    const el = containerRef.current!;
    const rect = el.getBoundingClientRect();
    if (e.touches.length === 1 && dragOrigin.current) {
      const t = e.touches[0];
      const dx = t.clientX - dragOrigin.current.mx;
      const dy = t.clientY - dragOrigin.current.my;
      setView((v) => ({ ...v, x: dragOrigin.current!.vx + dx, y: dragOrigin.current!.vy + dy }));
    } else if (e.touches.length === 2 && lastTouch.current) {
      const t0 = e.touches[0], t1 = e.touches[1];
      const dist = Math.hypot(t1.clientX - t0.clientX, t1.clientY - t0.clientY);
      const factor = dist / lastTouch.current.dist;
      const px = (t0.clientX + t1.clientX) / 2 - rect.left;
      const py = (t0.clientY + t1.clientY) / 2 - rect.top;
      zoomAt(factor, px, py);
      lastTouch.current = { dist, cx: px, cy: py };
    }
  }, [zoomAt]);

  const onTouchEnd = useCallback(() => {
    dragOrigin.current = null;
    lastTouch.current = null;
  }, []);

  // ── Layout ────────────────────────────────────────────────────────────────
  const layout = useMemo(() => computeLayout(ideas), [ideas]);

  const posMap = useMemo(() => {
    // Primary: map canonical IDs from the deduplicated layout
    const m = new Map<string, { x: number; y: number }>();
    const textToPos = new Map<string, { x: number; y: number }>();
    for (const b of layout) {
      m.set(b.id, { x: b.x, y: b.y });
      textToPos.set(b.text.trim().toLowerCase(), { x: b.x, y: b.y });
    }
    // Fallback: map every original idea ID via text match.
    // Needed when deduplication keeps only one ID per text group.
    for (const idea of ideas) {
      if (!m.has(idea.id)) {
        const pos = textToPos.get(idea.text.trim().toLowerCase());
        if (pos) m.set(idea.id, pos);
      }
    }
    return m;
  }, [layout, ideas]);

  const layoutMap = useMemo(() => {
    const m = new Map<string, BubbleLayout>();
    for (const b of layout) m.set(b.id, b);
    return m;
  }, [layout]);

  // ── Zoom controls ─────────────────────────────────────────────────────────
  const zoomIn = () => {
    const el = containerRef.current!;
    zoomAt(1.2, el.clientWidth / 2, el.clientHeight / 2);
  };
  const zoomOut = () => {
    const el = containerRef.current!;
    zoomAt(0.83, el.clientWidth / 2, el.clientHeight / 2);
  };
  const zoomFit = () => {
    const el = containerRef.current!;
    const scale = 0.85;
    setView({ x: el.clientWidth / 2 - CX * scale, y: el.clientHeight / 2 - CY * scale, scale });
  };

  if (ideas.length === 0) {
    return (
      <div className="cloud-container" ref={containerRef}>
        <div className="cloud-empty">
          <div className="cloud-empty-icon">💡</div>
          <p>Ideas will appear here as people submit them</p>
          <p className="cloud-empty-sub">Popular ideas grow larger in the center</p>
        </div>
      </div>
    );
  }

  return (
    <div
      ref={containerRef}
      className="cloud-container"
      style={{ cursor: isDragging ? "grabbing" : "grab", userSelect: "none" }}
      onMouseDown={onMouseDown}
      onMouseMove={onMouseMove}
      onMouseUp={onMouseUp}
      onMouseLeave={onMouseUp}
      onTouchStart={onTouchStart}
      onTouchMove={onTouchMove}
      onTouchEnd={onTouchEnd}
    >
      {/* Infinite canvas */}
      <div
        className="cloud-canvas"
        style={{
          width: CANVAS_W,
          height: CANVAS_H,
          position: "absolute",
          transformOrigin: "0 0",
          transform: `translate(${view.x}px, ${view.y}px) scale(${view.scale})`,
          willChange: "transform",
        }}
      >
        {/* SVG relationship lines */}
        <svg
          style={{ position: "absolute", top: 0, left: 0, width: CANVAS_W, height: CANVAS_H, pointerEvents: "none", overflow: "visible" }}
        >
          <LinkLines links={links} posMap={posMap} layoutMap={layoutMap} />
        </svg>

        {/* Bubbles */}
        {layout.map((b, i) => {
          const displayText = b.text.length > 28 ? b.text.slice(0, 26) + "…" : b.text;
          return (
            <div
              key={b.id}
              className={`idea-bubble-wrapper${b.isTop ? " idea-bubble-top-wrapper" : ""}`}
              style={{
                left: b.x,
                top: b.y,
                animationDelay: `${i * 40}ms`,
                pointerEvents: "none",
              }}
            >
              <div
                className={`idea-bubble${b.isTop ? " idea-bubble-top" : ""}${b.isAI ? " idea-bubble-ai" : ""}`}
                style={{
                  background: b.color.bg,
                  borderColor: b.color.border,
                  color: b.color.text,
                  fontSize: b.font,
                  padding: `${b.padV}px ${b.padH}px`,
                }}
              >
                {b.isAI && <span className="ai-tag">AI</span>}
                <span>{displayText}</span>
                {b.count > 1 && (
                  <span className="idea-bubble-count">×{b.count}</span>
                )}
              </div>
            </div>
          );
        })}
      </div>

      {/* Zoom controls — fixed over the canvas */}
      <div className="zoom-controls">
        <button className="zoom-btn" onClick={zoomIn} title="Zoom in">+</button>
        <span className="zoom-level">{Math.round(view.scale * 100)}%</span>
        <button className="zoom-btn" onClick={zoomOut} title="Zoom out">−</button>
        <button className="zoom-btn zoom-fit" onClick={zoomFit} title="Fit to view">⊙</button>
      </div>
    </div>
  );
}
