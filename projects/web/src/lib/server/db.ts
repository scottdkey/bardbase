import { env } from '$env/dynamic/private';
import type { D1Database } from '@cloudflare/workers-types';

/**
 * DB access with two non-overlapping backends, discriminated by env var:
 *
 *   - `DB_PATH` set → open that SQLite file directly via `node:sqlite`.
 *     Used only at build time (prerendering in CI / local dev). Local file
 *     reads are ~100× faster than Turso HTTP, so prerendering 3K scenes
 *     takes seconds instead of minutes.
 *
 *   - Otherwise → libSQL `/web` client against Turso. Used at runtime in
 *     the Cloudflare Workers environment. Requires `TURSO_URL` +
 *     `TURSO_AUTH_TOKEN`.
 *
 * CI sets DB_PATH=/tmp/bardbase.db and TURSO_URL/TURSO_AUTH_TOKEN. The
 * build uses DB_PATH; the deployed Worker sees only TURSO_*.
 *
 * Both backends return a client that exposes the D1Database surface
 * (prepare / bind / first / all / run) so `api.ts` stays backend-agnostic.
 * Dynamic imports keep each backend's dependencies out of the other's
 * bundle — the CF Workers build never statically pulls in `node:sqlite`.
 */

let shim: D1Database | null = null;

export function getDb(_platform?: App.Platform | undefined): D1Database {
	if (!shim) shim = env.DB_PATH ? makeLocalShim(env.DB_PATH) : makeTursoShim();
	return shim;
}

// ── Turso (runtime) ──────────────────────────────────────────────────────────

function makeTursoShim(): D1Database {
	type LibSqlArg = string | number | bigint | boolean | ArrayBuffer | null;

	const clientPromise = (async () => {
		if (!env.TURSO_URL) {
			throw new Error(
				'TURSO_URL is not set. Required at runtime on CF Pages (Production + Preview env). For local dev, set TURSO_URL + TURSO_AUTH_TOKEN in projects/web/.env.local OR set DB_PATH to a local bardbase.db file.'
			);
		}
		const { createClient } = await import('@libsql/client/web');
		return createClient({ url: env.TURSO_URL, authToken: env.TURSO_AUTH_TOKEN });
	})();

	function stmt(sql: string, args: LibSqlArg[]) {
		return {
			bind(...newArgs: LibSqlArg[]) {
				return stmt(sql, newArgs);
			},
			async first<T>(): Promise<T | null> {
				const c = await clientPromise;
				const res = await c.execute({ sql, args });
				const row = res.rows[0];
				return row
					? (Object.fromEntries(res.columns.map((col, i) => [col, row[i]])) as T)
					: null;
			},
			async all<T>(): Promise<{ results: T[] }> {
				const c = await clientPromise;
				const res = await c.execute({ sql, args });
				const results = res.rows.map(
					(r) => Object.fromEntries(res.columns.map((col, i) => [col, r[i]])) as T
				);
				return { results };
			},
			async run() {
				const c = await clientPromise;
				return c.execute({ sql, args });
			}
		};
	}

	return { prepare: (sql: string) => stmt(sql, []) } as unknown as D1Database;
}

// ── node:sqlite (build time, local dev) ──────────────────────────────────────

function makeLocalShim(path: string): D1Database {
	type SQLVal = import('node:sqlite').SQLInputValue;
	type DB = import('node:sqlite').DatabaseSync;

	const dbPromise: Promise<DB> = (async () => {
		const { DatabaseSync } = await import('node:sqlite');
		return new DatabaseSync(path);
	})();

	function stmt(sql: string, args: SQLVal[]) {
		return {
			bind(...newArgs: SQLVal[]) {
				return stmt(sql, newArgs);
			},
			async first<T>(): Promise<T | null> {
				const db = await dbPromise;
				const row = db.prepare(sql).get(...args) as T | undefined;
				return row ?? null;
			},
			async all<T>(): Promise<{ results: T[] }> {
				const db = await dbPromise;
				const results = db.prepare(sql).all(...args) as T[];
				return { results };
			},
			async run() {
				const db = await dbPromise;
				return db.prepare(sql).run(...args);
			}
		};
	}

	return { prepare: (sql: string) => stmt(sql, []) } as unknown as D1Database;
}
