import { Processor } from "./processor.ts";

if (import.meta.main) {
  const args = Deno.args;
  if (args.length !== 1) {
    console.error("Usage: cca <github-issue-url>");
    console.error("Example: cca https://github.com/owner/repo/issues/123");
    Deno.exit(1);
  }

  const issueURL = args[0];
  if (!issueURL.includes("github.com") || !issueURL.includes("/issues/")) {
    console.error(`Error: Invalid GitHub issue URL: ${issueURL}`);
    console.error("URL must contain 'github.com' and '/issues/'");
    Deno.exit(1);
  }

  const processor = new Processor();
  try {
    await processor.processIssue(issueURL);
    console.log("\u2705 Pull request created successfully!");
  } catch (err) {
    console.error(`Error: ${err}`);
    Deno.exit(1);
  }
}
