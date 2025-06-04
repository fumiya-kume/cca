// deno-lint-ignore-file no-explicit-any require-await
import {
  assert,
  assertEquals,
} from "https://deno.land/std@0.204.0/testing/asserts.ts";
import { Processor } from "../src/processor.ts";
import { createPR, gitOperations, setGitHelpers } from "../src/git.ts";
import { CodeChanges, Issue } from "../src/types.ts";

// utility to stub Deno.Command during tests
let commandResponses: Array<any> = [];
class MockCommand {
  cmd: string;
  args: string[];
  constructor(cmd: string, options: { args?: string[] } = {}) {
    this.cmd = cmd;
    this.args = options.args ?? [];
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

Deno.test("fetchIssue parses gh output", async () => {
  (Deno as any).Command = MockCommand;
  const issue: Issue = {
    number: 1,
    title: "t",
    body: "b",
    repository: "r",
    url: "u",
  };
  commandResponses = [{
    code: 0,
    stdout: JSON.stringify(issue),
    stderr: "",
  }];
  const p = new Processor();
  const got = await (p as any).fetchIssue("https://github.com/o/r/issues/1");
  assertEquals(got.title, "t");
});

Deno.test("applyChanges writes files and deletes", async () => {
  const tmp = await Deno.makeTempDir();
  try {
    const p = new Processor();
    const changes: CodeChanges = {
      files: {
        [tmp + "/a.txt"]: "hello",
      },
      new_files: [""],
      deleted_files: [],
      summary: "s",
    };
    await (p as any).applyChanges(changes);
    const content = await Deno.readTextFile(tmp + "/a.txt");
    assertEquals(content, "hello");
  } finally {
    await Deno.remove(tmp, { recursive: true });
  }
});

Deno.test("applyChanges deletes existing file", async () => {
  const tmp = await Deno.makeTempDir();
  const file = `${tmp}/old.txt`;
  await Deno.writeTextFile(file, "x");
  const p = new Processor();
  const changes: CodeChanges = {
    files: {},
    new_files: [],
    deleted_files: [file],
    summary: "s",
  };
  await (p as any).applyChanges(changes);
  const exists = await Deno.stat(file).then(() => true).catch(() => false);
  assert(!exists);
  await Deno.remove(tmp, { recursive: true });
});

Deno.test("runVerification creates stub when missing", async () => {
  (Deno as any).Command = MockCommand;
  commandResponses = [{ code: 0, stdout: "", stderr: "" }];
  const tmp = await Deno.makeTempDir();
  try {
    const prev = Deno.cwd();
    Deno.chdir(tmp);
    const p = new Processor();
    const res = await (p as any).runVerification();
    assertEquals(res, undefined);
    const verifyContent = await Deno.readTextFile(".cca/verify.sh");
    assert(verifyContent.includes("No verification script"));
    Deno.chdir(prev);
  } finally {
    await Deno.remove(tmp, { recursive: true });
  }
});

Deno.test("gitOperations runs commands", async () => {
  (Deno as any).Command = MockCommand;
  commandResponses = [
    { code: 0 }, // checkout
    { code: 0 }, // add
    { code: 0 }, // commit
    { code: 0 }, // push
  ];
  const issue: Issue = {
    number: 1,
    title: "t",
    body: "b",
    repository: "r",
    url: "u",
  };
  await gitOperations(issue);
  assertEquals(commandResponses.length, 0);
});

for (
  const [idx, msg] of [
    "create branch",
    "add files",
    "commit",
    "push",
  ].entries()
) {
  Deno.test(`gitOperations fails to ${msg}`, async () => {
    (Deno as any).Command = MockCommand;
    commandResponses = Array(idx).fill({ code: 0 }).concat([{ code: 1 }]);
    const issue: Issue = {
      number: 1,
      title: "t",
      body: "b",
      repository: "r",
      url: "u",
    };
    let threw = false;
    try {
      await gitOperations(issue);
    } catch (e) {
      if (e instanceof Error) threw = true;
    }
    assert(threw);
  });
}

Deno.test("createPR returns last line", async () => {
  (Deno as any).Command = MockCommand;
  commandResponses = [{
    code: 0,
    stdout: "line1\nhttps://github.com/pr",
    stderr: "",
  }];
  const issue: Issue = {
    number: 1,
    title: "t",
    body: "b",
    repository: "r",
    url: "u",
  };
  const url = await createPR(issue);
  assertEquals(url, "https://github.com/pr");
});

Deno.test("createPR returns trimmed output when no url", async () => {
  (Deno as any).Command = MockCommand;
  commandResponses = [{ code: 0, stdout: "done", stderr: "" }];
  const issue: Issue = {
    number: 1,
    title: "t",
    body: "b",
    repository: "r",
    url: "u",
  };
  const url = await createPR(issue);
  assertEquals(url, "done");
});

Deno.test("createPR throws on failure", async () => {
  (Deno as any).Command = MockCommand;
  commandResponses = [{ code: 1, stdout: "", stderr: "bad" }];
  const issue: Issue = {
    number: 1,
    title: "t",
    body: "b",
    repository: "r",
    url: "u",
  };
  let threw = false;
  try {
    await createPR(issue);
  } catch (e) {
    if (e instanceof Error) {
      threw = e.message.includes("failed to create PR");
    }
  }
  assert(threw);
});

Deno.test("runVerification existing script fails", async () => {
  (Deno as any).Command = MockCommand;
  const tmp = await Deno.makeTempDir();
  const script = `${tmp}/verify.sh`;
  await Deno.writeTextFile(script, "exit 1");
  await Deno.chmod(script, 0o700);
  commandResponses = [{ code: 1, stdout: "nope", stderr: "" }];
  const p = new Processor();
  const prev = Deno.cwd();
  try {
    Deno.chdir(tmp);
    const err = await (p as any).runVerification();
    assertEquals(err, "nope");
  } finally {
    Deno.chdir(prev);
    await Deno.remove(tmp, { recursive: true });
  }
});

Deno.test("runVerification propagates other stat errors", async () => {
  const originalStat = Deno.stat;
  (Deno as any).stat = () => {
    throw new Error("bad");
  };
  const p = new Processor();
  let threw = false;
  try {
    await (p as any).runVerification();
  } catch (e) {
    if (e instanceof Error) threw = e.message === "bad";
  }
  assert(threw);
  (Deno as any).stat = originalStat;
});

Deno.test("generateCode throws on failure", async () => {
  const p = new Processor();
  (p as any).claude = {
    chat: () => Promise.resolve({ success: false, error: { result: "bad" } }),
  };
  let threw = false;
  try {
    await (p as any).generateCode({
      number: 1,
      title: "t",
      body: "b",
      repository: "r",
      url: "u",
    });
  } catch (e) {
    if (e instanceof Error) threw = e.message.includes("bad");
  }
  assert(threw);
});

Deno.test("applyChanges throws on delete failure", async () => {
  const p = new Processor();
  const origRemove = Deno.remove;
  (Deno as any).remove = () => {
    throw new Error("oops");
  };
  const ch: CodeChanges = {
    files: {},
    new_files: [],
    deleted_files: ["x"],
    summary: "",
  };
  let threw = false;
  try {
    await (p as any).applyChanges(ch);
  } catch (e) {
    if (e instanceof Error) threw = e.message.includes("failed to delete");
  }
  (Deno as any).remove = origRemove;
  assert(threw);
});

Deno.test("runVerification uses stderr output", async () => {
  (Deno as any).Command = MockCommand;
  const tmp = await Deno.makeTempDir();
  const script = `${tmp}/verify.sh`;
  await Deno.writeTextFile(script, "exit 1");
  await Deno.chmod(script, 0o700);
  commandResponses = [{ code: 1, stdout: "", stderr: "bad" }];
  const p = new Processor();
  const prev = Deno.cwd();
  try {
    Deno.chdir(tmp);
    const err = await (p as any).runVerification();
    assertEquals(err, "bad");
  } finally {
    Deno.chdir(prev);
    await Deno.remove(tmp, { recursive: true });
  }
});

Deno.test("fetchIssue throws on gh error", async () => {
  (Deno as any).Command = MockCommand;
  commandResponses = [{ code: 1, stdout: "", stderr: "fail" }];
  const p = new Processor();
  let threw = false;
  try {
    await (p as any).fetchIssue("https://github.com/o/r/issues/1");
  } catch (e) {
    if (e instanceof Error) {
      threw = e.message.includes("gh command failed");
    }
  }
  assert(threw);
});

Deno.test("fixWithClaude throws on failure", async () => {
  const p = new Processor();
  (p as any).claude = {
    chat: () => Promise.resolve({ success: false, error: { result: "oops" } }),
  };
  let threw = false;
  try {
    await (p as any).fixWithClaude({
      files: {},
      new_files: [],
      deleted_files: [],
      summary: "",
    }, "err");
  } catch (e) {
    if (e instanceof Error) {
      threw = e.message.includes("oops");
    }
  }
  assert(threw);
});

Deno.test("processIssue end-to-end with verification retry", async () => {
  (Deno as any).Command = MockCommand;
  const issueOut = { number: 1, title: "t", body: "b", url: "u" };
  commandResponses = [{
    code: 0,
    stdout: JSON.stringify(issueOut),
    stderr: "",
  }];

  const p = new Processor();

  const first: CodeChanges = {
    files: { "f.txt": "1" },
    new_files: ["f.txt"],
    deleted_files: [],
    summary: "s",
  };
  const second: CodeChanges = {
    files: { "f.txt": "2" },
    new_files: ["f.txt"],
    deleted_files: [],
    summary: "s2",
  };

  let chat = 0;
  (p as any).claude = {
    chat: (_: any) => {
      chat++;
      const result = chat === 1 ? first : second;
      return Promise.resolve({
        success: true,
        message: { result: JSON.stringify(result) },
      });
    },
  };

  (p as any).applyChanges = async (_: CodeChanges) => {};

  let verify = 0;
  (p as any).runVerification = async () => {
    verify++;
    return verify === 1 ? "err" : undefined;
  };

  setGitHelpers({
    gitOperations: async (_: Issue) => {},
    createPR: async (_: Issue) => "pr",
  });

  await p.processIssue("https://github.com/o/r/issues/1");
  assertEquals(chat, 2);
  setGitHelpers({ gitOperations, createPR });
});

Deno.test("processIssue throws after max retries", async () => {
  (Deno as any).Command = MockCommand;
  const issueOut = { number: 2, title: "t2", body: "b", url: "u" };
  commandResponses = [{
    code: 0,
    stdout: JSON.stringify(issueOut),
    stderr: "",
  }];

  const p = new Processor();
  const changes: CodeChanges = {
    files: { "f.txt": "1" },
    new_files: ["f.txt"],
    deleted_files: [],
    summary: "s",
  };
  (p as any).claude = {
    chat: () =>
      Promise.resolve({
        success: true,
        message: { result: JSON.stringify(changes) },
      }),
  };
  (p as any).applyChanges = async () => {};
  (p as any).runVerification = async () => "err";
  setGitHelpers({ gitOperations: async () => {}, createPR: async () => "pr" });

  let threw = false;
  try {
    await p.processIssue("https://github.com/o/r/issues/2");
  } catch {
    threw = true;
  }
  setGitHelpers({ gitOperations, createPR });
  assert(threw);
});

Deno.test("parseClaudeResponse handles fenced block", () => {
  const p = new Processor();
  const ch: CodeChanges = {
    files: { "a.txt": "hi" },
    new_files: [],
    deleted_files: [],
    summary: "s",
  };
  const text = "```json\n" + JSON.stringify(ch) + "\n```";
  const res = (p as any).parseClaudeResponse(text);
  assertEquals(res.files["a.txt"], "hi");
});

Deno.test("parseClaudeResponse extracts JSON substring", () => {
  const p = new Processor();
  const ch: CodeChanges = {
    files: {},
    new_files: [],
    deleted_files: [],
    summary: "done",
  };
  const text = "random text " + JSON.stringify(ch) + " more";
  const res = (p as any).parseClaudeResponse(text);
  assertEquals(res.summary, "done");
});