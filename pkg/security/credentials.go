package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fumiya-kume/cca/pkg/errors"
)

// CredentialStore manages secure storage of credentials
type CredentialStore interface {
	Get(service string) (string, error)
	Set(service string, credential string) error
	Delete(service string) error
	List() ([]string, error)
}

// credentialData represents encrypted credential data
type credentialData struct {
	Service   string `json:"service"`
	Data      string `json:"data"`      // encrypted credential
	Nonce     string `json:"nonce"`     // encryption nonce
	Timestamp int64  `json:"timestamp"` // creation timestamp
}

// credentialStorage represents the storage file structure
type credentialStorage struct {
	Version     string                     `json:"version"`
	Credentials map[string]credentialData  `json:"credentials"`
}

// KeychainStore implements credential storage using system keychain with encrypted file fallback
type KeychainStore struct {
	prefix      string
	mutex       sync.RWMutex
	storageFile string
	encryptKey  []byte
}

// NewKeychainStore creates a new keychain-based credential store
func NewKeychainStore(prefix string) *KeychainStore {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "." // Fallback to current directory
	}
	storageDir := filepath.Join(homeDir, "."+prefix)
	if err := os.MkdirAll(storageDir, 0700); err != nil {
		// Continue with current directory if home dir creation fails
		storageDir = "."
	}
	
	storageFile := filepath.Join(storageDir, "credentials.enc")
	
	// Generate or load encryption key based on machine-specific data
	encryptKey := generateEncryptionKey(prefix)
	
	return &KeychainStore{
		prefix:      prefix,
		storageFile: storageFile,
		encryptKey:  encryptKey,
	}
}

// generateEncryptionKey creates a machine-specific encryption key
func generateEncryptionKey(prefix string) []byte {
	// Use machine-specific information to derive a consistent key
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost" // Fallback hostname
	}
	user := os.Getenv("USER")
	if user == "" {
		user = os.Getenv("USERNAME") // Windows fallback
	}
	if user == "" {
		user = "unknown" // Final fallback
	}
	
	// Create a deterministic key from machine-specific data
	keyData := fmt.Sprintf("%s-%s-%s", prefix, hostname, user)
	hash := sha256.Sum256([]byte(keyData))
	return hash[:]
}

// Get retrieves a credential from the keychain
func (k *KeychainStore) Get(service string) (string, error) {
	k.mutex.RLock()
	defer k.mutex.RUnlock()

	if err := validateServiceName(service); err != nil {
		return "", err
	}

	// Try keychain first (platform-specific implementation would go here)
	// For now, fall back to environment variables
	envKey := fmt.Sprintf("%s_%s_TOKEN", strings.ToUpper(k.prefix), strings.ToUpper(service))
	if value := os.Getenv(envKey); value != "" {
		return value, nil
	}

	// Try common environment variable patterns
	commonPatterns := []string{
		fmt.Sprintf("%s_TOKEN", strings.ToUpper(service)),
		fmt.Sprintf("%s_API_KEY", strings.ToUpper(service)),
		fmt.Sprintf("%s_KEY", strings.ToUpper(service)),
	}

	for _, pattern := range commonPatterns {
		if value := os.Getenv(pattern); value != "" {
			return value, nil
		}
	}
	
	// Try encrypted storage
	if credential, err := k.loadCredential(service); err == nil {
		return credential, nil
	}

	return "", errors.NewError(errors.ErrorTypeAuthentication).
		WithMessage(fmt.Sprintf("credential not found for service '%s'", service)).
		WithSeverity(errors.SeverityMedium).
		WithRecoverable(true).
		WithContext("service", service).
		WithSuggestion(fmt.Sprintf("Set %s environment variable", envKey)).
		WithSuggestion(fmt.Sprintf("Use 'ccagents credential set %s <token>' to store credentials", service)).
		Build()
}

// Set stores a credential in the keychain
func (k *KeychainStore) Set(service string, credential string) error {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	if err := validateServiceName(service); err != nil {
		return err
	}

	if err := validateCredential(credential); err != nil {
		return err
	}

	// Try to store in encrypted file
	if err := k.storeCredential(service, credential); err != nil {
		return fmt.Errorf("failed to store credential: %w", err)
	}
	
	return nil
}

// Delete removes a credential from the keychain
func (k *KeychainStore) Delete(service string) error {
	k.mutex.Lock()
	defer k.mutex.Unlock()

	if err := validateServiceName(service); err != nil {
		return err
	}

	// Try to delete from encrypted file
	if err := k.deleteCredential(service); err != nil {
		return fmt.Errorf("failed to delete credential: %w", err)
	}
	
	return nil
}

// List returns all available credential services
func (k *KeychainStore) List() ([]string, error) {
	k.mutex.RLock()
	defer k.mutex.RUnlock()

	var services []string

	// Check environment variables for known patterns
	knownServices := []string{"github", "claude", "anthropic"}

	for _, service := range knownServices {
		if _, err := k.Get(service); err == nil {
			services = append(services, service)
		}
	}

	return services, nil
}

// validateServiceName ensures service name is safe
func validateServiceName(service string) error {
	if service == "" {
		return errors.ValidationError("service name cannot be empty")
	}

	if len(service) > 50 {
		return errors.ValidationError("service name too long")
	}

	// Allow only alphanumeric characters, hyphens, and underscores
	for _, char := range service {
		if (char < 'a' || char > 'z') &&
			(char < 'A' || char > 'Z') &&
			(char < '0' || char > '9') &&
			char != '-' && char != '_' {
			return errors.ValidationError("service name contains invalid characters")
		}
	}

	return nil
}

