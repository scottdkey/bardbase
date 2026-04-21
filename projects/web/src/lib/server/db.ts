import type { D1Database } from '@cloudflare/workers-types';
import { error } from '@sveltejs/kit';

/**
 * Resolve the D1 search database binding.
 *
 * In production (Cloudflare Workers), platform.env.SEARCH_DB is the real
 * D1 binding. In local dev (Node), hooks.server.ts injects a D1-compatible
 * shim backed by node:sqlite reading build/bardbase.db. Either way, this
 * function returns something that speaks the D1 interface.
 *
 * Throws 503 if no DB is available — load functions should let this bubble up.
 */
export function getDb(platform: App.Platform | undefined): D1Database {
    const db = platform?.env?.SEARCH_DB;
    if (!db) throw error(503, 'Search database unavailable');
    return db;
}
