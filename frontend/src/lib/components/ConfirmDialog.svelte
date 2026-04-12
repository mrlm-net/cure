<script lang="ts">
	interface Props {
		title?: string;
		message: string;
		confirmLabel?: string;
		cancelLabel?: string;
		danger?: boolean;
		onConfirm: () => void;
		onCancel: () => void;
	}

	let {
		title = 'Confirm',
		message,
		confirmLabel = 'Confirm',
		cancelLabel = 'Cancel',
		danger = false,
		onConfirm,
		onCancel,
	}: Props = $props();

	let dialog: HTMLDialogElement | undefined = $state();

	$effect(() => {
		if (dialog) {
			dialog.showModal();
		}
	});

	function handleBackdrop(e: MouseEvent) {
		if (e.target === dialog) onCancel();
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') onCancel();
	}
</script>

<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<dialog
	bind:this={dialog}
	onclick={handleBackdrop}
	onkeydown={handleKeydown}
	class="m-auto rounded-xl border border-[var(--border)] bg-[var(--bg-secondary)] p-0 text-[var(--text-primary)] shadow-2xl backdrop:bg-black/60 backdrop:backdrop-blur-sm"
	aria-labelledby="dialog-title"
	aria-describedby="dialog-message"
>
	<div class="w-[min(90vw,420px)] p-6">
		<h2 id="dialog-title" class="mb-2 text-base font-semibold text-[var(--text-primary)]">
			{title}
		</h2>
		<p id="dialog-message" class="text-sm leading-relaxed text-[rgba(230,237,243,0.6)]">
			{message}
		</p>

		<div class="mt-6 flex justify-end gap-3">
			<button
				onclick={onCancel}
				class="cursor-pointer rounded-md bg-[var(--bg-tertiary)]/50 px-4 py-2 text-sm text-[rgba(230,237,243,0.7)] transition-colors hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]"
			>
				{cancelLabel}
			</button>
			<button
				onclick={onConfirm}
				class="cursor-pointer rounded-md px-4 py-2 text-sm font-medium transition-colors
					{danger
					? 'bg-[#f85149]/15 text-[var(--danger)] hover:bg-[#f85149]/25'
					: 'bg-[var(--accent)] text-white hover:bg-[var(--accent-hover)]'}"
			>
				{confirmLabel}
			</button>
		</div>
	</div>
</dialog>

<style>
	dialog::backdrop {
		background: rgba(0, 0, 0, 0.6);
		backdrop-filter: blur(4px);
	}
</style>
