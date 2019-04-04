package crypto

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"go.uber.org/zap"

	"k8s.io/api/core/v1"
)

type SecretWrapper struct {
	PrivateKey     *rsa.PrivateKey
	CACertificates []*x509.Certificate
	Certificates   []*x509.Certificate
}

func ParseSecretToCertContainer(secret *v1.Secret) (*SecretWrapper, error) {
	var err error

	keyBlock, _ := pem.Decode(secret.Data["tls.key"])
	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)

	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	certs := []*x509.Certificate{}
	caCerts := []*x509.Certificate{}

	raw := secret.Data["tls.crt"]

	for {
		block, rest := pem.Decode(raw)
		if block == nil {
			break
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}

		if cert.KeyUsage == 5 {
			certs = append(certs, cert)

		} else {
			caCerts = append(caCerts, cert)
		}

		raw = rest
	}

	return &SecretWrapper{
		Certificates:   certs,
		CACertificates: caCerts,
		PrivateKey:     key,
	}, nil
}

func AdditionalCaCerts(caSecret v1.Secret) (*[]*x509.Certificate, error) {
	caCerts := []*x509.Certificate{}

	for caName, caData := range caSecret.Data {
		zap.S().Debugf("Getting ca %s", caName)

		caCert, err := x509.ParseCertificate(caData)

		if err != nil {
			zap.S().Error("Error loading CA %s", caName)
			return nil, err
		}

		caCerts = append(caCerts, caCert)
	}

	return &caCerts, nil
}
