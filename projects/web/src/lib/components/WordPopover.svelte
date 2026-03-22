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
		const left = Math.min(x, window.innerWidth - 300);
		const above = y > 350;
		if (above) {
			return `left: ${Math.max(8, left)}px; bottom: ${window.innerHeight - y + 8}px;`;
		}
		return `left: ${Math.max(8, left)}px; top: ${y + 8}px;`;
	});

	// Group references by source
	let bySource = $derived.by(() => {
		const groups = new Map<string, LineReference[]>();
		for (const ref of references) {
			const key = ref.source_code;
			const list = groups.get(key) ?? [];
			list.push(ref);
			groups.set(key, list);
		}
		return groups;
	});

	const SOURCE_LABELS: Record<string, string> = {
		schmidt: 'Schmidt',
		onions: 'Onions',
		abbott: 'Abbott',
		bartlett: 'Bartlett',
		henley_farmer: 'Henley & Farmer'
	};
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
					<span class="entry-key">{ref.entry_key}</span>
					{#if ref.definition}
						<p class="entry-def">{ref.definition}</p>
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
		width: 300px;
		max-height: 360px;
		overflow-y: auto;
		background: var(--color-elevated);
		border: 1px solid var(--color-border);
		border-radius: 10px;
		box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
		padding: 8px 0;
	}

	.popover-header {
		padding: 4px 12px 6px;
		border-bottom: 1px solid var(--color-border);
	}

	.popover-word {
		font-size: 0.9rem;
		font-weight: 700;
		color: var(--color-accent);
	}

	.source-group {
		padding: 4px 0 0;
	}

	.source-label {
		padding: 2px 12px;
		font-size: 0.6rem;
		font-weight: 700;
		color: var(--color-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.popover-entry {
		display: block;
		width: 100%;
		padding: 6px 12px;
		border: none;
		border-bottom: 1px solid color-mix(in srgb, var(--color-border) 30%, transparent);
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

	.entry-def {
		margin: 3px 0 0;
		font-size: 0.7rem;
		color: var(--color-text-secondary);
		line-height: 1.4;
		display: -webkit-box;
		-webkit-line-clamp: 3;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	.entry-quote {
		margin: 3px 0 0;
		font-size: 0.65rem;
		color: var(--color-text-muted);
		font-style: italic;
		line-height: 1.3;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
</style>
