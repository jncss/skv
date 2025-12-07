#!/bin/bash

# Advanced recovery demonstration
# This script shows how to recover from various corruption scenarios

CLI="./skv"
echo "=== Advanced SKV Recovery Demonstration ==="
echo

# Cleanup
rm -f demo_*.skv recovery_*.json

# Scenario 1: Partial corruption in the middle
echo "Scenario 1: Partial Corruption (simulated disk error)"
echo "======================================================="
echo

# Create a database with many records
echo "Creating database with 20 records..."
for i in {1..20}; do
    $CLI put demo_original.skv "key$i" "Value number $i with some data" > /dev/null
done

echo "✓ Created 20 records"
$CLI count demo_original.skv
echo

# Corrupt bytes 200-250 (middle of file)
cp demo_original.skv demo_corrupted1.skv
dd if=/dev/urandom of=demo_corrupted1.skv bs=1 count=50 seek=200 conv=notrunc 2>/dev/null

echo "Injected corruption at bytes 200-250"
echo

# Try to verify (will fail)
echo "Verifying corrupted database:"
if $CLI verify demo_corrupted1.skv 2>&1 | grep -q "CRC mismatch\|corrupted"; then
    echo "✗ Corruption detected by verify"
else
    echo "⚠ No error detected (corruption may be in deleted records)"
fi
echo

# Recover
echo "Attempting recovery..."
$CLI recover demo_corrupted1.skv demo_recovered1.skv
echo

# Compare
original_count=$($CLI count demo_original.skv)
recovered_count=$($CLI count demo_recovered1.skv)
recovery_rate=$(( recovered_count * 100 / original_count ))

echo "Recovery results:"
echo "  Original records:  $original_count"
echo "  Recovered records: $recovered_count"
echo "  Recovery rate:     ${recovery_rate}%"
echo

# Scenario 2: Truncated file
echo
echo "Scenario 2: Truncated File (simulated incomplete write)"
echo "========================================================"
echo

# Create database and truncate it
$CLI put demo_original2.skv test1 "First value" > /dev/null
$CLI put demo_original2.skv test2 "Second value" > /dev/null
$CLI put demo_original2.skv test3 "Third value" > /dev/null

cp demo_original2.skv demo_truncated.skv
# Truncate to 70% of original size
original_size=$(stat -f%z demo_truncated.skv 2>/dev/null || stat -c%s demo_truncated.skv)
truncated_size=$((original_size * 70 / 100))
dd if=demo_truncated.skv of=demo_truncated_temp.skv bs=1 count=$truncated_size 2>/dev/null
mv demo_truncated_temp.skv demo_truncated.skv

echo "File truncated to 70% of original size"
echo

# Recover
echo "Attempting recovery from truncated file..."
$CLI recover demo_truncated.skv demo_recovered2.skv
echo

# Scenario 3: Multiple corrupted zones
echo
echo "Scenario 3: Multiple Corrupted Zones"
echo "====================================="
echo

# Create larger database
echo "Creating database with 30 records..."
for i in {1..30}; do
    $CLI put demo_original3.skv "record$i" "Data for record $i - some longer content here" > /dev/null
done

echo "✓ Created 30 records"
echo

# Corrupt multiple zones
cp demo_original3.skv demo_corrupted3.skv
dd if=/dev/urandom of=demo_corrupted3.skv bs=1 count=30 seek=100 conv=notrunc 2>/dev/null
dd if=/dev/urandom of=demo_corrupted3.skv bs=1 count=30 seek=300 conv=notrunc 2>/dev/null
dd if=/dev/urandom of=demo_corrupted3.skv bs=1 count=30 seek=500 conv=notrunc 2>/dev/null

echo "Injected corruption at 3 different locations"
echo

# Recover
echo "Attempting recovery..."
$CLI recover demo_corrupted3.skv demo_recovered3.skv
echo

# Final comparison
echo
echo "=== Recovery Summary ==="
echo
echo "Scenario 1 (Partial corruption):"
echo "  Recovery rate: ${recovery_rate}%"
echo

echo "Scenario 2 (Truncated file):"
scenario2_count=$($CLI count demo_recovered2.skv)
echo "  Recovered: $scenario2_count records"
echo

echo "Scenario 3 (Multiple zones):"
original3_count=$($CLI count demo_original3.skv)
recovered3_count=$($CLI count demo_recovered3.skv)
recovery3_rate=$(( recovered3_count * 100 / original3_count ))
echo "  Original:  $original3_count records"
echo "  Recovered: $recovered3_count records"  
echo "  Rate:      ${recovery3_rate}%"
echo

# Cleanup
echo
echo "Cleaning up demo files..."
rm -f demo_*.skv

echo
echo "=== Demonstration Complete ==="
echo
echo "Key Takeaways:"
echo "• Recovery depends on location and extent of corruption"
echo "• CRC verification ensures only valid data is recovered"
echo "• Multiple corruption zones may reduce recovery rate"
echo "• Truncated files can still yield partial recovery"
echo "• Always maintain regular backups for critical data"
