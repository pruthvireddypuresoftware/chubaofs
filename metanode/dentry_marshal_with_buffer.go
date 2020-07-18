package metanode

import (
	"encoding/binary"
	"fmt"
	"github.com/chubaofs/chubaofs/util/bytes"
	"github.com/chubaofs/chubaofs/util/encoding"
)

// Marshal marshals a dentry into a byte array.
func (d *Dentry) MarshalWithBuffer(buff *bytes.Buffer) (err error) {
	if err=d.MarshalKeyWithBuffer(buff);err!=nil {
		return
	}
	if err=d.MarshalValueWithBuffer(buff);err!=nil {
		return
	}
	return
}

func (d *Dentry) MarshalValueWithBuffer(buff *bytes.Buffer)(err error){
	valueLenData:=buff.NeedDataForWrite(4)
	valueLen:=0
	if err := encoding.Write(buff, binary.BigEndian, &d.Inode); err != nil {
		panic(err)
	}
	valueLen+=encoding.IntDataSize(d.Inode)
	if err := encoding.Write(buff, binary.BigEndian, &d.Type); err != nil {
		panic(err)
	}
	valueLen+=encoding.IntDataSize(d.Type)
	binary.BigEndian.PutUint32(valueLenData[0:4],uint32(valueLen))
	return
}

// MarshalKey is the bytes version of the MarshalKey method which returns the byte slice result.
func (d *Dentry) MarshalKeyWithBuffer(buff *bytes.Buffer) (err error) {
	keyLenData:=buff.NeedDataForWrite(4)
	keyLen:=0
	if err=encoding.Write(buff,binary.BigEndian,d.ParentId);err!=nil {
		return
	}
	keyLen+=encoding.IntDataSize(d.ParentId)
	nameData:=[]byte(d.Name)
	buff.Append(nameData)
	keyLen+=len(nameData)
	binary.BigEndian.PutUint32(keyLenData[0:4],uint32(keyLen))
	return
}


// Unmarshal unmarshals the dentry from a byte array.
func (d *Dentry) UnmarshalWithBuffer(buff *bytes.Buffer) (err error) {
	var (
		keyLen uint32
		valLen uint32
	)
	if _,err = encoding.Read(buff, binary.BigEndian, &keyLen); err != nil {
		return
	}
	if err = d.UnmarshalKeyWithBuffer(buff,int(keyLen)); err != nil {
		return
	}
	if _,err = encoding.Read(buff, binary.BigEndian, &valLen); err != nil {
		return
	}
	err = d.UnmarshalValueWithBuffer(buff,int(valLen))
	return
}

// UnmarshalKey unmarshals the exporterKey from bytes.
func (d *Dentry) UnmarshalKeyWithBuffer(buff *bytes.Buffer,keyLen int) (err error) {
	hasRead:=0
	var (
		readN int
	)
	if readN,err = encoding.Read(buff, binary.BigEndian, &d.ParentId); err != nil {
		return
	}
	hasRead+=readN
	d.Name = string(buff.CopyData(keyLen-hasRead))
	return
}

// UnmarshalValue unmarshals the value from bytes.
func (d *Dentry) UnmarshalValueWithBuffer(buff *bytes.Buffer,valueLen int) (err error) {
	hasRead :=0
	var (
		readN int
	)
	if readN,err = encoding.Read(buff, binary.BigEndian, &d.Inode); err != nil {
		return
	}
	hasRead +=readN
	if readN,err = encoding.Read(buff, binary.BigEndian, &d.Type);err!=nil {
		return
	}
	hasRead +=readN
	if hasRead !=valueLen{
		panic(fmt.Errorf("dentry value unmashal failed,expectValueL(%v) actualValueLen(%v)",valueLen, hasRead))
	}
	return
}
