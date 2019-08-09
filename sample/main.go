package main

import (
	"fmt"
	clt "github.com/QuarkChain/goqkcclient/client"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
)

var (
	client      = clt.NewClient("http://jrpc.devnet.quarkchain.io:38391")
	fullShardId = uint32(0)
)

func main() {
	address := common.HexToAddress("0xc9D14ADBff9F0f27725fceaB08577a6F34729c99")
	prvkey, _ := crypto.ToECDSA(common.FromHex("0xf4a9a6275a8476e7008d5e0d8dd860cbdb41b69dce18624a5c610cf29c4f7904"))

	context := make(map[string]string)
	context["address"] = address.Hex()
	getBalance(context)
	_, qkcToAddr, err := client.NewAddress(0)
	if err != nil {
		fmt.Println("NewAddress error: ", err.Error())
	}

	height := getHeight(nil)
	context["height"] = "0x" + common.Bytes2Hex(new(big.Int).SetUint64(height-25).Bytes())
	block := getBlock(context)

	txs := block["transactions"]
	for _, tx := range txs.([]interface{}) {
		info := tx.(map[string]interface{})
		context["txid"] = (info["id"]).(string)
		getTransaction(context)
		getReceipt(context)
	}

	context["from"] = address.Hex()
	context["to"] = qkcToAddr.Recipient.Hex()
	context["amount"] = "0"
	context["price"] = "100000000000"
	context["privateKey"] = common.Bytes2Hex(prvkey.D.Bytes())

	txid := sent(context)
	context["txid"] = txid
	getTransaction(context)
	getReceipt(context)

}

//获取余额
func getBalance(ctx map[string]string) {
	//address := common.HexToAddress(ctx.FormValue("address"))
	balance, err := client.GetBalance(&clt.QkcAddress{common.HexToAddress(ctx["address"]), 0})
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(balance)
}

//获取区块和交易内容
func getBlock(ctx map[string]string) map[string]interface{} {
	heightStr := ctx["height"]
	height := new(big.Int).SetBytes(common.FromHex(heightStr))
	result, err := client.GetRootBlockByHeight(height)
	if err != nil {
		fmt.Println(err.Error())
	}

	headers := result.Result.(map[string]interface{})["minorBlockHeaders"]
	headerIdList := make([]string, 0)
	txList := make([]interface{}, 0)
	for _, h := range headers.([]interface{}) {
		info := h.(map[string]interface{})
		id := (info["id"]).(string)
		headerIdList = append(headerIdList, id)
	}
	fmt.Println("headerIdList len", len(headerIdList))
	for _, headerId := range headerIdList {
		mresult, err := client.GetMinorBlockById(headerId)
		if err != nil {
			fmt.Println(err.Error())
		}
		txs := mresult.Result.(map[string]interface{})["transactions"]
		for _, tx := range txs.([]interface{}) {
			txList = append(txList, tx)
		}
	}
	result.Result.(map[string]interface{})["transactions"] = txList
	fmt.Println("txList len", len(txList))
	fmt.Println(result.Result)
	return result.Result.(map[string]interface{})
}

//获取交易回执
func getReceipt(ctx map[string]string) {
	txid, err := clt.ByteToTransactionId(common.FromHex(ctx["txid"]))
	if err != nil {
		fmt.Println(err.Error())
	}
	result, err := client.GetTransactionReceipt(txid)
	if err != nil {
		fmt.Println("getTransactionReceipt error: ", err.Error())
	}
	fmt.Println(result.Result)
}

func getHeight(ctx map[string]string) uint64 {
	height, err := client.GetRootBlockHeight()
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(height)
	return height
}

func getTransaction(ctx map[string]string) {
	txid, err := clt.ByteToTransactionId(common.FromHex(ctx["txid"]))
	if err != nil {
		fmt.Println(err.Error())
	}
	result, err := client.GetTransactionById(txid)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println("txid", result.Result.(map[string]interface{})["id"])
	fmt.Println(result.Result)
}

func sent(ctx map[string]string) string {
	from := common.HexToAddress(ctx["from"])
	to := common.HexToAddress(ctx["to"])
	amount, _ := new(big.Int).SetString(ctx["amount"], 10)
	gasPrice, _ := new(big.Int).SetString(ctx["price"], 10)
	privateKey := ctx["privateKey"]
	prvkey, _ := crypto.ToECDSA(common.FromHex(privateKey))
	tx, err := client.CreateTransaction(&clt.QkcAddress{from, 0}, &clt.QkcAddress{to, 0}, amount, uint64(30000), gasPrice)
	if err != nil {
		fmt.Println(err.Error())
	}
	tx, err = clt.SignTx(tx, prvkey)
	if err != nil {
		fmt.Println(err.Error())
	}
	txid, err := client.SendTransaction(tx)
	if err != nil {
		fmt.Println("SendTransaction error: ", err.Error())
	}

	fmt.Println(common.Bytes2Hex(txid))
	return common.Bytes2Hex(txid)
}
