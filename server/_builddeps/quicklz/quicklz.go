package quicklz
const(COMPRESSION_LEVEL_1=1;STREAMING_BUFFER_0=0)
type Encoder struct{};func New(level,streaming int)(*Encoder,error){return &Encoder{},nil};func(*Encoder)Compress(src,dst *[]byte)(int,error){return copy(*dst,*src),nil}
