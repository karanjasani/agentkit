# Examples: agent discovery

Without a discovery path, agents default to `grep`. These snippets tell them to
prefer `repomap` for structural questions and to scope changes before running
tests.

## Cursor

Copy the rule into your project so Cursor loads it automatically:

```bash
mkdir -p .cursor/rules
cp examples/.cursor/rules/repomap.mdc .cursor/rules/repomap.mdc
```

`.mdc` rules with `alwaysApply: true` are injected into the agent's context on
every request. Adjust the frontmatter (for example, scope it with `globs:`) to
taste.

## AGENTS.md (Cursor, and other agents that read AGENTS.md)

Append the block from [`AGENTS.md`](AGENTS.md) to your repository's root
`AGENTS.md`:

```bash
cat examples/AGENTS.md >> AGENTS.md   # then delete the HTML comment header
```

## Verifying it works

Ask your agent "where is `FetchWidget` defined and who calls it?" — it should run
`repomap symbol` and `repomap callers` instead of grepping. Ask it to make a
change and run tests — it should run `repomap impact --base <ref>` first and run
the recommended tests.
