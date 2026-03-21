---
name: feedback_resolve_warnings
description: Always resolve warnings/issues from changes; ask about pre-existing ones
type: feedback
---

If a change introduces warnings or leaves unresolved issues, fix them before finishing. If warnings are pre-existing (from before the current task), ask the user if they want to correct them rather than ignoring them silently.

**Why:** User expects clean output — no leftover warnings from changes, and transparency about pre-existing issues.
**How to apply:** After any code change, check for warnings/errors. Fix anything caused by the current work. For pre-existing issues spotted during the work, ask the user before fixing.
