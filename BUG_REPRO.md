# Bug reproduction

## What happens
Storage replay, append, recovery and copy operations continue their work after the caller cancels the request because the downstream operation uses a detached context.

## How to trigger
Run the four targeted storage tests in `verify_cmds` with an already-canceled context.

## Observed error
The tests report a nil error and multiple callback invocations instead of `context canceled`.
