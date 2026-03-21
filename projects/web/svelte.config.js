import adapter from '@sveltejs/adapter-static';
import Database from 'better-sqlite3';
import { resolve } from 'path';

function getPrerenderedEntries() {
	const dbPath = process.env.BARDBASE_DB_PATH ?? resolve('../../build/bardbase.db');
	try {
		const db = new Database(dbPath, { readonly: true });
		const entries = [];

		// All lexicon entry IDs
		const entryRows = db.prepare('SELECT MIN(id) as id FROM lexicon_entries GROUP BY base_key').all();
		for (const row of entryRows) {
			entries.push(`/api/lexicon/entry/${row.id}`);
		}

		// All work/act/scene combinations for text viewer
		const sceneRows = db.prepare(`
			SELECT DISTINCT work_id, act, scene
			FROM text_lines
			WHERE edition_id IN (1, 2, 3, 4, 5)
			  AND act IS NOT NULL AND scene IS NOT NULL
			GROUP BY work_id, act, scene
		`).all();
		for (const row of sceneRows) {
			entries.push(`/api/text/scene/${row.work_id}/${row.act}/${row.scene}`);
		}

		db.close();
		return entries;
	} catch {
		console.warn('Could not read database for prerender entries');
		return [];
	}
}

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		adapter: adapter({
			pages: 'build',
			assets: 'build',
			fallback: '404.html',
			precompress: false,
			strict: false
		}),
		prerender: {
			entries: ['/', '/glossary', '/editions', '/corrections', ...getPrerenderedEntries()],
			handleHttpError: 'warn'
		},
		paths: {
			// Cloudflare Pages serves from root
			base: ''
		}
	},
	vitePlugin: {
		dynamicCompileOptions: ({ filename }) =>
			filename.includes('node_modules') ? undefined : { runes: true }
	}
};

export default config;
