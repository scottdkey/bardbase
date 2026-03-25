<script lang="ts">
	import type { CitationSpan } from '$lib/server/api';
	import IconBack from '$lib/components/icons/IconBack.svelte';
	import IconButton from '$lib/components/ui/IconButton.svelte';
	import CollapsibleSection from '$lib/components/ui/CollapsibleSection.svelte';
	import { goto } from '$app/navigation';
	import { groupBy } from '$lib/utils';

	let { data } = $props();
	let entry = $derived(data.entry);

	const SOURCE_LABELS: Record<string, string> = {
		onions: 'Onions Shakespeare Glossary',
		abbott: 'Abbott Shakespearian Grammar',
		bartlett: "Bartlett's Concordance",
		henley_farmer: 'Henley & Farmer Slang'
	};

	interface Citation {
		work_title: string | null;
		act: number | null;
		scene: number | null;
		line: number | null;
		work_slug: string | null;
	}

	let citationsByWork = $derived.by(() => {
		return groupBy(entry.citations as Citation[], (c) => c.work_title || 'Other');
	});

	// Split raw_text into alternating plain/citation segments for annotated rendering.
	interface TextSegment {
		text: string;
		span?: CitationSpan;
	}

	let textSegments = $derived.by((): TextSegment[] => {
		const spans: CitationSpan[] = entry.citation_spans ?? [];
		if (spans.length === 0) return [{ text: entry.raw_text }];

		const segments: TextSegment[] = [];
		let pos = 0;
		for (const sp of spans) {
			if (sp.start > pos) {
				segments.push({ text: entry.raw_text.slice(pos, sp.start) });
			}
			segments.push({ text: entry.raw_text.slice(sp.start, sp.end), span: sp });
			pos = sp.end;
		}
		if (pos < entry.raw_text.length) {
			segments.push({ text: entry.raw_text.slice(pos) });
		}
		return segments;
	});

	function formatLoc(c: Citation): string {
		if (c.act == null) return c.line != null ? String(c.line) : '';
		let loc = `${c.act}`;
		if (c.scene != null) loc += `.${c.scene}`;
		if (c.line != null) loc += `.${c.line}`;
		return loc;
	}

	function openScene(c: Citation) {
		if (!c.work_slug || c.act == null) return;
		const scene = c.scene ?? 1;
		goto(`/text/${c.work_slug}/${c.act}/${scene}`);
	}
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
		<p class="definition">{#each textSegments as seg}{#if seg.span && seg.span.work_slug && seg.span.act != null}<button class="citation-inline" onclick={() => goto(`/text/${seg.span!.work_slug}/${seg.span!.act}/${seg.span!.scene ?? 1}`)}>{seg.text}</button>{:else}{seg.text}{/if}{/each}</p>

		{#if entry.citations.length > 0}
			{#if entry.source_code === 'bartlett'}
				<!-- Bartlett's: show citations below definition, always visible -->
				<section class="citations-below">
					<h3 class="citations-heading">Citations ({entry.citations.length})</h3>
					<div class="citation-groups">
						{#each [...citationsByWork.entries()] as [workName, citations] (workName)}
							<div class="citation-work-group">
								<h4 class="work-group-title">{workName}</h4>
								<ul class="citation-list">
									{#each citations as c, i (i)}
										<li>
											{#if c.work_slug && c.act != null}
												<button class="citation-item clickable" onclick={() => openScene(c)}>
													<span class="citation-ref">{formatLoc(c)}</span>
												</button>
											{:else}
												<span class="citation-item">
													<span class="citation-ref">{formatLoc(c)}</span>
												</span>
											{/if}
										</li>
									{/each}
								</ul>
							</div>
						{/each}
					</div>
				</section>
			{:else}
				<CollapsibleSection label="References" count={entry.citations.length} open={true}>
					<div class="citation-groups">
						{#each [...citationsByWork.entries()] as [workName, citations] (workName)}
							<div class="citation-work-group">
								<h4 class="work-group-title">{workName}</h4>
								<ul class="citation-list">
									{#each citations as c, i (i)}
										<li>
											{#if c.work_slug && c.act != null}
												<button class="citation-item clickable" onclick={() => openScene(c)}>
													<span class="citation-ref">{formatLoc(c)}</span>
												</button>
											{:else}
												<span class="citation-item">
													<span class="citation-ref">{formatLoc(c)}</span>
												</span>
											{/if}
										</li>
									{/each}
								</ul>
							</div>
						{/each}
					</div>
				</CollapsibleSection>
			{/if}
		{/if}
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

	.definition {
		margin: 0 0 20px;
		font-size: 0.95rem;
		color: var(--color-text-secondary);
		line-height: 1.8;
		white-space: pre-wrap;
	}

	.citation-inline {
		display: inline;
		padding: 0;
		margin: 0;
		border: none;
		background: none;
		font: inherit;
		color: var(--color-accent);
		cursor: pointer;
		text-decoration: underline;
		text-decoration-style: dotted;
		text-underline-offset: 2px;
	}

	.citation-inline:hover {
		text-decoration-style: solid;
		background: var(--color-hover);
		border-radius: 2px;
	}

	/* ─── Bartlett's citations below ─── */
	.citations-below {
		margin-top: 20px;
		padding-top: 16px;
		border-top: 1px solid var(--color-border);
	}

	.citations-heading {
		margin: 0 0 10px;
		font-size: 0.8rem;
		font-weight: 700;
		color: var(--color-text);
		text-transform: uppercase;
		letter-spacing: 0.03em;
	}

	/* ─── Citations ─── */
	.citation-groups {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.work-group-title {
		margin: 0 0 2px;
		font-size: 0.75rem;
		font-weight: 600;
		color: var(--color-accent);
		opacity: 0.8;
	}

	.citation-list {
		list-style: none;
		padding: 0;
		margin: 0;
		display: flex;
		flex-wrap: wrap;
		gap: 4px;
	}

	.citation-item {
		display: inline-block;
		padding: 3px 8px;
		font-size: 0.75rem;
		font-family: inherit;
		border-radius: 4px;
		color: var(--color-text-secondary);
	}

	.citation-item.clickable {
		border: none;
		background: none;
		cursor: pointer;
		color: var(--color-accent);
		font-weight: 600;
		transition: background 0.15s;
	}

	.citation-item.clickable:hover {
		background: var(--color-hover);
	}

	.citation-ref {
		font-weight: 600;
	}
</style>
