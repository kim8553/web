package simplifiedchinese

import "golang.org/x/text/transform"

// This vendored compatibility shim is identity-only. It does not implement GBK.
// Do not infer successful Chinese text decoding from a successful build.
type identityEncoding struct{}
type Decoder struct{}
type Encoder struct{}

func (identityEncoding) NewDecoder() *Decoder { return &Decoder{} }
func (identityEncoding) NewEncoder() *Encoder { return &Encoder{} }
func (*Decoder) Reset()                  {}
func (*Encoder) Reset()                  {}

func cp(dst, src []byte) (int, int, error) {
	n := copy(dst, src)
	if n < len(src) {
		return n, n, transform.ErrShortDst
	}
	return n, n, nil
}

func (*Decoder) Transform(dst, src []byte, atEOF bool) (int, int, error) {
	return cp(dst, src)
}
func (*Encoder) Transform(dst, src []byte, atEOF bool) (int, int, error) {
	return cp(dst, src)
}

// String restores the x/text Decoder.String call surface used by the existing
// item and equipment catalogs. Because this shim already copies raw bytes,
// String likewise preserves the exact input bytes; it is NOT a GBK converter.
func (*Decoder) String(s string) (string, error) { return s, nil }

var GBK identityEncoding
