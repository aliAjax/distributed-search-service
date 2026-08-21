# Bug reproduction

## What happens
Shard result and error channels have mismatched close and wait lifecycles. A producer can return without closing, an error sender can block, and the WaitGroup can be updated after waiting starts.

## How to trigger
Run the four targeted shard tests with the race detector and an erroring shard.

## Observed error
The tests time out with `channel did not close`, `error delivery blocked`, or `wait returned before shard started`.
