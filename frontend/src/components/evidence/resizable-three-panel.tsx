'use client';
import { useRef, useState, useCallback, useEffect } from 'react';
import { clsx } from 'clsx';

interface ResizableThreePanelProps {
  left: React.ReactNode;
  center: React.ReactNode;
  right: React.ReactNode;
  defaultLeftWidth?: number;   // percentage
  defaultRightWidth?: number;  // percentage
  minPanelWidth?: number;      // pixels
  className?: string;
}

const DRAG_HANDLE_CLASS =
  'w-1.5 bg-slate-100 hover:bg-blue-300 active:bg-blue-400 cursor-col-resize flex-shrink-0 transition-colors relative group';

const DRAG_INDICATOR_CLASS =
  'absolute inset-y-0 left-1/2 -translate-x-1/2 w-0.5 bg-slate-300 group-hover:bg-blue-400 transition-colors';

export function ResizableThreePanel({
  left,
  center,
  right,
  defaultLeftWidth = 25,
  defaultRightWidth = 28,
  minPanelWidth = 180,
  className,
}: ResizableThreePanelProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [leftPct, setLeftPct] = useState(defaultLeftWidth);
  const [rightPct, setRightPct] = useState(defaultRightWidth);

  const draggingRef = useRef<'left-handle' | 'right-handle' | null>(null);
  const startXRef = useRef(0);
  const startPctRef = useRef(0);

  const onMouseMove = useCallback(
    (e: MouseEvent) => {
      if (!draggingRef.current || !containerRef.current) return;
      const rect = containerRef.current.getBoundingClientRect();
      const containerWidth = rect.width;
      if (containerWidth === 0) return;

      const minPct = (minPanelWidth / containerWidth) * 100;
      const delta = ((e.clientX - startXRef.current) / containerWidth) * 100;

      if (draggingRef.current === 'left-handle') {
        const newLeft = Math.max(minPct, Math.min(startPctRef.current + delta, 100 - rightPct - minPct));
        setLeftPct(newLeft);
      } else {
        const newRight = Math.max(minPct, Math.min(startPctRef.current - delta, 100 - leftPct - minPct));
        setRightPct(newRight);
      }
    },
    [leftPct, rightPct, minPanelWidth],
  );

  const onMouseUp = useCallback(() => {
    draggingRef.current = null;
    document.body.style.cursor = '';
    document.body.style.userSelect = '';
  }, []);

  useEffect(() => {
    document.addEventListener('mousemove', onMouseMove);
    document.addEventListener('mouseup', onMouseUp);
    return () => {
      document.removeEventListener('mousemove', onMouseMove);
      document.removeEventListener('mouseup', onMouseUp);
    };
  }, [onMouseMove, onMouseUp]);

  const startDrag = (
    e: React.MouseEvent,
    handle: 'left-handle' | 'right-handle',
    currentPct: number,
  ) => {
    e.preventDefault();
    draggingRef.current = handle;
    startXRef.current = e.clientX;
    startPctRef.current = currentPct;
    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';
  };

  const centerPct = 100 - leftPct - rightPct;

  return (
    <div ref={containerRef} className={clsx('flex h-full overflow-hidden', className)}>
      {/* Left panel */}
      <div
        className="flex flex-col overflow-hidden border-r border-slate-200"
        style={{ width: `${leftPct}%` }}
      >
        {left}
      </div>

      {/* Drag handle left */}
      <div
        className={DRAG_HANDLE_CLASS}
        onMouseDown={(e) => startDrag(e, 'left-handle', leftPct)}
        role="separator"
        aria-orientation="vertical"
        aria-label="Resize left panel"
      >
        <div className={DRAG_INDICATOR_CLASS} />
      </div>

      {/* Center panel */}
      <div
        className="flex flex-col overflow-hidden border-r border-slate-200"
        style={{ width: `${centerPct}%` }}
      >
        {center}
      </div>

      {/* Drag handle right */}
      <div
        className={DRAG_HANDLE_CLASS}
        onMouseDown={(e) => startDrag(e, 'right-handle', rightPct)}
        role="separator"
        aria-orientation="vertical"
        aria-label="Resize right panel"
      >
        <div className={DRAG_INDICATOR_CLASS} />
      </div>

      {/* Right panel */}
      <div
        className="flex flex-col overflow-hidden"
        style={{ width: `${rightPct}%` }}
      >
        {right}
      </div>
    </div>
  );
}
