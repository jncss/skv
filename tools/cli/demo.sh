#!/bin/bash

# SKV CLI Demo Script
# This script demonstrates all the features of the SKV CLI

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== SKV CLI Demo ===${NC}\n"

# Cleanup
rm -rf demo_data
mkdir -p demo_data

DB="demo_data/example.skv"

echo -e "${GREEN}1. Basic Operations${NC}"
echo "$ skv put $DB username 'alice'"
./skv put $DB username 'alice'

echo "$ skv put $DB email 'alice@example.com'"
./skv put $DB email 'alice@example.com'

echo "$ skv put $DB role 'admin'"
./skv put $DB role 'admin'

echo ""
echo "$ skv get $DB username"
./skv get $DB username
echo ""

echo "$ skv update $DB role 'superadmin'"
./skv update $DB role 'superadmin'

echo "$ skv exists $DB username"
./skv exists $DB username

echo "$ skv count $DB"
./skv count $DB

echo ""
echo -e "${GREEN}2. List and Iterate${NC}"
echo "$ skv keys $DB"
./skv keys $DB

echo ""
echo "$ skv foreach $DB"
./skv foreach $DB

echo ""
echo -e "${GREEN}3. File Operations${NC}"
echo "Creating test files..."
echo "Configuration data" > demo_data/config.txt
echo "Logs from application" > demo_data/app.log

echo "$ skv putfile $DB config demo_data/config.txt"
./skv putfile $DB config demo_data/config.txt

echo "$ skv putfile $DB logs demo_data/app.log"
./skv putfile $DB logs demo_data/app.log

echo "$ skv getfile $DB config demo_data/retrieved_config.txt"
./skv getfile $DB config demo_data/retrieved_config.txt

echo "Retrieved content:"
cat demo_data/retrieved_config.txt

echo ""
echo -e "${GREEN}4. Batch Operations${NC}"
echo "$ skv putbatch $DB setting1 'value1' setting2 'value2' setting3 'value3'"
./skv putbatch $DB setting1 'value1' setting2 'value2' setting3 'value3'

echo "$ skv getbatch $DB setting1 setting2 setting3"
./skv getbatch $DB setting1 setting2 setting3

echo ""
echo -e "${GREEN}5. Streaming Operations (for large files)${NC}"
# Create a larger file
dd if=/dev/urandom of=demo_data/large.bin bs=1024 count=100 2>/dev/null

echo "$ skv putstream $DB largefile demo_data/large.bin"
./skv putstream $DB largefile demo_data/large.bin

echo "$ skv getstream $DB largefile demo_data/retrieved_large.bin"
./skv getstream $DB largefile demo_data/retrieved_large.bin

echo "Verifying file size..."
ls -lh demo_data/large.bin demo_data/retrieved_large.bin | awk '{print $5, $9}'

echo ""
echo -e "${GREEN}6. Database Statistics${NC}"
echo "$ skv verify $DB"
./skv verify $DB

echo ""
echo -e "${GREEN}7. Backup & Restore${NC}"
echo "$ skv backup $DB demo_data/backup.json"
./skv backup $DB demo_data/backup.json

echo "Backup content (first few entries):"
head -20 demo_data/backup.json

echo ""
echo "$ skv clear $DB"
./skv clear $DB

echo "$ skv count $DB (after clear)"
./skv count $DB

echo "$ skv restore $DB demo_data/backup.json"
./skv restore $DB demo_data/backup.json

echo "$ skv count $DB (after restore)"
./skv count $DB

echo ""
echo -e "${GREEN}8. Cleanup and Compact${NC}"
echo "$ skv delete $DB setting1"
./skv delete $DB setting1

echo "$ skv delete $DB setting2"
./skv delete $DB setting2

echo "$ skv verify $DB (before compact)"
./skv verify $DB | head -15

echo ""
echo "$ skv compact $DB"
./skv compact $DB

echo ""
echo -e "${BLUE}=== Demo Complete! ===${NC}"
echo ""
echo "Database file: $DB"
echo "Test data: demo_data/"
echo ""
echo "Try more commands:"
echo "  ./skv help"
echo "  ./skv foreach $DB"
echo ""
