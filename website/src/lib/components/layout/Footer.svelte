<script lang="ts">
  import { base } from '$app/paths';
  import { siteConfig } from '$lib/config/site.js';
</script>

<footer class="border-t border-[#30363d] bg-[#0d1117] py-8">
  <div class="mx-auto max-w-screen-xl px-4 md:px-6">
    <div class="flex flex-col items-center justify-between gap-4 sm:flex-row">
      <p class="text-sm text-[#848d97]">
        &copy; 2026 mrlm-net &middot; {siteConfig.license}
      </p>
      <div class="flex items-center gap-4">
        <a
          href={siteConfig.repoUrl}
          target="_blank"
          rel="noopener noreferrer"
          class="text-sm text-[#9198a1] hover:text-[#58a6ff]"
        >
          GitHub
        </a>
        <a href="{base}/changelog" class="text-sm text-[#9198a1] hover:text-[#58a6ff]">
          Changelog
        </a>
      </div>
    </div>
  </div>
</footer>
