# Database Recovery

The `recover` command allows you to salvage valid records from a corrupted SKV database file.

## How It Works

The recovery process works by:

1. **Scanning byte-by-byte** through the corrupted file
2. **Looking for type bytes** (0x01, 0x02, 0x04, or 0x08) that could indicate the start of a record
3. **Attempting to parse** each potential record according to the SKV format
4. **Verifying the CRC** checksum to confirm the record is valid
5. **Saving valid records** to a new database file

## Usage

```bash
skv recover <corrupted-file> <recovered-file>
```

**Arguments:**
- `corrupted-file`: Path to the corrupted SKV database
- `recovered-file`: Path where recovered records will be saved (must not exist)

## Example

```bash
# Try to recover from a corrupted database
skv recover broken.skv repaired.skv

# Verify the recovered database
skv verify repaired.skv

# Check which records were recovered
skv keys repaired.skv
```

## What Gets Recovered

✅ **Recovered:**
- Active records with valid CRC checksums
- Records where the structure is intact
- Complete key-value pairs

❌ **Not Recovered:**
- Deleted records (marked with DeletedFlag)
- Records with invalid CRC (corrupted data)
- Incomplete records (truncated or missing data)
- Random bytes that don't form valid records

## Example Output

```
Attempting to recover records from 'broken.skv'...
Found valid SKV header, skipping...
Scanning 15438 bytes for valid records...
  Recovered 100 records...
  Recovered 200 records...

✓ Recovery complete
  Total records recovered: 247
  Invalid bytes skipped: 1834
  Recovered database: repaired.skv
```

## When to Use Recovery

Use the `recover` command when:

- **Database corruption** detected by `verify` command
- **Disk errors** have damaged the file
- **Interrupted writes** left the file in an inconsistent state
- **Power failure** during database operations
- **File system corruption** affected the database file

## Recovery Limitations

1. **Not a substitute for backups**: Regular backups (`skv backup`) are essential
2. **No guarantee**: Heavily corrupted files may have few or no recoverable records
3. **Order may differ**: Recovered records are written in discovery order, not original order
4. **Duplicates possible**: If a key appears multiple times, the last valid occurrence wins

## Best Practices

### Before Recovery

1. **Make a copy** of the corrupted file:
   ```bash
   cp broken.skv broken.skv.backup
   ```

2. **Verify the corruption**:
   ```bash
   skv verify broken.skv
   ```

### After Recovery

1. **Verify the recovered database**:
   ```bash
   skv verify repaired.skv
   ```

2. **Check record count**:
   ```bash
   skv count repaired.skv
   ```

3. **Inspect recovered keys**:
   ```bash
   skv keys repaired.skv
   ```

4. **Test critical data**:
   ```bash
   skv get repaired.skv important-key
   ```

5. **Compact if needed** (to optimize the recovered database):
   ```bash
   skv compact repaired.skv
   ```

## Recovery Strategy

For maximum data recovery, try these strategies in order:

### 1. Direct Recovery (Best Case)
```bash
skv recover broken.skv repaired.skv
```

### 2. Compare with Backup
If you have a backup, compare what was recovered:
```bash
skv backup repaired.skv recovered.json
skv backup original.skv original.json
diff recovered.json original.json
```

### 3. Manual Inspection
For critical missing data, use hexdump to inspect the corrupted file:
```bash
skv --hex foreach broken.skv
```

## Technical Details

### Record Detection

The recovery algorithm identifies potential records by:

1. Finding bytes matching valid type values: `0x01`, `0x02`, `0x04`, `0x08`
2. Reading the key size (1 byte) and validating it's reasonable
3. Reading the key data
4. Reading the data size (1/2/4/8 bytes based on type)
5. Reading the data content
6. Reading and verifying the CRC-16 or CRC-32 checksum

### CRC Verification

Each record's CRC is calculated over:
- Type byte
- Key size byte
- Key data
- Data size (1/2/4/8 bytes)
- Data content

If the calculated CRC matches the stored CRC, the record is considered valid.

### False Positives

The algorithm minimizes false positives by:
- Requiring valid type bytes (only 4 possible values)
- Sanity checking data sizes (rejecting sizes larger than remaining file)
- Verifying CRC checksums (very low probability of random match)

## See Also

- `skv backup` - Create regular backups for disaster recovery
- `skv verify` - Check database integrity
- `skv compact` - Optimize database after recovery
