package tls

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGenerateSelfSignedCert(t *testing.T) {
	// Создаем временную директорию для тестов
	tempDir, err := os.MkdirTemp("", "cert-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Для теста с ошибкой создадим директорию, в которой нельзя создать файлы
	// Но в современных ОС это сложно сделать без специальных прав
	// Поэтому будем использовать несуществующий путь с некорректными символами

	tests := []struct {
		name     string
		certFile string
		keyFile  string
		wantErr  bool
	}{
		{
			name:     "Успешная генерация в существующей директории",
			certFile: filepath.Join(tempDir, "server.crt"),
			keyFile:  filepath.Join(tempDir, "server.key"),
			wantErr:  false,
		},
		{
			name:     "Успешная генерация с созданием поддиректории",
			certFile: filepath.Join(tempDir, "certs", "server.crt"),
			keyFile:  filepath.Join(tempDir, "certs", "server.key"),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GenerateSelfSignedCert(tt.certFile, tt.keyFile)

			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateSelfSignedCert() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Проверяем, что файлы созданы
				if !fileExists(tt.certFile) {
					t.Errorf("Certificate file was not created: %s", tt.certFile)
				}
				if !fileExists(tt.keyFile) {
					t.Errorf("Key file was not created: %s", tt.keyFile)
				}

				// Проверяем права доступа к ключевому файлу
				info, err := os.Stat(tt.keyFile)
				if err != nil {
					t.Errorf("Failed to stat key file: %v", err)
				} else if info.Mode().Perm() != 0600 {
					t.Errorf("Key file has wrong permissions: got %v, want 0600", info.Mode().Perm())
				}

				// Проверяем, что сертификат валидный
				validateCertificate(t, tt.certFile)
			}
		})
	}
}

func TestCheckCertFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cert-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	certFile := filepath.Join(tempDir, "test.crt")
	keyFile := filepath.Join(tempDir, "test.key")

	tests := []struct {
		name     string
		setup    func()
		certFile string
		keyFile  string
		want     bool
	}{
		{
			name:     "Оба файла не существуют",
			setup:    func() {},
			certFile: certFile,
			keyFile:  keyFile,
			want:     false,
		},
		{
			name: "Существует только сертификат",
			setup: func() {
				os.WriteFile(certFile, []byte("test"), 0644)
				os.Remove(keyFile)
			},
			certFile: certFile,
			keyFile:  keyFile,
			want:     false,
		},
		{
			name: "Существует только ключ",
			setup: func() {
				os.Remove(certFile)
				os.WriteFile(keyFile, []byte("test"), 0600)
			},
			certFile: certFile,
			keyFile:  keyFile,
			want:     false,
		},
		{
			name: "Оба файла существуют",
			setup: func() {
				os.WriteFile(certFile, []byte("test"), 0644)
				os.WriteFile(keyFile, []byte("test"), 0600)
			},
			certFile: certFile,
			keyFile:  keyFile,
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			got := CheckCertFiles(tt.certFile, tt.keyFile)
			if got != tt.want {
				t.Errorf("CheckCertFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnsureCertificates(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cert-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	certFile := filepath.Join(tempDir, "server.crt")
	keyFile := filepath.Join(tempDir, "server.key")

	t.Run("Генерация при отсутствии файлов", func(t *testing.T) {
		// Убеждаемся, что файлов нет
		os.Remove(certFile)
		os.Remove(keyFile)

		err := EnsureCertificates(certFile, keyFile)
		if err != nil {
			t.Errorf("EnsureCertificates failed: %v", err)
		}

		// Проверяем, что файлы созданы
		if !CheckCertFiles(certFile, keyFile) {
			t.Error("Certificates were not generated")
		}
	})

	t.Run("Использование существующих файлов", func(t *testing.T) {
		// Первая генерация
		err := EnsureCertificates(certFile, keyFile)
		if err != nil {
			t.Fatal(err)
		}

		// Получаем информацию о первом сертификате
		firstCertInfo, err := os.Stat(certFile)
		if err != nil {
			t.Fatal(err)
		}
		firstKeyInfo, err := os.Stat(keyFile)
		if err != nil {
			t.Fatal(err)
		}

		// Небольшая задержка, чтобы время модификации точно изменилось если файл будет перезаписан
		time.Sleep(10 * time.Millisecond)

		// Второй вызов - должен использовать существующие файлы
		err = EnsureCertificates(certFile, keyFile)
		if err != nil {
			t.Errorf("EnsureCertificates failed on second call: %v", err)
		}

		// Проверяем, что файлы не были перезаписаны
		secondCertInfo, err := os.Stat(certFile)
		if err != nil {
			t.Fatal(err)
		}
		secondKeyInfo, err := os.Stat(keyFile)
		if err != nil {
			t.Fatal(err)
		}

		if !firstCertInfo.ModTime().Equal(secondCertInfo.ModTime()) {
			t.Error("Certificate file was modified")
		}
		if !firstKeyInfo.ModTime().Equal(secondKeyInfo.ModTime()) {
			t.Error("Key file was modified")
		}
	})
}

func TestFileExists(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cert-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	testDir := filepath.Join(tempDir, "testdir")

	tests := []struct {
		name  string
		setup func()
		file  string
		want  bool
	}{
		{
			name:  "Файл не существует",
			setup: func() {},
			file:  testFile,
			want:  false,
		},
		{
			name: "Файл существует",
			setup: func() {
				os.WriteFile(testFile, []byte("test"), 0644)
			},
			file: testFile,
			want: true,
		},
		{
			name: "Путь является директорией",
			setup: func() {
				os.MkdirAll(testDir, 0755)
			},
			file: testDir,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			got := fileExists(tt.file)
			if got != tt.want {
				t.Errorf("fileExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Вспомогательная функция для валидации сертификата
func validateCertificate(t *testing.T, certFile string) {
	// Читаем файл сертификата
	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		t.Fatalf("Failed to read certificate file: %v", err)
	}

	// Декодируем PEM блок
	block, _ := pem.Decode(certPEM)
	if block == nil {
		t.Fatal("Failed to decode PEM block")
	}

	// Парсим сертификат
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("Failed to parse certificate: %v", err)
	}

	// Проверяем основные поля сертификата
	if cert.Subject.CommonName != "localhost" {
		t.Errorf("Expected CommonName 'localhost', got '%s'", cert.Subject.CommonName)
	}

	// Проверяем срок действия
	now := time.Now()
	if now.Before(cert.NotBefore) {
		t.Error("Certificate is not yet valid")
	}
	if now.After(cert.NotAfter) {
		t.Error("Certificate has expired")
	}

	// Проверяем DNS имена
	expectedDNS := map[string]bool{
		"localhost":   false,
		"*.localhost": false,
		"127.0.0.1":   false,
	}
	for _, dns := range cert.DNSNames {
		if _, ok := expectedDNS[dns]; ok {
			expectedDNS[dns] = true
		}
	}
	for dns, found := range expectedDNS {
		if !found {
			t.Logf("Warning: Expected DNS name '%s' not found in certificate", dns)
		}
	}

	// Проверяем IP адреса
	expectedIPs := map[string]bool{
		"127.0.0.1": false,
		"::1":       false,
	}
	for _, ip := range cert.IPAddresses {
		if _, ok := expectedIPs[ip.String()]; ok {
			expectedIPs[ip.String()] = true
		}
	}
	for ip, found := range expectedIPs {
		if !found {
			t.Logf("Warning: Expected IP address '%s' not found in certificate", ip)
		}
	}
}

// Тест на параллельное выполнение
func TestParallelExecution(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cert-test-parallel-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	t.Run("parallel", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			t.Run("parallel-gen", func(t *testing.T) {
				t.Parallel()
				certFile := filepath.Join(tempDir, "certs", "parallel", "server.crt")
				keyFile := filepath.Join(tempDir, "certs", "parallel", "server.key")

				err := GenerateSelfSignedCert(certFile, keyFile)
				if err != nil {
					t.Errorf("Parallel generation failed: %v", err)
				}
			})
		}
	})
}

// Тест на производительность
func BenchmarkGenerateSelfSignedCert(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "cert-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	for i := 0; i < b.N; i++ {
		certFile := filepath.Join(tempDir, "bench.crt")
		keyFile := filepath.Join(tempDir, "bench.key")

		err := GenerateSelfSignedCert(certFile, keyFile)
		if err != nil {
			b.Fatal(err)
		}
	}
}
