# File Format Example

Demonstration of the SKV file format with header and version information.

## Overview

This example shows:
- How the file header is structured
- Version information storage
- Backward compatibility with old format files

## File Structure

Every SKV file starts with a 6-byte header:

```
+---+---+---+---+---+---+
| S | K | V | M | m | p |
+---+---+---+---+---+---+
  0   1   2   3   4   5

S K V = Magic bytes (0x53 0x4B 0x56)
M     = Major version (e.g., 0)
m     = Minor version (e.g., 1)
p     = Patch version (e.g., 0)
```

After the header, records follow using the standard SKV record format.

**Important:** All SKV files must have a valid header. Files without the header will be rejected.

## Running the Example

```bash
go run demo.go
```

## Expected Output

```
=== SKV File Format Demo ===

1. File Header Information:
   Magic: SKV
   Version: 0.1.0

2. Database Statistics:
   Total records: 3
   Active records: 3
   Deleted records: 0

3. Stored Data:
   name = John Doe
   email = john@example.com
   city = Barcelona

4. File Size:
   Total size: 78 bytes
   Header: 6 bytes
   Data: 72 bytes

✓ Demo completed successfully!
```

## Version Information

- **Current version:** 0.1.0
- Version is stored as 3 separate bytes (major, minor, patch)
- Future versions can check compatibility based on version numbers

## Related Examples

- **01-basics/** - Basic SKV operations
- **02-advanced/** - Advanced features including Compact
