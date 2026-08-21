# Failure Drills

Stop after WAL fsync and restart `cmd/indexer` to validate checksum/replay. Corrupt one segment and restore the last snapshot generation. Cancel a multi-shard query context and verify partial-result semantics. Create analyzer version 2 and verify old documents retain their analyzer version.
