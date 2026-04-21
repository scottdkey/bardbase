// See https://svelte.dev/docs/kit/types#app.d.ts
// for information about these interfaces
declare global {
	namespace App {
		// interface Error {}
		// interface Locals {}
		// interface PageData {}
		// interface PageState {}
		// Platform is kept for SvelteKit load-function typing but is no longer
		// used — DB access is owned by $lib/server/db, which uses TURSO_URL +
		// TURSO_AUTH_TOKEN from $env/dynamic/private.
		interface Platform {
			env?: Record<string, unknown>;
		}
	}
}

export {};
