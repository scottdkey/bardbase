<script lang="ts">
	import type { LineReference } from '$lib/server/api';

	let {
		word,
		references,
		x,
		y,
		onclose,
		onselect
	}: {
		word: string;
		references: LineReference[];
		x: number;
		y: number;
		onclose: () => void;
		onselect: (ref: LineReference) => void;
	} = $props();

	let popoverStyle = $derived.by(() => {
		const left = Math.min(x, window.innerWidth - 320);
		const above = y > 350;
		if (above) {
			return `left: ${Math.max(8, left)}px; bottom: ${window.innerHeight - y + 8}px;`;
		}
		return `left: ${Math.max(8, left)}px; top: ${y + 8}px;`;
	});

	// Group references by source, ordered: schmidt first, then onions, abbott, bartlett
	const SOURCE_ORDER = ['schmidt', 'onions', 'abbott', 'bartlett', 'henley_farmer'];

	let bySource = $derived.by(() => {
		const groups = new Map<string, LineReference[]>();
		for (const ref of references) {
			const list = groups.get(ref.source_code) ?? [];
			list.push(ref);
			groups.set(ref.source_code, list);
		}
		// Sort by source order
		const sorted = new Map<string, LineReference[]>();
		for (const code of SOURCE_ORDER) {
			if (groups.has(code)) sorted.set(code, groups.get(code)!);
		}
		// Add any remaining
		for (const [code, refs] of groups) {
			if (!sorted.has(code)) sorted.set(code, refs);
		}
		return sorted;
	});

	const SOURCE_LABELS: Record<string, string> = {
		schmidt: 'Schmidt Lexicon',
		onions: 'Onions Glossary',
		abbott: 'Abbott Grammar',
		bartlett: 'Bartlett Concordance',
		henley_farmer: 'Henley & Farmer'
	};

	// For Bartlett, show the headword as a quote snippet, not the full raw_text
	function formatEntryKey(ref: LineReference): string {
		const key = ref.entry_key.replace(/\s+/g, ' ').trim();
		if (key.length > 60) return key.slice(0, 57) + '...';
		return key;
	}

	// For definition display, clean up OCR noise
	function formatDef(def: string | null): string {
		if (!def) return '';
		return def.replace(/\s+/g, ' ').trim();
	}

	// Is this a phrase entry (Bartlett-style)?
	function isPhrase(ref: LineReference): boolean {
		return ref.entry_key.includes(' ') && ref.entry_key.length > 20;
	}
</script>

<div class="popover-backdrop" onclick={onclose} onkeydown={(e) => e.key === 'Escape' && onclose()} role="presentation"></div>
<div class="word-popover" style={popoverStyle}>
	<div class="popover-header">
		<span class="popover-word">{word}</span>
	</div>
	{#each [...bySource.entries()] as [sourceCode, refs] (sourceCode)}
		<div class="source-group">
			<div class="source-label">{SOURCE_LABELS[sourceCode] ?? sourceCode}</div>
			{#each refs as ref, i (ref.entry_id + '-' + (ref.sense_id ?? i))}
				<button class="popover-entry" onclick={() => onselect(ref)}>
					{#if isPhrase(ref)}
						<p class="entry-quote-key">&ldquo;{formatEntryKey(ref)}&rdquo;</p>
					{:else}
						<span class="entry-key">{ref.entry_key}</span>
					{/if}
					{#if ref.definition && !isPhrase(ref)}
						<p class="entry-def">{formatDef(ref.definition)}</p>
					{/if}
					{#if ref.quote_text}
						<p class="entry-quote">{ref.quote_text}</p>
					{/if}
				</button>
			{/each}
		</div>
	{/each}
</div>

<style>
	.popover-backdrop {
		position: fixed;
		inset: 0;
		z-index: 500;
	}

	.word-popover {
		position: fixed;
		z-index: 600;
		width: 320px;
		max-height: 400px;
		overflow-y: auto;
		background: var(--color-elevated);
		border: 1px solid var(--color-border);
		border-radius: 10px;
		box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
		padding: 6px 0;
	}

	.popover-header {
		padding: 6px 12px 8px;
		border-bottom: 1px solid var(--color-border);
	}

	.popover-word {
		font-size: 0.95rem;
		font-weight: 700;
		color: var(--color-accent);
	}

	.source-group {
		padding: 4px 0 0;
	}

	.source-label {
		padding: 4px 12px 2px;
		font-size: 0.55rem;
		font-weight: 700;
		color: var(--color-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.popover-entry {
		display: block;
		width: 100%;
		padding: 5px 12px;
		border: none;
		border-bottom: 1px solid color-mix(in srgb, var(--color-border) 20%, transparent);
		background: none;
		text-align: left;
		cursor: pointer;
		font-family: inherit;
		color: var(--color-text);
		transition: background 0.1s;
	}

	.popover-entry:last-child {
		border-bottom: none;
	}

	.popover-entry:hover {
		background: var(--color-hover);
	}

	.entry-key {
		font-size: 0.8rem;
		font-weight: 700;
		color: var(--color-text);
	}

	.entry-quote-key {
		margin: 0;
		font-size: 0.72rem;
		font-style: italic;
		color: var(--color-text-secondary);
		line-height: 1.3;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	.entry-def {
		margin: 2px 0 0;
		font-size: 0.68rem;
		color: var(--color-text-secondary);
		line-height: 1.4;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	.entry-quote {
		margin: 2px 0 0;
		font-size: 0.62rem;
		color: var(--color-text-muted);
		font-style: italic;
		line-height: 1.3;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
</style>
