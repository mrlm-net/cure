<script lang="ts">
	import { apiFetch, ApiError } from '$lib/api';
	import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';
	import ErrorBanner from '$lib/components/ErrorBanner.svelte';

	interface Template {
		name: string;
		description: string;
	}

	let loading = $state(true);
	let error = $state<string | null>(null);
	let notImplemented = $state(false);
	let templates = $state<Template[]>([]);

	async function fetchTemplates() {
		loading = true;
		error = null;
		notImplemented = false;
		try {
			templates = await apiFetch<Template[]>('/api/generate/list');
		} catch (err) {
			if (err instanceof ApiError && err.status === 501) {
				notImplemented = true;
			} else {
				error = err instanceof Error ? err.message : 'Unknown error';
			}
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		fetchTemplates();
	});
</script>

<svelte:head>
	<title>Generate - cure</title>
</svelte:head>

<div class="space-y-6">
	<h1 class="text-xl font-semibold tracking-tight text-[#e6edf3]">Generate</h1>

	{#if loading}
		<div class="flex items-center justify-center py-12">
			<LoadingSpinner />
		</div>
	{:else if error}
		<ErrorBanner message={error} onDismiss={() => (error = null)} />
	{:else if notImplemented}
		<div class="flex flex-col items-center justify-center py-16 text-center">
			<svg
				width="48"
				height="48"
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="1.5"
				stroke-linecap="round"
				stroke-linejoin="round"
				class="mb-4 text-white/30"
				aria-hidden="true"
			>
				<path d="M12 4v16m8-8H4" />
			</svg>
			<p class="text-sm text-white/50">Generate commands are not yet available</p>
			<p class="mt-1 text-xs text-white/30">This feature is under development</p>
		</div>
	{:else if templates.length === 0}
		<p class="py-8 text-center text-sm text-[rgba(230,237,243,0.5)]">No templates available</p>
	{:else}
		<div class="space-y-2">
			{#each templates as template (template.name)}
				<div class="rounded-lg border border-white/10 bg-[#161b22] px-4 py-3">
					<p class="text-sm font-medium text-[#e6edf3]">{template.name}</p>
					<p class="mt-0.5 text-xs text-[rgba(230,237,243,0.5)]">{template.description}</p>
				</div>
			{/each}
		</div>
	{/if}
</div>
