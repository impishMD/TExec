# Releasing

**English** | [Русский](../ru/releasing.md)

`VERSION` contains the version without `v`. Tags, release titles and container tags
use `v<major>.<minor>.<patch>`. The default branch is `main`.

Choose an unused version and tag for every release. Container repositories are
`impishmd/texec` and `ghcr.io/impishmd/texec`.

## Release relay

The code phrase **«Релиз-эстафета»** (release relay) means: prepare and start a release,
then hand the remaining steps to the owner without waiting for GitHub Actions.
First state the selected version and planned actions, then proceed:

1. Integrate ready fixes and features through PRs into `dev`; never push directly to `dev`.
   In this mode the owner explicitly permits merging without waiting for CI when repository
   rules allow a normal merge. Do not bypass branch protection; if it blocks the merge,
   provide continuation commands starting at that stage.
2. Create `release/<version>` from current `dev`; update application/chart versions,
   changelog, EN/RU application and chart release notes, and installation examples.
   Check version and documentation consistency. Do not repeat completed code or local
   container checks unless further changes warrant it.
3. Push the release branch and application tag `v<version>` pointing to the same commit.
   Release waits for that commit's branch CI itself; failed CI blocks publication.
   Open a PR from the release branch back into `dev` for release preparation changes.
4. Stop after pushing, without waiting for CI or rerunning jobs. Provide PR/Actions links
   and ready-to-run commands for the remaining steps.
5. The owner checks CI, Packages, Helm CI and Release. After the image is published,
   push `chart-v<chart-version>` at the same commit, check Helm Release/Pages, merge
   the release branch into `main` and its PR back into `dev`.
   Use `[skip ci]` for the `main` merge only if it introduces no code changes or conflict resolutions.

Do not push the chart tag together with the application tag: Helm Release requires the
published image. Both tags are immutable. Workflows use the committed release notes.

## Prepare

Use `dev` for day-to-day integration:

- Start `fix/<task>` and `feature/<task>` from current `dev`, validate changes locally and open a pull request targeting `dev`. Merge through the PR after its checks pass; do not push changes directly into `dev`.
- PRs into `dev` run **Dev PR checks**: Go vet, frontend lint, documentation and version checks, translations and GitHub Actions validation. They do not run test suites, build containers/packages or install Helm charts. New commits cancel obsolete checks for the same PR.
- Merging into `dev` does not run CI again or publish images/releases. Full checks remain enabled for `main`, `release/*`, PRs targeting branches other than `dev`, and manual workflow dispatch. Run tests relevant to each change locally before opening its PR.
- `main` contains released code. Do not bump versions or create tags for each fix; prepare a version for a batch of changes.
- Create `release/<version>` from `dev` when the batch is ready. Only release fixes enter that branch; new features continue in `dev`.
- For an urgent production fix, start `hotfix/<task>` from `main`, publish it through a separate `release/*` branch, then merge it back into both `main` and `dev`.

1. Create `release/<version>` from `dev`. Update `VERSION`, frontend package versions and deployment examples.
2. Update `CHANGELOG.md` and add `release-notes/en/v<version>.md` and `release-notes/ru/v<version>.md` for GitHub Releases. Keep user guides focused on current behavior.
3. Run `make test`, `make check`, `make build`, runtime smoke tests, and `make release-check`.
4. Commit/push the release branch and wait for CI, Packages and Helm CI to pass. The tag must point to the same verified commit; merging into `main` is not required before publication.
5. Tag that verified commit and push the tag:

   ```sh
   release_tag="v$(cat VERSION)"
   git tag -a "$release_tag" -m "$release_tag"
   git push origin "$release_tag"
   ```

Never move an existing release tag. Local `make release-snapshot` builds without publishing.
GoReleaser 2.18.2 defines four Linux/macOS archives (amd64/arm64), four Linux DEB/RPM packages,
a source archive and `checksums.txt`. Packages are not GPG signed. Binary archives include
both documentation languages and license notices.

## Merge back into main and dev

After the release branch passes its checks, merge it into `main`. After publication,
merge the release branch back into `dev`, including version changes and release fixes.
Keep the application and chart tags on the verified
release-branch commit. When the merge introduces no code changes, skip duplicate CI:

```sh
git fetch origin
git switch main
git pull --ff-only origin main
git merge --no-ff release/1.2.0 -m "Merge release/1.2.0 into main [skip ci]"
git push origin main
gh pr create --base dev --head release/1.2.0 \
  --title "Merge release/1.2.0 back into dev" \
  --body "Bring the released version and release fixes back into development."
gh pr checks release/1.2.0 --watch
gh pr merge release/1.2.0 --merge
```

