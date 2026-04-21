<script lang="ts">
	import PageHeader from '$lib/components/ui/PageHeader.svelte';
	import SearchInput from '$lib/components/ui/SearchInput.svelte';
	import { goto } from '$app/navigation';
	import { tick } from 'svelte';

	let { data } = $props();

	interface RefResult {
		id: number;
		headword: string;
		raw_text: string;
		source_code: string;
		source_name: string;
	}

	const SOURCE_LABELS: Record<string, string> = {
		schmidt: 'Schmidt',
		onions: 'Onions',
		abbott: 'Abbott',
		bartlett: 'Bartlett',
		henley_farmer: 'Henley & Farmer'
	};

	const SOURCE_DESCRIPTIONS: Record<string, string> = {
		schmidt:
			'Alexander Schmidt\'s Shakespeare Lexicon (1902) is the definitive dictionary of Shakespeare\'s language. Every word Shakespeare used is catalogued with definitions and citations. Use it to look up the meaning of any word as Shakespeare used it.',
		onions:
			'C. T. Onions\' Shakespeare Glossary (1911) explains words and phrases that have changed meaning since Shakespeare\'s time or are no longer in common use. More concise than Schmidt, it focuses on words that modern readers are likely to find unfamiliar.',
		abbott:
			'E. A. Abbott\'s Shakespearian Grammar (1877) explains Shakespeare\'s grammar, syntax, and rhetorical devices. Use it to understand unusual sentence structures, word order, and grammatical constructions that differ from modern English.',
		bartlett:
			'John Bartlett\'s Complete Concordance (1896) indexes every significant word in Shakespeare with the line where it appears. Use it to find every occurrence of a specific word across all plays and poems.',
		henley_farmer:
			'W. E. Henley and John S. Farmer\'s Slang and Its Analogues (1890\u20131904) documents historical slang, cant, and colloquial language. Use it to decode bawdy, vulgar, or underworld language in Shakespeare\'s text.'
	};

	const ALL_SOURCES = Object.keys(SOURCE_LABELS);

	// Purely driven by URL — updated automatically when SvelteKit re-runs load on navigation
	let activeSource = $derived(data.initialSource ?? '');

	// Local input state; syncs from URL on navigation, otherwise updated by typing
	let query = $state('');
	$effect(() => { query = data.initialQuery ?? ''; });

	// Multi-source toggles — only used on the "all" page (activeSource === '')
	let enabledSources = $state<Set<string>>(new Set(ALL_SOURCES));

	function toggleSource(src: string) {
		const next = new Set(enabledSources);
		if (next.has(src)) {
			if (next.size > 1) next.delete(src); // keep at least one active
		} else {
			next.add(src);
		}
		enabledSources = next;
	}

	let results = $state<RefResult[]>([]);
	let loading = $state(false);
	let hasMore = $state(true);
	let offset = $state(0);
	const PAGE_SIZE = 50;

	let debounceTimer: ReturnType<typeof setTimeout>;

	// Fetch results
	async function fetchResults(reset = true) {
		if (reset) {
			offset = 0;
			hasMore = true;
		}
		loading = true;

		const params = new URLSearchParams();
		if (query.trim()) params.set('q', query.trim());
		if (activeSource) {
			params.set('source', activeSource);
		} else if (enabledSources.size < ALL_SOURCES.length) {
			params.set('sources', [...enabledSources].join(','));
		}
		params.set('limit', String(PAGE_SIZE));
		params.set('offset', String(offset));

		try {
			const res = await fetch(`/api/reference/search?${params}`);
			if (res.ok) {
				const fetched: RefResult[] = await res.json();
				results = reset ? fetched : [...results, ...fetched];
				hasMore = fetched.length === PAGE_SIZE;
			}
		} catch (err) {
			console.error('[references]', err);
		} finally {
			loading = false;
			// After a reset, the sentinel may already be visible but the observer
			// won't re-fire (it only triggers on intersection changes). Check manually.
			if (reset && hasMore && sentinelEl) {
				await tick();
				const rect = sentinelEl.getBoundingClientRect();
				if (rect.top < window.innerHeight + 200) loadMore();
			}
		}
	}

	// Load more for infinite scroll
	async function loadMore() {
		if (loading || !hasMore) return;
		offset += PAGE_SIZE;
		await fetchResults(false);
	}

	// React to filter/search/source-toggle changes
	$effect(() => {
		const _s = activeSource;
		const _q = query;
		const _e = enabledSources;

		clearTimeout(debounceTimer);
		debounceTimer = setTimeout(() => fetchResults(true), query ? 250 : 0);
	});

	// Infinite scroll observer
	let sentinelEl = $state<HTMLElement | null>(null);

	$effect(() => {
		if (!sentinelEl) return;
		const observer = new IntersectionObserver(
			(entries) => {
				if (entries[0].isIntersecting && hasMore && !loading) {
					loadMore();
				}
			},
			{ rootMargin: '200px' }
		);
		observer.observe(sentinelEl);
		return () => observer.disconnect();
	});

	function openEntry(r: RefResult) {
		if (r.source_code === 'schmidt') {
			goto(`/lexicon/entry/${r.id}`);
		} else {
			goto(`/reference/${r.id}`);
		}
	}
