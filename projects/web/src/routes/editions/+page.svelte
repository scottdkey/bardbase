<script lang="ts">
	import PageHeader from '$lib/components/ui/PageHeader.svelte';
</script>

<svelte:head>
	<title>Editions &mdash; Bardbase</title>
</svelte:head>

<div class="editions-page">
	<PageHeader title="Editions" />

	<p class="placeholder">Edition browser coming soon.</p>
</div>

<style>
	.editions-page {
		max-width: 720px;
		margin: 0 auto;
		padding: 0 16px 60px;
	}

	.placeholder {
		color: var(--color-text-muted);
		text-align: center;
		padding: 40px 0;
	}
</style>
