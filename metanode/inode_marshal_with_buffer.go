package metanode

import (
	"encoding/binary"
	"fmt"
	"github.com/chubaofs/chubaofs/proto"
	"github.com/chubaofs/chubaofs/util/bytes"
	"github.com/chubaofs/chubaofs/util/encoding"
)

// Marshal marshals the inode into a byte array.
func (i *Inode) MarshalWithBuffer(b *bytes.Buffer) (err error) {
	if err=encoding.Write(b,binary.BigEndian,uint32(encoding.IntDataSize(i.Inode)));err!=nil {
		panic(err)
	}
	if err=encoding.Write(b,binary.BigEndian,i.Inode);err!=nil {
		panic(err)
	}
	vLen:=b.NeedDataForWrite(4)
	valStart:=b.Len()
	i.MarshalValueWithBuffer(b)
	valBytes:=b.Len()-valStart
	binary.BigEndian.PutUint32(vLen[0:4],uint32(valBytes))

	return
}

// MarshalValue marshals the value to bytes.
func (i *Inode) MarshalValueWithBuffer(buff *bytes.Buffer) (hasWrite int) {
	var err error
	i.RLock()
	if err = encoding.Write(buff, binary.BigEndian, &i.Type); err != nil {
		panic(err)
	}
	if err = encoding.Write(buff, binary.BigEndian, &i.Uid); err != nil {
		panic(err)
	}
	if err = encoding.Write(buff, binary.BigEndian, &i.Gid); err != nil {
		panic(err)
	}
	if err = encoding.Write(buff, binary.BigEndian, &i.Size); err != nil {
		panic(err)
	}
	if err = encoding.Write(buff, binary.BigEndian, &i.Generation); err != nil {
		panic(err)
	}
	if err = encoding.Write(buff, binary.BigEndian, &i.CreateTime); err != nil {
		panic(err)
	}
	if err = encoding.Write(buff, binary.BigEndian, &i.AccessTime); err != nil {
		panic(err)
	}
	if err = encoding.Write(buff, binary.BigEndian, &i.ModifyTime); err != nil {
		panic(err)
	}
	// write SymLink
	symSize := uint32(len(i.LinkTarget))
	if err = encoding.Write(buff, binary.BigEndian, &symSize); err != nil {
		panic(err)
	}
	buff.Append(i.LinkTarget)

	if err = encoding.Write(buff, binary.BigEndian, &i.NLink); err != nil {
		panic(err)
	}
	if err = encoding.Write(buff, binary.BigEndian, &i.Flag); err != nil {
		panic(err)
	}
	if err = encoding.Write(buff, binary.BigEndian, &i.Reserved); err != nil {
		panic(err)
	}
	// marshal ExtentsKey
	err = i.Extents.MarshalBinaryWithBuffer(buff)
	if err != nil {
		panic(err)
	}
	i.RUnlock()
	return
}


func (se *SortedExtents) MarshalBinaryWithBuffer(buff *bytes.Buffer) error{
	se.RLock()
	defer se.RUnlock()

	for _, ek := range se.eks {
		err := ek.MarshalBinaryWithBuffer(buff)
		if err != nil {
			return err
		}
	}
	return nil
}


func (se *SortedExtents) UnmarshalBinaryWithBuffer(buf *bytes.Buffer) (int,error) {
	var (
		ek proto.ExtentKey
		hasRead int
	)
	for {
		if buf.ReadRemainBytes()== 0 {
			break
		}
		readN,err := ek.UnmarshalBinaryWithBuffer(buf)
		if err != nil {
			return 0,err
		}
		hasRead+=readN
		se.Append(ek)
	}
	return hasRead,nil
}


// Unmarshal unmarshals the inode.
func (i *Inode) UnmarshalWithBuffer(buff *bytes.Buffer) (err error) {
	var (
		keyLen uint32
		valLen uint32
	)
	if _,err = encoding.Read(buff, binary.BigEndian, &keyLen); err != nil {
		return
	}
	if err=i.UnmarshalKeyWithBuffer(buff,int(keyLen));err!=nil {
		return
	}
	if _,err = encoding.Read(buff, binary.BigEndian, &valLen); err != nil {
		return
	}
	if err=i.UnmarshalValueWithBuffer(buff,int(valLen));err!=nil {
		return
	}
	return
}

func (i *Inode) UnmarshalKeyWithBuffer(buff *bytes.Buffer,expectRead int) (err error) {
	readN,err:=encoding.Read(buff,binary.BigEndian,&i.Inode)
	if readN!=expectRead{
		panic(fmt.Errorf("cannot unmarsha inode key ,expect(%v),actual(%v)",expectRead,readN))
	}
	return
}

func (i *Inode) UnmarshalValueWithBuffer(buff *bytes.Buffer,expectRead int) (err error) {
	var (
		hasRead,readN int
	)
	if readN,err = encoding.Read(buff, binary.BigEndian, &i.Type); err != nil {
		return
	}
	hasRead+=readN
	if readN,err = encoding.Read(buff, binary.BigEndian, &i.Uid); err != nil {
		return
	}
	hasRead+=readN
	if readN,err = encoding.Read(buff, binary.BigEndian, &i.Gid); err != nil {
		return
	}
	hasRead+=readN
	if readN,err = encoding.Read(buff, binary.BigEndian, &i.Size); err != nil {
		return
	}
	hasRead+=readN
	if readN,err = encoding.Read(buff, binary.BigEndian, &i.Generation); err != nil {
		return
	}
	hasRead+=readN
	if readN,err = encoding.Read(buff, binary.BigEndian, &i.CreateTime); err != nil {
		return
	}
	hasRead+=readN
	if readN,err = encoding.Read(buff, binary.BigEndian, &i.AccessTime); err != nil {
		return
	}
	hasRead+=readN
	if readN,err = encoding.Read(buff, binary.BigEndian, &i.ModifyTime); err != nil {
		return
	}
	hasRead+=readN
	// read symLink
	symSize := uint32(0)
	if readN,err = encoding.Read(buff, binary.BigEndian, &symSize); err != nil {
		return
	}
	hasRead+=readN
	if symSize > 0 {
		i.LinkTarget = buff.CopyData(int(symSize))
		hasRead+=int(symSize)
	}

	if readN,err = encoding.Read(buff, binary.BigEndian, &i.NLink); err != nil {
		return
	}
	hasRead+=readN
	if readN,err = encoding.Read(buff, binary.BigEndian, &i.Flag); err != nil {
		return
	}
	hasRead+=readN
	if readN,err = encoding.Read(buff, binary.BigEndian, &i.Reserved); err != nil {
		return
	}
	hasRead+=readN
	if buff.ReadRemainBytes() == 0 {
		return
	}
	// unmarshal ExtentsKey
	if i.Extents == nil {
		i.Extents = NewSortedExtents()
	}
	if readN,err = i.Extents.UnmarshalBinaryWithBuffer(buff); err != nil {
		return
	}
	hasRead+=readN
	if hasRead!=expectRead{
		panic(fmt.Errorf("cannot unmarsha inode value ,expect(%v),actual(%v)",expectRead,hasRead))
	}
	return
}