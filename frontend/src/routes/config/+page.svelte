<script lang="ts">
	import { apiFetch, getBaseUrl } from '$lib/api';
	import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';
	import ErrorBanner from '$lib/components/ErrorBanner.svelte';

	interface Settings {
		workdir: string;
		default_provider: string;
		default_model: string;
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

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-xl font-semibold text-[var(--text-primary)]">Settings</h1>
			<p class="mt-1 text-xs text-[var(--text-tertiary)]">Global cure user configuration (~/.cure/config.json). Project-specific settings are managed per project.</p>
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
			<p class="mt-1 text-xs text-[var(--text-tertiary)]">Root directory for cure-managed repository clones and isolated agent workspaces</p>
		</section>

		<!-- Default AI Provider -->
		<section class="rounded-lg border border-[var(--border)] bg-[var(--bg-secondary)] p-5">
			<h2 class="mb-4 text-xs font-semibold uppercase tracking-wider text-[var(--text-tertiary)]">Default AI Provider</h2>
			<p class="mb-3 text-xs text-[var(--text-tertiary)]">Used when a project doesn't specify its own provider. Override per project in project configuration.</p>
			<div class="grid gap-4 sm:grid-cols-2">
				<div>
					<label class="block mb-1 text-sm text-[var(--text-secondary)]">Provider</label>
					<select
						bind:value={settings.default_provider}
						class="w-full rounded-md border border-[var(--border)] bg-[var(--bg-primary)] px-3 py-2 text-sm text-[var(--text-primary)] focus:border-[var(--accent)]/50 focus:outline-none"
					>
						<option value="claude-code">Claude Code (CLI)</option>
						<option value="claude">Claude (API)</option>
						<option value="openai">OpenAI</option>
						<option value="gemini">Gemini</option>
					</select>
				</div>
				<div>
					<label class="block mb-1 text-sm text-[var(--text-secondary)]">Model</label>
					<input
						bind:value={settings.default_model}
						type="text"
						class="w-full rounded-md border border-[var(--border)] bg-[var(--bg-primary)] px-3 py-2 font-mono text-sm text-[var(--text-primary)] focus:border-[var(--accent)]/50 focus:outline-none"
					/>
				</div>
			</div>
		</section>
	{/if}
</div>
