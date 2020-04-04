// Modified from go-ethereum under GNU Lesser General Public License

package client

import (
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	EvmTx = 0
)

//go:generate gencodec -type txdata -field-override txdataMarshaling -out gen_tx_json.go

var (
	ErrInvalidSig     = errors.New("invalid transaction v, r, s values")
	prefixOfRlpUint32 = byte(0x84)
	lenOfRlpUint32    = 5
)

type Uint32 uint32

func (u *Uint32) getValue() uint32 {
	return uint32(*u)
}

func (u *Uint32) EncodeRLP(w io.Writer) error {
	bytes := make([]byte, lenOfRlpUint32)
	bytes[0] = prefixOfRlpUint32
	binary.BigEndian.PutUint32(bytes[1:], uint32(*u))
	_, err := w.Write(bytes)
	return err
}

func (u *Uint32) DecodeRLP(s *rlp.Stream) error {
	data, err := s.Raw()
	if err != nil {
		return err
	}
	if len(data) != lenOfRlpUint32 {
		return fmt.Errorf("len is %v should %v", len(data), lenOfRlpUint32)
	}

	if data[0] != prefixOfRlpUint32 {
		return fmt.Errorf("preString is wrong, is %v should %v", data[0], lenOfRlpUint32)

	}

	*u = Uint32(binary.BigEndian.Uint32(data[1:]))
	return nil
}

type EvmTransaction struct {
	data txdata
	// caches
	updated       bool
	hash          atomic.Value
	size          atomic.Value
	from          atomic.Value
	FromShardsize uint32
	ToShardsize   uint32
}

type txdata struct {
	AccountNonce     uint64          `json:"nonce"              gencodec:"required"`
	Price            *big.Int        `json:"gasPrice"           gencodec:"required"`
	GasLimit         uint64          `json:"gas"                gencodec:"required"`
	Recipient        *common.Address `json:"to"                 rlp:"nil"` // nil means contract creation
	Amount           *big.Int        `json:"value"              gencodec:"required"`
	Payload          []byte          `json:"data"            	gencodec:"required"`
	NetworkId        uint32          `json:"networkId"          gencodec:"required"`
	FromFullShardKey *Uint32         `json:"fromFullShardKey"   gencodec:"required"`
	ToFullShardKey   *Uint32         `json:"toFullShardKey"     gencodec:"required"`
	GasTokenID       uint64          `json:"gasTokenId"    		gencodec:"required"`
	TransferTokenID  uint64          `json:"transferTokenId"    gencodec:"required"`
	Version          uint32          `json:"version"            gencodec:"required"`
	// Signature values
	V *big.Int `json:"v"             gencodec:"required"`
	R *big.Int `json:"r"             gencodec:"required"`
	S *big.Int `json:"s"             gencodec:"required"`

	// This is only used when marshaling to JSON.
	Hash *common.Hash `json:"-"              rlp:"-"`
}

func (e *EvmTransaction) MarshalJSON() ([]byte, error) {
	return json.Marshal(&e.data)
}

func NewEvmTransaction(nonce uint64, to *common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, fromFullShardKey uint32, toFullShardKey uint32, gasTokenID uint64, transferTokenID uint64, networkId uint32, version uint32, data []byte) *EvmTransaction {
	return newEvmTransaction(nonce, to, amount, gasLimit, gasPrice, fromFullShardKey, toFullShardKey, gasTokenID, transferTokenID, networkId, version, data)
}

func (e *EvmTransaction) SetGas(data uint64) {
	e.data.GasLimit = data
	e.updated = true
}

func (e *EvmTransaction) SetNonce(data uint64) {
	e.data.AccountNonce = data
	e.updated = true
}

func (e *EvmTransaction) SetVRS(v, r, s *big.Int) {
	e.data.V = v
	e.data.R = r
	e.data.S = s
	e.updated = true
}

func newEvmTransaction(nonce uint64, to *common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, fromFullShardKey uint32, toFullShardKey uint32, gasTokenID uint64, transferTokenID uint64, networkId uint32, version uint32, data []byte) *EvmTransaction {
	newFromFullShardKey := Uint32(fromFullShardKey)
	newToFullShardKey := Uint32(toFullShardKey)
	if len(data) > 0 {
		data = common.CopyBytes(data)
	}
	d := txdata{
		AccountNonce:     nonce,
		Recipient:        to,
		Payload:          data,
		Amount:           new(big.Int),
		GasLimit:         gasLimit,
		Price:            new(big.Int),
		FromFullShardKey: &newFromFullShardKey,
		ToFullShardKey:   &newToFullShardKey,
		GasTokenID:       gasTokenID,
		TransferTokenID:  transferTokenID,
		NetworkId:        networkId,
		Version:          version,
		V:                new(big.Int),
		R:                new(big.Int),
		S:                new(big.Int),
	}
	if amount != nil {
		d.Amount.Set(amount)
	}
	if gasPrice != nil {
		d.Price.Set(gasPrice)
	}

	return &EvmTransaction{data: d}
}

