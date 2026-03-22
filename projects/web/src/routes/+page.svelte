<script lang="ts">
	import EntryModal from '$lib/components/EntryModal.svelte';
	import PageHeader from '$lib/components/ui/PageHeader.svelte';
	import SearchInput from '$lib/components/ui/SearchInput.svelte';
	import type { LexiconEntryDetail, SearchResult } from '$lib/types';

	const MIN_QUERY = 2;
	let query = $state('');
	let results = $state<SearchResult[]>([]);
	let searching = $state(false);
	let searchError = $state<string | null>(null);

	let selectedEntry = $state<LexiconEntryDetail | null>(null);
	let selectedEntryId = $state<number | null>(null);

	let debounceTimer: ReturnType<typeof setTimeout>;

	$effect(() => {
		const q = query.trim();
		clearTimeout(debounceTimer);

		if (q.length < MIN_QUERY) {
			results = [];
			return;
		}

		debounceTimer = setTimeout(async () => {
			searching = true;
			searchError = null;
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

	async function openEntry(id: number) {
		selectedEntryId = id;
		try {
			const res = await fetch(`/api/lexicon/entry/${id}`);
			if (res.ok) selectedEntry = await res.json();
		} finally {
			// loading done
		}
	}

	function closeEntry() {
		selectedEntry = null;
		selectedEntryId = null;
	}
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
	{:else if query.trim().length > 0 && query.trim().length < MIN_QUERY}
		<div class="status-text">Keep typing&hellip;</div>
	{:else if searching}
		<div class="status-text">Searching&hellip;</div>
	{:else if query.trim().length >= MIN_QUERY && results.length === 0}
		<div class="status-text">No entries found for "{query}"</div>
	{:else if results.length > 0}
		<div class="result-count">{results.length} results</div>
	{:else}
		<div class="status-text">Start typing to search the lexicon</div>
	{/if}

	{#if results.length > 0}
		<ul class="entry-list" role="list" aria-label="Lexicon entries">
			{#each results as entry (entry.id)}
				<li>
					<button
						class="entry-item"
						class:selected={selectedEntryId === entry.id}
						onclick={() => openEntry(entry.id)}
						aria-label="View definition of {entry.key}"
					>
						<span class="entry-key">{entry.key}</span>
					</button>
				</li>
			{/each}
		</ul>
	{/if}
</div>

<EntryModal entry={selectedEntry} onclose={closeEntry} />

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

	.entry-item.selected {
		background: var(--color-active);
		border-color: var(--color-accent);
	}

	.entry-key {
		font-size: 1.05rem;
		font-weight: 500;
	}
</style>
