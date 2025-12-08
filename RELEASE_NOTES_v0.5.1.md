# SKV v0.5.1 Release Notes

**Release Date:** December 8, 2025

This is a patch release that fixes critical bugs related to compression and data integrity verification.

## 🐛 Bug Fixes

### Compression with Record Reuse
Fixed data corruption when updating uncompressed records with compressed data. The issue occurred when `writeRecord()` reused space from deleted records:

- **Problem**: Padding calculation used estimated size (before compression) instead of actual written size
- **Impact**: Left garbage data in file that could be misread as invalid records
- **Solution**: `writeRecordAtPosition()` now returns actual record size for accurate padding
- **Result**: Clean padding bytes (0x80) fill unused space correctly

**Example scenario that was failing:**
```bash
skv putfile db.skv key1 /etc/hosts          # 200 bytes uncompressed
skv -c lz4 updatefile db.skv key1 /etc/hosts  # Compressed to ~40 bytes
skv verify db.skv                           # Would fail with "unknown record type"
```

### Verify Statistics Accuracy
Fixed incorrect efficiency and padding calculations in `Verify()` when database contained compressed records:

- **Problem**: Final padding bytes were not counted
- **Impact**: `Efficiency` and `Wasted Percent` showed incorrect values
- **Solution**: Added padding to statistics before breaking from EOF loop
- **Result**: Accurate statistics for all compression scenarios

**Before fix:**
```
Padding Bytes:    0 bytes
Efficiency:       71.65%
Wasted Percent:   0.00%
```

**After fix:**
```
Padding Bytes:    31 bytes
Efficiency:       85.24%
Wasted Percent:   14.76%
```

## 📊 Technical Changes

### API Changes
- `writeRecordAtPosition()` signature changed: `(int64, error)` → `(int64, uint64, error)`
  - Returns actual record size as second value
  - **Internal API**: No impact on public API

### Code Improvements
- Renamed `activeDataSize` → `activeRecordSize` for clarity
- Added comments distinguishing on-disk size (compressed) vs decompressed size
- Improved padding detection logic in `Verify()`

## 🧪 Testing

- **All 228 tests passing** ✓
- **Coverage: 80.9%** (improved from 80.8%)
- Verified with multiple compression scenarios:
  - Uncompressed → Compressed updates
  - Mixed compression databases
  - Databases with padding
  - Deleted records with compression

## 📦 Installation

```bash
go get github.com/jncss/skv@v0.5.1
```

## 🔄 Upgrade Notes

This is a **bug fix release** - fully backward compatible with v0.5.0.

- No breaking changes
- No database migration required
- Existing databases work without modification
- CLI tools automatically use new version when rebuilt

## 📝 Full Changelog

See [CHANGELOG.md](CHANGELOG.md) for complete details.

## 🙏 Acknowledgments

Thanks to users who reported the compression-related verify issues!

---

**Previous Release:** [v0.5.0](RELEASE_NOTES_v0.5.0.md) - Atomic Transactions
**Next Steps:** See [TODO](todo.txt) for upcoming features
