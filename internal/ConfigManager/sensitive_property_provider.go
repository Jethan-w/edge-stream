package ConfigManager

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// SensitivePropertyProvider 敏感属性提供者接口
type SensitivePropertyProvider interface {
	// Encrypt 加密敏感属性
	Encrypt(plaintext string) (string, error)

	// Decrypt 解密敏感属性
	Decrypt(ciphertext string) (string, error)

	// GetAlgorithm 获取加密算法
	GetAlgorithm() string
}

// EncryptionAlgorithm 加密算法枚举
type EncryptionAlgorithm string

const (
	AESGCM256 EncryptionAlgorithm = "AES/GCM/NoPadding"
	AESCBC256 EncryptionAlgorithm = "AES/CBC/PKCS5Padding"
	RSA2048   EncryptionAlgorithm = "RSA/ECB/PKCS1Padding"
	PBKDF2    EncryptionAlgorithm = "PBKDF2WithHmacSHA256"
)

// AESSensitivePropertyProvider AES-GCM 加密实现
type AESSensitivePropertyProvider struct {
	secretKey []byte
	algorithm EncryptionAlgorithm
}

// NewAESSensitivePropertyProvider 创建新的AES敏感属性提供者
func NewAESSensitivePropertyProvider(secretKey []byte) *AESSensitivePropertyProvider {
	return &AESSensitivePropertyProvider{
		secretKey: secretKey,
		algorithm: AESGCM256,
	}
}

// Encrypt 加密敏感属性
func (a *AESSensitivePropertyProvider) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// 创建AES cipher
	block, err := aes.NewCipher(a.secretKey)
	if err != nil {
		return "", fmt.Errorf("创建AES cipher失败: %w", err)
	}

	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM模式失败: %w", err)
	}

	// 生成随机nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("生成随机nonce失败: %w", err)
	}

	// 加密
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// 返回Base64编码的结果
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密敏感属性
func (a *AESSensitivePropertyProvider) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// Base64解码
	encryptedData, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("Base64解码失败: %w", err)
	}

	// 创建AES cipher
	block, err := aes.NewCipher(a.secretKey)
	if err != nil {
		return "", fmt.Errorf("创建AES cipher失败: %w", err)
	}

	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM模式失败: %w", err)
	}

	// 检查数据长度
	nonceSize := gcm.NonceSize()
	if len(encryptedData) < nonceSize {
		return "", fmt.Errorf("加密数据长度不足")
	}

	// 分离nonce和密文
	nonce, ciphertextBytes := encryptedData[:nonceSize], encryptedData[nonceSize:]

	// 解密
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("解密失败: %w", err)
	}

	return string(plaintext), nil
}

// GetAlgorithm 获取加密算法
func (a *AESSensitivePropertyProvider) GetAlgorithm() string {
	return string(a.algorithm)
}

// KeyStoreManager 密钥管理器
type KeyStoreManager struct {
	keystorePath string
	keyAlias     string
	password     string
}

// NewKeyStoreManager 创建新的密钥管理器
func NewKeyStoreManager(keystorePath, keyAlias, password string) *KeyStoreManager {
	return &KeyStoreManager{
		keystorePath: keystorePath,
		keyAlias:     keyAlias,
		password:     password,
	}
}

// GenerateOrLoadKey 生成或加载密钥
func (ksm *KeyStoreManager) GenerateOrLoadKey() ([]byte, error) {
	// 这里简化实现，实际应该使用PKCS12或JKS格式的密钥库
	// 为了演示，我们使用基于密码的密钥派生

	if ksm.password == "" {
		ksm.password = "nifi-secret"
	}

	// 使用SHA256哈希密码作为密钥
	hash := sha256.Sum256([]byte(ksm.password))
	return hash[:], nil
}

// EncryptionException 加密异常
type EncryptionException struct {
	Message string
	Cause   error
}

func (e *EncryptionException) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("加密异常: %s, 原因: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("加密异常: %s", e.Message)
}

// DecryptionException 解密异常
type DecryptionException struct {
	Message string
	Cause   error
}

func (e *DecryptionException) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("解密异常: %s, 原因: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("解密异常: %s", e.Message)
}

// KeyManagementException 密钥管理异常
type KeyManagementException struct {
	Message string
	Cause   error
}

func (e *KeyManagementException) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("密钥管理异常: %s, 原因: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("密钥管理异常: %s", e.Message)
}
