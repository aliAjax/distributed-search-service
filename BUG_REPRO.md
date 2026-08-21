# Bug reproduction

## What happens
Maintenance defers resource releases until the outer batch ends, while cleanup and rollback defer functions overwrite the primary operation error.

## How to trigger
Run the four targeted maintenance tests with multiple leases and independent operation, rollback and close errors.

## Observed error
The tests report `active=0 peak=4`, return only `cleanup` or `close`, and show that rollback was not called.
