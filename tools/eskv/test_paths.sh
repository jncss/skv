#!/bin/bash
echo "=== Test de Sanitització de Paths ==="

# Crear una BD de test
./eskv &
SERVER_PID=$!
sleep 2

echo ""
echo "1. Test amb nom normal: test.skv"
curl -s -X POST http://localhost:9090/api/add \
  -d "filename=test.skv" \
  -d "key=key1" \
  -d "value=value1"

echo ""
echo "2. Test amb path absolut: /workspaces/skv/tools/eskv/data/test2.skv"
curl -s -X POST http://localhost:9090/api/add \
  -d "filename=/workspaces/skv/tools/eskv/data/test2.skv" \
  -d "key=key2" \
  -d "value=value2"

echo ""
echo "3. Test amb path relatiu: ../test3.skv"
curl -s -X POST http://localhost:9090/api/add \
  -d "filename=../test3.skv" \
  -d "key=key3" \
  -d "value=value3"

sleep 1
kill $SERVER_PID 2>/dev/null

echo ""
echo ""
echo "=== Arxius creats en data/ ==="
ls -la data/*.skv 2>/dev/null | awk '{print $9}'

echo ""
echo "=== Buscant directoris estranys ==="
find data -type d

echo ""
echo "✓ Test completat"
