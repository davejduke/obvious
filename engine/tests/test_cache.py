"""Tests for engine.shared.cache — session, ratelimit, scope, and invalidation."""
from __future__ import annotations

import asyncio
from datetime import datetime, timedelta, timezone
from typing import Any
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from engine.shared.cache import (
    SESSION_TTL_SECONDS,
    SCOPE_TTL_SECONDS,
    CacheClient,
    InvalidationHandler,
    RateLimitCache,
    ScopeCacheEntry,
    SessionCache,
    SessionEntry,
    ScopeCache,
)


# ─── Fixtures ────────────────────────────────────────────────────────────────


def make_redis_mock() -> AsyncMock:
    """Return an AsyncMock that mimics an aioredis.Redis interface."""
    mock = AsyncMock()
    # pipeline() is called synchronously; return a sync-callable pipeline mock.
    mock.pipeline = MagicMock(return_value=AsyncMock())
    return mock


def sample_session(user_id: str = "user-1") -> SessionEntry:
    return SessionEntry(
        user_id=user_id,
        org_id="org-1",
        token_id="jti-abc",
        roles=["internal_auditor"],
        issued_at=datetime.now(timezone.utc),
        expires_at=datetime.now(timezone.utc) + timedelta(hours=8),
    )


def sample_scope(engagement_id: str = "eng-1", version: int = 1) -> ScopeCacheEntry:
    return ScopeCacheEntry(
        engagement_id=engagement_id,
        version=version,
        dag={"nodes": ["ctrl-1", "ctrl-2"], "edges": [["ctrl-1", "ctrl-2"]]},
        computed_at=datetime.now(timezone.utc),
    )


# ─── SessionCache ─────────────────────────────────────────────────────────────


class TestSessionCache:
    @pytest.mark.asyncio
    async def test_set_calls_setex_with_ttl(self) -> None:
        r = make_redis_mock()
        cache = SessionCache(r)
        entry = sample_session()

        await cache.set("user-1", entry)

        r.setex.assert_called_once()
        call_args = r.setex.call_args
        assert call_args[0][0] == "session:user-1"
        assert call_args[0][1] == SESSION_TTL_SECONDS

    @pytest.mark.asyncio
    async def test_get_returns_none_on_miss(self) -> None:
        r = make_redis_mock()
        r.get.return_value = None
        cache = SessionCache(r)

        result = await cache.get("nonexistent")

        assert result is None

    @pytest.mark.asyncio
    async def test_get_returns_entry_on_hit(self) -> None:
        import json

        r = make_redis_mock()
        entry = sample_session()
        r.get.return_value = json.dumps(entry.to_dict()).encode()
        cache = SessionCache(r)

        result = await cache.get("user-1")

        assert result is not None
        assert result.user_id == "user-1"
        assert result.token_id == "jti-abc"

    @pytest.mark.asyncio
    async def test_delete_calls_del(self) -> None:
        r = make_redis_mock()
        cache = SessionCache(r)

        await cache.delete("user-1")

        r.delete.assert_called_once_with("session:user-1")

    @pytest.mark.asyncio
    async def test_refresh_ttl_calls_expire(self) -> None:
        r = make_redis_mock()
        r.expire.return_value = 1
        cache = SessionCache(r)

        result = await cache.refresh_ttl("user-1")

        assert result is True
        r.expire.assert_called_once_with("session:user-1", SESSION_TTL_SECONDS)


# ─── RateLimitCache ───────────────────────────────────────────────────────────


