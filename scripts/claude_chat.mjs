import { readFile } from 'fs/promises';
import { ClaudeCode } from 'claude-code-js';

async function main() {
  const file = process.argv[2];
  if (!file) {
    console.error('Usage: claude_chat.mjs <prompt-file>');
    process.exit(1);
  }
  const prompt = await readFile(file, 'utf8');
  const claude = new ClaudeCode();
  const res = await claude.chat({ prompt });
  if (!res.success || !res.message?.result) {
    console.error(res.error?.result || 'claude failed');
    process.exit(1);
  }
  process.stdout.write(res.message.result);
}

main().catch(err => {
  console.error(err);
  process.exit(1);
});
