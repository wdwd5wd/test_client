package client

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	RecipientLength    = 20
	FullShardKeyLength = 4
)

// Address include recipient and fullShardKey
type QkcAddress struct {
	Recipient    common.Address
	FullShardKey uint32
}

// ToHex return bytes included recipient and fullShardKey
func (Self QkcAddress) ToHex() string {
	address := Self.ToBytes()
	return hexutil.Encode(address)
}

func (Self QkcAddress) ToBytes() []byte {
	address := Self.Recipient.Bytes()
	shardKey := Uint32ToBytes(Self.FullShardKey)
	address = append(address, shardKey...)
	return address
}

func (Self QkcAddress) FullShardKeyToHex() string {
	return hexutil.Encode(Uint32ToBytes(Self.FullShardKey))
}

// Uint32ToBytes trans uint32 num to bytes
func Uint32ToBytes(n uint32) []byte {
	Bytes := make([]byte, 4)
	binary.BigEndian.PutUint32(Bytes, n)
	return Bytes
}

// CreatAddressFromBytes creat address from bytes
func CreatAddressFromBytes(bs []byte) (QkcAddress, error) {
	if len(bs) != RecipientLength+FullShardKeyLength {
		return QkcAddress{}, fmt.Errorf("bs length excepted %d,unexcepted %d", RecipientLength+FullShardKeyLength, len(bs))
	}

	buffer := bytes.NewBuffer(bs[RecipientLength:])
	var x uint32
	err := binary.Read(buffer, binary.BigEndian, &x)
	if err != nil {
		return QkcAddress{}, err
	}
	recipient := common.BytesToAddress(bs[0:RecipientLength])
	return QkcAddress{recipient, x}, nil
}
