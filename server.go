package GeeRPC

import (
	"encoding/json"
	"fmt"
	"github.com/lwyOAO/GeeRPC/codec"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
)

const MagicNumber = 0x3bef5c

// Option
// 为提升性能，一般在报文的最开始会规划固定的字节来协商相关信息
// 如第一个字节表示序列化方式，但为了实现简单，GeeRPC唯一要协商的是消息的编解码方式
// 我们将这部分信息放到Option结构体里，然后Option固定用Json方式解析(Option固定在报文的最开始)
// 后续对header、body的解析则根据Option里的CodecType决定
type Option struct {
	MagicNumber int        // MagicNumber marks this is a GeeRPC request
	CodecType   codec.Type // client may choose different Codec to encode body
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   codec.GobType,
}

// Server represents an RPC Server.
type Server struct{}

func NewServer() *Server {
	return &Server{}
}

var DefaultServer = NewServer()

// Accept accepts connections on the listener and serves requests
// for each incoming connection.
func (server *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error:", err)
			return
		}
		go server.ServeConn(conn)
	}
}

func Accept(lis net.Listener) {
	DefaultServer.Accept(lis)
}

// ServeConn 处理连接
func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	defer func() {
		_ = conn.Close()
	}()

	// 解码option
	var opt Option
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server: options error:", err)
		return
	}

	// 校验请求编号是否正确
	if opt.MagicNumber != MagicNumber {
		log.Printf("rpc server: invalid magic number %x", opt.MagicNumber)
		return
	}

	// 根据编解码器类型选择编解码器
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		log.Printf("rpc server: invalid codec type %s", opt.CodecType)
		return
	}

	// 将conn放入编解码器实例，继续处理
	server.serveCodec(f(conn))
}

// invalidRequest is a placeholder for response argv when error occurs
var invalidRequest = struct{}{}

// 处理编解码器
func (server *Server) serveCodec(cc codec.Codec) {
	sending := new(sync.Mutex) // make sure to send  a complete response
	wg := new(sync.WaitGroup)  // wait until all requests are handled

	// 一次连接多个请求，只有在header解析失败才终止连接
	for {
		// 读取请求
		req, err := server.readRequest(cc)
		if err != nil {
			if req == nil {
				break // it is not possible to recover
			}
			req.h.Error = err.Error()
			server.sendResponse(cc, req.h, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		// 并发处理请求, 但回复请求的报文必须逐个发送，使用sending(锁）保证
		go server.handleRequest(cc, req, sending, wg)
	}
	wg.Wait()
	_ = cc.Close()
}

func (server *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var h codec.Header
	if err := cc.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error:", err)
		}
		return nil, err
	}
	return &h, nil
}

type request struct {
	h            *codec.Header
	argv, replyv reflect.Value // argv、replyv of request
}

func (server *Server) readRequest(cc codec.Codec) (*request, error) {
	h, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}

	req := &request{h: h}
	// TODO now we don't know the type of request argv, now just suppose it's string
	req.argv = reflect.New(reflect.TypeOf(""))
	if err = cc.ReadBody(req.argv.Interface()); err != nil {
		log.Println("rpc server: read argv err:", err)
	}
	return req, nil
}

func (server *Server) sendResponse(cc codec.Codec, h *codec.Header, body any, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()

	if err := cc.Write(h, body); err != nil {
		log.Println("rpc server: write response error:", err)
	}
}

func (server *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {
	// TODO should call registered rpc methods to get the right replyv

	defer wg.Done()
	log.Println(req.h, req.argv.Elem())
	req.replyv = reflect.ValueOf(fmt.Sprintf("geerpc resp %d", req.h.Seq))
	server.sendResponse(cc, req.h, req.replyv.Interface(), sending)
}
