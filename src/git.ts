import { Issue } from "./types.ts";

export async function gitOperations(issue: Issue): Promise<void> {
  const branchName = `cca/issue-${issue.number}`;

  console.log(`git checkout -b ${branchName}`);
  let status = await new Deno.Command("git", {
    args: ["checkout", "-b", branchName],
  }).spawn().status;
  if (status.code !== 0) throw new Error("failed to create branch");

  console.log("git add .");
  status = await new Deno.Command("git", { args: ["add", "."] }).spawn().status;
  if (status.code !== 0) throw new Error("failed to add files");

  const commitMsg = `Implement: ${issue.title}`;
  console.log(`git commit -m \"${commitMsg}\"`);
  status = await new Deno.Command("git", { args: ["commit", "-m", commitMsg] })
    .spawn().status;
  if (status.code !== 0) throw new Error("failed to commit");

  console.log(`git push origin ${branchName}`);
  status = await new Deno.Command("git", {
    args: ["push", "origin", branchName],
  }).spawn().status;
  if (status.code !== 0) throw new Error("failed to push");
}

export async function createPR(issue: Issue): Promise<string> {
  const title = `Fix: ${issue.title}`;
  const body = `Resolves: ${issue.url}`;

  console.log('gh pr create --draft --title "' + title + '"');

  const cmd = new Deno.Command("gh", {
    args: ["pr", "create", "--draft", "--title", title, "--body", body],
    stdout: "piped",
    stderr: "piped",
  });

  const { code, stdout, stderr } = await cmd.output();
  const output = new TextDecoder().decode(stdout.length ? stdout : stderr);
  if (code !== 0) {
    throw new Error(`failed to create PR: ${output}`);
  }

  console.log("Pull request created: " + output.trim());

  const lines = output.trim().split("\n");
  const lastLine = lines[lines.length - 1];
  if (lastLine.includes("github.com")) {
    return lastLine;
  }
  return output.trim();
}

export const helpers = { gitOperations, createPR };

export function setGitHelpers(custom: {
  gitOperations(issue: Issue): Promise<void>;
  createPR(issue: Issue): Promise<string>;
}) {
  helpers.gitOperations = custom.gitOperations;
  helpers.createPR = custom.createPR;
}
