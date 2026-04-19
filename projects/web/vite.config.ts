import { sveltekit } from '@sveltejs/kit/vite';
import { SvelteKitPWA } from '@vite-pwa/sveltekit';
import { defineConfig } from 'vite';

export default defineConfig({
	ssr: {
		// node:sqlite is used only during prerender (Node.js). Mark it external
		// so it is never bundled into the Cloudflare Workers output.
		external: ['node:sqlite', 'node:path']
	},
	server: {
		host: '0.0.0.0',
		port: 5173,
		// Poll for file changes — inotify events don't propagate from macOS host
		// through the VM layer into the Linux container.
		watch: { usePolling: true },
		hmr: { port: 24678 }
	},
	plugins: [
		sveltekit(),
		SvelteKitPWA({
			srcDir: 'src',
			strategies: 'generateSW',
			registerType: 'autoUpdate',
			manifest: {
				name: 'Variorum',
				short_name: 'Variorum',
				description: 'Complete works of Shakespeare with multi-edition texts, lexicon, and full-text search',
				theme_color: '#1a1a2e',
				background_color: '#1a1a2e',
				display: 'standalone',
				scope: '/',
				start_url: '/',
				icons: [
					{
						src: '/icons/icon-192.png',
						sizes: '192x192',
						type: 'image/png'
					},
					{
						src: '/icons/icon-512.png',
						sizes: '512x512',
						type: 'image/png'
					},
					{
						src: '/icons/icon-512.png',
						sizes: '512x512',
						type: 'image/png',
						purpose: 'maskable'
					}
				]
			},
			workbox: {
				globPatterns: ['**/*.{js,css,svg,png,woff2}'],
				// Prerendered HTML scenes can be 7-11 MB (multi-edition text + references inline).
				// Let the plugin find them (avoids Workbox "empty pattern" warning), but strip
				// them from the SW manifest via manifestTransforms — they must never be precached.
				maximumFileSizeToCacheInBytes: 15 * 1024 * 1024,
				manifestTransforms: [
					(entries) => ({
						manifest: entries.filter((e) => !e.url.startsWith('prerendered/')),
						warnings: []
					})
				],
				runtimeCaching: [
					{
						// Static metadata — changes only when the DB is rebuilt, serve from cache immediately
						urlPattern: /\/api\/(attributions|works|stats)/,
						handler: 'CacheFirst',
						options: {
							cacheName: 'bardbase-meta',
							expiration: {
								maxEntries: 20,
								maxAgeSeconds: 60 * 60 * 24 * 30 // 30 days
							},
							cacheableResponse: { statuses: [0, 200] }
						}
					},
					{
						// Lexicon entries and scene text — stable content, cache aggressively
						urlPattern: /\/api\/(lexicon\/entry|text\/scene)\//,
						handler: 'CacheFirst',
						options: {
							cacheName: 'bardbase-data',
							expiration: {
								maxEntries: 2000,
								maxAgeSeconds: 60 * 60 * 24 * 30 // 30 days
							},
							cacheableResponse: { statuses: [0, 200] }
						}
					},
					{
						// Search — serve cached results instantly, refresh in background
						urlPattern: /\/api\/search/,
						handler: 'StaleWhileRevalidate',
						options: {
							cacheName: 'bardbase-search',
							expiration: {
								maxEntries: 500,
								maxAgeSeconds: 60 * 60 * 24 * 7 // 7 days
							},
							cacheableResponse: { statuses: [0, 200] }
						}
					}
				]
			}
		})
	]
});
