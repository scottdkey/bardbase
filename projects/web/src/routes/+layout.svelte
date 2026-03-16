<script lang="ts">
	import { onMount } from 'svelte';
	import { pwaInfo } from 'virtual:pwa-info';

	let { children } = $props();

	onMount(async () => {
		if (pwaInfo) {
			const { registerSW } = await import('virtual:pwa-register');
			registerSW({
				immediate: true,
				onRegisteredSW(swUrl: string) {
					console.log(`SW registered: ${swUrl}`);
				},
				onOfflineReady() {
					console.log('App ready to work offline');
				}
			});
		}
	});
</script>

<svelte:head>
	{#if pwaInfo?.webManifest?.href}
		<link rel="manifest" href={pwaInfo.webManifest.href} />
	{/if}
</svelte:head>

<nav>
	<a href="/">Home</a>
	<a href="/works">Works</a>
	<a href="/lexicon">Lexicon</a>
	<a href="/search">Search</a>
</nav>

<main>
	{@render children()}
</main>

<style>
	:global(body) {
		margin: 0;
		font-family: Georgia, 'Times New Roman', serif;
		background: #1a1a2e;
		color: #e0e0e0;
	}

	nav {
		display: flex;
		gap: 1.5rem;
		padding: 1rem 2rem;
		background: #16213e;
		border-bottom: 1px solid #0f3460;
	}

	nav a {
		color: #e0e0e0;
		text-decoration: none;
		font-size: 1rem;
	}

	nav a:hover {
		color: #e94560;
	}

	main {
		max-width: 960px;
		margin: 0 auto;
		padding: 2rem;
	}
</style>
