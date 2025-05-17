package aesutil

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
)

// AES加密类型
const (
	AES_CBC = "CBC"
	AES_GCM = "GCM"
)

// Encrypt 使用AES加密数据
// mode: CBC或GCM
// key: 密钥，长度必须为16(AES-128)、24(AES-192)或32(AES-256)字节
// plaintext: 要加密的明文
func Encrypt(mode string, keytext string, plaintext []byte) ([]byte, error) {
	key := generateKey(keytext)
	switch mode {
	case AES_CBC:
		return encryptCBC(key, plaintext)
	case AES_GCM:
		return encryptGCM(key, plaintext)
	default:
		return nil, errors.New("unsupported AES mode")
	}
}

// Decrypt 使用AES解密数据
// mode: CBC或GCM
// key: 密钥，长度必须为16(AES-128)、24(AES-192)或32(AES-256)字节
// ciphertext: 要解密的密文
func Decrypt(mode string, keytext string, ciphertext []byte) ([]byte, error) {
	key := generateKey(keytext)
	switch mode {
	case AES_CBC:
		return decryptCBC(key, ciphertext)
	case AES_GCM:
		return decryptGCM(key, ciphertext)
	default:
		return nil, errors.New("unsupported AES mode")
	}
}

// CBC模式加密
func encryptCBC(key []byte, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 填充明文以满足块大小
	plaintext = pkcs7Pad(plaintext, aes.BlockSize)

	// IV需要是唯一的，但不一定是安全的
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], plaintext)

	return ciphertext, nil
}

// CBC模式解密
func decryptCBC(key []byte, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	// CBC模式总是工作在完整的块上
	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, errors.New("ciphertext is not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)

	// 去除填充
	ciphertext = pkcs7Unpad(ciphertext)

	return ciphertext, nil
}

// GCM模式加密
func encryptGCM(key []byte, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// GCM模式解密
func decryptGCM(key []byte, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// PKCS7填充
func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padtext...)
}

// PKCS7去除填充
func pkcs7Unpad(data []byte) []byte {
	length := len(data)
	unpadding := int(data[length-1])
	return data[:(length - unpadding)]
}

// EncryptToBase64 加密并返回base64编码字符串
func EncryptToBase64(mode string, key string, plaintext []byte) (string, error) {
	ciphertext, err := Encrypt(mode, key, plaintext)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptFromBase64 从base64解码并解密
func DecryptFromBase64(mode string, key string, base64Str string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return nil, err
	}
	return Decrypt(mode, key, ciphertext)
}

// EncryptToHex 加密并返回hex编码字符串
func EncryptToHex(mode string, key string, plaintext []byte) (string, error) {
	ciphertext, err := Encrypt(mode, key, plaintext)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(ciphertext), nil
}

// DecryptFromHex 从hex解码并解密
func DecryptFromHex(mode string, key string, hexStr string) ([]byte, error) {
	ciphertext, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, err
	}
	return Decrypt(mode, key, ciphertext)
}
func generateKey(key string) []byte {
	hash := sha256.Sum256([]byte(key))
	return hash[:32]
}
