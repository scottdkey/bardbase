import { sveltekit } from '@sveltejs/kit/vite';
import { SvelteKitPWA } from '@vite-pwa/sveltekit';
import { defineConfig } from 'vite';

export default defineConfig({
	server: {
		host: '0.0.0.0',
		port: 5173
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
				description:
					'Complete works of Shakespeare with multi-edition texts, lexicon, and full-text search',
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
				globPatterns: ['**/*.{js,css,html,svg,png,woff2}'],
				runtimeCaching: [
					{
						// Lexicon entries and scene text — cache first, rarely changes
						urlPattern: /\/api\/(lexicon\/entry|text\/scene)\//,
						handler: 'StaleWhileRevalidate',
						options: {
							cacheName: 'bardbase-data',
							expiration: {
								maxEntries: 500,
								maxAgeSeconds: 60 * 60 * 24 * 7 // 7 days
							},
							cacheableResponse: { statuses: [0, 200] }
						}
					},
					{
						// Search and metadata — try network first, fall back to cache
						urlPattern: /\/api\/(search|attributions|works|stats)/,
						handler: 'NetworkFirst',
						options: {
							cacheName: 'bardbase-api',
							networkTimeoutSeconds: 5,
							expiration: {
								maxEntries: 100,
								maxAgeSeconds: 60 * 60 * 24 // 1 day
							},
							cacheableResponse: { statuses: [0, 200] }
						}
					}
				]
			}
		})
	]
});
