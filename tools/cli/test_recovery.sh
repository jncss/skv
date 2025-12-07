#!/bin/bash

# Test script for the recover command

set -e

CLI="./skv"
TEST_DB="test_recovery.skv"
CORRUPTED_DB="test_corrupted.skv"
RECOVERED_DB="test_recovered.skv"

echo "=== SKV Recovery Test ==="
echo

# Clean up any existing test files
rm -f "$TEST_DB" "$CORRUPTED_DB" "$RECOVERED_DB"

# 1. Create a test database with some data
echo "1. Creating test database with sample data..."
$CLI put "$TEST_DB" user:1 "Alice Johnson"
$CLI put "$TEST_DB" user:2 "Bob Smith"
$CLI put "$TEST_DB" user:3 "Charlie Brown"
$CLI put "$TEST_DB" config:host "localhost"
$CLI put "$TEST_DB" config:port "8080"
$CLI put "$TEST_DB" data:large "$(printf 'X%.0s' {1..1000})"  # 1000 bytes

echo "✓ Created database with 6 records"
$CLI count "$TEST_DB"
echo

# 2. Make a copy and corrupt it
echo "2. Corrupting database..."
cp "$TEST_DB" "$CORRUPTED_DB"

# Corrupt by writing random bytes in the middle
# This simulates disk corruption
dd if=/dev/urandom of="$CORRUPTED_DB" bs=1 count=50 seek=100 conv=notrunc 2>/dev/null

echo "✓ Injected 50 random bytes at position 100"
echo

# 3. Try to verify corrupted database (should fail)
echo "3. Verifying corrupted database (should fail)..."
if $CLI verify "$CORRUPTED_DB" 2>&1 | grep -q "CRC mismatch\|corrupted"; then
    echo "✓ Corruption detected as expected"
else
    echo "⚠ Warning: Corruption not detected in verify"
fi
echo

# 4. Recover the database
echo "4. Recovering valid records from corrupted database..."
$CLI recover "$CORRUPTED_DB" "$RECOVERED_DB"
echo

# 5. Verify recovered database
echo "5. Verifying recovered database..."
$CLI verify "$RECOVERED_DB"
echo

# 6. Check which records were recovered
echo "6. Checking recovered records..."
echo "Records in recovered database:"
$CLI keys "$RECOVERED_DB"
echo

# 7. Try to read some values
echo "7. Testing recovered values:"
for key in "user:2" "user:3" "config:host" "config:port"; do
    if value=$($CLI get "$RECOVERED_DB" "$key" 2>/dev/null); then
        echo "  ✓ $key = $value"
    else
        echo "  ✗ $key (not recovered)"
    fi
done
echo

# 8. Compare sizes
echo "8. File size comparison:"
ls -lh "$TEST_DB" "$CORRUPTED_DB" "$RECOVERED_DB" | awk '{print "  " $9 ": " $5}'
echo

# Clean up
echo "Cleaning up test files..."
rm -f "$TEST_DB" "$CORRUPTED_DB" "$RECOVERED_DB"

echo "=== Test Complete ==="
