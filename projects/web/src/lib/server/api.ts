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
	const url =
		(typeof process !== 'undefined' ? process.env.API_URL : undefined) ??
		env.API_URL ??
		'http://localhost:8080';
	return url;
}

function getApiKey(): string | undefined {
	return (
		(typeof process !== 'undefined' ? process.env.API_KEY : undefined) ??
		env.API_KEY ??
		undefined
	);
}

// Exposed for proxy routes that need to pass through query params
export async function apiProxy(path: string): Promise<Response> {
	const base = getBaseUrl();
	const url = `${base}${path}`;
	const headers: Record<string, string> = {};
	const apiKey = getApiKey();
	if (apiKey) headers['Authorization'] = `Bearer ${apiKey}`;
	return fetch(url, { headers });
}

async function apiFetch<T>(path: string): Promise<T> {
	const base = getBaseUrl();
	const url = `${base}${path}`;
	const headers: Record<string, string> = {};
	const apiKey = getApiKey();
	if (apiKey) {
		headers['Authorization'] = `Bearer ${apiKey}`;
	}
	let res: Response;
	try {
		res = await fetch(url, { headers });
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

export interface ReferenceEntryCitation {
	work_title: string | null;
	act: number | null;
	scene: number | null;
	line: number | null;
	work_slug: string | null;
}

export interface CitationSpan {
	start: number;
	end: number;
	work_slug?: string;
	act?: number;
	scene?: number;
	line?: number;
}

export interface ReferenceEntryDetail {
	id: number;
	headword: string;
	raw_text: string;
	source_name: string;
	source_code: string;
	citations: ReferenceEntryCitation[];
	citation_spans: CitationSpan[];
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
	getScene: (workIdOrSlug: number | string, act: number, scene: number) =>
		apiFetch<MultiEditionScene>(`/api/text/scene/${workIdOrSlug}/${act}/${scene}`),
	getSceneReferences: (workIdOrSlug: number | string, act: number, scene: number) =>
		apiFetch<Record<string, LineReference[]>>(
			`/api/text/scene/${workIdOrSlug}/${act}/${scene}/references`
		),
	getWorkBySlug: (slug: string) =>
		apiFetch<{ id: number; title: string; slug: string }>(`/api/resolve/${slug}`),
	getWorks: () => apiFetch<{ plays: Work[]; poetry: Work[] }>('/api/works'),
	getWorkEditions: (idOrSlug: number | string) =>
		apiFetch<WorkEdition[]>(`/api/works/${idOrSlug}/editions`),
	getWorkTOC: (idOrSlug: number | string) =>
		apiFetch<WorkDivision[]>(`/api/works/${idOrSlug}/toc`),
	search: (q: string, limit = 20) =>
		apiFetch<SearchResult[]>(`/api/search?q=${encodeURIComponent(q)}&limit=${limit}`)
};
