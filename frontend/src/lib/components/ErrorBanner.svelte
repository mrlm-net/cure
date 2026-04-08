<script lang="ts">
	interface Props {
		message: string;
		onDismiss?: () => void;
	}

	let { message, onDismiss }: Props = $props();
</script>

<div
	role="alert"
	class="flex items-center justify-between gap-3 rounded-lg border border-red-500/40 bg-red-500/10 px-4 py-3 text-sm text-red-400"
>
	<span>{message}</span>
	{#if onDismiss}
		<button
			onclick={onDismiss}
			class="shrink-0 rounded p-1 text-red-400 hover:bg-red-500/20"
			aria-label="Dismiss error"
		>
			<svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
				<path
					d="M4 4l8 8M12 4l-8 8"
					stroke="currentColor"
					stroke-width="1.5"
					stroke-linecap="round"
				/>
			</svg>
		</button>
	{/if}
</div>
