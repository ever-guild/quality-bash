# Contributing

Read [user behavior](readme.md), then [development commands](develop.md). Keep changes focused on an issue or an agreed requirement.

## Human and agent workflow

1. Inspect `git status`, the affected code and existing tests. Preserve unrelated edits.
2. Implement the smallest complete fix. Add a regression test for changed behavior; keep documentation and examples executable and current.
3. Run the relevant tests, then `make check`. For tool/measurement changes also run `make integration` with the real tools and inspect skipped tests.
4. Update `changelog.md` under Unreleased. Describe behavior, migration needs and known limitations; do not list unimplemented plans as features.
5. Inspect the diff. Create signed Conventional Commits (`fix(cli): ...`, `feat(baseline): ...`). Accumulate commits before pushing to avoid redundant Actions runs.
6. In the PR/handoff, state what changed, checks run and any remaining limitation. Follow the release gates in `develop.md` before publication.

Agents use these same commands and rules. Do not introduce a second agent-only workflow, generated planning files or new automation unless the task needs them. Keep local `spec.md` and scratch files out of Git. Never claim a skipped test or pending audit passed.
