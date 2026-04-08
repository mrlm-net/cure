<script lang="ts">
	interface Props {
		role: 'user' | 'assistant';
		content: string;
		streaming?: boolean;
	}

	let { role, content, streaming = false }: Props = $props();

	const isUser = $derived(role === 'user');
</script>

<div class="flex {isUser ? 'justify-end' : 'justify-start'}">
	<div
		class="rounded-lg px-4 py-3 text-sm leading-relaxed max-w-[85%] md:max-w-[70%]
			{isUser ? 'bg-[#58a6ff]/10 text-[#e6edf3]' : 'bg-[#161b22] text-[#e6edf3]'}"
	>
		<p class="whitespace-pre-wrap break-words">{content}{#if streaming}<span
					class="ml-1 inline-block w-1.5 h-4 bg-[#58a6ff] animate-pulse align-middle"
					aria-label="Streaming"
				></span>{/if}</p>
	</div>
</div>
