<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		label,
		count,
		children
	}: {
		label: string;
		count?: number;
		children: Snippet;
	} = $props();
</script>

<details class="collapsible">
	<summary class="toggle">{label}{count != null ? ` (${count})` : ''}</summary>
	<div class="content">
		{@render children()}
	</div>
</details>

<style>
	.collapsible {
		margin-top: 8px;
	}

	.toggle {
		margin: 0 0 4px;
		font-size: 0.7rem;
		font-weight: 600;
		color: var(--color-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.06em;
		cursor: pointer;
		list-style: none;
	}

	.toggle::-webkit-details-marker {
		display: none;
	}

	.toggle::before {
		content: '\25B6';
		display: inline-block;
		font-size: 0.55rem;
		margin-right: 6px;
		transition: transform 0.15s;
	}

	:global(details[open]) > .toggle::before {
		transform: rotate(90deg);
	}
</style>
