<script lang="ts">
	import IconBack from '$lib/components/icons/IconBack.svelte';
	import IconButton from '$lib/components/ui/IconButton.svelte';
	import { goto } from '$app/navigation';

	let { data } = $props();
	let entry = $derived(data.entry);

	const SOURCE_LABELS: Record<string, string> = {
		onions: 'Onions Shakespeare Glossary',
		abbott: 'Abbott Shakespearian Grammar',
		bartlett: "Bartlett's Concordance",
		henley_farmer: 'Henley & Farmer Slang'
	};
</script>

<svelte:head>
	<title>{entry.headword} &mdash; {SOURCE_LABELS[entry.source_code] ?? entry.source_name} &mdash; Bardbase</title>
</svelte:head>

<div class="reference-page">
	<div class="reference-header">
		<IconButton onclick={() => goto(`/references?source=${entry.source_code}`)} label="Back to References" size={36}>
			<IconBack size={20} />
		</IconButton>
		<div class="header-text">
			<h1 class="entry-word">{entry.headword}</h1>
			<span class="source-label">{SOURCE_LABELS[entry.source_code] ?? entry.source_name}</span>
		</div>
	</div>

	<div class="entry-body">
		<p class="raw-text">{entry.raw_text}</p>
	</div>
</div>

<style>
	.reference-page {
		max-width: 800px;
		margin: 0 auto;
		padding: 0 16px 48px;
	}

	.reference-header {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 16px 0;
		position: sticky;
		top: 0;
		z-index: 10;
		background: var(--color-bg);
	}

	.header-text {
		flex: 1;
	}

	.entry-word {
		margin: 0;
		font-size: 1.5rem;
		font-weight: 700;
		color: var(--color-text);
	}

	.source-label {
		font-size: 0.75rem;
		color: var(--color-text-muted);
	}

	.entry-body {
		padding: 0;
	}

	.raw-text {
		margin: 0;
		font-size: 0.95rem;
		color: var(--color-text-secondary);
		line-height: 1.8;
		white-space: pre-wrap;
	}
</style>
