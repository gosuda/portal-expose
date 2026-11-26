# CI
A single Depot Linux PR/merge-queue job runs the current linter, unit tests,
coverage upload and native build. Duplicate standalone lint and unit workflows
are deleted. The Go version comes from go.mod.
Main pushes run the same checks before publishing the Docker image. PRs no
longer repeat Docker builds. Kind end-to-end tests run weekly and manually,
so cluster integration regressions are not checked on every PR.
