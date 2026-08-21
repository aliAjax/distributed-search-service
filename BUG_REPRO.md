# Bug reproduction

## What happens
HTTP request context bridges replace the caller's context with `context.Background`, so cancellation and deadline state do not reach downstream work.

## How to trigger
Run the four targeted HTTP context tests with canceled and deadline-bound requests.

## Observed error
The tests observe a downstream callback after cancellation and a missing deadline.