class TestRateLimitCache:
    def _make_pipe(self, count: int) -> AsyncMock:
        pipe = AsyncMock()
        # pipeline execute returns [zremrangebyscore, zadd, zcard, expire]
        pipe.execute.return_value = [0, 1, count, 1]
        return pipe

    @pytest.mark.asyncio
    async def test_allowed_when_under_limit(self) -> None:
        r = make_redis_mock()
        r.pipeline = MagicMock(return_value=self._make_pipe(count=3))
        cache = RateLimitCache(r)

        result = await cache.increment("org-1", "/api/findings", 60, limit=10)

        assert result.allowed is True
        assert result.count == 3

    @pytest.mark.asyncio
    async def test_denied_when_over_limit(self) -> None:
        r = make_redis_mock()
        r.pipeline = MagicMock(return_value=self._make_pipe(count=11))
        import time
        oldest_ms = int(time.time() * 1000) - 30_000  # 30s ago
        r.zrangebyscore.return_value = [(b"member", float(oldest_ms))]
        cache = RateLimitCache(r)

        result = await cache.increment("org-1", "/api/findings", 60, limit=10)

        assert result.allowed is False
        assert result.retry_after_seconds > 0

    @pytest.mark.asyncio
    async def test_reset_deletes_key(self) -> None:
        r = make_redis_mock()
        cache = RateLimitCache(r)

        await cache.reset("org-1", "/api/findings")

        r.delete.assert_called_once_with("ratelimit:org-1:/api/findings")

    @pytest.mark.asyncio
    async def test_count_uses_zcount(self) -> None:
        r = make_redis_mock()
        r.zcount.return_value = 5
        cache = RateLimitCache(r)

        count = await cache.count("org-1", "/api/findings", 60)

        assert count == 5
        r.zcount.assert_called_once()


# ─── ScopeCache ───────────────────────────────────────────────────────────────


class TestScopeCache:
    @pytest.mark.asyncio
    async def test_set_uses_scope_ttl(self) -> None:
        r = make_redis_mock()
        cache = ScopeCache(r)
        entry = sample_scope()

        await cache.set("eng-1", 1, entry)

        r.setex.assert_called_once()
        call_args = r.setex.call_args
        assert call_args[0][0] == "scope:eng-1:1"
        assert call_args[0][1] == SCOPE_TTL_SECONDS

    @pytest.mark.asyncio
    async def test_get_returns_none_on_miss(self) -> None:
        r = make_redis_mock()
        r.get.return_value = None
        cache = ScopeCache(r)

        result = await cache.get("eng-1", 1)

        assert result is None

    @pytest.mark.asyncio
    async def test_get_deserialises_entry(self) -> None:
        import json

        r = make_redis_mock()
        entry = sample_scope()
        r.get.return_value = json.dumps(entry.to_dict()).encode()
        cache = ScopeCache(r)

        result = await cache.get("eng-1", 1)

        assert result is not None
        assert result.engagement_id == "eng-1"
        assert result.version == 1

    @pytest.mark.asyncio
    async def test_invalidate_scans_and_deletes(self) -> None:
        r = make_redis_mock()
        # First scan returns 2 keys, cursor=0 signals completion.
        r.scan.return_value = (0, [b"scope:eng-1:1", b"scope:eng-1:2"])
        r.delete.return_value = 2
        cache = ScopeCache(r)

        deleted = await cache.invalidate("eng-1")

        assert deleted == 2
        r.scan.assert_called_once()
        r.delete.assert_called_once()

    @pytest.mark.asyncio
    async def test_invalidate_version_deletes_specific_key(self) -> None:
        r = make_redis_mock()
        cache = ScopeCache(r)

        await cache.invalidate_version("eng-1", 3)

        r.delete.assert_called_once_with("scope:eng-1:3")


# ─── InvalidationHandler ──────────────────────────────────────────────────────


class TestInvalidationHandler:
    @pytest.mark.asyncio
    async def test_on_scope_changed_invalidates_all_versions(self) -> None:
        r = make_redis_mock()
        r.scan.return_value = (0, [b"scope:eng-1:1", b"scope:eng-1:2"])
        r.delete.return_value = 2
        session = SessionCache(r)
        scope = ScopeCache(r)
        handler = InvalidationHandler(session, scope)

        deleted = await handler.on_scope_changed("eng-1")

        assert deleted == 2

    @pytest.mark.asyncio
    async def test_on_user_logout_removes_session(self) -> None:
        r = make_redis_mock()
        session = SessionCache(r)
        scope = ScopeCache(r)
        handler = InvalidationHandler(session, scope)

        await handler.on_user_logout("user-logout")

        r.delete.assert_called_once_with("session:user-logout")


# ─── CacheClient TTL constants ────────────────────────────────────────────────


def test_session_ttl_is_eight_hours() -> None:
    assert SESSION_TTL_SECONDS == 8 * 60 * 60


def test_scope_ttl_is_thirty_minutes() -> None:
    assert SCOPE_TTL_SECONDS == 30 * 60
