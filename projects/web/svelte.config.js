import adapterCloudflare from '@sveltejs/adapter-cloudflare';
import adapterNode from '@sveltejs/adapter-node';

// Use adapter-cloudflare only when building on Cloudflare Pages (CF_PAGES=1).
// Locally and in Docker, adapter-node avoids the workerd binary requirement.
const adapter = process.env.CF_PAGES
	? adapterCloudflare({
			routes: { include: ['/*'], exclude: ['<all>'] },
			// Use in-memory D1 during prerendering so concurrent workers don't
			// fight over the same Miniflare SQLite state files (SQLITE_BUSY).
			platformProxy: { persist: false }
	  })
	: adapterNode();

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		adapter,
		paths: {
			base: ''
		},
		prerender: {
			concurrency: 4,
			handleHttpError: ({ status, path, referrer }) => {
				if (status === 404) {
					console.warn(`[prerender] 404 ${path} (linked from ${referrer})`);
					return;
				}
				throw new Error(`${status} ${path}`);
			}
		}
	},
	vitePlugin: {
		dynamicCompileOptions: ({ filename }) =>
			filename.includes('node_modules') ? undefined : { runes: true }
	}
};

export default config;
