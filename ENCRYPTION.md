# Encryption in SKV

SKV provides optional encryption for both keys and values using two encryption libraries:
- **AES** (via EasyAES): AES encryption with automatic key sizing (recommended for production)
- **SimpleCipher**: Custom lightweight cipher (suitable for obfuscation, not high-security)

## Features

✅ **Transparent Encryption**: Keys and values are encrypted separately  
✅ **No Format Changes**: No encryption flags in file format  
✅ **Works with Compression**: Encryption happens BEFORE compression  
✅ **Base64 Encoding**: Encrypted data uses Base64 encoding  
✅ **Simple API**: Just add password and encryption type to options  

## Quick Start

### Without Encryption (Default)
```go
db, err := skv.Open("mydb.skv")
```

### With AES Encryption
```go
opts := &skv.Options{
    Encryption:         skv.EncryptionAES,
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

## Encryption + Compression

Encryption and compression work together seamlessly. The order is:

**Write**: `Original Data → Encrypt → Compress → Write to disk`  
**Read**: `Read from disk → Decompress → Decrypt → Original Data`

```go
opts := &skv.Options{
    Encryption:         skv.EncryptionAES,
    EncryptionPassword: "my-secret-password",
    Compression:        skv.CompressionSnappy,  // or CompressionLZ4
}
db, err := skv.OpenWithOptions("mydb.skv", opts)
```

## Encryption Types

### EncryptionNone (Default)
No encryption is applied. Data is stored in plaintext.

### EncryptionAES (Recommended)
- Uses **AES encryption** in CFB mode
- Automatic key sizing (128/192/256 bits)
- Random initialization vector (IV)
- Base64 encoding
- **Best for**: Production applications requiring real security

### EncryptionSimpleCipher
- Custom XOR-based cipher with FNV-1a hash
- Lightweight and fast
- Base64 encoding
- **Best for**: Data obfuscation, testing, learning
- **⚠️ Warning**: Not suitable for high-security applications

## How It Works

### Key and Value Encryption

Both keys and values are encrypted separately using the same password:

```go
// Simplified internal flow
encryptedKey   := encrypt(key, password)
encryptedValue := encrypt(value, password)
// Store: [encryptedKey, encryptedValue]
```

### Encryption Size Overhead

Due to Base64 encoding and encryption:
- **AES**: ~33% size increase (Base64) + IV overhead
- **SimpleCipher**: ~33% size increase (Base64) + IV overhead

### Encrypted Key Size Limit

The maximum key size is **255 bytes** (after encryption). Since encryption expands data:
- Original keys should be kept reasonably small (< 150 bytes recommended)
- The library will return an error if encrypted key exceeds 255 bytes

## Security Considerations

### AES via EasyAES (Recommended for Production)

✅ **Use for**:
- Sensitive personal data
- Financial information
- API keys and tokens
- Production databases

**Best Practices**:
- Use strong, random passwords (16+ characters)
- Store passwords securely (environment variables, key vaults)
- Use different passwords for different databases
- Never hardcode passwords in source code

### SimpleCipher (Use with Caution)

⚠️ **Warning**: This is a custom encryption algorithm that has NOT been audited by security experts.

✅ **Suitable for**:
- Learning and experimentation
- Simple data obfuscation
- Non-sensitive testing data
- Personal projects

❌ **NOT suitable for**:
- Sensitive production data
- Personal or financial information
- Compliance requirements (GDPR, HIPAA, etc.)
- Any application requiring security guarantees

## Examples

### Basic Usage

```go
package main

import (
    "log"
    "github.com/jncss/skv"
)

func main() {
    opts := &skv.Options{
        Encryption:         skv.EncryptionAES,  // AES encryption
        EncryptionPassword: "super-secret-password",
    }
    
    db, err := skv.OpenWithOptions("secure.skv", opts)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // Store encrypted data
    db.Put([]byte("username"), []byte("alice"))
    db.Put([]byte("password"), []byte("secret123"))
    
    // Retrieve decrypted data
    username, _ := db.Get([]byte("username"))
    log.Printf("Username: %s", username)
}
```

### With Environment Variable Password

```go
import "os"

password := os.Getenv("DB_PASSWORD")
if password == "" {
    log.Fatal("DB_PASSWORD environment variable not set")
}

opts := &skv.Options{
    Encryption:         skv.EncryptionAES,
    EncryptionPassword: password,
}

db, err := skv.OpenWithOptions("secure.skv", opts)
```

### Multiple Databases with Different Passwords

```go
// User database
userOpts := &skv.Options{
    Encryption:         skv.EncryptionAES,
    EncryptionPassword: os.Getenv("USER_DB_PASSWORD"),
}
userDB, _ := skv.OpenWithOptions("users.skv", userOpts)

// Config database  
configOpts := &skv.Options{
    Encryption:         skv.EncryptionAES,
    EncryptionPassword: os.Getenv("CONFIG_DB_PASSWORD"),
}
configDB, _ := skv.OpenWithOptions("config.skv", configOpts)
```

## Backup and Restore with Encryption

**IMPORTANT**: Backup files preserve encryption state!

### How It Works

When you backup an encrypted database:
- ✅ Keys and values remain **encrypted** in the JSON file
- ✅ Encrypted data is encoded as Base64 in the JSON
- ✅ **Nobody can read the backup without the password**
- ✅ The backup file is secure even if stolen

Example of encrypted backup content:
```json
[
  {
    "key": "OQ_DJIsZ2TXU2tUcNb-NANk6ATE=",
    "value": "notqYE46bmB9po-cSPShhFR0VcvEqO9_LxvQCu8I4vbazkt8DccX",
    "is_binary": false
  }
]
```
☝️ This is encrypted data - completely unreadable without the password!

### Creating Encrypted Backups

```go
// Open encrypted database
opts := &skv.Options{
    Encryption:         skv.EncryptionAES,
    EncryptionPassword: "secret123",
}
db, _ := skv.OpenWithOptions("secure.skv", opts)

