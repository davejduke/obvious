"""AIAUDITOR Redis cache client — session, ratelimit, and scope namespaces.

All three caches are available via the CacheClient facade:

    from engine.shared.cache import CacheClient

    cache = CacheClient(redis_url="redis://localhost:6379")
    await cache.session.set("user-1", entry)
    result = await cache.ratelimit.increment("org-1", "/api/v1/findings", 60, 100)
    await cache.scope.set("eng-1", 1, dag_entry)
    await cache.invalidation.on_scope_changed("eng-1")
"""
from __future__ import annotations

import json
import time
from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import Any, Optional

import redis.asyncio as aioredis

# ─── TTL constants ────────────────────────────────────────────────────────────

SESSION_TTL_SECONDS: int = 8 * 60 * 60  # 8 hours
SCOPE_TTL_SECONDS: int = 30 * 60        # 30 minutes

# ─── Data models ──────────────────────────────────────────────────────────────


@dataclass
class SessionEntry:
    """Cached session data for a user. Key: session:{user_id}."""

    user_id: str
    org_id: str
    token_id: str  # current valid refresh-token JTI
    roles: list[str] = field(default_factory=list)
    issued_at: datetime = field(default_factory=lambda: datetime.now(timezone.utc))
    expires_at: datetime = field(default_factory=lambda: datetime.now(timezone.utc))

    def to_dict(self) -> dict[str, Any]:
        return {
            "user_id": self.user_id,
            "org_id": self.org_id,
            "token_id": self.token_id,
            "roles": self.roles,
            "issued_at": self.issued_at.isoformat(),
            "expires_at": self.expires_at.isoformat(),
        }

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> "SessionEntry":
        return cls(
            user_id=data["user_id"],
            org_id=data["org_id"],
            token_id=data["token_id"],
            roles=data.get("roles", []),
            issued_at=datetime.fromisoformat(data["issued_at"]),
            expires_at=datetime.fromisoformat(data["expires_at"]),
        )


@dataclass
class RateLimitResult:
    """Result of a sliding-window rate-limit increment."""

    count: int
    allowed: bool
    retry_after_seconds: float = 0.0


@dataclass
class ScopeCacheEntry:
    """Cached compiled scope DAG. Key: scope:{engagement_id}:{version}."""

    engagement_id: str
    version: int
    dag: dict[str, Any]  # serialised scope DAG
    computed_at: datetime = field(default_factory=lambda: datetime.now(timezone.utc))

    def to_dict(self) -> dict[str, Any]:
        return {
            "engagement_id": self.engagement_id,
            "version": self.version,
            "dag": self.dag,
            "computed_at": self.computed_at.isoformat(),
        }

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> "ScopeCacheEntry":
        return cls(
            engagement_id=data["engagement_id"],
            version=data["version"],
            dag=data["dag"],
            computed_at=datetime.fromisoformat(data["computed_at"]),
        )


# ─── Namespace helpers ────────────────────────────────────────────────────────


class SessionCache:
    """Manages the session:{user_id} namespace (8-hour TTL)."""

    PREFIX = "session:"

    def __init__(self, client: aioredis.Redis) -> None:
        self._r = client

    def _key(self, user_id: str) -> str:
        return f"{self.PREFIX}{user_id}"

    async def set(self, user_id: str, entry: SessionEntry) -> None:
        """Store a session entry with SESSION_TTL_SECONDS TTL."""
        await self._r.setex(
            self._key(user_id),
            SESSION_TTL_SECONDS,
            json.dumps(entry.to_dict()),
        )

    async def get(self, user_id: str) -> Optional[SessionEntry]:
        """Retrieve a session entry. Returns None on cache miss."""
        raw = await self._r.get(self._key(user_id))
        if raw is None:
            return None
        return SessionEntry.from_dict(json.loads(raw))

    async def delete(self, user_id: str) -> None:
        """Remove the session entry (e.g. on logout)."""
        await self._r.delete(self._key(user_id))

    async def refresh_ttl(self, user_id: str) -> bool:
        """Reset the TTL on an existing session. Returns False if key missing."""
        return bool(await self._r.expire(self._key(user_id), SESSION_TTL_SECONDS))


class RateLimitCache:
    """Manages the ratelimit:{org_id}:{endpoint} namespace.

    Uses a Redis sorted set for sliding-window counting.
    Members are nanosecond timestamps; scores are millisecond timestamps.
    """

    PREFIX = "ratelimit:"

    def __init__(self, client: aioredis.Redis) -> None:
        self._r = client

    def _key(self, org_id: str, endpoint: str) -> str:
        return f"{self.PREFIX}{org_id}:{endpoint}"

    async def increment(
        self,
        org_id: str,
        endpoint: str,
        window_seconds: int,
        limit: int,
    ) -> RateLimitResult:
        """Record a request and check whether it is within the rate limit."""
        now_ms = int(time.time() * 1000)
        window_start_ms = now_ms - (window_seconds * 1000)
        member = str(time.time_ns())
        key = self._key(org_id, endpoint)
        expire_seconds = window_seconds + 1

        pipe = self._r.pipeline()
        pipe.zremrangebyscore(key, "-inf", window_start_ms)
        pipe.zadd(key, {member: now_ms})
        pipe.zcard(key)
        pipe.expire(key, expire_seconds)
        results = await pipe.execute()

        count = int(results[2])
        allowed = count <= limit

        retry_after: float = 0.0
        if not allowed:
            oldest = await self._r.zrangebyscore(
                key, "-inf", "+inf", start=0, num=1, withscores=True
            )
            if oldest:
                oldest_ms = int(oldest[0][1])
                resets_at_ms = oldest_ms + (window_seconds * 1000)
                delta_ms = resets_at_ms - now_ms
                retry_after = max(0.0, delta_ms / 1000)

        return RateLimitResult(count=count, allowed=allowed, retry_after_seconds=retry_after)

    async def count(self, org_id: str, endpoint: str, window_seconds: int) -> int:
        """Return the current request count without recording a new request."""
        window_start_ms = int(time.time() * 1000) - (window_seconds * 1000)
        return int(
            await self._r.zcount(self._key(org_id, endpoint), window_start_ms, "+inf")
        )

    async def reset(self, org_id: str, endpoint: str) -> None:
        """Clear the rate-limit counter for an org+endpoint."""
        await self._r.delete(self._key(org_id, endpoint))


