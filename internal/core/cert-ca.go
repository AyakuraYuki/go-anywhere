package core

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/AyakuraYuki/go-anywhere/internal/log"
)

const (
	caName = "go-anywhere Root CA"
)

func loadOrCreateCA() (*x509.Certificate, *ecdsa.PrivateKey, error) {
	if cert, key, err := loadCA(); err == nil {
		return cert, key, nil
	}

	cert, key, err := createCA()
	if err != nil {
		return nil, nil, err
	}

	if err = InstallCA(); err != nil {
		log.Warn().Str("scope", "cert-ca").Msgf(`Cannot auto-install CA into trust store: %v
  You can manually trust the CA cert at %s
  Or run: sudo anywhere --install-ca (required run anywhere at least once)
`, err, caCertPath())
	} else {
		log.Debug().Str("scope", "cert-ca").Msgf(`Local CA installed into system trust store.
  Browsers will trust certificates from this server.
  CA cert location: %s
`, caCertPath())
	}

	return cert, key, nil
}

func loadCA() (*x509.Certificate, *ecdsa.PrivateKey, error) {
	certPEM, err := os.ReadFile(caCertPath())
	if err != nil {
		return nil, nil, err
	}

	certKey, err := os.ReadFile(caKeyPath())
	if err != nil {
		return nil, nil, err
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, nil, errors.New("invalid CA cert PEM")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}

	if time.Now().After(cert.NotAfter) {
		return nil, nil, errors.New("CA certificate expired")
	}

	keyBlock, _ := pem.Decode(certKey)
	if keyBlock == nil {
		return nil, nil, errors.New("invalid CA key")
	}

	key, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}

	return cert, key, nil
}

func createCA() (*x509.Certificate, *ecdsa.PrivateKey, error) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Country:      []string{"CN"},
			Organization: []string{"go-anywhere static file server"},
			CommonName:   caName,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, privKey.Public(), privKey)
	if err != nil {
		return nil, nil, err
	}

	caCert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, err
	}

	// Persist to disk
	if err = os.MkdirAll(caDir(), os.ModePerm); err != nil {
		return nil, nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	if err = os.WriteFile(caCertPath(), certPEM, 0644); err != nil {
		return nil, nil, err
	}

	keyDER, err := x509.MarshalECPrivateKey(privKey)
	if err != nil {
		return nil, nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	if err = os.WriteFile(caKeyPath(), keyPEM, 0600); err != nil {
		return nil, nil, err
	}

	return caCert, privKey, nil
}

func InstallCA() error {
	switch runtime.GOOS {
	case "darwin":
		return runCmd("sudo", "security", "add-trusted-cert",
			"-d",
			"-r", "trustRoot",
			"-k", "/Library/Keychains/System.keychain",
			caCertPath())

	case "windows":
		return runCmd("certutil", "-addstore", "-user", "Root", caCertPath())

	case "linux":
		// Debian/Ubuntu
		if _, err := exec.LookPath("update-ca-certificates"); err == nil {
			dst := "/usr/local/share/ca-certificates/go-anywhere-ca.crt"
			if err = runCmd("sudo", "cp", caCertPath(), dst); err != nil {
				return err
			}
			return runCmd("sudo", "update-ca-certificates", "--fresh")
		}

		// RHEL/Fedora/Arch
		if _, err := exec.LookPath("trust"); err == nil {
			return runCmd("sudo", "trust", "anchor", "--store", caCertPath())
		}

		return fmt.Errorf("no supported trust store manager found (need update-ca-certificates or trust)")

	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func UninstallCA() error {
	defer func(path string) { _ = os.RemoveAll(path) }(caDir())

	switch runtime.GOOS {
	case "darwin":
		return runCmd("sudo", "security", "remove-trusted-cert", "-d", caCertPath())

	case "windows":
		return runCmd("certutil", "-delstore", "-user", "Root", caName)

	case "linux":
		// Debian/Ubuntu
		if _, err := exec.LookPath("update-ca-certificates"); err == nil {
			dst := "/usr/local/share/ca-certificates/go-anywhere-ca.crt"
			if err = runCmd("sudo", "rm", "-f", dst); err != nil {
				return err
			}
			return runCmd("sudo", "update-ca-certificates", "--fresh")
		}

		// RHEL/Fedora/Arch
		if _, err := exec.LookPath("trust"); err == nil {
			return runCmd("sudo", "trust", "anchor", "--remove", caCertPath())
		}

		return fmt.Errorf("no supported trust store manager found (need update-ca-certificates or trust)")
	}

	return nil
}

func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
