# Fuzzing Tests

SKV includes comprehensive fuzzing tests to discover edge cases and ensure robustness with random, unexpected, or malformed inputs.

## Available Fuzz Tests

### FuzzPutGet
Tests basic Put/Get operations with random keys and values.

```bash
go test -fuzz=FuzzPutGet -fuzztime=30s
```

**Coverage:**
- Random key/value combinations
- Empty values
- Large values (automatically generated up to memory limits)
- Binary data with special characters

### FuzzUpdate
Tests Update operations with random data transitions.

```bash
go test -fuzz=FuzzUpdate -fuzztime=30s
```

**Coverage:**
- Small to large value updates
- Large to small value updates
- Empty value updates

### FuzzDelete
Tests Delete operations with various key/value patterns.

```bash
go test -fuzz=FuzzDelete -fuzztime=30s
```

**Coverage:**
- Delete after Put
- Verification of ErrKeyNotFound
- Cache consistency after deletion

### FuzzMultipleOperations
Tests random sequences of operations to find state inconsistencies.

```bash
go test -fuzz=FuzzMultipleOperations -fuzztime=1m
```

**Coverage:**
- Random sequences of Put, Update, Delete, Get
- Database consistency validation
- Count verification

### FuzzReopenPersistence
Tests data persistence across database close/reopen cycles.

```bash
go test -fuzz=FuzzReopenPersistence -fuzztime=30s
```

**Coverage:**
- Random data persistence
- Cache reconstruction
- File format integrity

### FuzzCompaction
Tests compaction with random operation sequences.

```bash
go test -fuzz=FuzzCompaction -fuzztime=1m
```

**Coverage:**
- Random Put/Update/Delete sequences
- Compaction correctness
- Post-compaction verification
- Record count consistency

### FuzzBinaryKeys
Tests with binary keys containing special characters.

```bash
go test -fuzz=FuzzBinaryKeys -fuzztime=30s
```

**Coverage:**
- Keys with 0x00 (null byte)
- Keys with 0xFF (all bits set)
- Keys with 0x80 (deleted flag bit)
- Mixed binary data
- Persistence with binary keys

## Running Fuzzing Tests

### Quick Test (5 seconds each)
```bash
go test -fuzz=FuzzPutGet -fuzztime=5s
go test -fuzz=FuzzBinaryKeys -fuzztime=5s
```

### Standard Test (30 seconds each)
```bash
go test -fuzz=FuzzPutGet -fuzztime=30s
go test -fuzz=FuzzUpdate -fuzztime=30s
go test -fuzz=FuzzDelete -fuzztime=30s
go test -fuzz=FuzzReopenPersistence -fuzztime=30s
go test -fuzz=FuzzBinaryKeys -fuzztime=30s
```

### Extended Test (1+ minutes)
```bash
go test -fuzz=FuzzMultipleOperations -fuzztime=2m
go test -fuzz=FuzzCompaction -fuzztime=2m
```

### Continuous Fuzzing (until failure)
```bash
go test -fuzz=FuzzPutGet
```
Press Ctrl+C to stop.

## Corpus

Fuzzing automatically builds a corpus of interesting inputs in `testdata/fuzz/`. These are:
- Automatically saved when new code paths are discovered
- Replayed in regular test runs to prevent regressions
- Committed to version control for continuous testing

## What Fuzzing Discovers

Fuzzing has helped find and prevent:

✅ **Edge Cases**
- Very long keys (near 255 byte limit)
- Empty values
- Binary data with special bytes

✅ **Boundary Conditions**
- Type transitions (Type1Byte → Type2Bytes, etc.)
- Maximum data sizes for each type

✅ **Persistence Issues**
- Data integrity across reopen
- Cache reconstruction correctness

✅ **Concurrency Issues** (when combined with `-race`)
```bash
go test -fuzz=FuzzMultipleOperations -race -fuzztime=1m
```

✅ **Resource Leaks**
- File handle leaks
- Memory leaks with large values

## Interpreting Results

### Success
```
fuzz: elapsed: 3s, execs: 4563 (1521/sec), new interesting: 4 (total: 10)
PASS
```
- **execs**: Total fuzzing executions
- **new interesting**: Inputs that discovered new code paths
- **total**: Total corpus size

### Failure
If fuzzing finds a crash, it will:
1. Show the failure message
2. Save the failing input to `testdata/fuzz/FuzzName/`
3. Create a reproducible test case

Example:
```
--- FAIL: FuzzPutGet (3.45s)
    fuzz_test.go:42: Data mismatch!
    
    Failing input written to testdata/fuzz/FuzzPutGet/abc123
```

You can then reproduce with:
```bash
go test -run=FuzzPutGet/abc123
```

## Best Practices

1. **Run regularly**: Include fuzzing in CI/CD pipelines
2. **Commit corpus**: Check in `testdata/fuzz/` to prevent regressions
3. **Combine with race detector**: `go test -fuzz=Fuzz -race`
4. **Start small**: Begin with 5-10 second runs, extend for deeper testing
5. **Monitor resources**: Long fuzzing runs can use significant CPU/memory

## Integration with CI/CD

Example GitHub Actions workflow:

```yaml
- name: Run fuzzing tests
  run: |
    go test -fuzz=FuzzPutGet -fuzztime=30s
    go test -fuzz=FuzzBinaryKeys -fuzztime=30s
    go test -fuzz=FuzzCompaction -fuzztime=1m
```

## See Also

- [Go Fuzzing Documentation](https://go.dev/security/fuzz/)
- [TESTING.md](TESTING.md) - General testing documentation
- [Test Coverage Report](https://github.com/jncss/skv)
