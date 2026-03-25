import { getAllDocMeta } from '$lib/content/pipeline.server.js';
import { buildNavSections } from '$lib/config/navigation.js';

export const prerender = true;

export function load() {
  const docs = getAllDocMeta();
  const sections = buildNavSections(docs);
  return { sections };
}
