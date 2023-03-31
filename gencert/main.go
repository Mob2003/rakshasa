package main

import (
	"bytes"
	cr "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"net"
	"os"
	"time"
)
type RandReader struct {
	rand.Source
}
func main() {
	rand.Seed(time.Now().Unix())
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
	//Write(ca, "../cert/server")
	crt, err := Req(ca.CSR, subj, 365, []string{"test.default.svc", "test"}, []net.IP{})

	if err != nil {
		log.Panic(err)
	}
	Write(crt, "../cert/server")
	GenerateRSAKey(2096)
	b, _ := os.ReadFile("../cert/private.pem")
	data := fmt.Sprintf("package cert\r\n func init(){\r\nprivateKey=%#v\r\n}\r\n", b)
	os.WriteFile("../cert/private.go", []byte(data), 0655)
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

func GenerateRSAKey(bits int) {
	//GenerateKey函数使用随机数据生成器random生成一对具有指定字位数的RSA密钥
	//Reader是一个全局、共享的密码用强随机数生成器
	privateKey, err := rsa.GenerateKey(cr.Reader, bits)
	if err != nil {
		panic(err)
	}
	//保存私钥
	//通过x509标准将得到的ras私钥序列化为ASN.1 的 DER编码字符串
	X509PrivateKey := x509.MarshalPKCS1PrivateKey(privateKey)
	//使用pem格式对x509输出的内容进行编码
	//创建文件保存私钥
	privateFile, err := os.Create("../cert/private.pem")
	if err != nil {
		panic(err)
	}
	defer privateFile.Close()
	//构建一个pem.Block结构体对象
	privateBlock := pem.Block{Type: "RSA Private Key", Bytes: X509PrivateKey}
	//将数据保存到文件
	pem.Encode(privateFile, &privateBlock)

	//保存公钥
	//获取公钥的数据
	publicKey := privateKey.PublicKey
	//X509对公钥编码
	X509PublicKey, err := x509.MarshalPKIXPublicKey(&publicKey)
	if err != nil {
		panic(err)
	}
	//pem格式编码
	//创建用于保存公钥的文件
	publicFile, err := os.Create("../cert/public.pem")
	if err != nil {
		panic(err)
	}
	defer publicFile.Close()
	//创建一个pem.Block结构体对象
	publicBlock := pem.Block{Type: "RSA Public Key", Bytes: X509PublicKey}
	//保存到文件
	pem.Encode(publicFile, &publicBlock)
}

