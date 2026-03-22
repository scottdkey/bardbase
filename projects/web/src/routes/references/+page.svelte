<script lang="ts">
	import PageHeader from '$lib/components/ui/PageHeader.svelte';
	import SearchInput from '$lib/components/ui/SearchInput.svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';

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

	// Initialize from URL params
	let activeSource = $state<string>(page.url.searchParams.get('source') ?? '');
	let query = $state(page.url.searchParams.get('q') ?? '');
	let workFilter = $state(page.url.searchParams.get('work') ?? '');

	// Sync state to URL
	function updateUrl() {
		const params = new URLSearchParams();
		if (activeSource) params.set('source', activeSource);
		if (query.trim()) params.set('q', query.trim());
		if (workFilter) params.set('work', workFilter);
		const qs = params.toString();
		const url = qs ? `/references?${qs}` : '/references';
		goto(url, { replaceState: true, keepFocus: true, noScroll: true });
	}
	let results = $state<RefResult[]>([]);
	let loading = $state(false);
	let hasMore = $state(true);
	let offset = $state(0);
	const PAGE_SIZE = 50;

	let debounceTimer: ReturnType<typeof setTimeout>;
	let scrollEl: HTMLElement | null = null;

	// Fetch results
	async function fetchResults(reset = true) {
		if (reset) {
			offset = 0;
			hasMore = true;
		}
		loading = true;

		const params = new URLSearchParams();
		if (query.trim()) params.set('q', query.trim());
		if (activeSource) params.set('source', activeSource);
		if (workFilter) params.set('work_id', workFilter);
		params.set('limit', String(PAGE_SIZE));
		params.set('offset', String(offset));

		try {
			const res = await fetch(`/api/reference/search?${params}`);
			if (res.ok) {
				const data: RefResult[] = await res.json();
				if (reset) {
					results = data;
				} else {
					results = [...results, ...data];
				}
				hasMore = data.length === PAGE_SIZE;
			}
		} catch (err) {
			console.error('[references]', err);
		} finally {
			loading = false;
		}
	}

	// Load more for infinite scroll
	async function loadMore() {
		if (loading || !hasMore) return;
		offset += PAGE_SIZE;
		await fetchResults(false);
	}

	// React to filter/search changes
	$effect(() => {
		const _s = activeSource;
		const _w = workFilter;
		const _q = query;

		clearTimeout(debounceTimer);
		debounceTimer = setTimeout(() => {
			fetchResults(true);
			updateUrl();
		}, query ? 250 : 0);
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

	<!-- Source tabs -->
	<div class="source-tabs">
		<button
			class="tab-btn"
			class:active={activeSource === ''}
			onclick={() => (activeSource = '')}
		>
			All
		</button>
		{#each data.sources as src (src.code)}
			<button
				class="tab-btn"
				class:active={activeSource === src.code}
				onclick={() => (activeSource = src.code)}
			>
				{SOURCE_LABELS[src.code] ?? src.code}
				<span class="tab-count">{src.count.toLocaleString()}</span>
			</button>
		{/each}
	</div>

	<!-- Search + filter -->
	<div class="filter-bar">
		<div class="search-wrap">
			<SearchInput bind:value={query} placeholder="Search references..." />
		</div>
		<select class="work-filter" bind:value={workFilter}>
			<option value="">All works</option>
			{#each data.works as work (work.id)}
				<option value={String(work.id)}>{work.title}</option>
			{/each}
		</select>
	</div>

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

	/* ─── Source tabs ─── */
	.source-tabs {
		display: flex;
		gap: 4px;
		overflow-x: auto;
		padding-bottom: 8px;
		border-bottom: 1px solid var(--color-border);
		margin-bottom: 12px;
	}

	.tab-btn {
		display: flex;
		align-items: center;
		gap: 4px;
		padding: 6px 12px;
		border: none;
		background: none;
		color: var(--color-text-muted);
		font-family: inherit;
		font-size: 0.75rem;
		font-weight: 600;
		cursor: pointer;
		border-bottom: 2px solid transparent;
		white-space: nowrap;
		transition: color 0.15s, border-color 0.15s;
	}

	.tab-btn:hover {
		color: var(--color-text);
	}

	.tab-btn.active {
		color: var(--color-accent);
		border-bottom-color: var(--color-accent);
	}

	.tab-count {
		font-size: 0.6rem;
		color: var(--color-text-muted);
		font-weight: 400;
	}

	/* ─── Filter bar ─── */
	.filter-bar {
		display: flex;
		gap: 8px;
		margin-bottom: 16px;
		align-items: stretch;
	}

	.search-wrap {
		flex: 1;
	}

	.work-filter {
		padding: 6px 10px;
		border: 1px solid var(--color-border);
		background: var(--color-surface);
		color: var(--color-text);
		font-family: inherit;
		font-size: 0.8rem;
		border-radius: 8px;
		cursor: pointer;
		max-width: 200px;
	}

	.work-filter:focus {
		outline: none;
		border-color: var(--color-accent);
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
