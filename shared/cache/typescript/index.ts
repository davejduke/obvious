/**
 * @aiauditor/cache — Redis cache client for AIAUDITOR Node.js services.
 *
 * Provides three cache namespaces:
 *   - session:{user_id}              — JWT refresh-token validation, 8-hour TTL
 *   - ratelimit:{org_id}:{endpoint}  — sliding-window counters
 *   - scope:{engagement_id}:{version} — compiled scope DAG, 30-min TTL
 *
 * Usage:
 *   import { CacheClient } from "@aiauditor/cache";
 *   const cache = new CacheClient({ url: "redis://redis:6379" });
 *   await cache.session.set("user-1", entry);
 *   const result = await cache.ratelimit.increment("org-1", "/api/v1/findings", 60, 100);
 *   await cache.scope.set("eng-1", 1, dagEntry);
 *   await cache.invalidation.onScopeChanged("eng-1");
 */

import Redis, { type RedisOptions } from "ioredis";

// ─── TTL constants ──────────────────────────────────────────────────────────

/** 8 hours in seconds. */
export const SESSION_TTL_SECONDS = 8 * 60 * 60;
/** 30 minutes in seconds. */
export const SCOPE_TTL_SECONDS = 30 * 60;

// ─── Data types ─────────────────────────────────────────────────────────────

export interface SessionEntry {
  userId: string;
  orgId: string;
  /** Current valid refresh-token JTI. */
  tokenId: string;
  roles: string[];
  issuedAt: string;  // ISO 8601
  expiresAt: string; // ISO 8601
}

export interface RateLimitResult {
  count: number;
  allowed: boolean;
  /** Seconds until the oldest request exits the window (0 when allowed). */
  retryAfterSeconds: number;
}

export interface ScopeCacheEntry {
  engagementId: string;
  version: number;
  /** Serialised scope DAG produced by the engine. */
  dag: Record<string, unknown>;
  computedAt: string; // ISO 8601
}

// ─── Config ─────────────────────────────────────────────────────────────────

export interface CacheConfig {
  /** Redis URL, e.g. "redis://redis:6379" or "rediss://..." for TLS. */
  url: string;
  /** Maximum connections in the pool (default 10). */
  maxConnections?: number;
  /** Connection timeout in ms (default 5000). */
  connectTimeout?: number;
  /** Command timeout in ms (default 3000). */
  commandTimeout?: number;
}

// ─── SessionCache ────────────────────────────────────────────────────────────

export class SessionCache {
  private static readonly PREFIX = "session:";

  constructor(private readonly redis: Redis) {}

  private key(userId: string): string {
    return `${SessionCache.PREFIX}${userId}`;
  }

  /** Store a session entry with SESSION_TTL_SECONDS TTL. */
  async set(userId: string, entry: SessionEntry): Promise<void> {
    await this.redis.setex(this.key(userId), SESSION_TTL_SECONDS, JSON.stringify(entry));
  }

  /** Retrieve a session entry. Returns null on cache miss. */
  async get(userId: string): Promise<SessionEntry | null> {
    const raw = await this.redis.get(this.key(userId));
    if (!raw) return null;
    return JSON.parse(raw) as SessionEntry;
  }

  /** Remove the session entry (e.g. on logout). */
  async delete(userId: string): Promise<void> {
    await this.redis.del(this.key(userId));
  }

  /**
   * Reset the TTL on an existing session to SESSION_TTL_SECONDS.
   * Returns false if the key does not exist.
   */
  async refreshTtl(userId: string): Promise<boolean> {
    const result = await this.redis.expire(this.key(userId), SESSION_TTL_SECONDS);
    return result === 1;
  }
}

// ─── RateLimitCache ──────────────────────────────────────────────────────────

export class RateLimitCache {
  private static readonly PREFIX = "ratelimit:";

  constructor(private readonly redis: Redis) {}

  private key(orgId: string, endpoint: string): string {
    return `${RateLimitCache.PREFIX}${orgId}:${endpoint}`;
  }

  /**
   * Record a new request in the sliding window and check whether it is
   * allowed under the configured limit.
   *
   * @param orgId         Organisation identifier.
   * @param endpoint      API endpoint path (key discriminator).
   * @param windowSeconds Length of the sliding window in seconds.
   * @param limit         Maximum requests allowed per window.
   */
  async increment(
    orgId: string,
    endpoint: string,
    windowSeconds: number,
    limit: number,
  ): Promise<RateLimitResult> {
    const nowMs = Date.now();
    const windowStartMs = nowMs - windowSeconds * 1000;
    const member = `${process.hrtime.bigint()}`;
    const expireSeconds = windowSeconds + 1;
    const key = this.key(orgId, endpoint);

    const pipe = this.redis.pipeline();
    pipe.zremrangebyscore(key, "-inf", windowStartMs);
    pipe.zadd(key, nowMs, member);
    pipe.zcard(key);
    pipe.expire(key, expireSeconds);
    const results = await pipe.exec();

    const count = (results?.[2]?.[1] as number) ?? 0;
    const allowed = count <= limit;

    let retryAfterSeconds = 0;
    if (!allowed) {
      const oldest = await this.redis.zrangebyscore(key, "-inf", "+inf", "WITHSCORES", "LIMIT", 0, 1);
      if (oldest.length >= 2) {
        const oldestMs = parseInt(oldest[1], 10);
        const resetsAtMs = oldestMs + windowSeconds * 1000;
        retryAfterSeconds = Math.max(0, (resetsAtMs - nowMs) / 1000);
      }
    }

    return { count, allowed, retryAfterSeconds };
  }

