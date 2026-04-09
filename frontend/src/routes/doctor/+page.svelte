<script lang="ts">
	import { apiFetch } from '$lib/api';
	import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';
	import ErrorBanner from '$lib/components/ErrorBanner.svelte';

	interface CheckResult {
		name: string;
		status: 'pass' | 'warn' | 'fail';
		message: string;
	}

	let loading = $state(true);
	let error = $state<string | null>(null);
	let results = $state<CheckResult[]>([]);

	async function runDoctor() {
		loading = true;
		error = null;
		try {
			results = await apiFetch<CheckResult[]>('/api/doctor');
		} catch (err) {
			error = err instanceof Error ? err.message : 'Unknown error';
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		runDoctor();
	});

	function badgeClasses(status: string): string {
		switch (status) {
			case 'pass':
				return 'bg-[var(--success)]/15 text-[var(--success)]';
			case 'warn':
				return 'bg-[var(--warning)]/15 text-[var(--warning)]';
			case 'fail':
				return 'bg-[var(--danger)]/15 text-[var(--danger)]';
			default:
				return 'bg-[var(--bg-tertiary)] text-white/50';
		}
	}
</script>

<svelte:head>
	<title>Doctor - cure</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 class="text-xl font-semibold tracking-tight text-[var(--text-primary)]">Doctor</h1>
		<button
			onclick={runDoctor}
			disabled={loading}
			class="rounded-md border border-[var(--border)] bg-[var(--bg-tertiary)]/50 px-3 py-1.5 text-sm text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-tertiary)] disabled:cursor-not-allowed disabled:opacity-50"
		>
			Re-run
		</button>
	</div>

	{#if loading}
		<div class="flex items-center justify-center py-12">
			<LoadingSpinner />
		</div>
	{:else if error}
		<ErrorBanner message={error} onDismiss={() => (error = null)} />
	{:else if results.length === 0}
		<p class="py-8 text-center text-sm text-[var(--text-secondary)]">No checks returned</p>
	{:else}
		<div class="space-y-2">
			{#each results as check (check.name)}
				<div class="flex items-center justify-between rounded-lg border border-[var(--border)] bg-[var(--bg-secondary)] px-4 py-3">
					<div class="min-w-0 flex-1">
						<p class="text-sm font-medium text-[var(--text-primary)]">{check.name}</p>
						<p class="mt-0.5 text-xs text-[var(--text-secondary)]">{check.message}</p>
					</div>
					<span
						class="ml-3 shrink-0 rounded-full px-2 py-0.5 text-xs font-medium {badgeClasses(check.status)}"
						aria-label="{check.status} status"
					>
						{check.status}
					</span>
				</div>
			{/each}
		</div>
	{/if}
</div>
