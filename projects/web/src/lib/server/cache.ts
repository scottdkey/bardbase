/**
 * Cache-Control header profiles for SvelteKit `setHeaders` / `Response` init.
 *
 * Reality check: the content in this app only changes when a new `bardbase.db`
 * ships — i.e. on deploy. There are no runtime writes, no user-specific
 * content, nothing time-sensitive. Cache aggressively and rely on deploy-time
 * purge (CF Cache Rules / Pages purge-on-deploy) to invalidate.
 *
 * Profile choices:
 *   - Browser `max-age`: 1 hour. Long enough that back/forward and same-session
 *     revisits are instant; short enough that a user picking the tab back up
 *     tomorrow gets fresh content without needing to force-refresh.
 *   - Edge `s-maxage`: 30 days. The edge is what we actually care about —
 *     one hit per PoP per month, invalidated on deploy.
 *   - `stale-while-revalidate`: 1 day grace. If the cache expires between
 *     deploys (it won't, but), serve stale while revalidating in the
 *     background. Zero user-visible latency.
 *   - `immutable` is NOT used — we want purge-on-deploy to work.
 */

// Scene HTML, reference entry pages, lexicon entries, meta endpoints, TOC.
// Effectively immutable between deploys.
export const CACHE_STATIC =
    'public, max-age=3600, s-maxage=2592000, stale-while-revalidate=86400';

// Search responses — parameterized by query, so one cache entry per query+offset.
// Still safe to cache for a long time (same query → same answer until deploy).
export const CACHE_SHORT =
    'public, max-age=600, s-maxage=2592000, stale-while-revalidate=86400';
