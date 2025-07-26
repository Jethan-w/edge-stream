// Copyright 2025 EdgeStream Team
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	// EncryptedPrefix 加密数据前缀
	EncryptedPrefix = "ENC("
	// EncryptedSuffix 加密数据后缀
	EncryptedSuffix = ")"
)

// AESEncryptor AES加密器实现
type AESEncryptor struct {
	key []byte
}

// NewAESEncryptor 创建AES加密器
func NewAESEncryptor(password string) *AESEncryptor {
	// 使用SHA256生成32字节密钥
	hash := sha256.Sum256([]byte(password))
	return &AESEncryptor{
		key: hash[:],
	}
}

// Encrypt 加密数据
func (e *AESEncryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// 创建AES cipher
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// 生成随机nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// 加密数据
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Base64编码并添加前缀后缀
	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	return EncryptedPrefix + encoded + EncryptedSuffix, nil
}

// Decrypt 解密数据
func (e *AESEncryptor) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// 检查是否为加密格式
	if !e.IsEncrypted(ciphertext) {
		return ciphertext, nil // 如果不是加密格式，直接返回原文
	}

	// 移除前缀和后缀
	encoded := strings.TrimPrefix(ciphertext, EncryptedPrefix)
	encoded = strings.TrimSuffix(encoded, EncryptedSuffix)

	// Base64解码
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	// 创建AES cipher
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// 创建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// 检查数据长度
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	// 分离nonce和密文
	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]

	// 解密数据
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// IsEncrypted 检查是否为加密数据
func (e *AESEncryptor) IsEncrypted(data string) bool {
	return strings.HasPrefix(data, EncryptedPrefix) && strings.HasSuffix(data, EncryptedSuffix)
}

// EncryptSensitiveFields 加密结构体中的敏感字段
func (e *AESEncryptor) EncryptSensitiveFields(config interface{}) error {
	// 这里可以使用反射来自动加密标记为sensitive的字段
	// 为了简化，这里先返回nil，后续可以扩展
	return nil
}

// DecryptSensitiveFields 解密结构体中的敏感字段
func (e *AESEncryptor) DecryptSensitiveFields(config interface{}) error {
	// 这里可以使用反射来自动解密标记为sensitive的字段
	// 为了简化，这里先返回nil，后续可以扩展
	return nil
}

// GenerateKey 生成随机加密密钥
func GenerateKey() (string, error) {
	key := make([]byte, 32) // 256位密钥
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("failed to generate key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// IsSensitiveKey 检查是否为敏感配置键
func IsSensitiveKey(key string) bool {
	sensitiveKeys := []string{
		"password", "secret", "key", "token", "credential",
		"mysql.password", "postgresql.password", "redis.password",
		"security.encryption_key", "security.jwt_secret",
	}

	key = strings.ToLower(key)
	for _, sensitiveKey := range sensitiveKeys {
		if strings.Contains(key, sensitiveKey) {
			return true
		}
	}
	return false
}
