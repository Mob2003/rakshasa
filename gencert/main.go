package main

import (
	"bytes"
	cr "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"math/rand"
	"net"
	"os"
	"time"
)

func main() {
	subj := &pkix.Name{
		CommonName:    "chinamobile.com",
		Organization:  []string{"Company, INC."},
		Country:       []string{"US"},
		Province:      []string{""},
		Locality:      []string{"San Francisco"},
		StreetAddress: []string{"Golden Gate Bridge"},
		PostalCode:    []string{"94016"},
	}
	ca, err := CreateCA(subj, 10)
	if err != nil {
		log.Panic(err)
	}

	Write(ca, "../cert/server")

	crt, err := Req(ca.CSR, subj, 10, []string{"test.default.svc", "test"}, []net.IP{})

	if err != nil {
		log.Panic(err)
	}

	Write(crt, "../cert/server")
}

type CERT struct {
	CERT       []byte
	CERTKEY    *rsa.PrivateKey
	CERTPEM    *bytes.Buffer
	CERTKEYPEM *bytes.Buffer
	CSR        *x509.Certificate
}

func CreateCA(sub *pkix.Name, expire int) (*CERT, error) {
	var (
		ca  = new(CERT)
		err error
	)

	if expire < 1 {
		expire = 1
	}
	// 为ca生成私钥
	ca.CERTKEY, err = rsa.GenerateKey(cr.Reader, 4096)
	if err != nil {
		return nil, err
	}

	// 对证书进行签名
	ca.CSR = &x509.Certificate{
		SerialNumber: big.NewInt(rand.Int63n(2000)),
		Subject:      *sub,
		NotBefore:    time.Now(),                       // 生效时间
		NotAfter:     time.Now().AddDate(expire, 0, 0), // 过期时间
		IsCA:         true,                             // 表示用于CA
		// openssl 中的 extendedKeyUsage = clientAuth, serverAuth 字段
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		// openssl 中的 keyUsage 字段
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	// 创建证书
	// caBytes 就是生成的证书
	ca.CERT, err = x509.CreateCertificate(cr.Reader, ca.CSR, ca.CSR, &ca.CERTKEY.PublicKey, ca.CERTKEY)
	if err != nil {
		return nil, err
	}
	ca.CERTPEM = new(bytes.Buffer)
	pem.Encode(ca.CERTPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: ca.CERT,
	})
	ca.CERTKEYPEM = new(bytes.Buffer)
	pem.Encode(ca.CERTKEYPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(ca.CERTKEY),
	})

	// 进行PEM编码，编码就是直接cat证书里面内容显示的东西
	return ca, nil
}

func Req(ca *x509.Certificate, sub *pkix.Name, expire int, dns []string, ip []net.IP) (*CERT, error) {
	var (
		cert = &CERT{}
		err  error
	)
	cert.CERTKEY, err = rsa.GenerateKey(cr.Reader, 4096)
	if err != nil {
		return nil, err
	}
	if expire < 1 {
		expire = 1
	}
	cert.CSR = &x509.Certificate{
		SerialNumber: big.NewInt(rand.Int63n(2000)),
		Subject:      *sub,
		IPAddresses:  ip,
		DNSNames:     dns,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(expire, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	cert.CERT, err = x509.CreateCertificate(cr.Reader, cert.CSR, ca, &cert.CERTKEY.PublicKey, cert.CERTKEY)
	if err != nil {
		return nil, err
	}

	cert.CERTPEM = new(bytes.Buffer)
	pem.Encode(cert.CERTPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.CERT,
	})
	cert.CERTKEYPEM = new(bytes.Buffer)
	pem.Encode(cert.CERTKEYPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(cert.CERTKEY),
	})
	return cert, nil
}

func Write(cert *CERT, file string) error {
	keyFileName := file + ".key"
	certFIleName := file + ".crt"
	kf, err := os.Create(keyFileName)
	if err != nil {
		return err
	}
	defer kf.Close()

	if _, err := kf.Write(cert.CERTKEYPEM.Bytes()); err != nil {
		return err
	}

	cf, err := os.Create(certFIleName)
	if err != nil {
		return err
	}
	if _, err := cf.Write(cert.CERTPEM.Bytes()); err != nil {
		return err
	}
	return nil
}
