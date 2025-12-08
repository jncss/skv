# Encryption Example

This example demonstrates how to use encryption in SKV to protect sensitive data.

## Features Demonstrated

1. **AES Encryption** - AES encryption with automatic key sizing (via EasyAES)
2. **SimpleCipher Encryption** - Custom cipher algorithm
3. **Combined Encryption + Compression** - Data is encrypted BEFORE compression
4. **Automatic Key/Value Encryption** - Both keys and values are encrypted separately
5. **Base64 Encoding** - Encrypted data uses Base64 encoding

## How It Works

### Encryption Flow (Write)
```
Original Data → Encrypt → Compress (optional) → Write to disk
```

### Decryption Flow (Read)
```
Read from disk → Decompress (if compressed) → Decrypt → Original Data
```

## Key Points

- **No format flags**: The file format doesn't include encryption indicators
- **Password required**: You must use the correct password to decrypt
- **Separate encryption**: Keys and values are encrypted independently
- **Order matters**: Encryption happens BEFORE compression (as specified)

## Running the Example

```bash
cd examples/12-encryption
go run main.go
```

## Output

The example will:
1. Create databases with different encryption methods
2. Store and retrieve encrypted data
3. Demonstrate encryption + compression together
4. Show error handling for wrong passwords
5. Clean up all test files

## API Usage

### Without Encryption
```go
db, err := skv.Open("mydb.skv")
```

### With AES Encryption
```go
opts := &skv.Options{
    Encryption:         skv.EncryptionAES,  // AES encryption
    EncryptionPassword: "my-secret-password",
}
db, err := skv.OpenWithOptions("mydb.skv", opts)
```

### With SimpleCipher Encryption
```go
opts := &skv.Options{
    Encryption:         skv.EncryptionSimpleCipher,
    EncryptionPassword: "my-secret-password",
}
db, err := skv.OpenWithOptions("mydb.skv", opts)
```

### With Encryption + Compression
```go
opts := &skv.Options{
    Encryption:         skv.EncryptionAES,  // AES encryption
    EncryptionPassword: "my-secret-password",
    Compression:        skv.CompressionSnappy,
}
db, err := skv.OpenWithOptions("mydb.skv", opts)
```

## Security Notes

⚠️ **Important**:
- Store passwords securely (environment variables, key vaults, etc.)
- Never hardcode passwords in production code
- Use strong, random passwords
- Different databases should use different passwords
- Without the correct password, data is unrecoverable
- SimpleCipher is NOT recommended for high-security applications (use AES instead)

## Dependencies

This example requires:
- `github.com/jncss/easyaes` - for EasyAES encryption
- `github.com/jncss/simplecipher` - for SimpleCipher encryption

Both are automatically installed when you run `go get` or `go mod tidy`.
