// Server-side Go API client.
// Used only in server load functions and API proxy routes — never shipped to the browser.
import { env } from '$env/dynamic/private';
import type {
	FooterAttribution,
	LexiconEntryDetail,
	MultiEditionScene,
	SearchResult
} from '$lib/types';

function getBaseUrl(): string {
	// process.env is checked first so docker-compose environment vars always win
	// over any .env file that Vite might load into $env/dynamic/private.
	// In Cloudflare Workers process is undefined, so we fall through to env.API_URL.
	const url =
		(typeof process !== 'undefined' ? process.env.API_URL : undefined) ??
		env.API_URL ??
		'http://localhost:8080';
	return url;
}

async function apiFetch<T>(path: string): Promise<T> {
	const base = getBaseUrl();
	const url = `${base}${path}`;
	console.log('[api] ->', url);
	let res: Response;
	try {
		res = await fetch(url);
	} catch (err) {
		console.error('[api] unreachable:', url, err);
		throw new Error(`API unreachable at ${url}: ${err}`);
	}
	if (!res.ok) {
		console.error('[api] error', res.status, url);
		throw new Error(`API error ${res.status}: ${path}`);
	}
	return res.json() as Promise<T>;
}

export interface LexiconLetter {
	letter: string;
	count: number;
}

export interface CorrectionIssue {
	number: number;
	title: string;
	state: string;
	url: string;
	created_at: string;
	updated_at: string;
	labels: string[];
	body: string;
}

export interface Work {
	id: number;
	title: string;
	slug: string;
	work_type: string;
	date_composed: string | null;
}

export interface WorkEdition {
	id: number;
	name: string;
	short_code: string;
	year: number | null;
	source_name: string;
}

export interface LineReference {
	entry_id: number;
	entry_key: string;
	source: string;
	source_code: string;
	sense_id: number | null;
	definition: string | null;
	quote_text: string | null;
	line: number;
}

export interface ReferenceSource {
	code: string;
	name: string;
	count: number;
}

export interface ReferenceEntryDetail {
	id: number;
	headword: string;
	raw_text: string;
	source_name: string;
	source_code: string;
}

export interface WorkDivision {
	act: number;
	scene: number;
	description: string | null;
	line_count: number;
}

export const api = {
	getAttributions: () => apiFetch<FooterAttribution[]>('/api/attributions'),
	getCorrections: (state = 'all') =>
		apiFetch<CorrectionIssue[]>(`/api/corrections?state=${state}`),
	getLexiconEntry: (id: number) => apiFetch<LexiconEntryDetail>(`/api/lexicon/entry/${id}`),
	getReferenceEntry: (id: number) =>
		apiFetch<ReferenceEntryDetail>(`/api/reference/entry/${id}`),
	getReferenceSources: () =>
		apiFetch<ReferenceSource[]>('/api/reference/sources'),
	getLexiconKeys: () => apiFetch<string[]>('/api/lexicon/keys'),
	getLexiconLetters: () => apiFetch<LexiconLetter[]>('/api/lexicon/letters'),
	getScene: (workId: number, act: number, scene: number) =>
		apiFetch<MultiEditionScene>(`/api/text/scene/${workId}/${act}/${scene}`),
	getSceneReferences: (workId: number, act: number, scene: number) =>
		apiFetch<Record<string, LineReference[]>>(
			`/api/text/scene/${workId}/${act}/${scene}/references`
		),
	getWorkBySlug: (slug: string) =>
		apiFetch<{ id: number; title: string; slug: string }>(`/api/resolve/${slug}`),
	getWorks: () => apiFetch<{ plays: Work[]; poetry: Work[] }>('/api/works'),
	getWorkEditions: (id: number) => apiFetch<WorkEdition[]>(`/api/works/${id}/editions`),
	getWorkTOC: (id: number) => apiFetch<WorkDivision[]>(`/api/works/${id}/toc`),
	search: (q: string, limit = 20) =>
		apiFetch<SearchResult[]>(`/api/search?q=${encodeURIComponent(q)}&limit=${limit}`)
};
