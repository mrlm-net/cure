<script lang="ts">
	interface Props {
		size?: 'sm' | 'md';
	}

	let { size = 'md' }: Props = $props();

	const dimensions = $derived(size === 'sm' ? 16 : 24);
</script>

<svg
	width={dimensions}
	height={dimensions}
	viewBox="0 0 24 24"
	fill="none"
	role="status"
	aria-label="Loading"
	class="animate-spin"
>
	<circle cx="12" cy="12" r="10" stroke="rgba(255,255,255,0.1)" stroke-width="3" />
	<path
		d="M12 2a10 10 0 0 1 10 10"
		stroke="#58a6ff"
		stroke-width="3"
		stroke-linecap="round"
	/>
</svg>
