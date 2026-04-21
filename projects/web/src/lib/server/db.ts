import { createClient, type Client as LibSqlClient, type InValue } from '@libsql/client';
import { env } from '$env/dynamic/private';
import type { D1Database } from '@cloudflare/workers-types';

/**
 * DB access is served via libSQL (Turso in prod, local SQLite file in dev).
 *
 * In prod (Cloudflare Workers), TURSO_URL + TURSO_AUTH_TOKEN point at the
 * Turso DB. In local dev, we fall back to `file:../../build/bardbase.db` —
 * libSQL speaks the SQLite file format directly, so there's no shim layer
 * between dev and prod.
 *
 * The libSQL client is wrapped in a thin adapter that exposes the D1Database
 * surface (`prepare().bind().all()/first()`), so `api.ts` doesn't care which
 * backend is underneath.
 */

let client: LibSqlClient | null = null;
let shim: D1Database | null = null;

function getClient(): LibSqlClient {
    if (client) return client;
    const url = env.TURSO_URL ?? 'file:../../build/bardbase-search.db';
    client = createClient({ url, authToken: env.TURSO_AUTH_TOKEN });
    return client;
}

function makeStmt(c: LibSqlClient, sql: string, args: InValue[]) {
    return {
        bind(...newArgs: InValue[]) {
            return makeStmt(c, sql, newArgs);
        },
        async first<T>(): Promise<T | null> {
            const res = await c.execute({ sql, args });
            const row = res.rows[0];
            return row ? (Object.fromEntries(res.columns.map((col, i) => [col, row[i]])) as T) : null;
        },
        async all<T>(): Promise<{ results: T[] }> {
            const res = await c.execute({ sql, args });
            const results = res.rows.map(
                (r) => Object.fromEntries(res.columns.map((col, i) => [col, r[i]])) as T
            );
            return { results };
        },
        async run() {
            return c.execute({ sql, args });
        }
    };
}

function makeShim(c: LibSqlClient): D1Database {
    return { prepare: (sql: string) => makeStmt(c, sql, []) } as unknown as D1Database;
}

/**
 * Return a D1-shaped DB client. The `platform` arg is accepted for call-site
 * consistency but unused — the client is a module-level singleton.
 */
export function getDb(_platform?: App.Platform | undefined): D1Database {
    if (shim) return shim;
    shim = makeShim(getClient());
    return shim;
}
