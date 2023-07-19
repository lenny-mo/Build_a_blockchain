package main

import (
	"bytes"
	"math/big"
)

var alphabet = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")

// Base58Encode encodes a byte array to a modified base58 string
//
// Base58Encode 用于将一个字节切片（[]byte）编码为 Base58 格式。
func Base58Encode(input []byte) []byte {

	zeroPrefix := 0 // 用于记录前导0的个数
	for _, b := range input {
		if b == 0x00 {
			zeroPrefix++
		} else {
			break
		}
	}

	var result []byte //定义一个空的切片来存储 Base58 编码结果。

	x := big.NewInt(0).SetBytes(input)       // 将输入的字节切片转化为一个大整数（*big.Int）。
	base := big.NewInt(int64(len(alphabet))) // 创建一个大整数代表我们 Base58 字母表（alphabet）的长度。
	zero := big.NewInt(0)                    // 创建一个值为0的大整数，我们将用它来比较上面的 x 是否为0。

	mod := &big.Int{} // 创建一个新的大整数 mod 用于存储每次运算的余数。

	for x.Cmp(zero) != 0 {
		x.DivMod(x, base, mod)                         // 用 x 除以 alphabet 的长度（base），得到的结果保存在 x 中，余数保存在 mod 中。
		result = append(result, alphabet[mod.Int64()]) // 在结果的末尾添加一个字符，这个字符对应于字母表（alphabet）中 mod 所代表的索引。
	}

	// reverse
	reverseBytes(result) // 由于在上面的循环中，我们是从低位开始添加字符的，所以最后需要将结果反转。

	// 根据开头统计的前导0个数， add leading 1's
	for i := 0; i < zeroPrefix; i++ {
		result = append([]byte{alphabet[0]}, result...)
	}

	return result
}

// Base58Decode decodes a modified base58 string to bytes
//
// Base58Decode 用于将一个 Base58 编码的字符串解码为字节切片（[]byte）
func Base58Decode(input []byte) ([]byte, error) {
	result := big.NewInt(0)
	zeroBytes := 0

	for _, b := range input {
		if b == alphabet[0] {
			zeroBytes++
		} else {
			break
		}
	}

	payload := input[zeroBytes:]

	for b := range payload {
		charIndex := bytes.IndexByte(alphabet, payload[b])
		result.Mul(result, big.NewInt(58))
		result.Add(result, big.NewInt(int64(charIndex)))
	}

	decoded := result.Bytes()

	decoded = append(bytes.Repeat([]byte{byte(0x00)}, zeroBytes), decoded...)

	return decoded, nil
}
