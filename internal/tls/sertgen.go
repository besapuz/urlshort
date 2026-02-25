package tls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// GenerateSelfSignedCert генерирует самоподписанный сертификат для разработки
// Возвращает пути к сгенерированным файлам сертификата и ключа
func GenerateSelfSignedCert(certFile, keyFile string) error {
	// Создаем директорию для сертификатов если её нет
	certDir := filepath.Dir(certFile)
	if certDir != "." && certDir != "" {
		if err := os.MkdirAll(certDir, 0755); err != nil {
			return fmt.Errorf("failed to create cert directory: %w", err)
		}
	}

	// Генерируем приватный ключ
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Создаем шаблон сертификата
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "localhost",
			Organization: []string{"URL Shortener Dev"},
			Country:      []string{"RU"},
			Province:     []string{"Moscow"},
			Locality:     []string{"Moscow"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour), // Действителен 1 год

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,

		DNSNames: []string{
			"localhost",
			"*.localhost",
			"127.0.0.1",
		},
		IPAddresses: []net.IP{
			net.ParseIP("127.0.0.1"),
			net.ParseIP("::1"),
		},
	}

	// Создаем самоподписанный сертификат
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	// Сохраняем сертификат в файл
	certOut, err := os.Create(certFile)
	if err != nil {
		return fmt.Errorf("failed to open cert file for writing: %w", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return fmt.Errorf("failed to write cert to file: %w", err)
	}

	// Сохраняем приватный ключ в файл
	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open key file for writing: %w", err)
	}
	defer keyOut.Close()

	privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyBytes}); err != nil {
		return fmt.Errorf("failed to write key to file: %w", err)
	}

	fmt.Printf("✅ Generated self-signed certificate:\n")
	fmt.Printf("   Certificate: %s\n", certFile)
	fmt.Printf("   Private key: %s\n", keyFile)
	fmt.Printf("   Valid until: %s\n", template.NotAfter.Format(time.RFC3339))

	return nil
}

// CheckCertFiles проверяет существование файлов сертификатов
func CheckCertFiles(certFile, keyFile string) bool {
	certExists := fileExists(certFile)
	keyExists := fileExists(keyFile)
	return certExists && keyExists
}

// EnsureCertificates проверяет наличие сертификатов и генерирует их при необходимости
func EnsureCertificates(certFile, keyFile string) error {
	if CheckCertFiles(certFile, keyFile) {
		fmt.Printf("✅ Using existing certificates:\n")
		fmt.Printf("   Certificate: %s\n", certFile)
		fmt.Printf("   Private key: %s\n", keyFile)
		return nil
	}

	fmt.Printf("🔐 Certificate files not found, generating self-signed certificates...\n")
	return GenerateSelfSignedCert(certFile, keyFile)
}

// fileExists проверяет существование файла
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
