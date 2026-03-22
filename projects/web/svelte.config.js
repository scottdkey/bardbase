import adapterCloudflare from '@sveltejs/adapter-cloudflare';
import adapterNode from '@sveltejs/adapter-node';

// Use adapter-cloudflare only when building on Cloudflare Pages (CF_PAGES=1).
// Locally and in Docker, adapter-node avoids the workerd binary requirement.
const adapter = process.env.CF_PAGES
	? adapterCloudflare({ routes: { include: ['/*'], exclude: ['<all>'] } })
	: adapterNode();

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		adapter,
		paths: {
			base: ''
		}
	},
	vitePlugin: {
		dynamicCompileOptions: ({ filename }) =>
			filename.includes('node_modules') ? undefined : { runes: true }
	}
};

export default config;
