import { codeToHtml } from 'shiki';

export const prerender = true;

const contextCode = `# Start a new session
$ cure context new \\
    --provider claude \\
    --message "Summarise Go 1.25 release notes"

Go 1.25 introduces several improvements...
session saved: a3f2c1d0e4b5...

# Resume and continue
$ cure context resume a3f2c1d0e4b5 \\
    --message "Which change is most impactful?"

# List all sessions
$ cure context list

# Fork and explore a branch
$ cure context fork a3f2c1d0e4b5`;

export async function load(): Promise<{ codeHtml: string }> {
  const codeHtml = await codeToHtml(contextCode, { lang: 'bash', theme: 'github-dark' });
  return { codeHtml };
}
