package application

type Cipher interface {
	Encrypt(plain string) ([]byte, error)
	Decrypt(enc []byte) (string, error)
}
