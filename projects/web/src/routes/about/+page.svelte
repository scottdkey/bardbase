<script>
	export const prerender = true;
</script>

<svelte:head>
	<title>About &mdash; Bardbase</title>
</svelte:head>

<div class="about-page">
	<header class="page-header">
		<h1>About Bardbase</h1>
	</header>

	<section class="section">
		<h2>What it is</h2>
		<p>Bardbase is a free, open-source reading environment for Shakespeare's complete works. It pairs the texts with five major scholarly reference works — all integrated so you can look up a word, phrase, or grammatical construction without leaving the page.</p>
		<p>The goal is to make reading Shakespeare less frustrating. The language barrier is real. These reference works exist precisely to bridge it, but they're scattered across physical volumes and paywalled digitizations. Bardbase puts them in one place, cross-referenced against the text.</p>
	</section>

	<section class="section">
		<h2>The texts</h2>
		<p>Plays are presented in multiple editions side by side — primarily the First Folio alongside modern critical editions — so you can see where the texts diverge. Line numbers correspond to the source edition. Poetry is presented as a single text.</p>
	</section>

	<section class="section">
		<h2>The reference works</h2>
		<ul class="ref-list">
			<li><strong>Schmidt Shakespeare Lexicon</strong> (1902) — Alexander Schmidt's exhaustive dictionary of every word Shakespeare used, with definitions and citations from the plays and poems.</li>
			<li><strong>Onions Shakespeare Glossary</strong> (1911) — C. T. Onions' concise glossary of words that have changed meaning or fallen out of use since Shakespeare's time.</li>
			<li><strong>Abbott Shakespearian Grammar</strong> (1877) — E. A. Abbott's systematic account of Shakespeare's grammar, syntax, and rhetorical devices.</li>
			<li><strong>Bartlett's Complete Concordance</strong> (1896) — John Bartlett's index of every significant word across all works, with act/scene/line citations.</li>
			<li><strong>Henley &amp; Farmer Slang and Its Analogues</strong> (1890–1904) — W. E. Henley and John S. Farmer's historical dictionary of slang, cant, and colloquial language.</li>
		</ul>
		<p>All reference works are in the public domain. The compiled database is released under <strong>CC BY-SA 4.0</strong> due to Perseus content. Build tooling is <strong>MIT</strong>.</p>
	</section>

	<section class="section">
		<h2>Open source</h2>
		<p>The source code and compiled database are both publicly available.</p>
		<ul class="link-list">
			<li><a href="https://github.com/scottdkey/bardbase" target="_blank" rel="noopener">github.com/scottdkey/bardbase</a> — source code, build pipeline, and issue tracker</li>
			<li><a href="https://github.com/scottdkey/bardbase/releases" target="_blank" rel="noopener">Compiled SQLite database</a> — download the full bardbase.db for local use or integration</li>
		</ul>
	</section>

	<section class="section">
		<h2>Built by</h2>
		<p>Scott Key — <a href="https://scottkey.dev" target="_blank" rel="noopener">scottkey.dev</a></p>
	</section>
</div>

<style>
	.about-page {
		max-width: 660px;
		margin: 0 auto;
		padding: 32px 24px 80px;
	}

	.page-header {
		margin-bottom: 36px;
		padding-bottom: 20px;
		border-bottom: 1px solid var(--color-border);
	}

	.page-header h1 {
		margin: 0;
		font-size: 2rem;
		font-weight: 700;
		color: var(--color-text);
		letter-spacing: -0.01em;
	}

	.section {
		margin-bottom: 32px;
	}

	.section h2 {
		margin: 0 0 10px;
		font-size: 0.75rem;
		font-weight: 700;
		color: var(--color-accent);
		text-transform: uppercase;
		letter-spacing: 0.06em;
	}

	.section p {
		margin: 0 0 10px;
		font-size: 0.88rem;
		color: var(--color-text-secondary);
		line-height: 1.7;
	}

	.section p:last-child { margin-bottom: 0; }

	.ref-list, .link-list {
		margin: 6px 0 10px;
		padding-left: 20px;
		font-size: 0.88rem;
		color: var(--color-text-secondary);
		line-height: 1.8;
	}

	.ref-list li, .link-list li { margin-bottom: 4px; }
</style>
