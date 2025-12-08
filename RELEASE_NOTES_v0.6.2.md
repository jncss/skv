# SKV v0.6.2 - CLI Encryption Recovery Fix

**Release Date:** December 8, 2025

## Overview

SKV v0.6.2 is a bug fix release that resolves a critical issue with the CLI `recover` command when used with encrypted databases. This release ensures data recovery works correctly for encrypted databases without double-encryption corruption.

## 🐛 Bug Fixes

### CLI Recover Command with Encryption

**Problem:**
- The `recover` command was double-encrypting data when recovering encrypted databases
- Process: Read encrypted bytes from corrupted file → `Put()` re-encrypted them → unreadable double-encrypted data
- Result: Recovered databases were unreadable even with the correct password

**Solution:**
- Recover now temporarily disables encryption during the recovery operation
- Raw bytes are copied as-is from corrupted file to recovered file
- If original file was encrypted, recovered file maintains that encryption
- Must use the same encryption password to access recovered database

**Example:**
```bash
# Recover encrypted database (specify encryption to indicate original was encrypted)
./skv -e aes -p myPassword recover corrupted.skv recovered.skv

# Note: Recovered file preserves encryption, use same password to access
./skv -e aes -p myPassword get recovered.skv mykey
```

### Memory Safety Improvement

**Added Protection Against OOM Panics:**
- Added 100MB maximum size limit for individual record allocations
- Prevents Out-Of-Memory panics when corrupted data size fields contain invalid values
- Gracefully skips corrupted records instead of crashing

**Before:**
```
panic: runtime error: makeslice: len out of range
```

**After:**
```
Warning: data size exceeds maximum reasonable size, skipping...
✓ Recovery complete
  Total records recovered: 4
  Invalid bytes skipped: 996
```

## 📚 Documentation Updates

### CLI README.md

**Updated Example 9: Recovery with Encryption**
```bash
# Recover encrypted database
# Note: Specify encryption flags to indicate the file WAS encrypted
# Recovered file will preserve the encryption (data stays encrypted)
skv -e aes -p mySecretPassword recover corrupted_secure.skv repaired_secure.skv

# Access recovered encrypted database (use same password)
skv -e aes -p mySecretPassword get repaired_secure.skv mykey
```

**Added clarification:**
> Important: Recovery preserves the encryption state of the original file. If the corrupted file was encrypted, the recovered file will also be encrypted with the same data. You must use the same encryption password to access the recovered database.

### Enhanced Testing

**test_recovery.sh improvements:**
- Added dedicated encrypted database recovery test
- Verifies 3 encrypted records can be recovered and read correctly
- Tests corruption handling with encrypted data
- All tests passing ✅

## 🧪 Testing

**Test Results:**
- ✅ All 238 library tests passing
- ✅ All 10 encryption tests passing
- ✅ CLI recovery tests passing (normal + encrypted)
- ✅ Encrypted recovery: 3/3 records recovered successfully
- ✅ Memory safety: No OOM panics on corrupted data

## 🔧 Technical Details

### Implementation Changes

**File:** `tools/cli/commands.go`

**handleRecover() function:**
```go
// Save encryption settings
savedEncryptionType := encryptionType
savedEncryptionPassword := encryptionPassword

// Disable encryption during recovery (copy raw bytes)
encryptionType = "none"
encryptionPassword = ""

// ... perform recovery ...

// Restore encryption settings
encryptionType = savedEncryptionType
encryptionPassword = savedEncryptionPassword
```

**tryParseRecord() function:**
```go
// Additional sanity check: protect against unreasonably large allocations
const maxReasonableSize = 100 * 1024 * 1024 // 100 MB
if dataSize > maxReasonableSize {
    return nil, 0, fmt.Errorf("data size exceeds maximum reasonable size")
}
```

### Behavior Details

**Encryption During Recovery:**
1. User specifies `-e aes -p password` to indicate original file was encrypted
2. Recover temporarily disables encryption internally
3. Reads raw encrypted bytes from corrupted file
4. Writes raw encrypted bytes to recovered file (no re-encryption)
5. Displays message reminding user to use same password for access

**Why This Works:**
- Encrypted data in file is just bytes (Base64-encoded ciphertext)
- Recovery copies these bytes verbatim without interpretation
- Recovered file has identical encrypted content as original
- Same password successfully decrypts the preserved encrypted data

## ⬆️ Upgrade Guide

**From v0.6.1 to v0.6.2:**

This is a transparent upgrade for most users:

```bash
go get github.com/jncss/skv@v0.6.2
```

**For CLI Users:**

If you previously attempted to recover an encrypted database with v0.6.1 and got corrupted results:

1. Delete the incorrectly recovered file
2. Update to v0.6.2 CLI: `cd tools/cli && go build -o skv`
3. Re-run recovery with encryption flags: `./skv -e aes -p PASSWORD recover corrupted.skv recovered.skv`
4. Access with same password: `./skv -e aes -p PASSWORD get recovered.skv key`

**No Changes Required:**
- Library API unchanged
- Existing code continues to work
- Only affects CLI `recover` command behavior

## 📋 Compatibility

- **Go Version:** Requires Go 1.24.0 or higher
- **Platforms:** Linux, macOS, BSD, Windows (all platforms supported)
- **Backward Compatibility:** Full compatibility with v0.6.1 and v0.6.0
- **Database Format:** No changes, all v0.6.x databases compatible

## 🎯 Impact

**Who Should Upgrade:**

**High Priority:**
- Users who need to recover encrypted databases
- Users experiencing OOM panics during recovery of corrupted files
- CLI users working with sensitive encrypted data

**Standard Priority:**
- All CLI users (for improved safety and reliability)
- Anyone performing database recovery operations

**Low Priority:**
- Library-only users not using CLI recover command
- Users not using encryption feature

## 🔗 Related

- See [RELEASE_NOTES_v0.6.0.md](RELEASE_NOTES_v0.6.0.md) for the initial encryption feature
- See [RELEASE_NOTES_v0.6.1.md](RELEASE_NOTES_v0.6.1.md) for dependency updates
- See [ENCRYPTION.md](ENCRYPTION.md) for encryption documentation
- See [tools/cli/RECOVERY.md](tools/cli/RECOVERY.md) for recovery strategies

---

**Summary:** This release fixes a critical bug in CLI encrypted database recovery and adds safety protections against memory exhaustion. All users of the CLI `recover` command should upgrade, especially those working with encrypted databases.
