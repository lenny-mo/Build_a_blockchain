package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"

	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"golang.org/x/crypto/ripemd160"
)

const (
	// 用于生成地址的版本号
	MAINNET_VERSION byte = 0x00 // 主网版本号
	TESTNET_VERSION byte = 0x6f // 测试网版本号
)

type Wallet struct {
	PrivateKey ecdsa.PrivateKey // 用于签署交易，保证交易的非伪造性
	PublicKey  []byte           // 用于验证交易，保证交易的真实性
}

func CreateWallet() *Wallet {
	privatekey, publickey := GenerateKeyPair()
	return &Wallet{privatekey, publickey}
}

// GenerateKeyPair generates a new key pair
//
// 生成一个新的公私钥对
func GenerateKeyPair() (ecdsa.PrivateKey, []byte) {
	// P-256 的椭圆曲线，又称为 secp256r1 或者 prime256v1
	// curve := elliptic.P256()
	// 为了做到和以太坊兼容，我们使用 secp256k1 椭圆曲线
	curve := secp256k1.S256()

	// 生成一个私钥
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		panic(err)
	}

	publickey := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)

	return *private, publickey
}

// GetAddressWithPublickey returns the wallet address
//
// 生成钱包地址
func (w *Wallet) GetAddressWithPublickey(version byte) []byte {

	// 1. calculate the public key hash
	publickeyHash := PublickeyHash(w.PublicKey)

	// 2～4 concat [version, public key hash, checksum] into one
	//
	// 2. put the blockchain version and public key hash together
	versionedPayload := append([]byte{version}, publickeyHash...)

	// 3. calculate the checksum
	checkSum := GenerateChecksum(versionedPayload)

	// 4. append the checksum to the versioned payload
	fullPayload := append(versionedPayload, checkSum...)

	// 5. encode the full payload with base58
	address := Base58Encode(fullPayload)

	return address
}

// PublickeyHash returns the public key hash
//
// 根据公钥生成公钥哈希, 使用到了 SHA256 和 RIPEMD160 哈希函数，
// 先执行 SHA256 哈希，再执行 RIPEMD160 哈希, 最后返回 RIPEMD160 哈希的结果
func PublickeyHash(publickey []byte) []byte {
	// get the public key hash uising SHA256
	publickeyHash256 := sha256.Sum256(publickey)

	// 创建一个RIPEMD160哈希函数的实例。
	// RIPEMD160是一种可以将任意长度的数据映射到一个固定长度（160位）的哈希值的函数。
	RIPEMD160Hasher := ripemd160.New()

	// 使用 RIPEMD-160 再次对sha256哈希值进行哈希
	_, err := RIPEMD160Hasher.Write(publickeyHash256[:])
	if err != nil {
		panic(err)
	}
	// 将RIPEMD160哈希的结果赋值给publickeyHash，如果输入的参数是nil，那么Sum方法就只会返回当前的哈希结果。
	// 这个哈希值通常被用作区块链地址，因为它既短且可以代表公钥
	publickeyHash := RIPEMD160Hasher.Sum(nil)

	return publickeyHash
}

// ValidateAddress checks if the public key hash is valid
//
// 验证公钥地址是否有效
func ValidateAddress(address string) bool {
	// decode the address to byte slice
	publickeyHash, _ := Base58Decode([]byte(address))

	// the last 4 bytes is the checksum
	actualCheckSum := publickeyHash[len(publickeyHash)-4:]

	// publickeyHash[0] is the version
	actualPublickeyHash := publickeyHash[1 : len(publickeyHash)-4]

	actualVersion := publickeyHash[0]

	// recalculate the checksum according to the version and public key hash
	targetCheckSum := GenerateChecksum(append([]byte{actualVersion}, actualPublickeyHash...))

	return bytes.Compare(actualCheckSum, targetCheckSum) == 0
}

// GenerateChecksum generates a 4-byte checksum for a byte slice
//
// 对输入进行两次sha256操作，取结果的前4字节作为校验和
func GenerateChecksum(data []byte) []byte {
	firstSHA := sha256.Sum256(data)
	secondSHA := sha256.Sum256(firstSHA[:])

	return secondSHA[:4]
}

// AddressToPubkeyHash returns the public key hash from an address
//
// 根据输入的地址，返回公钥哈希
func AddressToPubkeyHash(addr string) []byte {
	decodedAddr, err := Base58Decode([]byte(addr))
	if err != nil {
		panic(err)
	}

	pubkeyHash := decodedAddr[1 : len(decodedAddr)-4]
	return pubkeyHash
}
