# Changelog

All notable changes to SKV will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **CRC Integrity Checking**: Every record now includes a CRC checksum for corruption detection
  - CRC-16-CCITT (2 bytes) for Type 0x01 records (data ≤ 255 bytes)
  - CRC-32-IEEE (4 bytes) for Types 0x02, 0x04, 0x08 (larger data)
  - Automatic verification on all read operations for active records
  - Deleted records skip CRC verification (type byte is modified during deletion)
- **Atomic Compaction**: Compact operation now uses temporary files for safe atomic updates
  - Prevents corruption if process is interrupted during compaction
  - Original file is only replaced after new file is completely written and verified
  - See `COMPACT_SAFETY.md` for detailed documentation
- **CLI Hexdump Mode**: Added `--hex` / `-x` flag to display keys and values in hexadecimal format
  - Classic hexdump format with offset, hex bytes, and ASCII representation
  - Available in: `get`, `keys`, `foreach`, `getbatch` commands
  - Useful for inspecting binary data and debugging
- **CLI Recovery Command**: New `recover` command to salvage valid records from corrupted databases
  - Byte-by-byte scanning for potential valid records
  - CRC verification ensures only valid data is recovered
  - Automatically skips corrupted sections and deleted records
  - Usage: `skv recover corrupted.skv repaired.skv`
  - See `tools/cli/RECOVERY.md` for detailed documentation
- **Example: CRC Verification** (`examples/07-integrity/crc_verification`)
  - Demonstrates automatic CRC checking on active records
  - Shows verification of database integrity

### Changed
- **BREAKING**: Record format changed to include CRC at end of each record
  - Old databases (without CRC) are **not compatible** with this version
  - You must recreate databases or migrate data using backup/restore
  - Record sizes increased by 2-4 bytes per record (for CRC)

### Fixed
- CRC verification is now skipped for deleted records (only type byte changes on delete)
- `readRecord()` now always reads data for CRC verification (no seeking back)

## [0.1.0] - Previous Version

### Features
- Sequential binary file format
- In-memory cache for O(1) lookups
- Thread-safe operations with mutex locks
- Free space reuse for deleted records
- Backup and restore with JSON
- Streaming operations for large values
- Cross-platform support (Linux, macOS, BSD, Windows)
- Command-line tool with 22 commands (before recover was added)
- 142 comprehensive tests
