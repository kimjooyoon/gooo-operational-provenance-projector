# Immutable release policy

Release input must identify one exact merged `main` commit and one successful `main` conformance run whose `head_sha` is that commit. The release job reruns the same CI-only evidence generation against that commit and records the run, job, main run, main artifact identity, denominator, exact metrics, inventory, operational audit, receipt chain, and replay result.

The release sequence is one-way:

1. refuse an existing tag;
2. create one annotated tag pointing at the merge commit;
3. refuse an existing release for that tag;
4. create a draft release through the GitHub API and keep its returned release ID;
5. upload the evidence bundle and release audit assets;
6. enable and verify the platform immutable-release policy;
7. publish the draft once;
8. verify `immutable=true`, the annotated tag object and peeled commit, exact asset count, and every asset's `sha256:` digest.

No cleanup path deletes or overwrites failed evidence. If a step fails, the resulting run, tag, draft, release, or asset remains available for audit.

