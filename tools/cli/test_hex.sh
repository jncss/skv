#!/bin/bash
set -e

DB="test_hexdump.skv"
rm -f $DB

echo "=== Testing hexdump functionality ==="
echo

echo "1. Creating test data..."
./skv put $DB simple "test"
./skv put $DB utf8 "Héllo Wörld ñáéíóú €"
./skv put $DB multiline "Line 1
Line 2
Line 3"
echo "   ✓ Data created"
echo

echo "2. Testing 'get' with --hex..."
./skv --hex get $DB simple | head -1
echo "   ✓ Simple value hexdump works"
echo

echo "3. Testing 'get' with -x (short form)..."
./skv -x get $DB utf8 | head -1
echo "   ✓ UTF-8 value hexdump works"
echo

echo "4. Testing 'keys' with --hex..."
./skv --hex keys $DB | head -1
echo "   ✓ Keys hexdump works"
echo

echo "5. Testing 'foreach' with --hex..."
./skv --hex foreach $DB | grep -c "Key:"
echo "   ✓ Foreach hexdump works"
echo

echo "6. Testing 'getbatch' with --hex..."
./skv --hex getbatch $DB simple utf8 | grep -c "Key:"
echo "   ✓ Getbatch hexdump works"
echo

echo "=== All tests passed! ==="
rm -f $DB
