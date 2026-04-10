<script lang="ts">
	import { apiFetch, getBaseUrl } from '$lib/api';
	import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';
	import ErrorBanner from '$lib/components/ErrorBanner.svelte';

	interface Settings {
		workdir: string;
		default_provider: string;
		default_model: string;
		max_tokens: number;
		output_format: string;
		timeout: number;
		verbose: boolean;
		redact: boolean;
	}

	let settings = $state<Settings | null>(null);
	let original = $state('');
	let loading = $state(true);
	let error = $state<string | null>(null);
	let saving = $state(false);
	let saved = $state(false);

	const dirty = $derived(settings !== null && JSON.stringify(settings) !== original);

	async function fetchSettings() {
		loading = true;
		try {
			settings = await apiFetch<Settings>('/api/settings');
			original = JSON.stringify(settings);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load settings';
		} finally {
			loading = false;
		}
	}

	async function saveSettings() {
		if (!settings || !dirty) return;
		saving = true;
		saved = false;
		error = null;
		try {
			const base = getBaseUrl();
			const res = await fetch(`${base}/api/settings`, {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(settings)
			});
			if (!res.ok) throw new Error(`Save failed: ${res.status}`);
			original = JSON.stringify(settings);
			saved = true;
			setTimeout(() => (saved = false), 2000);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to save';
		} finally {
			saving = false;
		}
	}

	$effect(() => { fetchSettings(); });
</script>

<svelte:head>
	<title>Settings - cure</title>
</svelte:head>

<div class="space-y-6 max-w-2xl">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-xl font-semibold text-[var(--text-primary)]">Settings</h1>
			<p class="mt-1 text-xs text-[var(--text-tertiary)]">Global cure configuration. Saved to ~/.cure/config.json</p>
		</div>
		<div class="flex items-center gap-2">
			{#if saved}
				<span class="text-xs text-[var(--success)]">Saved</span>
			{/if}
			<button
				onclick={saveSettings}
				disabled={!dirty || saving}
				class="rounded-md bg-[var(--accent)] px-4 py-2 text-sm font-medium text-white disabled:opacity-50"
			>
				{saving ? 'Saving...' : 'Save'}
			</button>
		</div>
	</div>

	{#if error}
		<ErrorBanner message={error} onDismiss={() => (error = null)} />
	{/if}

	{#if loading}
		<div class="flex items-center justify-center py-12"><LoadingSpinner /></div>
	{:else if settings}

		<!-- Workspace -->
		<section class="rounded-lg border border-[var(--border)] bg-[var(--bg-secondary)] p-5">
			<h2 class="mb-4 text-xs font-semibold uppercase tracking-wider text-[var(--text-tertiary)]">Workspace</h2>
			<label class="block mb-1 text-sm text-[var(--text-secondary)]">Working Directory</label>
			<input
				bind:value={settings.workdir}
				type="text"
				class="w-full rounded-md border border-[var(--border)] bg-[var(--bg-primary)] px-3 py-2 font-mono text-sm text-[var(--text-primary)] focus:border-[var(--accent)]/50 focus:outline-none"
				placeholder="~/.cure/workdir"
			/>
			<p class="mt-1 text-xs text-[var(--text-tertiary)]">Root directory for cure-managed repository clones</p>
		</section>

		<!-- AI Provider -->
		<section class="rounded-lg border border-[var(--border)] bg-[var(--bg-secondary)] p-5">
			<h2 class="mb-4 text-xs font-semibold uppercase tracking-wider text-[var(--text-tertiary)]">AI Provider</h2>
			<div class="grid gap-4 sm:grid-cols-2">
				<div>
					<label class="block mb-1 text-sm text-[var(--text-secondary)]">Default Provider</label>
					<select
						bind:value={settings.default_provider}
						class="w-full rounded-md border border-[var(--border)] bg-[var(--bg-primary)] px-3 py-2 text-sm text-[var(--text-primary)] focus:border-[var(--accent)]/50 focus:outline-none"
					>
						<option value="claude">Claude</option>
						<option value="openai">OpenAI</option>
						<option value="gemini">Gemini</option>
					</select>
				</div>
				<div>
					<label class="block mb-1 text-sm text-[var(--text-secondary)]">Default Model</label>
					<input
						bind:value={settings.default_model}
						type="text"
						class="w-full rounded-md border border-[var(--border)] bg-[var(--bg-primary)] px-3 py-2 font-mono text-sm text-[var(--text-primary)] focus:border-[var(--accent)]/50 focus:outline-none"
					/>
				</div>
				<div>
					<label class="block mb-1 text-sm text-[var(--text-secondary)]">Max Tokens</label>
					<input
						bind:value={settings.max_tokens}
						type="number"
						class="w-full rounded-md border border-[var(--border)] bg-[var(--bg-primary)] px-3 py-2 text-sm text-[var(--text-primary)] focus:border-[var(--accent)]/50 focus:outline-none"
					/>
				</div>
				<div>
					<label class="block mb-1 text-sm text-[var(--text-secondary)]">Timeout (seconds)</label>
					<input
						bind:value={settings.timeout}
						type="number"
						class="w-full rounded-md border border-[var(--border)] bg-[var(--bg-primary)] px-3 py-2 text-sm text-[var(--text-primary)] focus:border-[var(--accent)]/50 focus:outline-none"
					/>
				</div>
			</div>
		</section>

		<!-- Output -->
		<section class="rounded-lg border border-[var(--border)] bg-[var(--bg-secondary)] p-5">
			<h2 class="mb-4 text-xs font-semibold uppercase tracking-wider text-[var(--text-tertiary)]">Output</h2>
			<div class="grid gap-4 sm:grid-cols-2">
				<div>
					<label class="block mb-1 text-sm text-[var(--text-secondary)]">Format</label>
					<select
						bind:value={settings.output_format}
						class="w-full rounded-md border border-[var(--border)] bg-[var(--bg-primary)] px-3 py-2 text-sm text-[var(--text-primary)] focus:border-[var(--accent)]/50 focus:outline-none"
					>
						<option value="json">JSON</option>
						<option value="text">Text</option>
					</select>
				</div>
			</div>
			<div class="mt-4 space-y-3">
				<label class="flex items-center gap-3 cursor-pointer">
					<input type="checkbox" bind:checked={settings.verbose} class="rounded border-[var(--border)] bg-[var(--bg-primary)] text-[var(--accent)] focus:ring-[var(--accent)]" />
					<div>
						<span class="text-sm text-[var(--text-primary)]">Verbose</span>
						<p class="text-xs text-[var(--text-tertiary)]">Enable verbose logging output</p>
					</div>
				</label>
				<label class="flex items-center gap-3 cursor-pointer">
					<input type="checkbox" bind:checked={settings.redact} class="rounded border-[var(--border)] bg-[var(--bg-primary)] text-[var(--accent)] focus:ring-[var(--accent)]" />
					<div>
						<span class="text-sm text-[var(--text-primary)]">Redact</span>
						<p class="text-xs text-[var(--text-tertiary)]">Redact sensitive values in output</p>
					</div>
				</label>
			</div>
		</section>
	{/if}
</div>
