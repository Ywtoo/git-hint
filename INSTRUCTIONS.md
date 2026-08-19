# Setup — Linting & Git Hook

The project already has linting configured (`.golangci.yml` and a
`githooks/pre-commit` script are in the repo). You just need to install two
tools and point git at the hooks folder — once per clone. No extra
framework needed, and it won't interfere with any personal git hooks you
already have set up on your machine.

## 1. Install the required tools

```bash
# golangci-lint
brew install golangci-lint
# or, if you don't use Homebrew:
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# goimports
go install golang.org/x/tools/cmd/goimports@latest
```

Check everything installed correctly:

```bash
golangci-lint --version
goimports -h
```

## 2. Point git at the repo's hook (once per clone)

From inside the project folder:

```bash
git config core.hooksPath githooks
```

This only affects this repository. If you already have a personal global
git hook (for anything else, on any other project), it's untouched — this
command just tells git to use this repo's hook *for this repo only*.

## What happens now

Every time you run `git commit` in this repo, it will automatically:

1. Format your Go files with `goimports`.
2. Run `golangci-lint` to catch dead code, duplicated code, unchecked
   errors, and style issues — auto-fixing what it safely can.

If something can't be auto-fixed, the commit is blocked and you'll see
what needs fixing in the terminal. Fix it and commit again.

## Useful commands

| What you want to do | Command |
|---|---|
| Run the linter manually, anytime | `golangci-lint run` |
| Run the linter and auto-fix | `golangci-lint run --fix` |
| Format imports manually | `goimports -w .` |
| Skip the hook for one commit (avoid unless necessary) | `git commit --no-verify` |
| Check which hooks path is active | `git config core.hooksPath` |

## Editor tip

If you're using VS Code with the Go extension, it will run `goimports`
automatically on save once it's on your `PATH` — no extra config needed.