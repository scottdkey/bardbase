import { json } from '@sveltejs/kit';
import { getCorrections } from '$lib/server/api';

export const prerender = false;

export async function GET({ url }) {
	const state = url.searchParams.get('state') ?? 'all';
	const issues = await getCorrections(state);
	return json(issues);
}
