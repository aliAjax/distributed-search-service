# Bug reproduction

## What happens
Zero-value query options and facet maps are nil, and a typed-nil policy is treated as a live interface value.

## How to trigger
Run the four targeted query tests with omitted optional configuration and a typed-nil policy.

## Observed error
The tests panic with `assignment to entry in nil map` or `invalid memory address or nil pointer dereference`.
