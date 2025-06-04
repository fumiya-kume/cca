// deno-lint-ignore-file no-explicit-any
import {
  assert,
  assertEquals,
} from "https://deno.land/std@0.204.0/testing/asserts.ts";
import { generateRandomString, gitOperations } from "../src/git.ts";
import { Issue } from "../src/types.ts";

// utility to stub Deno.Command during tests
let commandResponses: Array<any> = [];
let capturedCommands: Array<{ cmd: string; args: string[] }> = [];

class MockCommand {
  cmd: string;
  args: string[];
  constructor(cmd: string, options: { args?: string[]; stdout?: string; stderr?: string } = {}) {
    this.cmd = cmd;
    this.args = options.args ?? [];
    capturedCommands.push({ cmd, args: this.args });
  }
  output() {
    const res = commandResponses.shift() ??
      { code: 0, stdout: "{}", stderr: "" };
    return Promise.resolve({
      code: res.code ?? 0,
      stdout: new TextEncoder().encode(res.stdout ?? ""),
      stderr: new TextEncoder().encode(res.stderr ?? ""),
    });
  }
  spawn() {
    const res = commandResponses.shift() ?? { code: 0 };
    return { status: Promise.resolve({ code: res.code ?? 0 }) };
  }
}

Deno.test("generateRandomString generates correct length", () => {
  const result = generateRandomString(6);
  assertEquals(result.length, 6);
  assert(/^[a-z0-9]+$/.test(result));
});

Deno.test("generateRandomString generates unique strings", () => {
  const results = new Set();
  for (let i = 0; i < 100; i++) {
    results.add(generateRandomString(6));
  }
  // With 36^6 possible combinations, getting duplicates in 100 tries is extremely unlikely
  assert(results.size > 95);
});

Deno.test("gitOperations creates branch with random suffix", async () => {
  (Deno as any).Command = MockCommand;
  capturedCommands = [];
  commandResponses = [
    { code: 0 }, // checkout
    { code: 0 }, // add
    { code: 0 }, // commit
    { code: 0 }, // push
  ];
  const issue: Issue = {
    number: 123,
    title: "Test Issue",
    body: "Test body",
    repository: "test/repo",
    url: "https://github.com/test/repo/issues/123",
  };
  
  await gitOperations(issue);
  
  // Check that the branch name includes the issue number and a random suffix
  const checkoutCommand = capturedCommands[0];
  assertEquals(checkoutCommand.cmd, "git");
  assertEquals(checkoutCommand.args[0], "checkout");
  assertEquals(checkoutCommand.args[1], "-b");
  const branchName = checkoutCommand.args[2];
  assert(branchName.startsWith("cca/issue-123-"));
  // Check that it has a random suffix of 6 characters
  const suffix = branchName.replace("cca/issue-123-", "");
  assertEquals(suffix.length, 6);
  assert(/^[a-z0-9]+$/.test(suffix));
  
  // Check that push uses the same branch name
  const pushCommand = capturedCommands[3];
  assertEquals(pushCommand.cmd, "git");
  assertEquals(pushCommand.args[0], "push");
  assertEquals(pushCommand.args[1], "origin");
  assertEquals(pushCommand.args[2], branchName);
});

Deno.test("gitOperations generates different branch names for same issue", async () => {
  (Deno as any).Command = MockCommand;
  const issue: Issue = {
    number: 456,
    title: "Test Issue",
    body: "Test body",
    repository: "test/repo",
    url: "https://github.com/test/repo/issues/456",
  };
  
  const branchNames = new Set();
  
  for (let i = 0; i < 5; i++) {
    capturedCommands = [];
    commandResponses = [
      { code: 0 }, // checkout
      { code: 0 }, // add
      { code: 0 }, // commit
      { code: 0 }, // push
    ];
    
    await gitOperations(issue);
    
    const checkoutCommand = capturedCommands[0];
    const branchName = checkoutCommand.args[2];
    branchNames.add(branchName);
  }
  
  // All 5 runs should generate different branch names
  assertEquals(branchNames.size, 5);
});