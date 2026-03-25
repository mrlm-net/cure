import { visit } from 'unist-util-visit';

/**
 * Adds `id` attributes to h1-h6 elements based on their text content,
 * slugified to lowercase-hyphenated form. Duplicate IDs are deduplicated
 * with a -2, -3, etc. suffix.
 */
export default function rehypeSlug() {
  return (tree) => {
    const counts = {};

    visit(tree, 'element', (node) => {
      if (!/^h[1-6]$/.test(node.tagName)) return;

      const text = extractText(node);
      const slug = slugify(text);

      if (!slug) return;

      counts[slug] = (counts[slug] || 0) + 1;
      const id = counts[slug] === 1 ? slug : `${slug}-${counts[slug]}`;

      node.properties = node.properties || {};
      if (!node.properties.id) {
        node.properties.id = id;
      }
    });
  };
}

function extractText(node) {
  let text = '';
  visit(node, 'text', (textNode) => {
    text += textNode.value;
  });
  return text;
}

function slugify(text) {
  return text
    .toLowerCase()
    .trim()
    .replace(/[^\w\s-]/g, '')
    .replace(/[\s_]+/g, '-')
    .replace(/^-+|-+$/g, '');
}
