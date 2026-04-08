<script lang="ts">
	import { apiFetch } from '$lib/api';
	import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';
	import ErrorBanner from '$lib/components/ErrorBanner.svelte';

	let loading = $state(true);
	let error = $state<string | null>(null);
	let rawConfig = $state<unknown>(null);

	const highlightedJson = $derived(rawConfig !== null ? highlightJson(rawConfig) : '');

	async function fetchConfig() {
		loading = true;
		error = null;
		try {
			rawConfig = await apiFetch<unknown>('/api/config');
		} catch (err) {
			error = err instanceof Error ? err.message : 'Unknown error';
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		fetchConfig();
	});

	/**
	 * Minimal JSON syntax highlighter. Input is the parsed JSON value
	 * (from JSON.parse / apiFetch), so the stringify output is safe for
	 * injection via {@html} — it contains only JSON primitives.
	 */
	function highlightJson(value: unknown): string {
		const json = JSON.stringify(value, null, 2);
		return json.replace(
			/("(?:\\.|[^"\\])*")\s*(:)?|(\b(?:true|false|null)\b)|(-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)/g,
			(match, str: string | undefined, colon: string | undefined, keyword: string | undefined, num: string | undefined) => {
				if (str) {
					const escaped = str
						.replace(/&/g, '&amp;')
						.replace(/</g, '&lt;')
						.replace(/>/g, '&gt;');
					if (colon) {
						return `<span style="color:#79c0ff">${escaped}</span>:`;
					}
					return `<span style="color:#a5d6ff">${escaped}</span>`;
				}
				if (keyword) {
					return `<span style="color:#ff7b72">${keyword}</span>`;
				}
				if (num) {
					return `<span style="color:#ff7b72">${num}</span>`;
				}
				return match;
			}
		);
	}
</script>

<svelte:head>
	<title>Config - cure</title>
</svelte:head>

<div class="space-y-6">
	<h1 class="text-xl font-semibold tracking-tight text-[#e6edf3]">Config</h1>

	{#if loading}
		<div class="flex items-center justify-center py-12">
			<LoadingSpinner />
		</div>
	{:else if error}
		<ErrorBanner message={error} onDismiss={() => (error = null)} />
	{:else}
		<div class="overflow-x-auto rounded-lg border border-white/10 bg-[#161b22] p-4">
			<pre class="text-sm leading-relaxed text-[#e6edf3]"><code>{@html highlightedJson}</code></pre>
		</div>
	{/if}
</div>
