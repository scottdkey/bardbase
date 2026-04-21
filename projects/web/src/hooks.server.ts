import type { Handle } from '@sveltejs/kit';

// In production (Cloudflare Workers), platform.env.SEARCH_DB is the real D1 binding.
// In local dev (Node/Vite), platform is undefined. This hook injects a D1-compatible
// shim backed by build/bardbase.db so all D1 code paths work without modification.

type SQLVal = import('node:sqlite').SQLInputValue;
type DB = import('node:sqlite').DatabaseSync;

let devDb: DB | null = null;
let dbInitPromise: Promise<void> | null = null;

async function initDevDb(): Promise<void> {
	try {
		const [{ DatabaseSync }, { existsSync, realpathSync }, { join }] = await Promise.all([
			import('node:sqlite'),
			import('node:fs'),
			import('node:path')
		]);

		const dbPath = process.env.DB_PATH ?? join(process.cwd(), '../../build/bardbase.db');
		if (!existsSync(dbPath)) {
			console.warn('[dev] No database found at', dbPath);
			return;
		}
		devDb = new DatabaseSync(realpathSync(dbPath));
	} catch (err) {
		console.warn('[dev] Could not open database:', err);
	}
}

function makeD1Shim(db: DB) {
	function makeStmt(sql: string, args: SQLVal[]) {
		return {
			bind(...newArgs: SQLVal[]) {
				return makeStmt(sql, newArgs);
			},
			async first<T>(): Promise<T | null> {
				const row = db.prepare(sql).get(...args) as T | undefined;
				return row ?? null;
			},
			async all<T>(): Promise<{ results: T[] }> {
				const rows = db.prepare(sql).all(...args) as T[];
				return { results: rows };
			},
			async run() {
				return db.prepare(sql).run(...args);
			}
		};
	}
	return { prepare: (sql: string) => makeStmt(sql, [] as SQLVal[]) };
}

export const handle: Handle = async ({ event, resolve }) => {
	// During prerender, adapter-cloudflare makes env property access throw intentionally.
	// Wrap in try/catch so prerendered routes get the node:sqlite fallback gracefully.
	let hasD1 = false;
	try { hasD1 = !!event.platform?.env?.SEARCH_DB; } catch { hasD1 = false; }

	if (!hasD1) {
		if (!dbInitPromise) dbInitPromise = initDevDb();
		await dbInitPromise;

		if (devDb) {
			const shim = makeD1Shim(devDb) as unknown as App.Platform['env']['SEARCH_DB'];
			// Don't spread event.platform?.env — during prerender its property getters throw.
			// We're in fallback mode so there are no other real bindings to preserve.
			event.platform = {
				...(event.platform ?? {}),
				env: { SEARCH_DB: shim } as App.Platform['env']
			} as App.Platform;
		}
	}
	return resolve(event);
};
