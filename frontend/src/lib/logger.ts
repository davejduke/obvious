/**
 * Structured JSON logger for AIAUDITOR frontend API routes.
 *
 * Emits log entries matching tech spec §10.1:
 *   {timestamp, level, service, trace_id, span_id, org_id, engagement_id, event, metadata}
 *
 * Usage:
 *   import { createLogger } from '@/lib/logger'
 *   const log = createLogger('frontend')
 *   log.info(ctx, 'user.login', { user_id: id })
 */

import type { NextRequest } from 'next/server'

/** W3C Trace Context header names. */
export const HEADER_TRACEPARENT = 'traceparent'
export const HEADER_TRACESTATE = 'tracestate'
export const HEADER_REQUEST_ID = 'x-request-id'

/** Log severity levels matching tech spec §10.1. */
export type LogLevel = 'INFO' | 'WARN' | 'ERROR' | 'CRITICAL'

/** A single structured log entry matching tech spec §10.1. */
export interface LogEntry {
  timestamp: string
  level: LogLevel
  service: string
  trace_id?: string
  span_id?: string
  org_id?: string
  engagement_id?: string
  event: string
  metadata?: Record<string, unknown>
}

/** LogContext carries trace and tenant IDs extracted from a request. */
export interface LogContext {
  traceId?: string
  spanId?: string
  orgId?: string
  engagementId?: string
  requestId?: string
}

/** Parse a W3C traceparent header into {traceId, spanId}. */
export function parseTraceparent(
  header: string | null | undefined
): { traceId: string; spanId: string } | null {
  if (!header) return null
  const parts = header.split('-')
  if (parts.length !== 4) return null
  const [version, traceId, spanId, flags] = parts
  if (version !== '00' || traceId.length !== 32 || spanId.length !== 16 || flags.length !== 2) {
    return null
  }
  return { traceId, spanId }
}

/** Format a W3C traceparent header value. */
export function formatTraceparent(traceId: string, spanId: string): string {
  return `00-${traceId}-${spanId}-01`
}

/** Extract LogContext from a Next.js API route request. */
export function contextFromRequest(req: NextRequest): LogContext {
  const traceparent = req.headers.get(HEADER_TRACEPARENT)
  const parsed = parseTraceparent(traceparent)
  return {
    traceId: parsed?.traceId,
    spanId: parsed?.spanId,
    orgId: req.headers.get('x-org-id') ?? undefined,
    requestId: req.headers.get(HEADER_REQUEST_ID) ?? undefined,
  }
}

/** A typed structured logger bound to a service name. */
export class Logger {
  constructor(private readonly service: string) {}

  private emit(
    level: LogLevel,
    ctx: LogContext,
    event: string,
    metadata?: Record<string, unknown>
  ): void {
    const entry: LogEntry = {
      timestamp: new Date().toISOString(),
      level,
      service: this.service,
      event,
    }
    if (ctx.traceId) entry.trace_id = ctx.traceId
    if (ctx.spanId) entry.span_id = ctx.spanId
    if (ctx.orgId) entry.org_id = ctx.orgId
    if (ctx.engagementId) entry.engagement_id = ctx.engagementId
    if (metadata && Object.keys(metadata).length > 0) entry.metadata = metadata

    // In production Next.js, console.log goes to stdout as JSON.
    // eslint-disable-next-line no-console
    console.log(JSON.stringify(entry))
  }

  info(ctx: LogContext, event: string, metadata?: Record<string, unknown>): void {
    this.emit('INFO', ctx, event, metadata)
  }

  warn(ctx: LogContext, event: string, metadata?: Record<string, unknown>): void {
    this.emit('WARN', ctx, event, metadata)
  }

  error(ctx: LogContext, event: string, metadata?: Record<string, unknown>): void {
    this.emit('ERROR', ctx, event, metadata)
  }

  critical(ctx: LogContext, event: string, metadata?: Record<string, unknown>): void {
    this.emit('CRITICAL', ctx, event, metadata)
  }
}

/** Create a structured logger bound to the given service name. */
export function createLogger(service: string): Logger {
  return new Logger(service)
}

/** Default frontend logger instance. */
export const logger = createLogger('frontend')

