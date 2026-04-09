<script lang="ts">
	import { marked } from 'marked';
	import hljs from 'highlight.js/lib/core';
	import go from 'highlight.js/lib/languages/go';
	import typescript from 'highlight.js/lib/languages/typescript';
	import javascript from 'highlight.js/lib/languages/javascript';
	import python from 'highlight.js/lib/languages/python';
	import bash from 'highlight.js/lib/languages/bash';
	import json from 'highlight.js/lib/languages/json';
	import yaml from 'highlight.js/lib/languages/yaml';
	import rust from 'highlight.js/lib/languages/rust';
	import sql from 'highlight.js/lib/languages/sql';
	import xml from 'highlight.js/lib/languages/xml';
	import css from 'highlight.js/lib/languages/css';
	import markdown from 'highlight.js/lib/languages/markdown';

	hljs.registerLanguage('go', go);
	hljs.registerLanguage('typescript', typescript);
	hljs.registerLanguage('ts', typescript);
	hljs.registerLanguage('javascript', javascript);
	hljs.registerLanguage('js', javascript);
	hljs.registerLanguage('python', python);
	hljs.registerLanguage('bash', bash);
	hljs.registerLanguage('sh', bash);
	hljs.registerLanguage('shell', bash);
	hljs.registerLanguage('json', json);
	hljs.registerLanguage('yaml', yaml);
	hljs.registerLanguage('yml', yaml);
	hljs.registerLanguage('rust', rust);
	hljs.registerLanguage('sql', sql);
	hljs.registerLanguage('html', xml);
	hljs.registerLanguage('xml', xml);
	hljs.registerLanguage('css', css);
	hljs.registerLanguage('markdown', markdown);
	hljs.registerLanguage('md', markdown);

	interface Props {
		role: 'user' | 'assistant';
		content: string;
		streaming?: boolean;
	}

	let { role, content, streaming = false }: Props = $props();

	const isUser = $derived(role === 'user');

	marked.setOptions({
		breaks: true,
		gfm: true,
	});

	const renderer = new marked.Renderer();
	renderer.code = function({ text, lang }: { text: string; lang?: string }) {
		const language = lang && hljs.getLanguage(lang) ? lang : undefined;
		const highlighted = language
			? hljs.highlight(text, { language }).value
			: hljs.highlightAuto(text).value;
		const langLabel = lang ? `<span class="code-lang">${lang}</span>` : '';
		return `<pre class="hljs-pre">${langLabel}<code class="hljs">${highlighted}</code></pre>`;
	};
	marked.use({ renderer });

	const renderedContent = $derived(
		isUser ? '' : (marked.parse(content) as string)
	);
</script>

{#if isUser}
	<!-- User: right-aligned compact bubble -->
	<div class="flex justify-end">
		<div class="rounded-lg bg-[var(--accent)]/10 px-4 py-2.5 text-sm text-[var(--text-primary)] max-w-[85%] md:max-w-[70%]">
			<p class="whitespace-pre-wrap break-words">{content}</p>
		</div>
	</div>
{:else}
	<!-- Assistant: full-width prose in subtle bubble -->
	<div class="prose rounded-lg bg-[var(--bg-secondary)] px-5 py-4">
		{@html renderedContent}
		{#if streaming}
			<span class="ml-1 inline-block w-1.5 h-4 bg-[var(--accent)] animate-pulse align-middle"></span>
		{/if}
	</div>
{/if}

<style>
	.prose {
		font-size: 0.875rem;
		line-height: 1.7;
		color: var(--text-primary);
	}
	.prose :global(p) { margin: 0.5em 0; }
	.prose :global(p:first-child) { margin-top: 0; }
	.prose :global(p:last-child) { margin-bottom: 0; }
	.prose :global(code) {
		background: var(--bg-tertiary);
		padding: 0.15em 0.4em;
		border-radius: 4px;
		font-size: 0.85em;
		color: var(--accent);
	}
	.prose :global(pre) {
		background: var(--bg-primary);
		border: 1px solid var(--border);
		border-radius: 8px;
		padding: 1em 1.25em;
		overflow-x: auto;
		margin: 0.75em 0;
	}
	.prose :global(pre code) {
		background: none;
		padding: 0;
		color: var(--text-primary);
		font-size: 0.85em;
	}
	.prose :global(ul), .prose :global(ol) {
		padding-left: 1.5em;
		margin: 0.5em 0;
	}
	.prose :global(li) { margin: 0.25em 0; }
	.prose :global(blockquote) {
		border-left: 3px solid var(--accent);
		padding-left: 1em;
		margin: 0.75em 0;
		color: var(--text-secondary);
	}
	.prose :global(a) {
		color: var(--accent);
		text-decoration: underline;
	}
	.prose :global(strong) { font-weight: 600; }
	.prose :global(em) { font-style: italic; }
	.prose :global(h1), .prose :global(h2), .prose :global(h3) {
		font-weight: 600;
		margin: 1em 0 0.25em;
	}
	.prose :global(h1) { font-size: 1.25em; }
	.prose :global(h2) { font-size: 1.125em; }
	.prose :global(h3) { font-size: 1em; }
	.prose :global(table) {
		border-collapse: collapse;
		margin: 0.75em 0;
		font-size: 0.85em;
		width: 100%;
	}
	.prose :global(th), .prose :global(td) {
		border: 1px solid var(--border);
		padding: 0.5em 0.75em;
		text-align: left;
	}
	.prose :global(th) {
		background: var(--bg-tertiary);
		font-weight: 600;
	}
	.prose :global(hr) {
		border: none;
		border-top: 1px solid var(--border);
		margin: 1em 0;
	}
	/* highlight.js syntax colors (GitHub-dark inspired) */
	.prose :global(.hljs-pre) { position: relative; }
	.prose :global(.code-lang) {
		position: absolute;
		top: 0.5em;
		right: 0.75em;
		font-size: 0.7em;
		color: var(--text-tertiary);
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}
	.prose :global(.hljs-keyword) { color: #ff7b72; }
	.prose :global(.hljs-string) { color: #a5d6ff; }
	.prose :global(.hljs-number) { color: #79c0ff; }
	.prose :global(.hljs-function) { color: #d2a8ff; }
	.prose :global(.hljs-title) { color: #d2a8ff; }
	.prose :global(.hljs-params) { color: #e6edf3; }
	.prose :global(.hljs-comment) { color: #8b949e; font-style: italic; }
	.prose :global(.hljs-type) { color: #79c0ff; }
	.prose :global(.hljs-built_in) { color: #ffa657; }
	.prose :global(.hljs-literal) { color: #79c0ff; }
	.prose :global(.hljs-attr) { color: #79c0ff; }
	.prose :global(.hljs-selector-class) { color: #7ee787; }
	.prose :global(.hljs-selector-tag) { color: #7ee787; }
	.prose :global(.hljs-meta) { color: #8b949e; }
	.prose :global(.hljs-variable) { color: #ffa657; }
	.prose :global(.hljs-property) { color: #79c0ff; }
</style>
