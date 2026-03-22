import { json, error } from '@sveltejs/kit';
import { dev } from '$app/environment';

export const prerender = false;

export async function POST({ request }) {
	if (!dev) {
		throw error(403, 'Corrections API only available in development');
	}

	const { writeFileSync, mkdirSync } = await import('node:fs');
	const { resolve } = await import('node:path');

	const dir = resolve(import.meta.dirname, '../../../../../../corrections');
	const correction = await request.json();
	mkdirSync(dir, { recursive: true });
	const filepath = resolve(dir, `${correction.id}.json`);
	writeFileSync(filepath, JSON.stringify(correction, null, 2));
	return json({ ok: true, path: filepath });
}

export async function GET() {
	if (!dev) {
		throw error(403, 'Corrections API only available in development');
	}

	const { mkdirSync, readdirSync, readFileSync } = await import('node:fs');
	const { resolve } = await import('node:path');

	const dir = resolve(import.meta.dirname, '../../../../../../corrections');
	mkdirSync(dir, { recursive: true });
	const files = readdirSync(dir).filter((f: string) => f.endsWith('.json'));
	const corrections = files.map((f: string) => {
		const content = readFileSync(resolve(dir, f), 'utf-8');
		return JSON.parse(content);
	});
	return json({ corrections });
}
