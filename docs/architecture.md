# Architecture Notes

The service separates domain objects from persistence and transport. A write is accepted only after schema validation and a durable WAL append. Segment readers use immutable snapshots while maintenance builds a replacement generation. The repository interface can be backed by PostgreSQL without changing domain code.
