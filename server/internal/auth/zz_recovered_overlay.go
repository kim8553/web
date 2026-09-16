package auth

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

func AccountKeyFor(account string) (string, error) {
	if account == "" {
		return "", fmt.Errorf("auth: empty account name")
	}
	if len(account) > 256 {
		return "", fmt.Errorf("auth: account name too long")
	}
	field1, _, err := EncryptFields(account, "x")
	if err != nil {
		return "", err
	}
	if len(field1) == 0 {
		return "", fmt.Errorf("auth: empty encrypted account")
	}
	digest := sha256.New()
	_, _ = digest.Write([]byte("nineyin/login-account/v1"))
	_, _ = digest.Write([]byte{0})
	var encodedLength [4]byte
	binary.LittleEndian.PutUint32(encodedLength[:], uint32(len(field1)))
	_, _ = digest.Write(encodedLength[:])
	_, _ = digest.Write(field1)
	return "acct:v1:" + hex.EncodeToString(digest.Sum(nil)), nil
}

var fcBYTEBIT = [8]uint8{0x80, 0x40, 0x20, 0x10, 0x08, 0x04, 0x02, 0x01}
var fcBIGBYTE = [24]uint32{0x00800000, 0x00400000, 0x00200000, 0x00100000, 0x00080000, 0x00040000, 0x00020000, 0x00010000, 0x00008000, 0x00004000, 0x00002000, 0x00001000, 0x00000800, 0x00000400, 0x00000200, 0x00000100, 0x00000080, 0x00000040, 0x00000020, 0x00000010, 0x00000008, 0x00000004, 0x00000002, 0x00000001}
var fcPC1 = [56]int{56, 48, 40, 32, 24, 16, 8, 0, 57, 49, 41, 33, 25, 17, 9, 1, 58, 50, 42, 34, 26, 18, 10, 2, 59, 51, 43, 35, 62, 54, 46, 38, 30, 22, 14, 6, 61, 53, 45, 37, 29, 21, 13, 5, 60, 52, 44, 36, 28, 20, 12, 4, 27, 19, 11, 3}
var fcTOTROT = [16]int{1, 2, 4, 6, 8, 10, 12, 14, 15, 17, 19, 21, 23, 25, 27, 28}
var fcPC2 = [48]int{13, 16, 10, 23, 0, 4, 2, 27, 14, 5, 20, 9, 22, 18, 11, 3, 25, 7, 15, 6, 26, 19, 12, 1, 40, 51, 30, 36, 46, 54, 29, 39, 50, 44, 32, 47, 43, 48, 38, 55, 33, 52, 45, 41, 49, 35, 28, 31}
var fcSPHex = [8]string{"00040101000000000000010004040101040001010404010004000000000001000004000000040101040401010004000004040001040001010000000104000000040400000004000100040001000401000004010000000101000001010404000104000100040000010400000104000100000000000404000004040100000000010000010004040101040000000000010100040101000000010000000100040000040001010000010000040100040000010004000004000000040400010404010004040101040001000000010104040001040000010404000004040100000401010404000000040001000400010000000004000100000401000000000004000101", "20801080008000800080000020801000000010002000000020001080208000802000008020801080008010800000008000800080000010002000000020001080008010002000100020800080000000000000008000800000208010000000108020001000200000800000000000801000208000000080108000001080208000000000000020801000200010800000100020800080000010800080108000800000000010800080008020000000208010802080100020000000008000000000008020800000008010800000100020000080200010002080008020000080200010000080100000000000008000802080000000000080200010802080108000801000", "08020000000202080000000008000208000200080000000008020200000200080800020008000008080000080000020008020208080002000000020808020000000000080800000000020208000200000002020000000208080002080802020008020008000202000000020008020008080000000802020800020000000000080002020800000008080002000802000000000200000202080002000800000000000200000800020008020208000200080800000800020000000000000800020808020008000002000000000808020208080000000802020000020200080000080000020808020008080200000000020808020200080000000800020800020200", "01208000812000008120000080000000802080008100800001008000012000000000000000208000002080008120800081000000000000008000800001008000010000000020000000008000012080008000000000008000012000008020000081008000010000008020000080008000002000008020800081208000810000008000800001008000002080008120800081000000000000000000000000208000802000008000800081008000010000000120800081200000812000008000000081208000810000000100000000200000010080000120000080208000810080000120000080200000000080000120800080000000000080000020000080208000", "00010000000108020000080200010042000008000001000000000040000008020001084000000800000100020001084000010042000008420001080000000040000000020000084000000840000000000001004000010842000108420001000200000842000100400000000000000042000108020000000200000042000108000000080000010042000100000000000200000040000008020001004200010840000100020000004000000842000108020001084000010000000000020000084200010842000108000000004200010842000008020000000000000840000000420001080000010002000100400000080000000000000008400001080200010040", "10000020000040200040000010404020000040201000000010404020000040000040002010404000000040001000002010004000004000200000002010400000000000001000400010400020004000000040400010400020100000001000402010004020000000001040400000404020104000000040400000404020000000200040002010000000100040200040400010404020000040001040000010000020000040000040002000000020104000001000002010404020004040000000402010404000004040200000000010004020100000000040000000004020104040000040000010004000104000200000000000404020000000201000400010400020", "00002000020020040208000400000000000800000208000402082000000820040208200400002000000000000200000402000000000000040200200402080000000800040208200002002000000800040200000400002004000820040200200000002004000800000208000002082004000820000200000000000004000820000000000400082000000020000208000402080004020020040200200402000000020020000000000400080004000020000008200402080000020820000008200402080000020000040208200400002004000820000000000002000000020820040000000002082000000020040008000002000004000800040008000002002000", "40100010001000000000040040100410000000104010001040000000000000104000040000000410401004100010040000100410401004000010000040000000000004104000001000100010401000000010040040000400400004100010041040100000000000000000000040000410400000100010001040100400000004004010040000000400001004100010000040000000400004100010000040100400001000104000000040000010000004104000041000000010000004004010001000000000401004104000040040000010000004100010001040100010000000004010041000100400001004004010000040100000400004000000001000100410"}
var fcSP [8][64]uint32
var fcField1Key = [16]uint8{0x66, 0xf9, 0x28, 0xe8, 0xa1, 0x00, 0x9b, 0x76, 0x29, 0xa0, 0x1f, 0x8d, 0xef, 0x45, 0x89, 0x19}
var fcField1Even = [4]uint32{0xf8ebd0a7, 0xc4badc9c, 0xa84fd0a2, 0x74cf24b7}
var fcField1Odd = [4]uint32{0x26e24c92, 0x38efda09, 0xc3b2d047, 0x28efd49c}
var fcField2Key = [16]uint8{0x10, 0x6e, 0xba, 0x55, 0x77, 0x61, 0xc7, 0x7e, 0x1e, 0xea, 0x0d, 0xeb, 0x67, 0x62, 0x93, 0xdc}
var fcField2Even = [4]uint32{0xf24d83e5, 0xab2526d8, 0x32b93688, 0xaffdfe20}
var fcField2Odd = [4]uint32{0xe2435ae5, 0x136126b1, 0x52ba6e85, 0x3fa3fb23}