// EncodeRLP implements rlp.Encoder
func (tx *EvmTransaction) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &tx.data)
}

// DecodeRLP implements rlp.Decoder
func (tx *EvmTransaction) DecodeRLP(s *rlp.Stream) error {
	_, size, _ := s.Kind()
	err := s.Decode(&tx.data)
	if err == nil {
		tx.size.Store(common.StorageSize(rlp.ListSize(size)))
	}

	return err
}

type txdataUnsigned struct {
	AccountNonce uint64          `json:"nonce"              gencodec:"required"`
	Price        *big.Int        `json:"gasPrice"           gencodec:"required"`
	GasLimit     uint64          `json:"gas"                gencodec:"required"`
	Recipient    *common.Address `json:"to"                 rlp:"nil"` // nil means contract creation
	Amount       *big.Int        `json:"value"              gencodec:"required"`
	Payload      []byte          `json:"input"              gencodec:"required"`
	NetworkId    uint32          `json:"networkid"          gencodec:"required"`
	// FromFullShardKey *Uint32         `json:"fromfullshardid"    gencodec:"required"`
	// ToFullShardKey   *Uint32         `json:"tofullshardid"      gencodec:"required"`
	GasTokenID      uint64 `json:"gasTokenID"      gencodec:"required"`
	TransferTokenID uint64 `json:"transferTokenID"      gencodec:"required"`
}

func (tx *EvmTransaction) getUnsignedHash() common.Hash {
	unsigntx := txdataUnsigned{
		AccountNonce: tx.data.AccountNonce,
		Price:        tx.data.Price,
		GasLimit:     tx.data.GasLimit,
		Recipient:    tx.data.Recipient,
		Amount:       tx.data.Amount,
		Payload:      tx.data.Payload,
		NetworkId:    tx.data.NetworkId,
		// FromFullShardKey: tx.data.FromFullShardKey,
		// ToFullShardKey:   tx.data.ToFullShardKey,
		GasTokenID:      tx.data.GasTokenID,
		TransferTokenID: tx.data.TransferTokenID,
	}

	return rlpHash(unsigntx)
}

func (tx *EvmTransaction) Data() []byte       { return common.CopyBytes(tx.data.Payload) }
func (tx *EvmTransaction) Gas() uint64        { return tx.data.GasLimit }
func (tx *EvmTransaction) GasPrice() *big.Int { return new(big.Int).Set(tx.data.Price) }
func (tx *EvmTransaction) Value() *big.Int    { return new(big.Int).Set(tx.data.Amount) }
func (tx *EvmTransaction) Nonce() uint64      { return tx.data.AccountNonce }
func (tx *EvmTransaction) CheckNonce() bool   { return true }
func (tx *EvmTransaction) FromFullShardId() uint32 {
	return tx.FromChainID()<<16 | tx.FromShardSize() | tx.FromShardID()
}
func (tx *EvmTransaction) ToFullShardId() uint32 {
	return tx.ToChainID()<<16 | tx.ToShardSize() | tx.ToShardID()
}
func (tx *EvmTransaction) NetworkId() uint32 { return tx.data.NetworkId }
func (tx *EvmTransaction) Version() uint32   { return tx.data.Version }
func (tx *EvmTransaction) IsCrossShard() bool {
	return !(tx.FromChainID() == tx.ToChainID() && tx.FromShardID() == tx.ToShardID())
}
func (tx *EvmTransaction) GasTokenID() uint64 {
	return tx.data.GasTokenID
}
func (tx *EvmTransaction) TransferTokenID() uint64 {
	return tx.data.TransferTokenID
}
func (tx *EvmTransaction) FromFullShardKey() uint32 { return tx.data.FromFullShardKey.getValue() }
func (tx *EvmTransaction) ToFullShardKey() uint32   { return tx.data.ToFullShardKey.getValue() }
func (tx *EvmTransaction) FromChainID() uint32      { return tx.data.FromFullShardKey.getValue() >> 16 }
func (tx *EvmTransaction) ToChainID() uint32        { return tx.data.ToFullShardKey.getValue() >> 16 }
func (tx *EvmTransaction) FromShardSize() uint32 {
	return tx.FromShardsize
}
func (tx *EvmTransaction) ToShardSize() uint32 {
	return tx.ToShardsize
}