  /** Return the current request count without recording a new request. */
  async count(orgId: string, endpoint: string, windowSeconds: number): Promise<number> {
    const windowStartMs = Date.now() - windowSeconds * 1000;
    const result = await this.redis.zcount(this.key(orgId, endpoint), windowStartMs, "+inf");
    return result;
  }

  /** Clear the rate-limit counter for an org+endpoint. */
  async reset(orgId: string, endpoint: string): Promise<void> {
    await this.redis.del(this.key(orgId, endpoint));
  }
}

// ─── ScopeCache ──────────────────────────────────────────────────────────────

export class ScopeCache {
  private static readonly PREFIX = "scope:";

  constructor(private readonly redis: Redis) {}

  private key(engagementId: string, version: number): string {
    return `${ScopeCache.PREFIX}${engagementId}:${version}`;
  }

  private pattern(engagementId: string): string {
    return `${ScopeCache.PREFIX}${engagementId}:*`;
  }

  /** Store a compiled scope DAG with SCOPE_TTL_SECONDS TTL. */
  async set(engagementId: string, version: number, entry: ScopeCacheEntry): Promise<void> {
    await this.redis.setex(this.key(engagementId, version), SCOPE_TTL_SECONDS, JSON.stringify(entry));
  }

  /** Retrieve a compiled scope DAG. Returns null on cache miss / expiry. */
  async get(engagementId: string, version: number): Promise<ScopeCacheEntry | null> {
    const raw = await this.redis.get(this.key(engagementId, version));
    if (!raw) return null;
    return JSON.parse(raw) as ScopeCacheEntry;
  }

  /**
   * Delete ALL cached scope DAGs for an engagement (all versions).
   * Uses SCAN+DEL to avoid blocking. Returns the number of keys deleted.
   */
  async invalidate(engagementId: string): Promise<number> {
    const pat = this.pattern(engagementId);
    let cursor = "0";
    let deleted = 0;

    do {
      const [nextCursor, keys] = await this.redis.scan(cursor, "MATCH", pat, "COUNT", 100);
      if (keys.length > 0) {
        deleted += await this.redis.del(...keys);
      }
      cursor = nextCursor;
    } while (cursor !== "0");

    return deleted;
  }

  /** Delete the cache entry for a specific engagement version. */
  async invalidateVersion(engagementId: string, version: number): Promise<void> {
    await this.redis.del(this.key(engagementId, version));
  }

  /**
   * Return remaining TTL in seconds.
   * -2 = key missing, -1 = key exists with no expiry.
   */
  async ttl(engagementId: string, version: number): Promise<number> {
    return this.redis.ttl(this.key(engagementId, version));
  }
}

// ─── InvalidationHandler ─────────────────────────────────────────────────────

export class InvalidationHandler {
  constructor(
    private readonly session: SessionCache,
    private readonly scope: ScopeCache,
  ) {}

  /**
   * Invalidate ALL cached scope DAGs for the engagement.
   * Call whenever scope configuration changes are detected.
   * Returns the number of keys deleted.
   */
  async onScopeChanged(engagementId: string): Promise<number> {
    return this.scope.invalidate(engagementId);
  }

  /** Invalidate a specific scope version. */
  async onScopeVersionChanged(engagementId: string, version: number): Promise<void> {
    return this.scope.invalidateVersion(engagementId, version);
  }

  /** Invalidate the session entry on explicit logout. */
  async onUserLogout(userId: string): Promise<void> {
    return this.session.delete(userId);
  }

  /** Invalidate the session entry for a deactivated user. */
  async onUserDeactivated(userId: string): Promise<void> {
    return this.session.delete(userId);
  }
}

// ─── CacheClient ─────────────────────────────────────────────────────────────

/**
 * Top-level cache client that wires together all namespace helpers.
 *
 * Call `close()` when the process shuts down.
 */
export class CacheClient {
  private readonly redis: Redis;

  readonly session: SessionCache;
  readonly ratelimit: RateLimitCache;
  readonly scope: ScopeCache;
  readonly invalidation: InvalidationHandler;

  constructor(config: CacheConfig) {
    const opts: RedisOptions = {
      maxRetriesPerRequest: 3,
      connectTimeout: config.connectTimeout ?? 5000,
      commandTimeout: config.commandTimeout ?? 3000,
      // Connection pool size is handled by cluster / sentinel mode or
      // can be configured via lazyConnect + enableOfflineQueue for basic client.
      enableOfflineQueue: true,
      retryStrategy: (times: number) => Math.min(times * 100, 3000),
    };

    this.redis = new Redis(config.url, opts);
    this.session = new SessionCache(this.redis);
    this.ratelimit = new RateLimitCache(this.redis);
    this.scope = new ScopeCache(this.redis);
    this.invalidation = new InvalidationHandler(this.session, this.scope);
  }

  /** Verify the Redis connection is alive. */
  async ping(): Promise<boolean> {
    const result = await this.redis.ping();
    return result === "PONG";
  }

  /** Release the underlying connection. */
  async close(): Promise<void> {
    await this.redis.quit();
  }
}
