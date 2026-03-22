// Server-side Go API client.
// Used only in server load functions and API proxy routes — never shipped to the browser.
import { env } from '$env/dynamic/private';
import type {
	FooterAttribution,
	LexiconEntryDetail,
	MultiEditionScene,
	SearchResult
} from '$lib/types';

async function apiFetch<T>(path: string): Promise<T> {
	// Read per-call so Cloudflare Workers gets the request-scoped env binding.
	const base = env.API_URL ?? 'http://localhost:8080';
	const url = `${base}${path}`;
	let res: Response;
	try {
		res = await fetch(url);
	} catch (err) {
		throw new Error(`API unreachable at ${url}: ${err}`);
	}
	if (!res.ok) throw new Error(`API error ${res.status}: ${path}`);
	return res.json() as Promise<T>;
}

export const api = {
	getAttributions: () => apiFetch<FooterAttribution[]>('/api/attributions'),
	getLexiconEntry: (id: number) => apiFetch<LexiconEntryDetail>(`/api/lexicon/entry/${id}`),
	getScene: (workId: number, act: number, scene: number) =>
		apiFetch<MultiEditionScene>(`/api/text/scene/${workId}/${act}/${scene}`),
	search: (q: string, limit = 20) =>
		apiFetch<SearchResult[]>(`/api/search?q=${encodeURIComponent(q)}&limit=${limit}`)
};
