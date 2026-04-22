<script lang="ts">
	import type { LexiconEntryDetail } from '$lib/types';
	import type { CitationSpan, LineReference } from '$lib/server/api';
	import IconClose from '$lib/components/icons/IconClose.svelte';
	import IconButton from '$lib/components/ui/IconButton.svelte';
	import CollapsibleSection from '$lib/components/ui/CollapsibleSection.svelte';
	import { groupBy } from '$lib/utils';
	import { goto } from '$app/navigation';

	interface TextSegment {
		text: string;
		span?: CitationSpan;
	}

	function buildSegments(rawText: string, spans: CitationSpan[]): TextSegment[] {
		if (!spans || spans.length === 0) return [{ text: rawText }];
		const segments: TextSegment[] = [];
		let pos = 0;
		for (const sp of spans) {
			if (sp.start > pos) segments.push({ text: rawText.slice(pos, sp.start) });
			segments.push({ text: rawText.slice(sp.start, sp.end), span: sp });
			pos = sp.end;
		}
		if (pos < rawText.length) segments.push({ text: rawText.slice(pos) });
		return segments;
	}

	let {
		ref,
		onclose
	}: {
		ref: LineReference;
		onclose: () => void;
	} = $props();

	const SOURCE_LABELS: Record<string, string> = {
		schmidt: 'Schmidt Shakespeare Lexicon',
		onions: 'Onions Shakespeare Glossary',
		abbott: 'Abbott Shakespearian Grammar',
		bartlett: "Bartlett's Concordance",
		henley_farmer: 'Henley & Farmer Slang'
	};

	const SOURCE_HINTS: Record<string, string> = {
		onions: 'Explains unfamiliar or archaic words',
		abbott: 'Explains grammar and syntax',
		bartlett: 'Every occurrence of this word in Shakespeare',
		henley_farmer: 'Historical slang and colloquial usage'
	};

	// Entry data - one of these will be populated
	let lexiconEntry = $state<LexiconEntryDetail | null>(null);
	let referenceEntry = $state<{
		id: number;
		headword: string;
		raw_text: string;
		source_name: string;
		source_code: string;
		citations: { work_title: string | null; act: number | null; scene: number | null; line: number | null; work_slug: string | null }[];
		citation_spans?: CitationSpan[];
	} | null>(null);

	let refTextSegments = $derived.by((): TextSegment[] => {
		if (!referenceEntry) return [];
		return buildSegments(referenceEntry.raw_text, referenceEntry.citation_spans ?? []);
	});
	let loading = $state(true);
	let error = $state<string | null>(null);

	$effect(() => {
		loading = true;
		error = null;
		lexiconEntry = null;
		referenceEntry = null;

		if (ref.source_code === 'schmidt') {
			// Use key-based lookup (stable across DB rebuilds) instead of
			// numeric entry_id, which is baked into pre-rendered HTML at build
			// time but may differ from the runtime Turso DB's auto-increment IDs.
			fetch(`/api/lexicon/key/${encodeURIComponent(ref.entry_key)}`)
				.then((res) => {
					if (!res.ok) throw new Error('Failed to load entry');
					return res.json();
				})
				.then((data) => {
					lexiconEntry = data;
					loading = false;
				})
				.catch((e) => {
					error = e.message;
					loading = false;
				});
		} else {
			// Use source + headword for stable lookup (same reason as above).
			const refUrl =
				`/api/reference/key` +
				`?source=${encodeURIComponent(ref.source_code)}` +
				`&headword=${encodeURIComponent(ref.entry_key)}`;
			fetch(refUrl)
				.then((res) => {
					if (!res.ok) throw new Error('Failed to load entry');
					return res.json();
				})
				.then((data) => {
					referenceEntry = data;
					loading = false;
				})
				.catch((e) => {
					error = e.message;
					loading = false;
				});
		}
	});

	// For Schmidt entries, get the primary definition
	let primaryDefinition = $derived.by(() => {
		if (!lexiconEntry) return null;
		for (const sub of lexiconEntry.subEntries) {
			for (const sense of sub.senses) {
				if (sense.definition_text) return sense.definition_text;
			}
			if (sub.full_text) return sub.full_text;
		}
		return lexiconEntry.senses?.[0]?.definition_text ?? null;
	});

	// For reference entries, get citations grouped by work
	let citationsByWork = $derived.by(() => {
		if (!referenceEntry?.citations) return new Map();
		return groupBy(referenceEntry.citations, (c) => c.work_title || 'Other');
	});

	function formatLoc(c: { act: number | null; scene: number | null; line: number | null }): string {
		if (c.act == null) return c.line != null ? String(c.line) : '';
		let loc = `${c.act}`;
		if (c.scene != null) loc += `.${c.scene}`;
		if (c.line != null) loc += `.${c.line}`;
		return loc;
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') onclose();
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="drawer-backdrop" onclick={onclose} role="presentation"></div>
<aside class="drawer" aria-label="Reference details">
	<div class="drawer-header">
		<div class="header-content">
			<h2 class="headword">{ref.entry_key}</h2>
			<span class="source-label">{SOURCE_LABELS[ref.source_code] ?? ref.source_code}</span>
			{#if !loading}
				{#if lexiconEntry && primaryDefinition}
					<p class="header-definition">{primaryDefinition}</p>
				{:else if referenceEntry && referenceEntry.source_code !== 'bartlett'}
					<p class="header-definition">{referenceEntry.raw_text.length > 200 ? referenceEntry.raw_text.slice(0, 200) + '...' : referenceEntry.raw_text}</p>
				{/if}
			{/if}
		</div>
		<IconButton onclick={onclose} label="Close" size={32}>
			<IconClose size={18} />
		</IconButton>
	</div>

	<div class="drawer-body">
		{#if loading}
			<div class="loading">Loading...</div>
		{:else if error}
			<div class="error">{error}</div>
		{:else if lexiconEntry}
			<!-- Schmidt lexicon entry -->
			{#each lexiconEntry.subEntries as sub (sub.id)}
				{#if lexiconEntry.subEntries.length > 1}
					<div class="sub-entry-header">
						<h3 class="sub-entry-key">{sub.key}</h3>
						{#if sub.entry_type}
							<span class="sub-entry-type">{sub.entry_type}</span>
						{/if}
					</div>
				{/if}

				{#if sub.senses.length > 0}
					{@const otherSenses = sub.senses.filter(s => s.definition_text !== primaryDefinition)}
					{#if otherSenses.length > 0}
						<section class="senses">
							{#each otherSenses as sense (sense.id)}
								<div class="sense-block" class:sub-sense={sense.sub_sense}>
									<div class="sense">
										{#if sense.sub_sense}
											<span class="sense-num sub">{sense.sub_sense})</span>
										{:else}
											<span class="sense-num">{sense.sense_number})</span>
										{/if}
										<p class="sense-def">{sense.definition_text}</p>
									</div>
								</div>
							{/each}
						</section>
					{/if}
				{:else if sub.full_text}
					<section class="full-text">
						<p>{sub.full_text}</p>
					</section>
				{/if}
			{/each}

			{#if lexiconEntry.references && lexiconEntry.references.length > 0}
				{@const refsBySource = groupBy(lexiconEntry.references, (r) => r.source_code)}
				<section class="reference-works">
					<h3 class="ref-section-title">Reference Works</h3>
					{#each [...refsBySource.entries()] as [sourceCode, refs] (sourceCode)}
						{@const sourceDesc = SOURCE_HINTS[sourceCode] ?? ''}
						<CollapsibleSection label={refs[0].source_name} count={refs.length} open={true}>
							{#if sourceDesc}
								<p class="source-hint">{sourceDesc}</p>
							{/if}
							<ul class="ref-citation-list">
								{#each refs as r, idx (idx)}
									<li class="ref-citation-item">
										{#if r.work_slug && r.act != null}
											<button class="ref-location clickable" onclick={() => goto(`/text/${r.work_slug}/${r.act}/${r.scene ?? 1}${r.line != null ? `?line=${r.line}` : ''}`)}>
												{r.work_title ?? r.work_abbrev ?? ''} {r.act}.{r.scene ?? ''}.{r.line ?? ''}
											</button>
										{:else}
											<span class="ref-location">
												{r.work_title ?? r.work_abbrev ?? ''}
												{r.act != null ? ` ${r.act}.${r.scene ?? ''}.${r.line ?? ''}` : ''}
											</span>
										{/if}
										{#if r.entry_id}
											<button class="ref-entry-link" onclick={() => goto(`/reference/${r.entry_id}`)}>view</button>
										{/if}
									</li>
								{/each}
							</ul>
						</CollapsibleSection>
					{/each}
				</section>
			{/if}
		{:else if referenceEntry}
			<!-- Reference entry (Abbott, Onions, Bartlett, Henley & Farmer) -->
			<div class="entry-text">
				<p class="definition">{#each refTextSegments as seg}{#if seg.span && seg.span.work_slug && seg.span.act != null}<button class="citation-inline" onclick={() => goto(`/text/${seg.span!.work_slug}/${seg.span!.act}/${seg.span!.scene ?? 1}`)}>{seg.text}</button>{:else}{seg.text}{/if}{/each}</p>
			</div>

			{#if referenceEntry.citations && referenceEntry.citations.length > 0}
				<section class="citations-section">
					<h3 class="citations-title">Citations ({referenceEntry.citations.length})</h3>
					<div class="citation-groups">
						{#each [...citationsByWork.entries()] as [workName, citations] (workName)}
							<div class="citation-work-group">
								<h4 class="work-group-title">{workName}</h4>
								<ul class="citation-list">
									{#each citations as c, i (i)}
										<li>
											<span class="citation-ref">{formatLoc(c)}</span>
										</li>
									{/each}
								</ul>
							</div>
						{/each}
					</div>
				</section>
			{/if}
		{/if}
	</div>
</aside>

<style>
	.drawer-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(0, 0, 0, 0.3);
		z-index: 500;
	}

	.drawer {
		position: fixed;
		top: 0;
		right: 0;
		bottom: 0;
		width: 380px;
		max-width: 90vw;
		background: var(--color-elevated);
		border-left: 1px solid var(--color-border);
		z-index: 600;
		display: flex;
		flex-direction: column;
		animation: drawer-slide-in 0.2s ease-out;
		box-shadow: -4px 0 24px rgba(0, 0, 0, 0.3);
	}

	@keyframes drawer-slide-in {
		from {
			transform: translateX(100%);
		}
		to {
			transform: translateX(0);
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.drawer {
			animation: none;
		}
	}

	.drawer-header {
		display: flex;
		align-items: flex-start;
		gap: 8px;
		padding: 16px 16px 12px;
		border-bottom: 1px solid var(--color-border);
		flex-shrink: 0;
	}

	.header-content {
		flex: 1;
		min-width: 0;
	}

	.headword {
		margin: 0;
		font-size: 1.3rem;
		font-weight: 700;
		color: var(--color-text);
	}

	.source-label {
		display: block;
		font-size: 0.65rem;
		color: var(--color-text-muted);
		margin-top: 2px;
	}

	.header-definition {
		margin: 8px 0 0;
		font-size: 0.8rem;
		color: var(--color-text-secondary);
		line-height: 1.5;
		display: -webkit-box;
		-webkit-line-clamp: 3;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	.drawer-body {
		flex: 1;
		overflow-y: auto;
		padding: 16px;
	}

	.loading,
	.error {
		text-align: center;
		padding: 24px 0;
		font-size: 0.85rem;
		color: var(--color-text-muted);
	}

	/* ─── Senses (Schmidt) ─── */
	.sub-entry-header {
		display: flex;
		align-items: baseline;
		gap: 8px;
		margin-top: 16px;
		padding-bottom: 4px;
		border-bottom: 1px solid var(--color-border);
	}

	.sub-entry-key {
		margin: 0;
		font-size: 1rem;
		font-weight: 700;
		color: var(--color-accent);
	}

	.sub-entry-type {
		font-size: 0.75rem;
		font-style: italic;
		color: var(--color-text-muted);
	}

	.senses {
		margin-bottom: 16px;
	}

	.sense-block {
		margin-bottom: 12px;
	}

	.sense-block:last-child {
		margin-bottom: 0;
	}

	.sense {
		display: flex;
		gap: 6px;
	}

	.sense-num {
		color: var(--color-accent);
		font-weight: 600;
		flex-shrink: 0;
		min-width: 20px;
		font-size: 0.85rem;
	}

	.sense-num.sub {
		font-weight: 500;
		font-style: italic;
	}

	.sense-block.sub-sense {
		margin-left: 16px;
	}

	.sense-def {
		margin: 0;
		color: var(--color-text-secondary);
		font-size: 0.85rem;
		line-height: 1.6;
	}

	.full-text {
		margin-bottom: 16px;
	}

	.full-text p {
		margin: 0;
		color: var(--color-text-secondary);
		font-size: 0.85rem;
		line-height: 1.6;
	}

	/* ─── Reference entry ─── */
	.entry-text {
		margin-bottom: 16px;
	}

	.definition {
		margin: 0;
		font-size: 0.85rem;
		color: var(--color-text-secondary);
		line-height: 1.7;
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

	/* ─── Citations ─── */
	.citations-section {
		border-top: 1px solid var(--color-border);
		padding-top: 12px;
	}

	.citations-title {
		margin: 0 0 8px;
		font-size: 0.75rem;
		font-weight: 700;
		color: var(--color-text);
		text-transform: uppercase;
		letter-spacing: 0.03em;
	}

	.citation-groups {
		display: flex;
		flex-direction: column;
		gap: 10px;
	}

	.work-group-title {
		margin: 0 0 2px;
		font-size: 0.7rem;
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

	.citation-ref {
		display: inline-block;
		padding: 2px 6px;
		font-size: 0.7rem;
		font-weight: 600;
		color: var(--color-text-secondary);
		border-radius: 3px;
	}

	/* ─── Reference works ─── */
	.reference-works {
		margin-top: 16px;
		padding-top: 12px;
		border-top: 1px solid var(--color-border);
	}

	.ref-section-title {
		margin: 0 0 6px;
		font-size: 0.75rem;
		font-weight: 700;
		color: var(--color-text);
	}

	.ref-citation-list {
		list-style: none;
		padding: 0 0 0 8px;
		margin: 4px 0 0;
	}

	.ref-citation-item {
		padding: 2px 0;
		font-size: 0.7rem;
		color: var(--color-text-secondary);
	}

	.ref-location {
		font-weight: 600;
		color: var(--color-text);
		margin-right: 4px;
	}

	.ref-location.clickable {
		border: none;
		background: none;
		font: inherit;
		font-weight: 600;
		color: var(--color-accent);
		cursor: pointer;
		padding: 0;
	}

	.ref-location.clickable:hover {
		text-decoration: underline;
	}

	.ref-entry-link {
		border: none;
		background: none;
		font: inherit;
		font-size: 0.6rem;
		color: var(--color-text-muted);
		cursor: pointer;
		padding: 1px 4px;
		text-decoration: underline;
		text-decoration-style: dotted;
	}

	.ref-entry-link:hover {
		color: var(--color-accent);
	}

	.source-hint {
		margin: 0 0 6px;
		font-size: 0.65rem;
		color: var(--color-text-muted);
		font-style: italic;
	}

	.ref-citation-item {
		display: flex;
		align-items: baseline;
		gap: 6px;
	}

	@media (max-width: 600px) {
		.drawer {
			width: 100vw;
			max-width: 100vw;
		}
	}
</style>
