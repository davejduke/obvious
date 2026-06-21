/**
 * Tests for @aiauditor/cache
 * Uses ioredis-mock so no Redis server is required.
 */
import RedisMock from "ioredis-mock";
import {
  SESSION_TTL_SECONDS,
  SCOPE_TTL_SECONDS,
  SessionCache,
  RateLimitCache,
  ScopeCache,
  InvalidationHandler,
  type SessionEntry,
  type ScopeCacheEntry,
} from "./index";

// ─── Helpers ────────────────────────────────────────────────────────────────

function makeRedis(): InstanceType<typeof RedisMock> {
  return new RedisMock();
}

function sampleSession(userId = "user-1"): SessionEntry {
  return {
    userId,
    orgId: "org-1",
    tokenId: "jti-abc",
    roles: ["internal_auditor"],
    issuedAt: new Date().toISOString(),
    expiresAt: new Date(Date.now() + SESSION_TTL_SECONDS * 1000).toISOString(),
  };
}

function sampleScope(engagementId = "eng-1", version = 1): ScopeCacheEntry {
  return {
    engagementId,
    version,
    dag: { nodes: ["ctrl-1", "ctrl-2"] },
    computedAt: new Date().toISOString(),
  };
}

// ─── SessionCache ────────────────────────────────────────────────────────────

describe("SessionCache", () => {
  test("set → get round-trip", async () => {
    const redis = makeRedis() as any;
    const cache = new SessionCache(redis);
    const entry = sampleSession();

    await cache.set("user-1", entry);
    const result = await cache.get("user-1");

    expect(result).not.toBeNull();
    expect(result?.userId).toBe("user-1");
    expect(result?.tokenId).toBe("jti-abc");
  });

  test("get returns null on miss", async () => {
    const redis = makeRedis() as any;
    const cache = new SessionCache(redis);

    const result = await cache.get("nonexistent");
    expect(result).toBeNull();
  });

  test("delete removes entry", async () => {
    const redis = makeRedis() as any;
    const cache = new SessionCache(redis);
    await cache.set("user-2", sampleSession("user-2"));
    await cache.delete("user-2");

    const result = await cache.get("user-2");
    expect(result).toBeNull();
  });

  test("key prefix is 'session:'", async () => {
    const redis = makeRedis() as any;
    const cache = new SessionCache(redis);
    await cache.set("user-3", sampleSession("user-3"));

    // The underlying key should use the correct prefix.
    const raw = await redis.get("session:user-3");
    expect(raw).not.toBeNull();
  });
});

// ─── RateLimitCache ──────────────────────────────────────────────────────────

describe("RateLimitCache", () => {
  test("allows requests under the limit", async () => {
    const redis = makeRedis() as any;
    const cache = new RateLimitCache(redis);
    const LIMIT = 5;

    for (let i = 0; i < LIMIT; i++) {
      const result = await cache.increment("org-1", "/api/findings", 60, LIMIT);
      expect(result.allowed).toBe(true);
    }
  });

  test("denies the request that exceeds the limit", async () => {
    const redis = makeRedis() as any;
    const cache = new RateLimitCache(redis);
    const LIMIT = 2;

    for (let i = 0; i < LIMIT; i++) {
      await cache.increment("org-2", "/api/findings", 60, LIMIT);
    }
    const result = await cache.increment("org-2", "/api/findings", 60, LIMIT);
    expect(result.allowed).toBe(false);
    expect(result.count).toBeGreaterThan(LIMIT);
  });

  test("reset clears the counter", async () => {
    const redis = makeRedis() as any;
    const cache = new RateLimitCache(redis);
    const LIMIT = 1;

    await cache.increment("org-3", "/health", 60, LIMIT);
    await cache.increment("org-3", "/health", 60, LIMIT);
    await cache.reset("org-3", "/health");

    const result = await cache.increment("org-3", "/health", 60, LIMIT);
    expect(result.allowed).toBe(true);
  });

  test("key format is 'ratelimit:{orgId}:{endpoint}'", async () => {
    const redis = makeRedis() as any;
    const cache = new RateLimitCache(redis);
    await cache.increment("org-4", "/api/v1/engagements", 60, 100);

    const keys = await redis.keys("ratelimit:*");
    expect(keys).toContain("ratelimit:org-4:/api/v1/engagements");
  });
});

// ─── ScopeCache ──────────────────────────────────────────────────────────────

describe("ScopeCache", () => {
  test("set → get round-trip", async () => {
    const redis = makeRedis() as any;
    const cache = new ScopeCache(redis);
    const entry = sampleScope();

    await cache.set("eng-1", 1, entry);
    const result = await cache.get("eng-1", 1);

    expect(result).not.toBeNull();
    expect(result?.engagementId).toBe("eng-1");
    expect(result?.version).toBe(1);
  });

  test("get returns null on miss", async () => {
    const redis = makeRedis() as any;
    const cache = new ScopeCache(redis);
    const result = await cache.get("eng-missing", 99);
    expect(result).toBeNull();
  });

  test("invalidate removes all versions", async () => {
    const redis = makeRedis() as any;
    const cache = new ScopeCache(redis);

    for (const ver of [1, 2, 3]) {
      await cache.set("eng-2", ver, sampleScope("eng-2", ver));
    }
    const deleted = await cache.invalidate("eng-2");
    expect(deleted).toBe(3);

    for (const ver of [1, 2, 3]) {
      expect(await cache.get("eng-2", ver)).toBeNull();
    }
  });

  test("key format is 'scope:{engagementId}:{version}'", async () => {
    const redis = makeRedis() as any;
    const cache = new ScopeCache(redis);
    await cache.set("eng-3", 7, sampleScope("eng-3", 7));

    const raw = await redis.get("scope:eng-3:7");
    expect(raw).not.toBeNull();
  });
});

// ─── InvalidationHandler ────────────────────────────────────────────────────

describe("InvalidationHandler", () => {
  test("onScopeChanged invalidates all versions", async () => {
    const redis = makeRedis() as any;
    const session = new SessionCache(redis);
    const scope = new ScopeCache(redis);
    const handler = new InvalidationHandler(session, scope);

    await scope.set("eng-4", 1, sampleScope("eng-4", 1));
    await scope.set("eng-4", 2, sampleScope("eng-4", 2));
    const deleted = await handler.onScopeChanged("eng-4");
    expect(deleted).toBe(2);

    expect(await scope.get("eng-4", 1)).toBeNull();
    expect(await scope.get("eng-4", 2)).toBeNull();
  });

  test("onUserLogout removes session", async () => {
    const redis = makeRedis() as any;
    const session = new SessionCache(redis);
    const scope = new ScopeCache(redis);
    const handler = new InvalidationHandler(session, scope);

    await session.set("user-out", sampleSession("user-out"));
    await handler.onUserLogout("user-out");

    expect(await session.get("user-out")).toBeNull();
  });
});

// ─── TTL constants ────────────────────────────────────────────────────────────

describe("TTL constants", () => {
  test("SESSION_TTL_SECONDS is 8 hours", () => {
    expect(SESSION_TTL_SECONDS).toBe(8 * 60 * 60);
  });

  test("SCOPE_TTL_SECONDS is 30 minutes", () => {
    expect(SCOPE_TTL_SECONDS).toBe(30 * 60);
  });
});
