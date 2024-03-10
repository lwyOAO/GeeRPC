package service

import (
	"reflect"
	"sync/atomic"
)

// MethodType 包含一个方法的完整信息
type MethodType struct {
	method   reflect.Method // 方法
	ArgType  reflect.Type   // 第一个参数的类型
	ReplyTpe reflect.Type   // 第二个参数的类型
	numCalls uint64         // 统计方法调用次数
}

func (m *MethodType) NumCalls() uint64 {
	// 原子地读numCalls的值
	return atomic.LoadUint64(&m.numCalls)
}

func (m *MethodType) NewArgv() reflect.Value {
	var argv reflect.Value
	// argv可能是指针类型，也可能是值类型
	if m.ArgType.Kind() == reflect.Ptr {
		// m.ArgType是指针，m.ArgType.Elem()返回指针指向的元素类型
		// 例如：*int，返回int
		// reflect.New()返回指针
		argv = reflect.New(m.ArgType.Elem())
	} else {
		// reflect.New().Elem() 返回指针指向的值类型
		argv = reflect.New(m.ArgType).Elem()
	}
	return argv
}

func (m *MethodType) NewReplyv() reflect.Value {
	// reply一定是指针类型
	replyv := reflect.New(m.ReplyTpe.Elem())
	switch m.ReplyTpe.Elem().Kind() {
	case reflect.Map:
		replyv.Elem().Set(reflect.MakeMap(m.ReplyTpe.Elem()))
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(m.ReplyTpe.Elem(), 0, 0))
	}
	return replyv
}
