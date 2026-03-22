<script lang="ts">
	import IconSearch from '$lib/components/icons/IconSearch.svelte';
	import IconClose from '$lib/components/icons/IconClose.svelte';

	let {
		value = $bindable(''),
		placeholder = 'Search...',
		autofocus = false
	}: {
		value?: string;
		placeholder?: string;
		autofocus?: boolean;
	} = $props();
</script>

<div class="search-wrapper">
	<span class="search-icon">
		<IconSearch size={18} />
	</span>
	<!-- svelte-ignore a11y_autofocus -->
	<input
		type="text"
		class="search-input"
		bind:value
		{placeholder}
		autofocus={autofocus || undefined}
	/>
	{#if value}
		<button class="clear-btn" onclick={() => value = ''} aria-label="Clear search">
			<IconClose size={16} />
		</button>
	{/if}
</div>

<style>
	.search-wrapper {
		position: relative;
		width: 100%;
	}

	.search-icon {
		position: absolute;
		left: 14px;
		top: 50%;
		transform: translateY(-50%);
		color: var(--color-text-muted);
		pointer-events: none;
		display: flex;
	}

	.search-input {
		width: 100%;
		padding: 10px 40px 10px 42px;
		border: 1px solid var(--color-border);
		background: var(--color-surface);
		color: var(--color-text);
		font-family: inherit;
		font-size: 0.95rem;
		border-radius: 12px;
		outline: none;
		box-sizing: border-box;
	}

	.search-input:focus {
		border-color: var(--color-accent);
		box-shadow: 0 0 0 3px var(--color-hover);
	}

	.search-input::placeholder {
		color: var(--color-text-muted);
	}

	.clear-btn {
		position: absolute;
		right: 10px;
		top: 50%;
		transform: translateY(-50%);
		display: flex;
		align-items: center;
		justify-content: center;
		width: 24px;
		height: 24px;
		border: none;
		background: none;
		color: var(--color-text-muted);
		cursor: pointer;
		border-radius: 4px;
	}

	.clear-btn:hover {
		color: var(--color-text);
	}
</style>
