# Bug reproduction

## What happens
Repository error wrappers detach sentinel errors while crossing the repository boundary. Missing, conflict, version and page failures cannot be classified with `errors.Is`.

## How to trigger
Run the four targeted repository tests in `verify_cmds` from the collection record against the published bug branch.

## Observed error
The targeted tests fail with messages such as `missing sentinel detached`, `conflict sentinel detached`, `version sentinel detached`, and `page sentinel detached`.
