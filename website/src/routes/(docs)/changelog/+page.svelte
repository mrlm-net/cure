<script lang="ts">
  import { siteConfig } from '$lib/config/site.js';
  import { onMount } from 'svelte';
  import Prism from 'prismjs';
  import 'prismjs/components/prism-bash.js';
  import type { DocPage } from '$lib/types/index.js';

  interface Props {
    data: { doc: DocPage; html: string };
  }

  let { data }: Props = $props();

  onMount(() => {
    Prism.highlightAll();
  });
</script>

<svelte:head>
  <title>Changelog &mdash; {siteConfig.title}</title>
  <meta name="description" content="Version history and release notes for Cure CLI." />
</svelte:head>

<article class="prose max-w-none">
  <h1>Changelog</h1>
  <p class="lead">Version history and release notes for Cure CLI.</p>

  <!-- eslint-disable-next-line svelte/no-at-html-tags -->
  {@html data.html}
</article>
