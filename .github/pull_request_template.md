## What does this change do?

<!-- One or two sentences: what changed and why. -->

## Related issue

Closes #

## Type of change

- [ ] Bug fix
- [ ] New command, check, or rule
- [ ] Change to an existing command
- [ ] Documentation
- [ ] CI / repo automation
- [ ] Other (describe above)

## Checklist

- [ ] `go vet ./...`, `go build ./...` and `go test -race ./...` pass
- [ ] `gofmt -s -l .` is empty and `go mod tidy -diff` prints nothing
- [ ] `golangci-lint run ./...` and `govulncheck ./...` pass
- [ ] Commit messages follow Conventional Commits (`CHANGELOG.md` is generated from them)
- [ ] If `.github/scripts/` changed: `python3 .github/scripts/tests/run_tests.py` and `ruff check .github/scripts` pass

## Security

- [ ] This change does not print, log, or store a secret value found during a scan.

The contribution is licensed under the MIT License of this repository
(see [LICENSE](../LICENSE)), and [docs/CONTRIBUTING.md](../docs/CONTRIBUTING.md) has been read.
