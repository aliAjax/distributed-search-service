# Bug reproduction

## What happens
Index snapshot readers expose mutable backing storage while writers update it concurrently.

## How to trigger
Run the four targeted index tests with the race detector and concurrent readers and writers.

## Observed error
The race detector reports `WARNING: DATA RACE` with conflicting reads and writes in snapshot and store methods.
