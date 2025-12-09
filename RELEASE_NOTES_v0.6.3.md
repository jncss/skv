# Release Notes - v0.6.3

**Release Date:** December 9, 2025

## 🐛 Bug Fixes

### Fixed `go install` failure due to invalid file paths

- **Issue**: Files with colons (`:`) in their names were tracked in Git, causing `go install` to fail with "malformed file path" errors when creating module zip files
- **Solution**: 
  - Removed problematic files from Git tracking in `examples/02-advanced/file_operations/data/files/extracted/`
  - Added `.gitignore` to prevent future tracking of extracted files
  - Updated `file_operations.go` to sanitize filenames by replacing colons with underscores during file extraction

### Files Fixed
- `examples/02-advanced/file_operations/data/files/extracted/assets:logo`
- `examples/02-advanced/file_operations/data/files/extracted/config:app`
- `examples/02-advanced/file_operations/data/files/extracted/scripts:main.js`
- `examples/02-advanced/file_operations/data/files/extracted/styles:main.css`
- `examples/02-advanced/file_operations/data/files/extracted/templates:footer.html`
- `examples/02-advanced/file_operations/data/files/extracted/templates:header.html`

## 📦 Installation

```bash
go install github.com/jncss/skv@v0.6.3
```

Or in your `go.mod`:

```go
require github.com/jncss/skv v0.6.3
```

## 🔄 Upgrade Notes

This is a patch release that fixes installation issues. No breaking changes or API modifications.

## 📝 Changes

- Added `.gitignore` file to workspace root
- Modified `examples/02-advanced/file_operations/file_operations.go` to sanitize extracted filenames
- Removed invalid files from Git repository

## ✅ Verification

After this release, the following command should work without errors:

```bash
go install github.com/jncss/skv@latest
```

---

**Full Changelog**: https://github.com/jncss/skv/compare/v0.6.2...v0.6.3
