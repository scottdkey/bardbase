import { api } from '$lib/server/api';

export async function load() {
	return { attributions: await api.getAttributions() };
}
