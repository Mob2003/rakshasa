package aes

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"unsafe"
)

var Key []byte

func AesCtrEncrypt(dst, plainText []byte) []byte {
	//1. 创建cipher.Block接口
	block, _ := aes.NewCipher(Key)

	//2. 创建分组模式，在crypto/cipher包中
	iv := bytes.Repeat([]byte("1"), block.BlockSize())
	stream := cipher.NewCTR(block, iv)
	//3. 加密

	stream.XORKeyStream(dst, plainText)

	return dst
}

func AesCtrDecrypt(encryptData []byte) []byte {
	data := make([]byte, len(encryptData))
	return AesCtrEncrypt(data, encryptData)
}

const hextable = "0123456789abcdef"

func MD5_B(str string) []byte {
	dst := make([]byte, 32)
	for k, v := range md5.Sum(Str2bytes(str)) {
		dst[k*2] = hextable[v>>4]
		dst[k*2+1] = hextable[v&0x0f]
	}
	return dst
}
func Str2bytes(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}
