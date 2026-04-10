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
	let platformChecks = $state<CheckResult[]>([]);
	let projectChecks = $state<CheckResult[]>([]);

	async function runDoctor() {
		loading = true;
		error = null;
		try {
			const [platform, project] = await Promise.all([
				apiFetch<CheckResult[]>('/api/doctor/platform').catch(() => []),
				apiFetch<CheckResult[]>('/api/doctor')
			]);
			platformChecks = platform;
			projectChecks = project;
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
			case 'pass': return 'bg-[var(--success)]/15 text-[var(--success)]';
			case 'warn': return 'bg-[var(--warning)]/15 text-[var(--warning)]';
			case 'fail': return 'bg-[var(--danger)]/15 text-[var(--danger)]';
			default: return 'bg-[var(--bg-tertiary)] text-[var(--text-secondary)]';
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
		<div class="flex items-center justify-center py-12"><LoadingSpinner /></div>
	{:else if error}
		<ErrorBanner message={error} onDismiss={() => (error = null)} />
	{:else}
		<!-- Platform / Control Plane -->
		{#if platformChecks.length > 0}
			<div>
				<h2 class="mb-2 text-xs font-semibold uppercase tracking-wider text-[var(--text-tertiary)]">Platform</h2>
				<div class="space-y-2">
					{#each platformChecks as check (check.name)}
						<div class="flex items-center justify-between rounded-lg border border-[var(--border)] bg-[var(--bg-secondary)] px-4 py-3">
							<div>
								<p class="text-sm font-medium text-[var(--text-primary)]">{check.name}</p>
								<p class="mt-0.5 text-xs text-[var(--text-secondary)]">{check.message}</p>
							</div>
							<span class="ml-3 shrink-0 rounded-full px-2 py-0.5 text-xs font-medium {badgeClasses(check.status)}">{check.status}</span>
						</div>
					{/each}
				</div>
			</div>
		{/if}

		<!-- Project checks -->
		<div>
			<h2 class="mb-2 text-xs font-semibold uppercase tracking-wider text-[var(--text-tertiary)]">Project</h2>
			{#if projectChecks.length === 0}
				<p class="py-4 text-center text-sm text-[var(--text-secondary)]">No project checks returned</p>
			{:else}
				<div class="space-y-2">
					{#each projectChecks as check (check.name)}
						<div class="flex items-center justify-between rounded-lg border border-[var(--border)] bg-[var(--bg-secondary)] px-4 py-3">
							<div>
								<p class="text-sm font-medium text-[var(--text-primary)]">{check.name}</p>
								<p class="mt-0.5 text-xs text-[var(--text-secondary)]">{check.message}</p>
							</div>
							<span class="ml-3 shrink-0 rounded-full px-2 py-0.5 text-xs font-medium {badgeClasses(check.status)}">{check.status}</span>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	{/if}
</div>
