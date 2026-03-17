---
name: Never run the build pipeline
description: Do not run make run or the build command - it wipes the existing DB. Tell the user to build instead.
type: feedback
---

Never run the build pipeline (`make run`, `go run ./cmd/build`, etc.) — it deletes and rebuilds the database, wiping out the user's current build.

**Why:** I ran a build that destroyed the user's existing DB. The build is slow (~6 minutes) and the user needs to control when it runs.

**How to apply:** When code changes need testing via a full build, tell the user to run the build themselves. Write code, run tests, but never trigger the pipeline.