</script>

<svelte:head>
	<title>References &mdash; Bardbase</title>
</svelte:head>

<div class="references-page">
	<PageHeader title="References" />

	<div class="filter-bar">
		<SearchInput bind:value={query} placeholder="Search references..." />
	</div>

	{#if !activeSource}
		<div class="source-toggles" role="group" aria-label="Toggle reference sources">
			{#each ALL_SOURCES as src}
				<button
					class="source-toggle"
					class:active={enabledSources.has(src)}
					onclick={() => toggleSource(src)}
					aria-pressed={enabledSources.has(src)}
				>
					{SOURCE_LABELS[src]}
				</button>
			{/each}
		</div>
	{/if}

	{#if activeSource && SOURCE_DESCRIPTIONS[activeSource]}
		<p class="source-description">{SOURCE_DESCRIPTIONS[activeSource]}</p>
	{/if}

	<!-- Results -->
	{#if results.length === 0 && !loading}
		<div class="empty-state">No references found.</div>
	{/if}

	<ul class="result-list">
		{#each results as r (r.id)}
			<li>
				<button class="result-item" onclick={() => openEntry(r)}>
					<div class="result-header">
						<span class="result-headword">{r.headword}</span>
						<span class="result-source">{SOURCE_LABELS[r.source_code] ?? r.source_code}</span>
					</div>
					<p class="result-text">{r.raw_text}</p>
				</button>
			</li>
		{/each}
	</ul>

	{#if loading}
		<div class="loading-indicator">Loading&hellip;</div>
	{/if}

	<!-- Infinite scroll sentinel -->
	<div bind:this={sentinelEl} class="scroll-sentinel"></div>
</div>

<style>
	.references-page {
		max-width: 800px;
		margin: 0 auto;
		padding: 0 16px 60px;
	}

	.source-description {
		margin: 0 0 12px;
		padding: 10px 14px;
		font-size: 0.78rem;
		color: var(--color-text-secondary);
		line-height: 1.6;
		background: var(--color-surface);
		border-radius: 6px;
		border: 1px solid var(--color-border);
	}

	/* ─── Filter bar ─── */
	.filter-bar {
		margin-bottom: 10px;
	}

	.source-toggles {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
		margin-bottom: 14px;
	}

	.source-toggle {
		padding: 4px 10px;
		border: 1px solid var(--color-border);
		border-radius: 20px;
		background: none;
		color: var(--color-text-muted);
		font-family: inherit;
		font-size: 0.72rem;
		font-weight: 600;
		cursor: pointer;
		transition: background 0.15s, color 0.15s, border-color 0.15s;
	}

	.source-toggle.active {
		background: var(--color-active);
		color: var(--color-accent);
		border-color: var(--color-accent);
	}

	.source-toggle:hover {
		color: var(--color-text);
		border-color: var(--color-text-muted);
	}

	/* ─── Results ─── */
	.empty-state {
		text-align: center;
		padding: 40px 0;
		color: var(--color-text-muted);
		font-size: 0.85rem;
	}

	.result-list {
		list-style: none;
		padding: 0;
		margin: 0;
	}

	.result-item {
		display: block;
		width: 100%;
		padding: 12px 14px;
		border: none;
		background: none;
		text-align: left;
		cursor: pointer;
		font-family: inherit;
		color: var(--color-text);
		border-bottom: 1px solid color-mix(in srgb, var(--color-border) 50%, transparent);
		transition: background 0.15s;
		border-radius: 0;
	}

	.result-item:hover {
		background: var(--color-hover);
	}

	.result-header {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		margin-bottom: 2px;
	}

	.result-headword {
		font-size: 0.95rem;
		font-weight: 700;
		color: var(--color-text);
	}

	.result-source {
		font-size: 0.6rem;
		font-weight: 600;
		color: var(--color-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.03em;
	}

	.result-text {
		margin: 0;
		font-size: 0.78rem;
		color: var(--color-text-secondary);
		line-height: 1.4;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	.loading-indicator {
		text-align: center;
		padding: 16px 0;
		font-size: 0.8rem;
		color: var(--color-text-muted);
	}

	.scroll-sentinel {
		height: 1px;
	}
</style>
