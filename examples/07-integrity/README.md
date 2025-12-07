# Data Integrity and CRC Verification

This example demonstrates SKV's built-in CRC (Cyclic Redundancy Check) integrity verification.

## What is CRC?

CRC is a checksum algorithm used to detect accidental changes to data. SKV uses:

- **CRC-16-CCITT** (2 bytes) for small records (≤ 255 bytes of data)
- **CRC-32-IEEE** (4 bytes) for larger records

The CRC covers the entire record structure and is stored at the end of each record.

## How It Works

Every record in an SKV file has this structure:

```
[Type][KeySize][Key][DataSize][Data][CRC]
                                      ^^^^
                                      Checksum covers all previous fields
```

1. **On Write**: SKV calculates the CRC over the entire record and appends it
2. **On Read**: SKV recalculates the CRC and compares it with the stored value (only for active records)
3. **On Delete**: Only the Type byte is modified with the DeletedFlag; CRC is not recalculated (deleted records are skipped in reads)

## When CRC Checks Happen

CRC verification is automatic and happens:

- **Every Get/GetString**: Reading any active value verifies its CRC
- **During Verify()**: Scans all active records and verifies each CRC (deleted records are counted but not verified)
- **During Compact()**: Reads all active records (CRC verified) and rewrites them
- **During rebuildCache()**: When opening a database, all active record CRCs are verified

**Note**: Deleted records skip CRC verification because the Type byte is modified during deletion.

## Running the Example

```bash
cd examples/07-integrity/crc_verification
go run crc_verification.go
```

**Expected output:**

```
Database verified successfully!
  Active records: 3
  Total records: 3
  CRC checks: All passed ✓

Read 'user:1': John Doe (CRC verified ✓)
Updated 'user:1': John Doe Updated (CRC verified ✓)

Database verified after update:
  Active records: 3
  Deleted records: 1 (CRC not verified)
```

## What Happens When Corruption is Detected

If a record's CRC doesn't match (e.g., due to disk corruption or file tampering):

```go
value, err := db.GetString("corrupted_key")
if err != nil {
    // Error will be something like:
    // "CRC mismatch: expected 0xD1E4, got 0x869A (record may be corrupted)"
}
```

## Best Practices

1. **Regular Verification**: Run `Verify()` periodically to check database integrity
2. **Deleted Records**: Don't worry about CRC of deleted records - they're ignored during reads
3. **Before Backups**: Verify integrity before creating backups
4. **After Recovery**: Verify after any system crash or unexpected shutdown

## Performance Impact

CRC calculation and verification is very fast:

- **CRC-16**: ~1-2 GB/s on modern CPUs
- **CRC-32**: ~500 MB/s - 1 GB/s on modern CPUs

The overhead is negligible compared to disk I/O, and the data integrity guarantee is worth it.