type fcSubkeys struct {
	ks1 [32]uint32
	ks2 [32]uint32
	ks3 [32]uint32
}

var fcF1KS fcSubkeys
var fcF2KS fcSubkeys

func init() {
	for i := 0; i < 8; i++ {
		s := fcSPHex[i]
		for j := 0; j < 64; j++ {
			var b [4]byte
			base := j * 8
			for k := 0; k < 4; k++ {
				b[k] = hexByte(s, base+k*2)
			}
			fcSP[i][j] = binary.LittleEndian.Uint32(b[:])
		}
	}
	fcF1KS = fcFieldSubkeys(fcField1Key)
	fcF2KS = fcFieldSubkeys(fcField2Key)
}
func hexNibble(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	default:
		return 0
	}
}
func hexByte(s string, index int) byte {
	return hexNibble(s[index])<<4 | hexNibble(s[index+1])
}
func rol32(v uint32, n int) uint32 {
	n &= 31
	return v<<n | v>>(32-n)
}
func fcCookey(raw [32]uint32) [32]uint32 {
	var cook [32]uint32
	for i := 0; i < 16; i++ {
		raw0 := raw[i*2]
		raw1 := raw[i*2+1]
		cook[i*2] = ((raw0 & 0x00fc0000) << 6) | ((raw0 & 0x00000fc0) << 10) | ((raw1 & 0x00fc0000) >> 10) | ((raw1 & 0x00000fc0) >> 6)
		cook[i*2+1] = ((raw0 & 0x0003f000) << 12) | ((raw0 & 0x0000003f) << 16) | ((raw1 & 0x0003f000) >> 4) | (raw1 & 0x0000003f)
	}
	return cook
}
func fcDeskey(key8 []byte, edf int) [32]uint32 {
	var pc1m [56]int
	var pcr [56]int
	var kn [32]uint32
	for j := 0; j < 56; j++ {
		l := fcPC1[j]
		m := l & 7
		if key8[l>>3]&fcBYTEBIT[m] != 0 {
			pc1m[j] = 1
		}
	}
	for i := 0; i < 16; i++ {
		var m int
		if edf == 1 {
			m = (15 - i) << 1
		} else {
			m = i << 1
		}
		n := m + 1
		kn[m], kn[n] = 0, 0
		for j := 0; j < 28; j++ {
			l := j + fcTOTROT[i]
			if l < 28 {
				pcr[j] = pc1m[l]
			} else {
				pcr[j] = pc1m[l-28]
			}
		}
		for j := 28; j < 56; j++ {
			l := j + fcTOTROT[i]
			if l < 56 {
				pcr[j] = pc1m[l]
			} else {
				pcr[j] = pc1m[l-28]
			}
		}
		for j := 0; j < 24; j++ {
			if pcr[fcPC2[j]] != 0 {
				kn[m] |= fcBIGBYTE[j]
			}
			if pcr[fcPC2[j+24]] != 0 {
				kn[n] |= fcBIGBYTE[j]
			}
		}
	}
	return fcCookey(kn)
}
func fcDesCore(a uint32, b uint32, ks [32]uint32) (uint32, uint32) {
	left, right := a, b
	work := ((left >> 4) ^ right) & 0x0f0f0f0f
	right ^= work
	left ^= work << 4
	work = ((left >> 16) ^ right) & 0x0000ffff
	right ^= work
	left ^= work << 16
	work = ((right >> 2) ^ left) & 0x33333333
	left ^= work
	right ^= work << 2
	work = ((right >> 8) ^ left) & 0x00ff00ff
	left ^= work
	right ^= work << 8
	right = rol32(right, 1)
	work = (left ^ right) & 0xaaaaaaaa
	left ^= work
	right ^= work
	left = rol32(left, 1)
	k := 0
	for i := 0; i < 8; i++ {
		work = rol32(right, 28) ^ ks[k]
		fval := fcSP[6][work&0x3f] | fcSP[4][(work>>8)&0x3f] | fcSP[2][(work>>16)&0x3f] | fcSP[0][(work>>24)&0x3f]
		work = right ^ ks[k+1]
		fval |= fcSP[7][work&0x3f] | fcSP[5][(work>>8)&0x3f] | fcSP[3][(work>>16)&0x3f] | fcSP[1][(work>>24)&0x3f]
		left ^= fval
		work = rol32(left, 28) ^ ks[k+2]
		fval = fcSP[6][work&0x3f] | fcSP[4][(work>>8)&0x3f] | fcSP[2][(work>>16)&0x3f] | fcSP[0][(work>>24)&0x3f]
		work = left ^ ks[k+3]
		fval |= fcSP[7][work&0x3f] | fcSP[5][(work>>8)&0x3f] | fcSP[3][(work>>16)&0x3f] | fcSP[1][(work>>24)&0x3f]
		right ^= fval
		k += 4
	}
	right = rol32(right, 31)
	work = (left ^ right) & 0xaaaaaaaa
	left ^= work
	right ^= work
	left = rol32(left, 31)
	work = ((left >> 8) ^ right) & 0x00ff00ff
	right ^= work
	left ^= work << 8
	work = ((left >> 2) ^ right) & 0x33333333
	right ^= work
	left ^= work << 2
	work = ((right >> 16) ^ left) & 0x0000ffff
	left ^= work
	right ^= work << 16
	work = ((right >> 4) ^ left) & 0x0f0f0f0f
	left ^= work
	right ^= work << 4
	return right, left
}
func fcTransform128(block16 []byte, ks1 [32]uint32, ks2 [32]uint32, ks3 [32]uint32) [16]byte {
	a := binary.BigEndian.Uint32(block16[0:4])
	b := binary.BigEndian.Uint32(block16[4:8])
	c := binary.BigEndian.Uint32(block16[8:12])
	d := binary.BigEndian.Uint32(block16[12:16])
	a, b = fcDesCore(a, b, ks1)
	c, d = fcDesCore(c, d, ks1)
	a, c = fcDesCore(a, c, ks2)
	b, d = fcDesCore(b, d, ks2)
	a, b = fcDesCore(a, b, ks3)
	c, d = fcDesCore(c, d, ks3)
	var out [16]byte
	binary.BigEndian.PutUint32(out[0:4], a)
	binary.BigEndian.PutUint32(out[4:8], b)
	binary.BigEndian.PutUint32(out[8:12], c)
	binary.BigEndian.PutUint32(out[12:16], d)
	return out
}
func fcFieldSubkeys(key16 [16]uint8) fcSubkeys {
	ks1 := fcDeskey(key16[:8], 0)
	ks2 := fcDeskey(key16[8:], 1)
	return fcSubkeys{ks1: ks1, ks2: ks2, ks3: ks1}
}
func pad16(data []byte) []byte {
	n := len(data)
	m := (n + 15) &^ 15
	if m == n {
		return data
	}
	out := make([]byte, m)
	copy(out, data)
	return out
}
func encryptField(data []byte, even [4]uint32, odd [4]uint32, ks fcSubkeys) []byte {
	blocks := len(data) / 16
	out := make([]byte, 0, blocks*16)
	for i := 0; i < blocks; i++ {
		block := data[i*16 : i*16+16]
		mask := even
		if i&1 != 0 {
			mask = odd
		}
		var mixed [16]byte
		binary.LittleEndian.PutUint32(mixed[0:4], binary.LittleEndian.Uint32(block[0:4])^mask[0])
		binary.LittleEndian.PutUint32(mixed[4:8], binary.LittleEndian.Uint32(block[4:8])^mask[1])
		binary.LittleEndian.PutUint32(mixed[8:12], binary.LittleEndian.Uint32(block[8:12])^mask[2])
		binary.LittleEndian.PutUint32(mixed[12:16], binary.LittleEndian.Uint32(block[12:16])^mask[3])
		encrypted := fcTransform128(mixed[:], ks.ks1, ks.ks2, ks.ks3)
		out = append(out, encrypted[:]...)
	}
	return out
}
func EncryptFields(account string, loginPassword string) (field1CT []byte, field2CT []byte, err error) {
	field1 := pad16([]byte(account))
	field2 := pad16([]byte(loginPassword))
	return encryptField(field1, fcField1Even, fcField1Odd, fcF1KS), encryptField(field2, fcField2Even, fcField2Odd, fcF2KS), nil
}
func ParseLoginCredentials(message []byte) (accountCT, passwordCT []byte, err error) {
	if len(message) < loginHeaderSize || message[0] != 0x02 {
		return nil, nil, fmt.Errorf("%w: missing login header", ErrInvalidLogin)
	}
	accountLength := binary.LittleEndian.Uint32(message[0x15:0x19])
	if err := validateSlotLength("account", accountLength, accountMin, accountMax); err != nil {
		return nil, nil, err
	}
	accountEnd, ok := advance(loginHeaderSize, accountLength, len(message))
	if !ok {
		return nil, nil, fmt.Errorf("%w: truncated account slot", ErrInvalidLogin)
	}
	cursor := accountEnd
	passwordLength, next, ok := readLength(message, cursor)
	if !ok {
		return nil, nil, fmt.Errorf("%w: missing password length", ErrInvalidLogin)
	}
	if err := validateSlotLength("password", passwordLength, secretMin, secretMax); err != nil {
		return nil, nil, err
	}
	passwordEnd, ok := advance(next, passwordLength, len(message))
	if !ok {
		return nil, nil, fmt.Errorf("%w: truncated password slot", ErrInvalidLogin)
	}
	return message[loginHeaderSize:accountEnd], message[next:passwordEnd], nil
}
func ComputePasswordVerifier(account, password string) ([]byte, error) {
	lp, err := ComputeLoginPassword(password)
	if err != nil {
		return nil, err
	}
	_, field2, err := EncryptFields(account, lp)
	if err != nil {
		return nil, err
	}
	return field2, nil
}

var passwordKey = append([]byte("nengbupojiemexiechengxuhaoxinku"), 0)

func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	return append(data, bytes.Repeat([]byte{byte(padding)}, padding)...)
}
func EncryptPassword(plain string) (string, error) {
	block, err := aes.NewCipher(passwordKey)
	if err != nil {
		return "", err
	}
	pt := pkcs7Pad([]byte(plain), aes.BlockSize)
	ct := make([]byte, len(pt))
	iv := make([]byte, aes.BlockSize)
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ct, pt)
	return base64.StdEncoding.EncodeToString(ct), nil
}
func VerifyPasswordVerifier(passwordCT, verifier []byte) bool {
	return bytes.Equal(passwordCT, verifier)
}
func swapHalves(value string) string {
	half := len(value) / 2
	return value[half:] + value[:half]
}
func ComputeLoginPassword(plain string) (string, error) {
	a, err := EncryptPassword(plain)
	if err != nil {
		return "", err
	}
	ab := []byte(a)
	sum := md5.Sum(ab)
	acc := 0
	for _, value := range sum {
		acc += int(value)
	}
	b := base64.StdEncoding.EncodeToString(append([]byte{byte(acc)}, ab...))
	return swapHalves(b), nil
}
