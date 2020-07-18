package bytes

import (
	"io"
	"sync"
)

var (
	commonBufferSize=1024
	commonPool=&sync.Pool{
		New: func() interface{} {
			return NewBufferWithData(make([]byte,commonBufferSize))
		},
	}
	inodeSize=8
	storeInodeNoPool=&sync.Pool{
		New: func() interface{} {
			return make([]byte,inodeSize)
		},
	}
)

type Buffer struct {
	buf []byte
	readOffset int
	writeOffset int
}


func GetCommonBytesBuffer()(b *Buffer){
	b=commonPool.Get().(*Buffer)
	b.readOffset=0
	b.writeOffset=0
	return
}


func PutCommonBytesBuffer(b *Buffer){
	b.readOffset=0
	b.writeOffset=0
	commonPool.Put(b)
}

func GetStoreInodeBuffer()(k []byte){
	k=storeInodeNoPool.Get().([]byte)
	for i:=0;i<len(k);i++{
		k[i]=0
	}
	return
}

func PutStoreInodeBuffer(k []byte)(){
	storeInodeNoPool.Put(k)
}

func NewBuffer()(b *Buffer) {
	b=new(Buffer)
	b.buf=make([]byte,0)
	return b
}

func NewBufferWithData(data []byte)(b *Buffer) {
	b=new(Buffer)
	b.buf=data
	b.writeOffset=len(data)
	return b
}

func (b *Buffer) NeedDataForWrite(n int)(data []byte){
	if b.writeOffset+n<=cap(b.buf){
		data=b.buf[b.writeOffset:b.writeOffset+n]
	}else {
		newBuf:=make([]byte,2*cap(b.buf))
		copy(newBuf,b.buf[0:b.writeOffset])
		b.buf=newBuf
		data=b.buf[b.writeOffset:b.writeOffset+n]
	}
	b.writeOffset+=n
	return
}

func (b *Buffer)Append(k []byte){
	if b.writeOffset+len(k)<=cap(b.buf){
		copy(b.buf[b.writeOffset:b.writeOffset+len(k)],k)
	}else {
		newBuf:=make([]byte,cap(b.buf)+len(k))
		copy(newBuf,b.buf[0:b.writeOffset])
		b.buf=newBuf
		copy(b.buf[b.writeOffset:b.writeOffset+len(k)],k)
	}
	b.writeOffset+=len(k)
}

func (b *Buffer)CopyData(size int)(data []byte){
	data=make([]byte,size)
	copy(data,b.buf[b.readOffset:b.readOffset+size])
	b.readOffset+=size
	return data
}

func (b *Buffer) NeedDataForRead(n int)(data []byte,err error){
	if b.readOffset+n>cap(b.buf){
		return nil,io.EOF
	}
	data=b.buf[b.readOffset:b.readOffset+n]
	b.readOffset+=n
	return data,nil
}

func (b *Buffer)GetData()(data []byte){
	return b.buf[b.readOffset:b.writeOffset]
}


func (b *Buffer)Len()int {
	return b.writeOffset
}

func (b *Buffer)ReadRemainBytes()int {
	return b.writeOffset-b.readOffset
}