func (tx *EvmTransaction) FromShardID() uint32 {
	shardMask := tx.FromShardSize() - 1
	return tx.data.FromFullShardKey.getValue() & shardMask
}
func (tx *EvmTransaction) ToShardID() uint32 {
	shardMask := tx.ToShardSize() - 1
	return tx.data.ToFullShardKey.getValue() & shardMask
}

// To returns the recipient address of the transaction.
// It returns nil if the transaction is a contract creation.
func (tx *EvmTransaction) To() *common.Address {
	if tx.data.Recipient == nil {
		return nil
	}

	to := *tx.data.Recipient
	return &to
}

// Hash hashes the RLP encoding of tx.
// It uniquely identifies the transaction.
func (tx *EvmTransaction) Hash() common.Hash {
	if hash := tx.hash.Load(); hash != nil && !tx.updated {
		return hash.(common.Hash)
	}
	v := rlpHash(tx)
	tx.hash.Store(v)
	return v
}

// WithSignature returns a new transaction with the given signature.
// This signature needs to be formatted as described in the yellow paper (v+27).
func (tx *EvmTransaction) WithSignature(sig []byte) (*EvmTransaction, error) {
	r, s, v, err := SignatureValues(tx, sig)
	if err != nil {
		return nil, err
	}
	cpy := &EvmTransaction{data: tx.data}
	cpy.data.R, cpy.data.S, cpy.data.V = r, s, v
	return cpy, nil
}

// SignTx signs the transaction using the given signer and private key
func SignTx(tx *EvmTransaction, prv *ecdsa.PrivateKey) (*EvmTransaction, error) {
	h := tx.getUnsignedHash()
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	return tx.WithSignature(sig)
}

// Cost returns amount + gasprice * gaslimit.
func (tx *EvmTransaction) Cost() *big.Int {
	total := new(big.Int).Mul(tx.data.Price, new(big.Int).SetUint64(tx.data.GasLimit))
	total.Add(total, tx.data.Amount)
	return total
}

func (tx *EvmTransaction) RawSignatureValues() (*big.Int, *big.Int, *big.Int) {
	return tx.data.V, tx.data.R, tx.data.S
}

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

type Transaction struct {
	TxType uint8
	EvmTx  *EvmTransaction

	hash atomic.Value
}

func (tx *Transaction) getNonce() uint64 {
	if tx.TxType == EvmTx {
		return tx.EvmTx.data.AccountNonce
	}

	//todo verify the default value when have more type of tx
	return 0
}

func (tx *Transaction) getPrice() *big.Int {
	if tx.TxType == EvmTx {
		return tx.EvmTx.data.Price
	}

	//todo verify the default value when have more type of tx
	return big.NewInt(0)
}

// SignatureValues returns signature values. This signature
// needs to be in the [R || S || V] format where V is 0 or 1.
func SignatureValues(tx *EvmTransaction, sig []byte) (R, S, V *big.Int, err error) {
	if len(sig) != 65 {
		panic(fmt.Sprintf("wrong size for signature: got %d, want 65", len(sig)))
	}
	R = new(big.Int).SetBytes(sig[:32])
	S = new(big.Int).SetBytes(sig[32:64])
	V = new(big.Int).SetBytes([]byte{sig[64] + 27})

	return R, S, V, nil
}

// Hash return the hash of the transaction it contained
func (tx *Transaction) Hash() (h common.Hash) {
	w := make([]byte, 5)
	w[0] = 0
	bytes, err := rlp.EncodeToBytes(tx.EvmTx)
	if err != nil {
		return common.Hash{}
	}
	binary.BigEndian.PutUint32(w[1:], uint32(len(bytes)))
	w = append(w, bytes...)

	hw := sha3.NewKeccak256()
	hw.Write(w)
	hw.Sum(h[:0])
	tx.hash.Store(h)
	return h
}

func recoverPlain(sighash common.Hash, R, S, Vb *big.Int, homestead bool) (common.Address, error) {
	if Vb.BitLen() > 8 {
		return common.Address{}, ErrInvalidSig
	}
	// QuarkChain use NetworkId to store the chain Id instead of added to V,
	// so do not need to remove chain Id from VB
	V := byte(Vb.Uint64() - 27)
	if !crypto.ValidateSignatureValues(V, R, S, homestead) {
		return common.Address{}, ErrInvalidSig
	}
	// encode the signature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, 65)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V
	// recover the public key from the signature
	pub, err := crypto.Ecrecover(sighash[:], sig)
	if err != nil {
		return common.Address{}, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return common.Address{}, errors.New("invalid public key")
	}
	var addr common.Address
	copy(addr[:], crypto.Keccak256(pub[1:])[12:])
	return addr, nil
}
