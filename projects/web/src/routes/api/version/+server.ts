import { json } from '@sveltejs/kit';

// Prerendered at build time — the build timestamp becomes the version string.
// The layout compares this against the cached value to detect deployments
// and clear stale browser caches.
export const prerender = true;

const BUILD = new Date().toISOString();

export const GET = () => json({ build: BUILD });
