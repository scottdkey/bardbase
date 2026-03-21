import { json, error } from '@sveltejs/kit';
import { dev } from '$app/environment';
import { writeFileSync, mkdirSync, readFileSync, readdirSync } from 'node:fs';
import { resolve } from 'node:path';

const CORRECTIONS_DIR = resolve(import.meta.dirname, '../../../../../../corrections');

export function POST({ request }) {
	if (!dev) {
		throw error(403, 'Corrections API only available in development');
	}

	return request.json().then((correction) => {
		mkdirSync(CORRECTIONS_DIR, { recursive: true });

		const filename = `${correction.id}.json`;
		const filepath = resolve(CORRECTIONS_DIR, filename);
		writeFileSync(filepath, JSON.stringify(correction, null, 2));

		return json({ ok: true, path: filepath });
	});
}

export function GET() {
	if (!dev) {
		throw error(403, 'Corrections API only available in development');
	}

	mkdirSync(CORRECTIONS_DIR, { recursive: true });

	const files = readdirSync(CORRECTIONS_DIR).filter((f: string) => f.endsWith('.json'));
	const corrections = files.map((f: string) => {
		const content = readFileSync(resolve(CORRECTIONS_DIR, f), 'utf-8');
		return JSON.parse(content);
	});

	return json({ corrections });
}
