<script lang="ts">
	import { apiFetch, getBaseUrl } from '$lib/api';
	import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';
	import ErrorBanner from '$lib/components/ErrorBanner.svelte';

	let loading = $state(true);
	let error = $state<string | null>(null);
	let configText = $state('');
	let originalText = $state('');
	let saving = $state(false);
	let editing = $state(false);

	const dirty = $derived(configText !== originalText);

	async function fetchConfig() {
		loading = true;
		error = null;
		try {
			const data = await apiFetch<unknown>('/api/config');
			const json = JSON.stringify(data, null, 2);
			configText = json;
			originalText = json;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Unknown error';
		} finally {
			loading = false;
		}
	}

	async function saveConfig() {
		saving = true;
		error = null;
		try {
			// Validate JSON
			JSON.parse(configText);
			const base = getBaseUrl();
			const res = await fetch(`${base}/api/config`, {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: configText
			});
			if (!res.ok) throw new Error(`Save failed: ${res.status}`);
			originalText = configText;
			editing = false;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Invalid JSON or save failed';
		} finally {
			saving = false;
		}
	}

	$effect(() => {
		fetchConfig();
	});
</script>

<svelte:head>
	<title>Config - cure</title>
</svelte:head>

<div class="space-y-4">
	<div class="flex items-center justify-between">
		<h1 class="text-xl font-semibold tracking-tight text-[var(--text-primary)]">Config</h1>
		<div class="flex gap-2">
			{#if !editing}
				<button
					onclick={() => (editing = true)}
					class="rounded-md bg-[var(--bg-tertiary)] px-3 py-1.5 text-xs text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
				>
					Edit
				</button>
			{:else}
				{#if dirty}
					<button
						onclick={() => { configText = originalText; editing = false; }}
						class="rounded-md bg-[var(--bg-tertiary)] px-3 py-1.5 text-xs text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
					>
						Discard
					</button>
				{/if}
				<button
					onclick={saveConfig}
					disabled={!dirty || saving}
					class="rounded-md bg-[var(--accent)] px-3 py-1.5 text-xs text-white disabled:opacity-50"
				>
					{saving ? 'Saving...' : 'Save'}
				</button>
			{/if}
		</div>
	</div>

	<p class="text-xs text-[var(--text-tertiary)]">Effective merged configuration (defaults + global + project + local + env). Edit saves to local .cure.json.</p>

	{#if loading}
		<div class="flex items-center justify-center py-12"><LoadingSpinner /></div>
	{:else if error}
		<ErrorBanner message={error} onDismiss={() => (error = null)} />
	{/if}

	{#if editing}
		<textarea
			bind:value={configText}
			class="w-full rounded-lg border border-[var(--border)] bg-[var(--bg-primary)] p-4 font-mono text-sm text-[var(--text-primary)] focus:border-[var(--accent)]/50 focus:outline-none"
			rows={20}
			spellcheck="false"
		></textarea>
	{:else if !loading}
		<pre class="overflow-x-auto rounded-lg border border-[var(--border)] bg-[var(--bg-secondary)] p-4 text-sm leading-relaxed font-mono text-[var(--text-primary)]">{configText}</pre>
	{/if}
</div>
