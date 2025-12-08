package skv

import (
	"fmt"
)

// EncryptionType represents the type of encryption algorithm to use
type EncryptionType int

const (
	// EncryptionNone means no encryption is applied
	EncryptionNone EncryptionType = iota
	// EncryptionAES uses AES encryption (via easyaes library)
	EncryptionAES
	// EncryptionSimpleCipher uses the simplecipher library with custom cipher
	EncryptionSimpleCipher
)

// String returns the string representation of the encryption type
func (e EncryptionType) String() string {
	switch e {
	case EncryptionNone:
		return "none"
	case EncryptionAES:
		return "aes"
	case EncryptionSimpleCipher:
		return "simplecipher"
	default:
		return "unknown"
	}
}

// encryptor is an internal interface for encryption operations
type encryptor interface {
	encrypt(data []byte) ([]byte, error)
	decrypt(data []byte) ([]byte, error)
}

// noEncryption is a no-op encryptor
type noEncryption struct{}

func (n *noEncryption) encrypt(data []byte) ([]byte, error) {
	return data, nil
}

func (n *noEncryption) decrypt(data []byte) ([]byte, error) {
	return data, nil
}

// easyAESEncryptor wraps AES encryption (easyaes library)
type easyAESEncryptor struct {
	key string
}

func (e *easyAESEncryptor) encrypt(data []byte) ([]byte, error) {
	// Import easyaes dynamically to avoid compile errors if not installed
	// For now, we'll use a placeholder that will be replaced with actual import
	return encryptWithAES(string(data), e.key)
}

func (e *easyAESEncryptor) decrypt(data []byte) ([]byte, error) {
	return decryptWithAES(string(data), e.key)
}

// simpleCipherEncryptor wraps the simplecipher library
type simpleCipherEncryptor struct {
	key string
}

func (s *simpleCipherEncryptor) encrypt(data []byte) ([]byte, error) {
	return encryptWithSimpleCipher(string(data), s.key)
}

func (s *simpleCipherEncryptor) decrypt(data []byte) ([]byte, error) {
	return decryptWithSimpleCipher(string(data), s.key)
}

// createEncryptor creates an encryptor based on the type and password
func createEncryptor(encType EncryptionType, password string) (encryptor, error) {
	switch encType {
	case EncryptionNone:
		return &noEncryption{}, nil
	case EncryptionAES:
		if password == "" {
			return nil, fmt.Errorf("password required for AES encryption")
		}
		return &easyAESEncryptor{key: password}, nil
	case EncryptionSimpleCipher:
		if password == "" {
			return nil, fmt.Errorf("password required for SimpleCipher encryption")
		}
		return &simpleCipherEncryptor{key: password}, nil
	default:
		return nil, fmt.Errorf("unknown encryption type: %v", encType)
	}
}
