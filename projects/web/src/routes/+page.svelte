<svelte:head>
	<title>Bardbase — Shakespeare with Scholarly References</title>
</svelte:head>

<div class="home">
	<header class="hero">
		<h1 class="title">Bardbase</h1>
		<p class="subtitle">Shakespeare's complete works with integrated scholarly references</p>
	</header>

	<section class="section">
		<p>Bardbase pairs Shakespeare's plays and poetry with five major scholarly reference works, cross-referenced against the text so you can look up a word or construction without leaving the page. Everything is free and open source.</p>
		<p>Open the <span class="key">☰</span> menu to start reading. <a href="/help">How to use Bardbase →</a></p>
	</section>

	<section class="section">
		<h2 class="section-title">References</h2>
		<ul class="ref-list">
			<li><strong>Schmidt Shakespeare Lexicon</strong> — definitions of every word Shakespeare used, with citations</li>
			<li><strong>Onions Glossary</strong> — concise glossary of archaic and changed-meaning words</li>
			<li><strong>Abbott Grammar</strong> — Shakespeare's grammar, syntax, and rhetorical devices</li>
			<li><strong>Bartlett's Concordance</strong> — index of every significant word across all works</li>
			<li><strong>Henley &amp; Farmer</strong> — historical slang, cant, and colloquial language</li>
		</ul>
		<p><a href="/references">Browse all references →</a></p>
	</section>

	<section class="section">
		<h2 class="section-title">Project</h2>
		<p>Bardbase is an open-source project. The source code and compiled SQLite database are publicly available on GitHub.</p>
		<p class="meta-links">
			<a href="/about">About &amp; goals</a>
			<span class="sep">·</span>
			<a href="https://github.com/scottdkey/bardbase" target="_blank" rel="noopener">GitHub</a>
			<span class="sep">·</span>
			<a href="/corrections">Corrections</a>
		</p>
	</section>
</div>

<style>
	.home {
		max-width: 600px;
		margin: 0 auto;
		padding: 40px 24px 80px;
	}

	.hero {
		margin-bottom: 32px;
		padding-bottom: 24px;
		border-bottom: 1px solid var(--color-border);
	}

	.title {
		margin: 0 0 8px;
		font-size: 2.2rem;
		font-weight: 700;
		color: var(--color-text);
		letter-spacing: -0.01em;
	}

	.subtitle {
		margin: 0;
		font-size: 1rem;
		color: var(--color-text-secondary);
		line-height: 1.5;
	}

	.section {
		margin-bottom: 28px;
	}

	.section-title {
		margin: 0 0 10px;
		font-size: 0.75rem;
		font-weight: 700;
		color: var(--color-accent);
		text-transform: uppercase;
		letter-spacing: 0.06em;
	}

	.section p {
		margin: 0 0 8px;
		font-size: 0.88rem;
		color: var(--color-text-secondary);
		line-height: 1.7;
	}

	.section p:last-child { margin-bottom: 0; }

	.ref-list {
		margin: 0 0 10px;
		padding-left: 20px;
		font-size: 0.88rem;
		color: var(--color-text-secondary);
		line-height: 1.8;
	}

	.ref-list li { margin-bottom: 2px; }

	.key {
		display: inline-block;
		padding: 0 5px;
		border: 1px solid var(--color-border);
		border-radius: 4px;
		font-size: 0.78rem;
		color: var(--color-text-muted);
		background: var(--color-surface);
		vertical-align: middle;
	}

	.meta-links {
		font-size: 0.85rem;
	}

	.sep {
		margin: 0 6px;
		color: var(--color-text-muted);
	}
</style>
