import { dirname, join } from "https://deno.land/std@0.204.0/path/mod.ts";
import { ensureDir } from "https://deno.land/std@0.204.0/fs/mod.ts";
// Lazy import to avoid issues with Node/Esm interop during testing
import type { ClaudeCode } from "npm:claude-code-js";
import { CodeChanges, Issue } from "./types.ts";
import { helpers } from "./git.ts";
import { withProgress } from "./progress.ts";

export class Processor {
  private claude?: ClaudeCode;

  constructor() {}

  private async getClaude(): Promise<ClaudeCode> {
    if (!this.claude) {
      console.log("🤖 Loading Claude Code SDK...");
      const mod = await import("npm:claude-code-js");
      this.claude = new mod.ClaudeCode();
      console.log("✅ Claude Code SDK loaded");
    }
    return this.claude;
  }

  private parseClaudeResponse(text: string): CodeChanges {
    let cleaned = text.trim();
    const fence = cleaned.match(/```(?:json)?\n([\s\S]*?)\n```/);
    if (fence) {
      cleaned = fence[1];
    }
    try {
      return JSON.parse(cleaned);
    } catch {
      const first = cleaned.indexOf("{");
      const last = cleaned.lastIndexOf("}");
      if (first !== -1 && last !== -1) {
        return JSON.parse(cleaned.slice(first, last + 1));
      }
      throw new SyntaxError("Invalid JSON from Claude");
    }
  }

  async processIssue(issueURL: string): Promise<void> {
    console.log("🔍 Fetching issue...");
    const issue = await this.fetchIssue(issueURL);
    console.log(`✅ Issue fetched: "${issue.title}"\n`);

    console.log("🤖 Generating code with Claude...");
    console.log(`Using issue data: ${JSON.stringify(issue)}`);
    let changes = await this.generateCode(issue);
    console.log(
      `✅ Code generated: ${
        Object.keys(changes.files).length
      } files changed\n`,
    );

    const maxRetries = 3;
    for (let attempt = 1; attempt <= maxRetries; attempt++) {
      console.log(`🔌 Applying changes (attempt ${attempt})...`);
      await this.applyChanges(changes);

      console.log("🔧 Running verification (.cca/verify.sh)...");
      const verifyErr = await this.runVerification();
      if (!verifyErr) {
        console.log("✅ Verification passed");
        break;
      }

      if (attempt === maxRetries) {
        throw new Error(
          `verification failed after ${maxRetries} attempts: ${verifyErr}`,
        );
      }

      console.log(`❌ Verification failed: ${verifyErr}\n`);
      console.log(
        `🔄 Verification failed (attempt ${attempt}/${maxRetries}), asking Claude to fix...`,
      );
      console.log("🤖 Claude fixing verification errors...");
      changes = await this.fixWithClaude(changes, verifyErr);
      console.log(
        `✅ Code updated: ${
          Object.keys(changes.files).length
        } files changed\n`,
      );
    }

    console.log(`📝 Creating randomized branch for issue ${issue.number}...`);
    await helpers.gitOperations(issue);
    console.log("✅ Changes committed and pushed");

    console.log("🎯 Creating pull request...");
    const prURL = await helpers.createPR(issue);
    console.log(`✅ Pull request created: ${prURL}\n`);
  }

  private async fetchIssue(issueURL: string): Promise<Issue> {
    const cmd = new Deno.Command("gh", {
      args: ["issue", "view", issueURL, "--json", "number,title,body,url"],
      stdout: "piped",
      stderr: "piped",
    });
    const { code, stdout, stderr } = await cmd.output();
    if (code !== 0) {
      throw new Error(`gh command failed: ${new TextDecoder().decode(stderr)}`);
    }
    const issue: Issue = JSON.parse(new TextDecoder().decode(stdout));
    const parts = issueURL.split("/");
    if (parts.length >= 5) {
      issue.repository = `${parts[3]}/${parts[4]}`;
    }
    return issue;
  }

