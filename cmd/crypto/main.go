package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"metrics-and-alerting/pkg/logpack"
)

func GenerateRsaKeyPair() (*rsa.PrivateKey, *rsa.PublicKey) {
	key, _ := rsa.GenerateKey(rand.Reader, 4096)
	return key, &key.PublicKey
}

func PrivateToString(key *rsa.PrivateKey) string {
	bytes := x509.MarshalPKCS1PrivateKey(key)
	pem := pem.EncodeToMemory(
		&pem.Block{

			Type:  "RSA PRIVATE KEY",
			Bytes: bytes,
		},
	)
	return string(pem)
}

func PublicToString(key *rsa.PublicKey) (string, error) {
	bytes, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return ``, err
	}

	pem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: bytes,
		},
	)

	return string(pem), nil
}

func ExportToFile(data string, file string) error {
	return ioutil.WriteFile(file, []byte(data), 0644)
}

func main() {

	privatePath := "private.key"
	publicPath := "public.key"

	logger := logpack.NewLogger()

	privateKey, publicKey := GenerateRsaKeyPair()

	privatePEM := PrivateToString(privateKey)
	publicPEM, _ := PublicToString(publicKey)

	if err := ExportToFile(privatePEM, privatePath); err != nil {
		logger.Err.Printf("failed export to file private key: %v\n", err)
	} else {
		logger.Info.Printf("success export private key to file: %s\n", privatePath)
	}

	if err := ExportToFile(publicPEM, publicPath); err != nil {
		logger.Err.Printf("failed export to file public key: %v\n", err)
	} else {
		logger.Info.Printf("success export public key to file: %s\n", publicPath)
	}
}
