# Stream Rollback Protection

## Overview

PutStream() and UpdateStream() now include rollback protection to ensure database integrity even when stream operations fail mid-write.

## Implementation

### PutStream()

**Checkpoint-based rollback:**
1. Save current file position as checkpoint
2. Write stream data via writeRecordStream()
3. On error: Truncate file to checkpoint and restore file position
4. On success: Sync to disk, then update cache

**Guarantees:**
- Atomic: Either the entire key-value is written or nothing
- No partial records in the database file
- Cache remains consistent with file state
- Failed operations leave database in clean state for retry

### UpdateStream()

**Write-then-delete approach:**
1. Check that key exists
2. Save current file position as checkpoint
3. Write new record via writeRecordStream()
4. On error: Truncate file to checkpoint and restore file position
5. On success: Sync to disk, delete old record, update cache to point to new record

**Guarantees:**
- Original value preserved if update fails
- No data loss on write errors
- Atomic update: Either completes fully or reverts to original state
- Old record only deleted after new record is safely written and synced

**Key difference from regular Update():**
- Regular Update() uses WAL (Write-Ahead Log) for rollback
- UpdateStream() uses checkpoint/truncate to avoid loading entire stream into WAL
- Trade-off: Slightly more disk I/O during update, but much lower memory usage for large values

## Error Scenarios

### Partial Write Failure
**Scenario:** writeRecordStream() fails after writing 100 bytes of a 1000-byte value

**Recovery:**
1. Error detected in writeRecordStream()
2. file.Truncate(checkpoint) removes the partial write
3. file.Seek(checkpoint, io.SeekStart) restores file position
4. Error returned to caller
5. Cache not updated - key still points to old value (UpdateStream) or doesn't exist (PutStream)

### Sync Failure
**Scenario:** Write succeeds but file.Sync() fails

**Recovery:**
1. Same rollback as partial write failure
2. Truncate and seek back to checkpoint
3. Cache not updated
4. Database remains in pre-operation state

### Rollback Failure
**Scenario:** Write fails AND truncate fails (very rare)

**Recovery:**
1. Error logged with both original error and rollback error
2. Combined error returned to caller
3. Database may be in inconsistent state
4. Recommendation: Close and reopen database to rebuild cache from file

## Testing

New test file: `stream_rollback_test.go`

### TestPutStreamRollbackOnError
- Verifies rollback on failed PutStream
- Checks file size unchanged after rollback
- Confirms failed key not added to cache
- Validates existing keys still work
- Tests successful write after failed rollback

### TestUpdateStreamRollbackOnError
- Verifies original value preserved on failed UpdateStream
- Checks file doesn't grow excessively
- Confirms other keys unaffected
- Tests successful write after failed rollback
- Validates database consistency after reopen

### TestStreamRollbackPreservesIntegrity
- Tests with 10 existing keys
- Failed stream operation in the middle
- All original keys still accessible
- Database reopens correctly
- Failed key doesn't exist

### errorReader Test Helper
- Simulates io.Reader that fails after N bytes
- Allows testing partial write scenarios
- Configurable failure point

## Performance Characteristics

### PutStream with Rollback
- **Best case (success):** +1 file.Sync() call
- **Worst case (failure):** +1 file.Truncate() + 1 file.Seek()
- **Memory:** No overhead (checkpoint is just int64)
- **Disk:** Failed writes cleaned up immediately

### UpdateStream with Rollback
- **Best case (success):** +1 file.Sync() before delete
- **Worst case (failure):** +1 file.Truncate() + 1 file.Seek()
- **Memory:** No overhead compared to WAL approach (avoids buffering entire stream)
- **Disk:** Temporary duplication (both old and new record) until old is deleted

### Comparison to WAL Approach

**Checkpoint/Truncate (implemented):**
- ✅ Constant memory usage (no stream buffering)
- ✅ Simple rollback mechanism
- ❌ Extra Sync() call on success
- ❌ Can't rollback deleteInternal() (UpdateStream)

**WAL (not implemented for streams):**
- ❌ Must buffer entire stream in WAL (memory issue for large values)
- ✅ Can rollback all operations atomically
- ❌ Complex recovery logic
- ❌ More disk I/O

**Decision:** Checkpoint/Truncate is better for large streams because it avoids memory overhead while providing atomic guarantees for the write operation.

## Logging

All rollback events are logged with appropriate levels:

**Warn:** Rollback succeeded
- Key name
- Error that triggered rollback

**Error:** Rollback failed
- Key name
- Original error
- Rollback error

**Debug:** Successful stream write
- Key name
- Size
- File position

## Future Improvements

1. **Two-phase UpdateStream:** 
   - Could use WAL entry with just key + flag (not the data)
   - On recovery: If WAL shows incomplete update, delete the new record
   - Would allow true atomic rollback of both write and delete

2. **Incremental Sync:**
   - Use file.Sync() with data-only flag where supported
   - Reduce sync overhead on successful writes

3. **Deferred Deletion:**
   - Mark old record as "deleted after" with pointer to new record
   - Garbage collection can clean up later
   - Avoids need for deleteInternal() during update

4. **Metrics:**
   - Track rollback frequency
   - Monitor partial write sizes
   - Measure performance impact

## Related Files

- `skv.go`: PutStream() and UpdateStream() implementations
- `stream_rollback_test.go`: Rollback test suite (3 tests)
- `file_test.go`: Existing stream tests (16 tests)
- `TESTING.md`: Overall testing documentation

## Version

Added in: SKV v0.3.1 (in development)
Previous version (v0.3.0): No rollback protection for streams
