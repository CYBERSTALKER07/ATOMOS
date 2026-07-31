# Gap closure build result — 2026-08-01

**Run time:** 2026-08-01 ~01:28 AM (UTC+5)  
**Target tag:** `ssmr-gap-closure-nomock`  
**Project:** `pegasus-503013` / ns `pegasusx-ssmr`

## Subagent execution note

The subagent **Shell tool was unavailable** for this run (`Service temporarily unavailable` / `Command failed to spawn` on every invocation). Steps below combine:

- Direct file verification (`application_default_credentials.json`)
- Parent-terminal logs from the same workstation session (timestamps ~2026-08-01 00:37–01:16 UTC+5)
- Prepared pipeline script at `/tmp/px_gap_closure_run.sh` (not executed — shell spawn failed)

---

## 1) ADC / project

| Check | Result |
|-------|--------|
| `quota_project_id` in `~/.config/gcloud/application_default_credentials.json` | **`pegasus-503013`** (verified by reading file) |
| `gcloud config set project pegasus-503013` | **Not executed** (subagent shell down) |
| `gcloud auth application-default set-quota-project pegasus-503013` | **Not executed** (subagent shell down) |

ADC quota project is already set correctly on disk. Prior parent session noted intermittent `USER_PROJECT_DENIED` on `v-o-i-d` for some APIs; Spanner migrations succeeded with `GOOGLE_CLOUD_QUOTA_PROJECT=pegasus-503013`.

---

## 2) Build

### Method: **failed** (Cloud Build → Docker fallback both blocked)

### Cloud Build (`gcloud builds submit`)

- **Status:** FAILED (`BUILD_EXIT:1`)
- **Log:** `/tmp/ssmr-build.log` (parent terminal, 2026-08-01 00:37 UTC+5)
- **Exact error:**

```
ERROR: (gcloud.builds.submit) The user is forbidden from accessing the bucket [pegasus-503013_cloudbuild]. Please check your organization's policy or if the user has the "serviceusage.services.use" permission. Giving the user a role with this permission such as Service Usage Admin may fix this issue. Alternatively, use the --no-source option and access your source code via a different method.
```

- **Note:** This run did **not** surface `USER_PROJECT_DENIED` / `v-o-i-d` quota — it failed on **Cloud Build bucket IAM** (`serviceusage.services.use` / bucket access).

### Docker fallback (local build + push)

- **Status:** FAILED (`DOCKER_BUILD_EXIT:1`)
- **Log:** `/tmp/ssmr-docker-build.log`
- **Dockerfile:** `apps/backend-go/Dockerfile` (build context: repo root)
- **Exact error:**

```
ERROR: failed to build: failed to solve: DeadlineExceeded: failed to fetch anonymous token: Get "https://auth.docker.io/token?scope=repository%3Alibrary%2Fdebian%3Apull&service=registry.docker.io": dial tcp: lookup auth.docker.io: i/o timeout
```

- **Push:** Not attempted (build failed before image existed).

---

## 3) Rollout

| Item | Result |
|------|--------|
| Image `…/backend-go:ssmr-gap-closure-nomock` | **Not available** |
| kubectl set image (backend-go / worker) | **Skipped** |
| Rollout status | **Not run** |

**Cluster still on prior image:** `asia-south1-docker.pkg.dev/pegasus-503013/pegasusx-ssmr-images/backend-go:ssmr-gap-closure-20260731g`

---

## 4) Smoke

| Check | Result |
|-------|--------|
| Requested curl with Host header to `136.69.43.141/v1/health` | **Not executed** (subagent shell down) |
| Last verified smoke (parent, 2026-07-31 19:44 UTC) | **OK** — `{"service":"pegasusx-backend","status":"ok"}` |

---

## 5) DNS / ManagedCert / Global Pay secrets

**Not re-queried this run** (subagent shell down). Last known from parent handoff:

| Check | Status |
|-------|--------|
| `dig api-ssmr.pegasusx.app A` | **No public A record** (pending) |
| `dig admin-ssmr.pegasusx.app A` | Not verified this run |
| ManagedCertificate | **Provisioning** (needs DNS A → `136.69.43.141`) |
| `backend-go-secrets` GP keys | Not re-checked; prior usage shows lowercase keys (`jwt-secret`, `global-pay-webhook-secret`) |

---

## Summary

| Field | Value |
|-------|-------|
| Build method | **failed** (cloudbuild → docker fallback) |
| Image tag deployed | **No** |
| Smoke (this run) | **Not run**; last known OK |
| DNS | Pending |
| ManagedCert | Provisioning (last known) |
| GP secrets | Not re-verified |

## Unblock

1. Cloud Build IAM on `pegasus-503013` / `pegasus-503013_cloudbuild` bucket.
2. Or fix Docker Hub network; local build + push to AR.
3. Re-run: `bash /tmp/px_gap_closure_run.sh`
