<script lang="ts">
	import PageHeader from '$lib/components/ui/PageHeader.svelte';
	import SearchInput from '$lib/components/ui/SearchInput.svelte';
	import type { SearchResult } from '$lib/types';
	import { goto } from '$app/navigation';

	let { data } = $props();

	const MIN_QUERY = 1;
	let query = $state('');
	let results = $state<SearchResult[]>([]);
	let searching = $state(false);
	let searchError = $state<string | null>(null);

	let debounceTimer: ReturnType<typeof setTimeout>;

	$effect(() => {
		const q = query.trim();
		clearTimeout(debounceTimer);
		searchError = null;

		if (q.length < MIN_QUERY) {
			results = [];
			return;
		}

		debounceTimer = setTimeout(async () => {
			searching = true;
			try {
				const res = await fetch(`/api/search?q=${encodeURIComponent(q)}&limit=100`);
				if (res.ok) {
					results = await res.json();
				} else {
					const body = await res.json().catch(() => ({}));
					searchError = body.message ?? `Search failed (${res.status})`;
					results = [];
				}
			} catch (err) {
				searchError = 'Could not reach the API server';
				results = [];
				console.error('[search]', err);
			} finally {
				searching = false;
			}
		}, 250);
	});

	function openEntry(id: number) {
		goto(`/lexicon/entry/${id}`);
	}

	function selectLetter(letter: string) {
		query = letter;
	}

	let isSearching = $derived(query.trim().length >= MIN_QUERY);
</script>

<svelte:head>
	<title>Lexicon &mdash; Bardbase</title>
</svelte:head>

<div class="lexicon-page">
	<div class="sticky-header">
		<PageHeader title="Bardbase" />
		<div class="search-bar">
			<SearchInput bind:value={query} placeholder="Search entries..." autofocus />
		</div>
	</div>

	{#if searchError}
		<div class="status-error">{searchError}</div>
	{:else if isSearching && searching}
		<div class="status-text">Searching&hellip;</div>
	{:else if isSearching && results.length === 0 && !searching}
		<div class="status-text">No entries found for "{query}"</div>
	{/if}

	{#if isSearching}
		{#if results.length > 0}
			<div class="result-count">{results.length} results</div>
			<ul class="entry-list" role="list" aria-label="Search results">
				{#each results as entry (entry.id)}
					<li>
						<button
							class="entry-item"
							onclick={() => openEntry(entry.id)}
							aria-label="View definition of {entry.key}"
						>
							<span class="entry-key">{entry.key}</span>
						</button>
					</li>
				{/each}
			</ul>
		{/if}
	{:else}
		<!-- Letter browser — shown when not searching -->
		<div class="letter-browser">
			{#each data.letters as l (l.letter)}
				<button
					class="letter-btn"
					onclick={() => selectLetter(l.letter)}
					title="{l.count} entries"
				>
					{l.letter.toUpperCase()}
					<span class="letter-count">{l.count}</span>
				</button>
			{/each}
		</div>
	{/if}
</div>


<style>
	.lexicon-page {
		max-width: 720px;
		margin: 0 auto;
		padding: 0 16px;
	}

	.sticky-header {
		position: sticky;
		top: 0;
		z-index: 50;
		background: var(--color-bg);
		padding-bottom: 8px;
	}

	.search-bar {
		position: relative;
	}

	/* ─── Status ─── */
	.status-text {
		padding: 12px 0;
		font-size: 0.85rem;
		color: var(--color-text-muted);
		text-align: center;
	}

	.status-error {
		padding: 12px 0;
		font-size: 0.85rem;
		color: var(--color-danger);
		text-align: center;
	}

	.result-count {
		padding: 4px 0 8px;
		font-size: 0.8rem;
		color: var(--color-text-muted);
	}

	/* ─── Letter browser ─── */
	.letter-browser {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		padding: 16px 0;
	}

	.letter-btn {
		display: flex;
		flex-direction: column;
		align-items: center;
		padding: 10px 14px;
		border: 1px solid var(--color-border);
		background: var(--color-surface);
		color: var(--color-text);
		border-radius: 8px;
		cursor: pointer;
		font-family: inherit;
		font-size: 1.1rem;
		font-weight: 600;
		min-width: 52px;
		transition: background 0.15s, border-color 0.15s;
	}

	.letter-btn:hover {
		background: var(--color-hover);
		border-color: var(--color-accent);
	}

	.letter-count {
		font-size: 0.65rem;
		font-weight: 400;
		color: var(--color-text-muted);
		margin-top: 2px;
	}

	/* ─── Entry list ─── */
	.entry-list {
		list-style: none;
		padding: 0;
		margin: 0;
	}

	.entry-item {
		display: block;
		width: 100%;
		padding: 14px 16px;
		border: none;
		background: none;
		text-align: left;
		cursor: pointer;
		font-family: inherit;
		border-radius: 8px;
		transition: background 0.15s;
		border-bottom: 1px solid var(--color-border);
		color: var(--color-text);
	}

	.entry-item:hover {
		background: var(--color-hover);
	}

	.entry-item:active {
		background: var(--color-active);
	}

	.entry-key {
		font-size: 1.05rem;
		font-weight: 500;
	}
</style>