Replace `1.2.0` with the release version. If the merge requires conflict resolutions
or other code changes, run CI for the resulting commit instead of adding `[skip ci]`.
The PR back into `dev` runs the same quick checks. Validate combined behavior locally
or dispatch full CI manually if conflict resolutions need additional tests.

## Credentials

The GitHub environment **docker_hub** contains **DOCKERHUB_USR** and **DOCKERHUB_TOKEN**.
GHCR uses the workflow `GITHUB_TOKEN` with `packages: write`. Build/test jobs use read access.
Environment protection rules remain effective. Do not store token values in this repository.

## Publication

The Release workflow requires successful CI from a push to `main` or `release/*` for the
exact tagged commit. It waits for an ongoing run; missing or failed CI blocks publication.
Tests (including `go test -race`) and temporary CI container builds run in branch CI only.
The check uses the [workflow runs API](https://docs.github.com/en/rest/actions/workflow-runs#list-workflow-runs-for-a-workflow).

Release validates the version and builds/tests release packages before creating a draft.
Branch packages are `-dev` snapshots; CI containers have the embedded version `dev` and
are not release artifacts. Native amd64/arm64 jobs build versioned server, runner, job and
helper images once, push them by digest to Docker Hub and GHCR, and smoke-test those images.
The final job reuses the verified digests without rebuilding: it creates and checks
multi-platform version tags, uploads `container-digests.txt`, updates stable aliases and
publishes the GitHub Release. Prereleases do not update `latest` aliases. Releases are serialized.

Images: `impishmd/texec` and `ghcr.io/impishmd/texec`. GitHub assets:
[Releases](https://github.com/impishMD/TExec/releases).
The pipeline follows [Docker multi-platform guidance](https://docs.docker.com/build/ci/github-actions/multi-platform/)
and [GitHub Container registry authentication](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry).

If publication fails, rerun failed jobs to resume the draft. Published releases are not
overwritten. Publication to two registries is not atomic; retries finish partial publication.
Do not publish an older stable tag after a newer one: that would move the stable aliases back.
Repository and package visibility are managed separately from the workflow.

## Helm chart releases

Application and chart tags are separate: `v1.0.0` publishes the application and
`chart-v1.0.0` publishes chart `1.0.0`. Chart-only changes increment `Chart.yaml`'s `version`;
`appVersion` follows `VERSION`. Helm CI validates Helm 3/4 renders and Kubernetes schemas,
then installs the built server in a disposable kind cluster and tests authentication,
upgrade/restart persistence, URL subpaths and retained-PVC reuse.

Before the first chart publication, enable **Settings → Pages → Source: GitHub Actions**.
Allow `chart-v*` tags in the `github-pages` environment's deployment rules. The workflow
uses `GITHUB_TOKEN` to create GitHub Releases and maintain `gh-pages`; no chart-specific
personal access token is required. Ensure package visibility permits pulling the default
GHCR image, or configure `imagePullSecrets` for users of a private installation.

1. Publish the matching application image first and wait for Release to finish.
2. Set `charts/texec/Chart.yaml` versions, validate the chart, and write release notes in
   `release-notes/charts/en/v<chart-version>.md` and `release-notes/charts/ru/v<chart-version>.md`.
3. Commit and push; wait for Helm CI. Tag that commit with `chart-v<chart-version>` and push the tag.
4. Helm Release reruns checks, verifies that the application image exists, packages the chart,
   uploads it to a separate GitHub Release and publishes the repository index through Pages.

Repository URL: `https://impishmd.github.io/TExec`. The first release creates `gh-pages`
automatically. Chart releases do not replace the application's GitHub “Latest” release.
Published archives are reused on retry so index digests stay consistent. Resume a failed
chart publication using **Helm Release → Run workflow**, specifying the existing chart tag;
never move an existing tag or reuse its version for different chart content.

Local checks (Python requires `PyYAML==6.0.3`):

```sh
make version-check helm-check
python3 tools/check-chart.py --release-tag chart-v1.0.0
helm package charts/texec --destination /tmp/texec-charts
```

To exercise installation locally with Docker and kind, use a dedicated kubeconfig:

```sh
export KUBECONFIG=$(mktemp)
kind create cluster --name texec-chart --kubeconfig "$KUBECONFIG"
docker build --target server --build-arg VERSION="v$(cat VERSION)" -t texec:helm-ci .
kind load docker-image texec:helm-ci --name texec-chart
bash tools/test-chart-install.sh
kind delete cluster --name texec-chart
rm "$KUBECONFIG"
unset KUBECONFIG
```

The install test refuses any current context other than `kind-texec-chart`.
