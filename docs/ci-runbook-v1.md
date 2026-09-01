# GitHub Actions runbook v1

GitHub Actions is the only validation authority for this repository. The workflow uses Go 1.27.0 on `ubuntu-latest`, writes all generated evidence below the runner's temporary directory, and verifies that the source checkout remains unchanged.

The CI artifact records the run ID, job ID, toolchain, runner, fixed denominator, exact state counts, case vectors, receipt-chain validity, deterministic replay, inventory, and authority boundary. Missing measurements are represented as `null` plus an UNKNOWN state; the workflow does not invent a zero.

The v0.49 local-validation history is an optional immutable input. Its exact five commands are retained with `OPERATIONAL_REFUTED`; the CI run does not internally close the operator API observation. `operator_api_attempts` remains `null` plus `UNKNOWN` until an independent authoritative operator receipt is supplied.

## PR-first sequence

1. Open a pull request from the implementation branch.
2. Wait for the pull-request conformance job to succeed.
3. Merge the pull request and wait for the successful `main` conformance run.
4. Dispatch the release workflow with the exact merge SHA and successful main run ID.

The release workflow uses only `${{ github.token }}`. It never deletes or overwrites a failed run, pull request, tag, draft, release, or asset. An existing tag or release for the requested version is a hard stop.

