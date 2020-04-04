package client

import (
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	TOKENBASE = uint64(36)
)

type TransactionId struct {
	Hash    common.Hash
	ShardId uint32
}

func (tid *TransactionId) Hex() string {
	bytes := make([]byte, 36)
	copy(bytes, tid.Hash.Bytes())
	binary.BigEndian.PutUint32(bytes[32:], tid.ShardId)
	return "0x" + common.Bytes2Hex(bytes)
}

func ByteToTransactionId(bytes []byte) (*TransactionId, error) {
	if len(bytes) != 36 {
		return nil, errors.New("wrong TransactionId fromat.")
	}
	hash := common.BytesToHash(bytes[:32])
	shardId := binary.BigEndian.Uint32(bytes[32:])
	return &TransactionId{hash, shardId}, nil
}

func TokenIDEncode(str string) uint64 {
	if len(str) >= 13 {
		panic(errors.New("name too long"))
	}

	id := TokenCharEncode(str[len(str)-1])
	base := TOKENBASE

	len := len(str)
	for index := len - 2; index >= 0; index-- {
		id += base * (TokenCharEncode(str[index]) + 1)
		base *= TOKENBASE
	}
	return id
}

func TokenCharEncode(char byte) uint64 {
	if char >= byte('A') && char <= byte('Z') {
		return 10 + uint64(char-byte('A'))
	}
	if char >= byte('0') && char <= byte('9') {
		return uint64(char - byte('0'))
	}
	panic(fmt.Errorf("unknown character %v", byte(char)))
}

func GetFullShardIdByFullShardKey(fullShardKey uint32) uint32 {
	chainID := fullShardKey >> 16
	shardsize := uint32(1)
	shardID := fullShardKey & (shardsize - 1)
	return (chainID << 16) | shardsize | shardID
}

func GetFullShardId(chainId, shardSize, shardId uint32) uint32 {
	return chainId<<16 | shardSize | shardId
}

func NewAddress(fullShardKey uint32) (*ecdsa.PrivateKey, *QkcAddress, error) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, nil, err
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		fmt.Println("no ok")
		return nil, nil, fmt.Errorf("")
	}

	address := QkcAddress{crypto.PubkeyToAddress(*publicKeyECDSA), fullShardKey}
	return privateKey, &address, nil
}
