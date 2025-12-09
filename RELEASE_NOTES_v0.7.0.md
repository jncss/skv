# Release Notes - v0.7.0

**Release Date:** December 9, 2025

## Overview

SKV v0.7.0 is a minor release that includes internal improvements and bug fixes. This release updates the version numbering and ensures consistency across all documentation.

## 🔧 Changes

### Version Updates
- Updated file format version to 0.7.0
- All version constants updated in `skv.go`
- Documentation updated to reflect new version

### Internal Improvements
- Fixed padding calculation in `rebuildCache()` for better free space tracking
- Enhanced error reporting in `Verify()` with position information for corruption detection
- Improved cache indexing accuracy

## 📦 Installation

```bash
go install github.com/jncss/skv@v0.7.0
```

Or in your `go.mod`:

```go
require github.com/jncss/skv v0.7.0
```

## 🔄 Upgrade Notes

This is a minor release with no breaking changes. All existing code and databases are fully compatible with v0.7.0.

- **Backward Compatibility:** Full compatibility with v0.6.x
- **Database Format:** No changes, all v0.6.x databases work with v0.7.0
- **API:** No changes to public API

## 📝 Technical Details

### File Format
- Version header: 0.7.0 (0x00 0x07 0x00)
- No changes to record structure
- Compatible with all v0.6.x databases

### Bug Fixes
- **Cache rebuild**: Fixed position calculation to correctly account for padding bytes
- **Verify**: Improved error messages with hexadecimal position information

## ✅ Verification

After installing, verify the version:

```bash
go version -m $(which skv)
```

## 🔗 Links

- **Repository**: https://github.com/jncss/skv
- **Documentation**: See README.md for complete feature list
- **Examples**: See `examples/` directory for usage examples

---

**Full Changelog**: https://github.com/jncss/skv/compare/v0.6.3...v0.7.0
