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

// GenSelfSignedCert generates self-signed cert for hosting TLS server
func GenSelfSignedCert(ips []string) (crt, key []byte, err error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

	tmpl := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"go-anywhere static file server"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(1, 0, 0),

		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,

		DNSNames: []string{"localhost"},
	}

	for _, ip := range ips {
		if parsed := net.ParseIP(ip); parsed != nil {
			tmpl.IPAddresses = append(tmpl.IPAddresses, parsed)
		}
	}
	if len(tmpl.IPAddresses) == 0 {
		tmpl.IPAddresses = append(tmpl.IPAddresses, net.ParseIP("127.0.0.1"))
	}

	// cert
	certDER, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, err
	}
	crt = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	// key
	privDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, nil, err
	}
	key = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privDER})

	return crt, key, nil
}
