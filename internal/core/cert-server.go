package core

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"time"
)

// GenSelfSignedCert generates a TLS certificate trusted by browsers.
//
// Workflow:
//  1. Check if a local root CA already exists
//  2. If not, generate one and attempt to install it into the system trust
//     store.
//  3. Use the root CA to sign a server certificate for the given IPs and
//     localhost.
//
// If CA installation fails, the cert is still generated and will work - just
// won't be auto-trusted by browsers.
func GenSelfSignedCert(ips []string) (crt, key []byte, err error) {
	caCert, caKey, err := loadOrCreateCA()
	if err != nil {
		return nil, nil, err
	}

	return genServerCert(caCert, caKey, ips)
}

func genServerCert(caCert *x509.Certificate, caKey *ecdsa.PrivateKey, ips []string) (crt, key []byte, err error) {
	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
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
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(1, 0, 0),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    []string{"localhost"},
	}

	seen := make(map[string]bool)
	for _, ip := range append(ips, "127.0.0.1", "::1") {
		if seen[ip] {
			continue
		}
		seen[ip] = true
		if parsed := net.ParseIP(ip); parsed != nil {
			tmpl.IPAddresses = append(tmpl.IPAddresses, parsed)
		}
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, serverKey.Public(), caKey)
	if err != nil {
		return nil, nil, err
	}

	crt = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	keyDER, err := x509.MarshalECPrivateKey(serverKey)
	if err != nil {
		return nil, nil, err
	}

	key = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return crt, key, nil
}