// validateCredential ensures credential is safe to store
func validateCredential(credential string) error {
	if credential == "" {
		return errors.ValidationError("credential cannot be empty")
	}

	if len(credential) > 1000 {
		return errors.ValidationError("credential too long")
	}

	// Check for suspicious patterns that might indicate injection attempts
	suspicious := []string{
		";", "|", "&", "$", "`", "$(", "&&", "||",
	}

	for _, pattern := range suspicious {
		if strings.Contains(credential, pattern) {
			return errors.ValidationError("credential contains suspicious characters")
		}
	}

	return nil
}

// CredentialManager provides high-level credential management
type CredentialManager struct {
	store CredentialStore
}

// NewCredentialManager creates a new credential manager
func NewCredentialManager() *CredentialManager {
	return &CredentialManager{
		store: NewKeychainStore("ccagents"),
	}
}

// GetGitHubToken retrieves GitHub authentication token
func (cm *CredentialManager) GetGitHubToken() (string, error) {
	return cm.store.Get("github")
}

// GetClaudeToken retrieves Claude API token
func (cm *CredentialManager) GetClaudeToken() (string, error) {
	return cm.store.Get("claude")
}

// GetAnthropicToken retrieves Anthropic API token
func (cm *CredentialManager) GetAnthropicToken() (string, error) {
	return cm.store.Get("anthropic")
}

// ValidateCredentials checks that required credentials are available
func (cm *CredentialManager) ValidateCredentials() error {
	requiredServices := map[string]string{
		"github": "GitHub integration",
		"claude": "Claude Code integration",
	}

	var missingServices []string

	for service, description := range requiredServices {
		if _, err := cm.store.Get(service); err != nil {
			missingServices = append(missingServices, fmt.Sprintf("%s (%s)", service, description))
		}
	}

	if len(missingServices) > 0 {
		return errors.NewError(errors.ErrorTypeAuthentication).
			WithMessage("missing required credentials").
			WithSeverity(errors.SeverityHigh).
			WithRecoverable(true).
			WithContext("missing_services", missingServices).
			WithSuggestion("Configure required environment variables").
			WithSuggestion("Run authentication setup for missing services").
			Build()
	}

	return nil
}

// loadStorage loads the credential storage from file
func (k *KeychainStore) loadStorage() (*credentialStorage, error) {
	if _, err := os.Stat(k.storageFile); os.IsNotExist(err) {
		// File doesn't exist, return empty storage
		return &credentialStorage{
			Version:     "1.0",
			Credentials: make(map[string]credentialData),
		}, nil
	}
	
	data, err := os.ReadFile(k.storageFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage file: %w", err)
	}
	
	var storage credentialStorage
	if err := json.Unmarshal(data, &storage); err != nil {
		return nil, fmt.Errorf("failed to parse storage file: %w", err)
	}
	
	return &storage, nil
}

// saveStorage saves the credential storage to file
func (k *KeychainStore) saveStorage(storage *credentialStorage) error {
	data, err := json.MarshalIndent(storage, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal storage: %w", err)
	}
	
	// Write with restricted permissions
	if err := os.WriteFile(k.storageFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write storage file: %w", err)
	}
	
	return nil
}

// encryptCredential encrypts a credential using AES-GCM
func (k *KeychainStore) encryptCredential(credential string) (string, string, error) {
	block, err := aes.NewCipher(k.encryptKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to create cipher: %w", err)
	}
	
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", fmt.Errorf("failed to create GCM: %w", err)
	}
	
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	
	encrypted := aesGCM.Seal(nil, nonce, []byte(credential), nil)
	
	return base64.StdEncoding.EncodeToString(encrypted), 
		   base64.StdEncoding.EncodeToString(nonce), nil
}

// decryptCredential decrypts a credential using AES-GCM
func (k *KeychainStore) decryptCredential(encryptedData, nonceStr string) (string, error) {
	block, err := aes.NewCipher(k.encryptKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}
	
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}
	
	encrypted, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", fmt.Errorf("failed to decode encrypted data: %w", err)
	}
	
	nonce, err := base64.StdEncoding.DecodeString(nonceStr)
	if err != nil {
		return "", fmt.Errorf("failed to decode nonce: %w", err)
	}
	
	decrypted, err := aesGCM.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}
	
	return string(decrypted), nil
}

// storeCredential stores an encrypted credential
func (k *KeychainStore) storeCredential(service, credential string) error {
	storage, err := k.loadStorage()
	if err != nil {
		return fmt.Errorf("failed to load storage: %w", err)
	}
	
	encryptedData, nonce, err := k.encryptCredential(credential)
	if err != nil {
		return fmt.Errorf("failed to encrypt credential: %w", err)
	}
	
	storage.Credentials[service] = credentialData{
		Service:   service,
		Data:      encryptedData,
		Nonce:     nonce,
		Timestamp: time.Now().Unix(),
	}
	
	if err := k.saveStorage(storage); err != nil {
		return fmt.Errorf("failed to save storage: %w", err)
	}
	
	return nil
}

// loadCredential loads and decrypts a credential
func (k *KeychainStore) loadCredential(service string) (string, error) {
	storage, err := k.loadStorage()
	if err != nil {
		return "", fmt.Errorf("failed to load storage: %w", err)
	}
	
	credData, exists := storage.Credentials[service]
	if !exists {
		return "", fmt.Errorf("credential not found")
	}
	
	credential, err := k.decryptCredential(credData.Data, credData.Nonce)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt credential: %w", err)
	}
	
	return credential, nil
}

// deleteCredential removes a credential from storage
func (k *KeychainStore) deleteCredential(service string) error {
	storage, err := k.loadStorage()
	if err != nil {
		return fmt.Errorf("failed to load storage: %w", err)
	}
	
	delete(storage.Credentials, service)
	
	if err := k.saveStorage(storage); err != nil {
		return fmt.Errorf("failed to save storage: %w", err)
	}
	
	return nil
}
