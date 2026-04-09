<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { apiFetch, getBaseUrl } from '$lib/api';
	import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';
	import ErrorBanner from '$lib/components/ErrorBanner.svelte';
	import { formatRelativeTime } from '$lib/utils';

	interface Session {
		id: string;
		name?: string;
		provider: string;
		project_name?: string;
		branch_name?: string;
		updated_at: string;
		turns: number;
	}

	const projectName = $derived($page.params.name ?? '');

	let projectJson = $state('');
	let originalJson = $state('');
	let sessions = $state<Session[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let saving = $state(false);
	let tab = $state<'sessions' | 'config'>('sessions');
	let dirty = $derived(projectJson !== originalJson);

	async function fetchData(): Promise<void> {
		try {
			const [proj, allSessions] = await Promise.all([
				apiFetch<any>(`/api/project/${projectName}`),
				apiFetch<Session[]>('/api/context/sessions').catch(() => [])
			]);
			const json = JSON.stringify(proj, null, 2);
			projectJson = json;
			originalJson = json;
			sessions = allSessions.filter(s => s.project_name === projectName);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load project';
		} finally {
			loading = false;
		}
	}

	async function saveProject(): Promise<void> {
		saving = true;
		error = null;
		try {
			const parsed = JSON.parse(projectJson);
			const base = getBaseUrl();
			const res = await fetch(`${base}/api/project/${projectName}`, {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: projectJson
			});
			if (!res.ok) throw new Error(`Save failed: ${res.status}`);
			originalJson = projectJson;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to save';
		} finally {
			saving = false;
		}
	}

	async function createSession(): Promise<void> {
		try {
			const data = await apiFetch<{ id: string }>('/api/context/sessions', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ project_name: projectName })
			});
			if (data?.id) goto(`/context/${data.id}`);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to create session';
		}
	}

	$effect(() => {
		fetchData();
	});
</script>

<svelte:head>
	<title>{projectName} - Projects - cure</title>
</svelte:head>

{#if loading}
	<div class="flex items-center justify-center py-20"><LoadingSpinner /></div>
{:else}
	<div class="space-y-4">
		<!-- Header -->
		<div class="flex items-center gap-3">
			<a href="/project" class="rounded-md p-1 text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]">
				<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M15 18l-6-6 6-6"/></svg>
			</a>
			<h1 class="text-xl font-semibold text-[var(--text-primary)]">{projectName}</h1>
		</div>

		{#if error}
			<ErrorBanner message={error} onDismiss={() => (error = null)} />
		{/if}

		<!-- Tabs -->
		<div class="flex gap-1 border-b border-[var(--border)]">
			<button
				onclick={() => (tab = 'sessions')}
				class="px-4 py-2 text-sm font-medium border-b-2 transition-colors
					{tab === 'sessions' ? 'border-[var(--accent)] text-[var(--accent)]' : 'border-transparent text-[var(--text-secondary)] hover:text-[var(--text-primary)]'}"
			>
				Sessions ({sessions.length})
			</button>
			<button
				onclick={() => (tab = 'config')}
				class="px-4 py-2 text-sm font-medium border-b-2 transition-colors
					{tab === 'config' ? 'border-[var(--accent)] text-[var(--accent)]' : 'border-transparent text-[var(--text-secondary)] hover:text-[var(--text-primary)]'}"
			>
				Configuration {dirty ? '*' : ''}
			</button>
		</div>

		<!-- Sessions tab -->
		{#if tab === 'sessions'}
			<div class="space-y-3">
				<div class="flex justify-end">
					<button
						onclick={createSession}
						class="rounded-md bg-[var(--accent)] px-4 py-2 text-sm font-medium text-white hover:opacity-90"
					>
						New Session
					</button>
				</div>

				{#if sessions.length === 0}
					<div class="py-12 text-center">
						<p class="text-sm text-[var(--text-secondary)]">No sessions for this project</p>
					</div>
				{:else}
					{#each sessions as s (s.id)}
						<a
							href="/context/{s.id}"
							class="flex items-center justify-between rounded-lg border border-[var(--border)] bg-[var(--bg-secondary)] px-4 py-3 hover:border-[var(--accent)]/30"
						>
							<div>
								<div class="flex items-center gap-2">
									<span class="font-mono text-sm text-[var(--accent)]">{s.name || s.id.slice(0, 8)}</span>
									<span class="rounded bg-[var(--bg-tertiary)] px-2 py-0.5 text-xs text-[var(--text-secondary)]">{s.provider}</span>
								</div>
								<div class="mt-1 flex gap-3 text-xs text-[var(--text-tertiary)]">
									<span>{formatRelativeTime(s.updated_at)}</span>
									<span>{s.turns} turns</span>
									{#if s.branch_name}
										<span class="font-mono">{s.branch_name}</span>
									{/if}
								</div>
							</div>
						</a>
					{/each}
				{/if}
			</div>
		{/if}

		<!-- Config tab (editable JSON) -->
		{#if tab === 'config'}
			<div class="space-y-3">
				<div class="flex items-center justify-between">
					<span class="text-xs text-[var(--text-tertiary)]">~/.cure/projects/{projectName}/project.json</span>
					<div class="flex gap-2">
						{#if dirty}
							<button
								onclick={() => (projectJson = originalJson)}
								class="rounded-md bg-[var(--bg-tertiary)] px-3 py-1.5 text-xs text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
							>
								Discard
							</button>
						{/if}
						<button
							onclick={saveProject}
							disabled={!dirty || saving}
							class="rounded-md bg-[var(--accent)] px-3 py-1.5 text-xs text-white disabled:opacity-50"
						>
							{saving ? 'Saving...' : 'Save'}
						</button>
					</div>
				</div>

				<textarea
					bind:value={projectJson}
					class="w-full rounded-lg border border-[var(--border)] bg-[var(--bg-primary)] p-4 font-mono text-sm text-[var(--text-primary)] focus:border-[var(--accent)]/50 focus:outline-none"
					rows={25}
					spellcheck="false"
				></textarea>
			</div>
		{/if}
	</div>
{/if}