// Put some data
db.Put([]byte("api_key"), []byte("sk_live_abc123..."))
db.Put([]byte("token"), []byte("bearer_xyz..."))

// Create backup - data stays encrypted!
db.Backup("backup.json")
db.Close()
```

The `backup.json` file contains **encrypted data only** - safe to store or transfer.

### Restoring Encrypted Backups

To restore, you **must use the same encryption password**:

```go
// Open database with SAME password
opts := &skv.Options{
    Encryption:         skv.EncryptionAES,
    EncryptionPassword: "secret123",  // Must match!
}
db, _ := skv.OpenWithOptions("restored.skv", opts)

// Restore from backup
db.Restore("backup.json")

// Data is now available (decrypted)
apiKey, _ := db.Get([]byte("api_key"))
fmt.Println(string(apiKey))  // sk_live_abc123...
```

### Security Notes

✅ **Backup files are secure**: Encrypted data stays encrypted in JSON  
✅ **Password required**: Cannot restore without the correct password  
✅ **No plaintext exposure**: Sensitive data never appears in backup files  
⚠️ **Same password required**: Backup and restore must use the same password  
⚠️ **Store backups safely**: While encrypted, still protect backup files  

### Wrong Password During Restore

If you try to restore with the wrong password:

```go
// Wrong password!
wrongOpts := &skv.Options{
    Encryption:         skv.EncryptionAES,
    EncryptionPassword: "WRONG-PASSWORD",
}
db, _ := skv.OpenWithOptions("db.skv", wrongOpts)

// Restore will succeed, but data will be corrupted
db.Restore("backup.json")

// Trying to read will fail or return garbage
value, err := db.Get([]byte("api_key"))
// Either error or unreadable bytes
```

### Best Practices

1. **Same password for backup and restore**: Use identical encryption settings
2. **Test restores**: Verify backups can be restored successfully
3. **Secure backup storage**: Even encrypted backups should be protected
4. **Document passwords**: Keep secure record of which password was used
5. **Regular backups**: Encrypted databases can become unreadable if corrupted

## Error Handling

### Wrong Password

If you open a database with the wrong password, you'll get decryption errors:

```go
// Wrong password
wrongOpts := &skv.Options{
    Encryption:         skv.EncryptionAES,
    EncryptionPassword: "WRONG-PASSWORD",
}

db, err := skv.OpenWithOptions("encrypted.skv", wrongOpts)
// Database opens successfully (no way to verify password at open time)

// But operations will fail:
value, err := db.Get([]byte("key"))
// err will contain decryption error
```

### No Password Provided

```go
opts := &skv.Options{
    Encryption:         skv.EncryptionAES,
    EncryptionPassword: "", // Empty!
}

db, err := skv.OpenWithOptions("db.skv", opts)
// Returns error: "password required for EasyAES encryption"
```

## File Format

**Important**: The SKV file format does NOT include encryption indicators or flags.

This means:
- ✅ Encrypted data looks like regular data
- ✅ No metadata about encryption in the file
- ✅ You must remember which database is encrypted and with which password
- ⚠️ Opening with wrong password won't fail until you try to read/write

## Performance

Encryption adds minimal overhead:
- **AES**: ~5-10% performance impact
- **SimpleCipher**: ~2-5% performance impact

Combined with compression:
- Encryption is applied BEFORE compression
- Compressed encrypted data is typically larger than compressed plaintext
- Overall: encryption first, then compression is the recommended order

## Testing

Run encryption tests:

```bash
go test -v -run TestEncryption
```

## Dependencies

```go
require (
    github.com/jncss/easyaes v0.0.0-20251208190620-9743bf4abb45
    github.com/jncss/simplecipher v1.0.0
)
```

Install with:
```bash
go get github.com/jncss/easyaes
go get github.com/jncss/simplecipher
```

## See Also

- [EasyAES Documentation](https://github.com/jncss/easyaes)
- [SimpleCipher Documentation](https://github.com/jncss/simplecipher)
- [Example: 12-encryption](../examples/12-encryption/)
- [Compression Documentation](./COMPRESSION.md)

## FAQ

**Q: Can I change the encryption password later?**  
A: No. You need to create a new database with the new password and copy the data.

**Q: Can I mix encrypted and non-encrypted data?**  
A: No. All data in a database is either encrypted or not, based on the Options used to open it.

**Q: What happens if I forget the password?**  
A: The data is unrecoverable. Always backup your passwords securely.

**Q: Can I switch between AES and SimpleCipher?**  
A: No. Once a database is created with one encryption type, you must continue using the same type.

**Q: Does encryption work with transactions?**  
A: Yes. Encryption is transparent to all SKV operations including transactions.

**Q: Does encryption work with indexes?**  
A: Yes. Keys in indexes are also encrypted.

**Q: Does encryption work with WAL?**  
A: Yes. WAL entries are also encrypted.

**Q: Is there a performance difference between encryption types?**  
A: SimpleCipher is slightly faster but less secure. Use AES (EncryptionAES) for production.
