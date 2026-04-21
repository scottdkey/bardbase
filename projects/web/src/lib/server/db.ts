import { createClient, type Client as LibSqlClient, type InValue } from '@libsql/client/web';
import { env } from '$env/dynamic/private';
import type { D1Database } from '@cloudflare/workers-types';

/**
 * DB access via libSQL's `/web` client — fetch-based, works on both the
 * Cloudflare Workers runtime (prod) and Node (local dev). Both environments
 * must set TURSO_URL + TURSO_AUTH_TOKEN.
 *
 * For local dev: put them in `projects/web/.env.local`. Point at the remote
 * Turso DB, or run `turso dev --db-file ../../build/bardbase-search.db --port
 * 8080` and set TURSO_URL=http://127.0.0.1:8080.
 *
 * The libSQL client is wrapped in a thin adapter exposing the D1Database
 * surface (prepare/bind/first/all) so `api.ts` stays backend-agnostic.
 */

let client: LibSqlClient | null = null;
let shim: D1Database | null = null;

function getClient(): LibSqlClient {
	if (client) return client;
	if (!env.TURSO_URL) {
		throw new Error(
			'TURSO_URL is not set. Set it in projects/web/.env.local (local dev) or the CF Pages dashboard (prod).'
		);
	}
	client = createClient({ url: env.TURSO_URL, authToken: env.TURSO_AUTH_TOKEN });
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
			return row
				? (Object.fromEntries(res.columns.map((col, i) => [col, row[i]])) as T)
				: null;
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

export function getDb(_platform?: App.Platform | undefined): D1Database {
	if (shim) return shim;
	const c = getClient();
	shim = { prepare: (sql: string) => makeStmt(c, sql, []) } as unknown as D1Database;
	return shim;
}
