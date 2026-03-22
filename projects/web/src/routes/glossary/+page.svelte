<script lang="ts">
	import { abbreviations } from '$lib/data/abbreviations';
	import PageHeader from '$lib/components/ui/PageHeader.svelte';
	import SearchInput from '$lib/components/ui/SearchInput.svelte';

	let filter = $state('');

	let filtered = $derived(
		filter.trim()
			? abbreviations.filter(a =>
				a.abbrev.toLowerCase().includes(filter.toLowerCase()) ||
				a.expansion.toLowerCase().includes(filter.toLowerCase()) ||
				(a.description ?? '').toLowerCase().includes(filter.toLowerCase())
			)
			: abbreviations
	);

	// Group by category based on position in the array
	const categories = [
		{ label: 'Grammatical', start: 0, end: 53 },
		{ label: 'Reference & Scholarly', start: 53, end: 75 },
		{ label: 'Editions & Texts', start: 75, end: 87 },
		{ label: 'Languages', start: 87, end: 93 },
		{ label: 'Schmidt-specific', start: 93, end: 97 },
		{ label: 'Scholars', start: 97, end: abbreviations.length }
	];
</script>

<svelte:head>
	<title>Glossary &mdash; Bardbase</title>
</svelte:head>

<div class="glossary-page">
	<PageHeader title="Glossary" subtitle="A–Z index of all lexicon headwords" />

	<div class="search-bar">
		<SearchInput bind:value={filter} placeholder="Filter entries..." />
	</div>

	<div class="glossary-list">
		{#each filtered as a (a.abbrev)}
			<div class="glossary-item">
				<span class="abbrev">{a.abbrev}</span>
				<div class="meaning">
					<span class="expansion">{a.expansion}</span>
					{#if a.description}
						<p class="description">{a.description}</p>
					{/if}
				</div>
			</div>
		{/each}
	</div>

	{#if filtered.length === 0}
		<p class="no-results">No abbreviations match "{filter}"</p>
	{/if}
</div>

<style>
	.glossary-page {
		max-width: 720px;
		margin: 0 auto;
		padding: 0 16px 60px;
	}

	.search-bar {
		margin-bottom: 12px;
	}

	.glossary-list {
		display: flex;
		flex-direction: column;
	}

	.glossary-item {
		display: flex;
		gap: 12px;
		padding: 8px 0;
		border-bottom: 1px solid var(--color-border);
		align-items: baseline;
	}

	.abbrev {
		font-weight: 700;
		color: var(--color-accent);
		min-width: 100px;
		flex-shrink: 0;
		font-size: 0.9rem;
	}

	.meaning {
		flex: 1;
	}

	.expansion {
		color: var(--color-text);
		font-size: 0.9rem;
	}

	.description {
		margin: 2px 0 0;
		font-size: 0.75rem;
		color: var(--color-text-muted);
		font-style: italic;
	}

	.no-results {
		text-align: center;
		color: var(--color-text-muted);
		padding: 20px 0;
	}
</style>
