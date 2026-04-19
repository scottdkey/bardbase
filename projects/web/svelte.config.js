import adapterCloudflare from '@sveltejs/adapter-cloudflare';
import adapterNode from '@sveltejs/adapter-node';

// Use adapter-cloudflare only when building on Cloudflare Pages (CF_PAGES=1).
// Locally and in Docker, adapter-node avoids the workerd binary requirement.
const adapter = process.env.CF_PAGES
	? adapterCloudflare({
			routes: { include: ['/*'], exclude: ['<all>'] },
			// Disable platformProxy — prerendered routes read from node:sqlite, not D1.
			// Leaving it enabled spawns a workerd child process that never terminates,
			// causing `vite build` to hang after prerender completes.
			platformProxy: false
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
			concurrency: 36,
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
