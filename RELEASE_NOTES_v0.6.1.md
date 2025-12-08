# SKV v0.6.1 - Dependency Update

**Release Date:** December 8, 2025

## Overview

SKV v0.6.1 is a maintenance release that updates the EasyAES encryption library to its stable v1.0.0 release. This is a minor update with no breaking changes or new features.

## 🔄 Changes

### Dependencies

**Updated:**
- `github.com/jncss/easyaes` - Upgraded from v0.0.0-20251208190620-9743bf4abb45 to **v1.0.0**
  - Now using the stable v1.0.0 release instead of the development version
  - No API changes, fully backward compatible
  - All encryption functionality remains identical

### Code Quality

**Fixed:**
- Removed redundant newline in `examples/12-encryption/main.go` (formatting improvement)

## ✅ Testing

- **All 238 tests passing** with the updated dependency
- No changes to test coverage (81.0%)
- Encryption tests verified working correctly with easyaes v1.0.0

## 🔧 Upgrade Guide

**From v0.6.0 to v0.6.1:**

This is a transparent upgrade - simply update your dependencies:

```bash
go get github.com/jncss/skv@v0.6.1
```

No code changes required. All existing code using v0.6.0 works identically with v0.6.1.

## 📋 Compatibility

- **Go Version:** Requires Go 1.24.0 or higher
- **Platforms:** Linux, macOS, BSD, Windows (all platforms supported)
- **Backward Compatibility:** Full compatibility with v0.6.0
- **Database Format:** No changes, all v0.6.0 databases work with v0.6.1

## 🔗 Related

- See [RELEASE_NOTES_v0.6.0.md](RELEASE_NOTES_v0.6.0.md) for the initial encryption feature release
- See [CHANGELOG.md](CHANGELOG.md) for complete version history
- See [ENCRYPTION.md](ENCRYPTION.md) for encryption documentation

---

**Note:** This is a maintenance release with no functional changes. Users of v0.6.0 can upgrade at their convenience.
