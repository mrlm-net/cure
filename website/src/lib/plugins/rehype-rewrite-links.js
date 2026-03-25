import { visit } from 'unist-util-visit';

/**
 * Rewrites links in markdown:
 * - Removes .md extension from relative links for SPA routing
 * - Adds target="_blank" rel="noopener noreferrer" to external links
 */
export default function rehypeRewriteLinks() {
  return (tree) => {
    visit(tree, 'element', (node) => {
      if (node.tagName !== 'a') return;

      const href = node.properties?.href;
      if (!href || typeof href !== 'string') return;

      // External links
      if (href.startsWith('http://') || href.startsWith('https://')) {
        node.properties.target = '_blank';
        node.properties.rel = 'noopener noreferrer';
        return;
      }

      // Relative .md links — strip the extension
      if (href.endsWith('.md')) {
        node.properties.href = href.slice(0, -3);
      }
    });
  };
}
