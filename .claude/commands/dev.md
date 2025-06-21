---
description: Developer workflow
---

Read @specs/architecture.md and then read @specs/prompt-plan.md - determine what needs to be implemented next.

Double check that it hasn't been implemented yet.

Create a new git branch using the pattern `claude/short-branch-description-here`.  I recommend referencing the prompt plan step if applicable, e.g. `claude/step-14-cli-watch-mode`.

Work to implement the next step of the plan. Add tests and run tests regularly.

Fix all test errors you find.

When done with the step, update @specs/prompt-plan.md to mark all the implemented components as complete then stop to let me review the functionality.

I will commit the changes to the repo if they look good.
