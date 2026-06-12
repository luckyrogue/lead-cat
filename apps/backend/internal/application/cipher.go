package application

// Cipher encrypts and decrypts secrets at rest (per-organization Google
// credentials). Implemented in infrastructure/crypto.
type Cipher interface {
	Encrypt(plain string) ([]byte, error)
	Decrypt(enc []byte) (string, error)
}
