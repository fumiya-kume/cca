import { readFile, writeFile, mkdir, rm } from 'fs/promises';
import { dirname } from 'path';

async function main() {
  const file = process.argv[2];
  if (!file) {
    console.error('Usage: apply_changes.mjs <changes-json-file>');
    process.exit(1);
  }
  const data = JSON.parse(await readFile(file, 'utf8'));
  for (const path of data.deleted_files || []) {
    await rm(path, { force: true });
    console.log(`Deleted ${path}`);
  }
  if (data.files) {
    for (const [path, content] of Object.entries(data.files)) {
      await mkdir(dirname(path), { recursive: true });
      await writeFile(path, content);
      console.log(`Wrote ${path}`);
    }
  }
}

main().catch(err => {
  console.error(err);
  process.exit(1);
});
