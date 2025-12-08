//go:build !noencryption
// +build !noencryption

package skv

import (
	"fmt"

	"github.com/jncss/easyaes"
	"github.com/jncss/simplecipher"
)

// encryptWithAES encrypts data using AES encryption (easyaes library) with Base64 encoding
func encryptWithAES(data, key string) ([]byte, error) {
	// Use Base64 encoding as specified
	encrypted, err := easyaes.EncryptStringB64(key, data)
	if err != nil {
		return nil, fmt.Errorf("AES encryption failed: %w", err)
	}
	return []byte(encrypted), nil
}

// decryptWithAES decrypts data using AES encryption (easyaes library) from Base64 encoding
func decryptWithAES(data, key string) ([]byte, error) {
	decrypted, err := easyaes.DecryptStringB64(key, data)
	if err != nil {
		return nil, fmt.Errorf("AES decryption failed: %w", err)
	}
	return []byte(decrypted), nil
}

// encryptWithSimpleCipher encrypts data using the simplecipher library with Base64 encoding
func encryptWithSimpleCipher(data, key string) ([]byte, error) {
	encrypted := simplecipher.EncryptStringB64(data, key)
	return []byte(encrypted), nil
}

// decryptWithSimpleCipher decrypts data using the simplecipher library from Base64 encoding
func decryptWithSimpleCipher(data, key string) ([]byte, error) {
	decrypted, err := simplecipher.DecryptStringB64(data, key)
	if err != nil {
		return nil, fmt.Errorf("simplecipher decryption failed: %w", err)
	}
	return []byte(decrypted), nil
}
