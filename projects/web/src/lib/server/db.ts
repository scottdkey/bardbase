import { env } from '$env/dynamic/private';
import type { D1Database } from '@cloudflare/workers-types';

/**
 * DB access, with two isolated backends:
 *
 *   - Prod (Cloudflare Workers): libSQL `/web` client against Turso. Uses
 *     fetch only — no Node APIs — so it works in the Workers runtime.
 *
 *   - Local dev (adapter-node): node:sqlite opened directly on
 *     build/bardbase-search.db. No network, no server, no auth token.
 *
 * Both paths return a client that exposes the D1Database surface (prepare /
 * bind / first / all) so `api.ts` stays backend-agnostic. Dynamic imports
 * keep each backend's dependencies out of the other's bundle — the CF
 * Workers build never sees `node:sqlite`, the Node dev build never pulls
 * in the web client's fetch plumbing.
 */

let shim: D1Database | null = null;

export function getDb(_platform?: App.Platform | undefined): D1Database {
	if (!shim) shim = env.TURSO_URL ? makeTursoShim() : makeLocalShim();
	return shim;
}

// ── Turso (prod) ─────────────────────────────────────────────────────────────

function makeTursoShim(): D1Database {
	const clientPromise = import('@libsql/client/web').then((m) =>
		m.createClient({ url: env.TURSO_URL!, authToken: env.TURSO_AUTH_TOKEN })
	);

	type LibSqlArg = string | number | bigint | boolean | ArrayBuffer | null;

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

// ── Local node:sqlite (dev) ──────────────────────────────────────────────────

function makeLocalShim(): D1Database {
	type SQLVal = import('node:sqlite').SQLInputValue;
	type DB = import('node:sqlite').DatabaseSync;

	const dbPromise: Promise<DB> = (async () => {
		const [{ DatabaseSync }, { join }] = await Promise.all([
			import('node:sqlite'),
			import('node:path')
		]);
		const path = env.DB_PATH ?? join(process.cwd(), '../../build/bardbase-search.db');
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
