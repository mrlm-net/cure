import adapter from '@sveltejs/adapter-static'
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte'

export default {
  preprocess: vitePreprocess(),
  kit: {
    adapter: adapter({
      pages: '../internal/gui/dist',
      assets: '../internal/gui/dist',
      fallback: 'index.html',
      precompress: false,
    })
  }
}
