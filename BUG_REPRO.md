# Bug reproduction

## What happens
Analyzer filtering reuses the caller's slice backing array. Later token, synonym, stopword and registry operations overwrite data retained by the caller.

## How to trigger
Run the four targeted analyzer tests after retaining the input slice and mutating the returned result.

## Observed error
The tests report `input mutated` or `result aliases input`.