class ScopeCache:
    """Manages the scope:{engagement_id}:{version} namespace (30-minute TTL)."""

    PREFIX = "scope:"

    def __init__(self, client: aioredis.Redis) -> None:
        self._r = client

    def _key(self, engagement_id: str, version: int) -> str:
        return f"{self.PREFIX}{engagement_id}:{version}"

    def _pattern(self, engagement_id: str) -> str:
        return f"{self.PREFIX}{engagement_id}:*"

    async def set(self, engagement_id: str, version: int, entry: ScopeCacheEntry) -> None:
        """Store a compiled scope DAG with SCOPE_TTL_SECONDS TTL."""
        await self._r.setex(
            self._key(engagement_id, version),
            SCOPE_TTL_SECONDS,
            json.dumps(entry.to_dict()),
        )

    async def get(
        self, engagement_id: str, version: int
    ) -> Optional[ScopeCacheEntry]:
        """Retrieve a compiled scope DAG. Returns None on cache miss / expiry."""
        raw = await self._r.get(self._key(engagement_id, version))
        if raw is None:
            return None
        return ScopeCacheEntry.from_dict(json.loads(raw))

    async def invalidate(self, engagement_id: str) -> int:
        """Delete ALL cached scope DAGs for an engagement (all versions).

        Uses SCAN+DEL to avoid blocking. Returns the number of keys deleted.
        """
        pattern = self._pattern(engagement_id)
        deleted = 0
        cursor: int = 0
        while True:
            cursor, keys = await self._r.scan(cursor, match=pattern, count=100)
            if keys:
                deleted += await self._r.delete(*keys)
            if cursor == 0:
                break
        return deleted

    async def invalidate_version(self, engagement_id: str, version: int) -> None:
        """Delete the cache entry for a specific version."""
        await self._r.delete(self._key(engagement_id, version))

    async def ttl(self, engagement_id: str, version: int) -> int:
        """Return remaining TTL in seconds. -2 = key missing, -1 = no expiry."""
        return int(await self._r.ttl(self._key(engagement_id, version)))


class InvalidationHandler:
    """Event-driven cache invalidation helpers.

    Wire to the Redpanda consumer or webhook handler to purge stale entries.
    """

    def __init__(self, session: SessionCache, scope: ScopeCache) -> None:
        self._session = session
        self._scope = scope

    async def on_scope_changed(self, engagement_id: str) -> int:
        """Invalidate all scope DAG entries for the engagement.

        Call whenever scope configuration changes are detected.
        Returns number of keys deleted.
        """
        return await self._scope.invalidate(engagement_id)

    async def on_scope_version_changed(self, engagement_id: str, version: int) -> None:
        """Invalidate a specific scope version."""
        await self._scope.invalidate_version(engagement_id, version)

    async def on_user_logout(self, user_id: str) -> None:
        """Invalidate the session entry for a user on explicit logout."""
        await self._session.delete(user_id)

    async def on_user_deactivated(self, user_id: str) -> None:
        """Invalidate the session entry for a deactivated user."""
        await self._session.delete(user_id)


# ─── Facade ───────────────────────────────────────────────────────────────────


class CacheClient:
    """Top-level cache client that wires together all namespace helpers.

    Usage::

        cache = CacheClient(redis_url="redis://redis:6379")
        await cache.ping()
        await cache.aclose()
    """

    def __init__(
        self,
        redis_url: str = "redis://localhost:6379",
        max_connections: int = 10,
        socket_timeout: float = 3.0,
        socket_connect_timeout: float = 5.0,
        retry_on_timeout: bool = True,
    ) -> None:
        pool = aioredis.ConnectionPool.from_url(
            redis_url,
            max_connections=max_connections,
            socket_timeout=socket_timeout,
            socket_connect_timeout=socket_connect_timeout,
            retry_on_timeout=retry_on_timeout,
            decode_responses=False,
        )
        self._r = aioredis.Redis(connection_pool=pool)
        self.session = SessionCache(self._r)
        self.ratelimit = RateLimitCache(self._r)
        self.scope = ScopeCache(self._r)
        self.invalidation = InvalidationHandler(self.session, self.scope)

    async def ping(self) -> bool:
        """Return True if the Redis server is reachable."""
        return (await self._r.ping()) is True

    async def aclose(self) -> None:
        """Release the connection pool."""
        await self._r.aclose()

    async def __aenter__(self) -> "CacheClient":
        return self

    async def __aexit__(self, *_: Any) -> None:
        await self.aclose()
