# Bug reproduction

## What happens
The ingest retry state machine leaves successful retries in an intermediate state. Its transition table, pending view and audit record disagree about the final state.

## How to trigger
Run the four targeted ingest tests through the retry-success path.

## Observed error
The tests observe `state=retrying`, a missing retrying-to-succeeded edge, a pending view that omits retrying, and an audit value of `retrying`.
