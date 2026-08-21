# Bug reproduction

## What happens
Configuration error wrappers create detached errors and use `%v`, preventing callers and monitoring from classifying open, scan, validation and startup failures.

## How to trigger
Run the four targeted configuration tests with each sentinel failure.

## Observed error
The tests report `open error detached`, `scan error detached`, `validation error detached`, and `classification error detached`.