  private async generateCode(issue: Issue): Promise<CodeChanges> {
    const prompt =
      `Implement a solution for this GitHub issue:\n\nIssue: ${issue.title}\nDescription: ${issue.body}\nRepository: ${issue.repository}\n\nAnalyze the issue and provide a complete implementation including:\n1. All necessary code changes\n2. Tests for the implementation\n3. Any documentation updates needed\n\nReturn the implementation as file paths and their complete content.\n\nFormat as JSON:\n{\n  "files": {\n    "path/to/file.ts": "complete file content..."\n  },\n  "new_files": ["list", "of", "new", "files"],\n  "deleted_files": ["list", "of", "deleted", "files"],\n  "summary": "Brief description of changes made"\n}`;
    console.log("Prompt sent to Claude:\n" + prompt);
    const claude = await this.getClaude();
    const res = await withProgress(
      "Waiting for Claude",
      () => claude.chat({ prompt }),
    );
    if (!res.success || !res.message?.result) {
      throw new Error(res.error?.result ?? "claude failed");
    }
    console.log("Claude response received");
    return this.parseClaudeResponse(res.message.result);
  }

  private async applyChanges(changes: CodeChanges): Promise<void> {
    for (const path of changes.deleted_files) {
      try {
        console.log(`Deleting ${path}`);
        await Deno.remove(path);
      } catch (err) {
        if (!(err instanceof Deno.errors.NotFound)) {
          throw new Error(`failed to delete ${path}: ${err}`);
        }
      }
    }

    for (const [path, content] of Object.entries(changes.files)) {
      const dir = dirname(path);
      await ensureDir(dir);
      console.log(`Writing ${path}`);
      await Deno.writeTextFile(path, content);
    }
  }

  private async runVerification(): Promise<string | undefined> {
    const verifyPath = ".cca/verify.sh";
    try {
      await Deno.stat(verifyPath);
    } catch (err) {
      if (err instanceof Deno.errors.NotFound) {
        await this.createVerificationScript();
        console.log("Created verification stub at " + verifyPath);
      } else {
        throw err;
      }
    }

    const cmd = new Deno.Command("bash", {
      args: [verifyPath],
      stdout: "piped",
      stderr: "piped",
    });
    console.log("Running verification script...");
    const { code, stdout, stderr } = await cmd.output();
    if (code !== 0) {
      const output = stdout.length ? stdout : stderr;
      console.log("Verification errors:\n" + new TextDecoder().decode(output));
      return new TextDecoder().decode(output);
    }
    console.log("Verification script exited successfully");
    return undefined;
  }

  private async createVerificationScript(): Promise<void> {
    const verifyDir = ".cca";
    const verifyPath = join(verifyDir, "verify.sh");
    await ensureDir(verifyDir);
    const content =
      `#!/bin/bash\n# Add your build, test, and lint commands here\n# Examples:\n# deno task build\n# deno test\n\necho "No verification script configured - skipping checks"\nexit 0\n`;
    await Deno.writeTextFile(verifyPath, content);
    await Deno.chmod(verifyPath, 0o700);
    console.log("Wrote verification stub to " + verifyPath);
  }

  private async fixWithClaude(
    currentChanges: CodeChanges,
    verifyErrors: string,
  ): Promise<CodeChanges> {
    const changesJSON = JSON.stringify(currentChanges, null, 2);
    const prompt =
      `The verification script failed with these errors:\n\n${verifyErrors}\n\nHere are the current code changes:\n${changesJSON}\n\nPlease fix the code to resolve these verification errors. Return the corrected implementation.\n\nFormat as JSON with the same structure as before:\n{\n  "files": {...},\n  "new_files": [...],\n  "deleted_files": [...],\n  "summary": "Description of fixes applied"\n}`;
    const claude = await this.getClaude();
    console.log("Sending fix prompt to Claude...");
    const res = await withProgress(
      "Waiting for Claude fix",
      () => claude.chat({ prompt }),
    );
    if (!res.success || !res.message?.result) {
      throw new Error(res.error?.result ?? "claude failed");
    }
    console.log("Claude returned fixed changes");
    return this.parseClaudeResponse(res.message.result);
  }
}