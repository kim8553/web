package simplifiedchinese
import "golang.org/x/text/transform"
type identityEncoding struct{}; type Decoder struct{}; type Encoder struct{}
func(identityEncoding)NewDecoder()*Decoder{return &Decoder{}};func(identityEncoding)NewEncoder()*Encoder{return &Encoder{}}
func(*Decoder)Reset(){};func(*Encoder)Reset(){}
func cp(dst,src []byte)(int,int,error){n:=copy(dst,src);if n<len(src){return n,n,transform.ErrShortDst};return n,n,nil}
func(*Decoder)Transform(dst,src []byte,atEOF bool)(int,int,error){return cp(dst,src)}
func(*Encoder)Transform(dst,src []byte,atEOF bool)(int,int,error){return cp(dst,src)}
var GBK identityEncoding
