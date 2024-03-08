package GeeRPC

import "github.com/lwyOAO/GeeRPC/codec"

const MagicNumber = 0x3bef5c

// Option
// 为提升性能，一般在报文的最开始会规划固定的字节来协商相关信息
// 如第一个字节表示序列化方式，但为了实现简单，GeeRPC唯一要协商的是消息的编解码方式
// 我们将这部分信息放到Option结构体里，然后Option固定用Json方式解析
// 后续对header、body的解析则根据Option里的CodecType决定
type Option struct {
	MagicNumber int        // MagicNumber marks this is a GeeRPC request
	CodecType   codec.Type // client may choose different Codec to encode body
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   codec.GobType,
}
