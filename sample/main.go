package main

import (
	"encoding/binary"
	"fmt"
	clt "github.com/QuarkChain/goqkcclient/client"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"

	"io/ioutil"
	"encoding/json"
	"encoding/csv"
	"log"
	"strings"
	
	"time"
	"runtime"
	"sync"
	"strconv"

)

var (
	client       = clt.NewClient("http://13.56.95.27:38391")
	fullShardKey = uint32(0)

	txCount = 40000
	CPUCount = 4
	MaxThreadCount = 4

)

type Account struct {
    Address  string `json:"address"`
    Key string `json:"key"`
}

type Parameter struct {
    Para []Account
}

//定义一个json空结构,为了实现一个方法
type JsonStruct struct {
}
//实现一个加载方法
func (js JsonStruct) Load(filename string, v interface{}) {
    data, err := ioutil.ReadFile(filename)
    if err != nil {
        return
    }
    json.Unmarshal([]byte(data), v)
}
//使用函数灵活调用方法
func NewJson() *JsonStruct {
    return &JsonStruct{}
}

// 设置并发channel
// var quit chan int = make(chan int, 100)

// 得到交易打包进块的时间
func GetBlockTimeStamp(sendingTx []Account, txCount int) {

	for i := 0; i < txCount; i++ {
		_, err := crypto.ToECDSA(common.FromHex(sendingTx[i].Key))

		if err == nil {
			txidval, err := clt.ByteToTransactionId(common.FromHex(TransactionID[i]))
			if err != nil {
				fmt.Println(err.Error())
			}
			result, err := client.GetTransactionReceipt(txidval)
			if err != nil {
				fmt.Println(err.Error())
			}
			timeStampResult[i] = result.Result.(map[string]interface{})["timestamp"]

			timeStampInt[i], _ = strconv.ParseInt(timeStampResult[i].(string), 0, 64)
		}
	}

}

var ReturnTime = make([]time.Duration, txCount)

var TransactionID = make([]string, txCount)

var timeUnix = make([]int64, txCount)

var timeStampResult = make([]interface{}, txCount)
var timeStampInt = make([]int64, txCount)



////////////////////////////////////////////////////////////////////////////////////////////////////

// // 用于在多shard上测试load unbalance

// 并行发送交易函数
func sendinBatch(sendingTx Account, wg *sync.WaitGroup, RandNum string, i int) {
	
	time.Sleep(15 * time.Millisecond)
	// fmt.Println(sendingNum)

	address, _ := hexutil.Decode(sendingTx.Address)
	prvkey, err := crypto.ToECDSA(common.FromHex(sendingTx.Key))
	
	if err == nil {
		context := make(map[string]string)
		// addr := account.NewAddress(common.BytesToAddress(address[:20]), binary.BigEndian.Uint32(address[20:]))
		addr := clt.QkcAddress{Recipient: common.BytesToAddress(address[:20]), FullShardKey: binary.BigEndian.Uint32(address[20:])}
		context["address"] = addr.Recipient.Hex()

		// 假设有4个shard，不同的随机数代表account在不同的shard上发送交易
		if RandNum == "1" {
			context["fromFullShardKey"] = "0x00000001"
			context["toFullShardKey"] = "0x00000001"
		}
		if RandNum == "2" {
			context["fromFullShardKey"] = "0x00010001"
			context["toFullShardKey"] = "0x00000001"
		}
		if RandNum == "3" {
			context["fromFullShardKey"] = "0x00020001"
			context["toFullShardKey"] = "0x00000001"
		}
		if RandNum == "4" {
			context["fromFullShardKey"] = "0x00030001"
			context["toFullShardKey"] = "0x00000001"
		}


		// context["fromFullShardKey"] = addr.FullShardKeyToHex()

		// getBalance(&addr)
		// _, qkcToAddr, err := clt.NewAddress(0)
		// if err != nil {
		// 	fmt.Println("NewAddress error: ", err.Error())
		// }
	
		context["from"] = addr.Recipient.Hex()
		context["to"] = addr.Recipient.Hex()
		context["amount"] = "0"
		context["price"] = "100000000000"
		// context["toFullShardKey"] = addr.FullShardKeyToHex()
		context["privateKey"] = common.Bytes2Hex(prvkey.D.Bytes())
	
		timeUnix[i] = time.Now().Unix()
		txid := sent(context)
		// context["txid"] = txid
		// getTransaction(context)
		// getReceipt(context)
		
		// sent(context)

		TransactionID[i] = txid

	}

	// runtime.Gosched()
	// quit <- 1

	// 消费完毕则调用 Done，减少需要等待的线程
	wg.Done()
}



func main() {

	// 读取及解析json文件
	json := make([]Account,0)
    NewJson().Load("./loadtest.json", &json)
    // fmt.Println(json[0].Address)

	fmt.Println("End reading json")

	// 读取用于生成随机数的csv文件
	dat, err := ioutil.ReadFile("testRandNumGen.csv")
    if err != nil {
        log.Fatal(err)
    }
	r := csv.NewReader(strings.NewReader(string(dat[:])))
    
	record, err := r.ReadAll()
	// if err == io.EOF {
	//     break
	// }
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println(record[0][0])
    

	// 参数设置

	runtime.GOMAXPROCS(CPUCount)

	// var accountBatch = 5000

	var startTime = time.Now()

	for iter := 0; iter < txCount/MaxThreadCount; iter++ {

		// 设置阻塞线程
		var wg sync.WaitGroup
		// 设置需要多少个线程阻塞
		wg.Add(MaxThreadCount)

		for i := iter*MaxThreadCount; i < (iter+1)*MaxThreadCount; i++{

			fmt.Println(i)
			
			// 生成的随机数，列代表生成随机数的类型，0是均匀分布，1是zipf分布
			var RandNum = record[i][0]

			// go sendinBatch(json[i%accountBatch])
			go sendinBatch(json[i], &wg, RandNum, i)

			// time.Sleep(10 * time.Millisecond)
		
		}

		// for i := 0; i < txCount; i++ {
		//     quit <- i
		//     // fmt.Println("sc:", sc, i)
		// }
		// close(quit)

		// 等待所有线程执行完毕的阻塞方法
		wg.Wait()

		// time.Sleep(10 * time.Second)

	}

	var endTime = time.Since(startTime)
	fmt.Println("Total running time: ", endTime)

	// 等待用户响应以后再获取每个transaction打包进块的时间
	fmt.Scanln()

	fmt.Println("Start calculating block time stamp...")
	GetBlockTimeStamp(json, txCount)
	// fmt.Println(timeStampInt[99])
	// fmt.Println(timeUnix[99])

	// 计算平均打包时延
	fmt.Println("Start calculating packing delay...")
	var diff = int64(0)
	var count = 0
	for i := 0; i < txCount; i++ {

		_, err := crypto.ToECDSA(common.FromHex(json[i].Key))

		if err == nil {
			diff = timeStampInt[i] - timeUnix[i] + diff
			count++
			// fmt.Println(i)
		}
	}
	diff = diff/int64(count)
	fmt.Println(diff)

}



////////////////////////////////////////////////////////////////////////////////////////////////////

// // 用于单shard上测试交易打包进块的延时

// var ReturnTime = make([]time.Duration, txCount)

// var TransactionID = make([]string, txCount)

// var timeUnix = make([]int64, txCount)

// var timeStampResult = make([]interface{}, txCount)
// var timeStampInt = make([]int64, txCount)

// // 并行发送交易函数
// func sendinBatch(sendingTx Account, wg *sync.WaitGroup, i int) {
// 	// time.Sleep(1 * time.Second)
// 	// fmt.Println(sendingNum)

// 	address, _ := hexutil.Decode(sendingTx.Address)
// 	prvkey, err := crypto.ToECDSA(common.FromHex(sendingTx.Key))
	
// 	if err == nil {
// 		context := make(map[string]string)
// 		// addr := account.NewAddress(common.BytesToAddress(address[:20]), binary.BigEndian.Uint32(address[20:]))
// 		addr := clt.QkcAddress{Recipient: common.BytesToAddress(address[:20]), FullShardKey: binary.BigEndian.Uint32(address[20:])}
// 		context["address"] = addr.Recipient.Hex()
// 		context["fromFullShardKey"] = addr.FullShardKeyToHex()

// 		// getBalance(&addr)
// 		// _, qkcToAddr, err := clt.NewAddress(0)
// 		// if err != nil {
// 		// 	fmt.Println("NewAddress error: ", err.Error())
// 		// }
	
// 		context["from"] = addr.Recipient.Hex()
// 		context["to"] = addr.Recipient.Hex()
// 		context["amount"] = "0"
// 		context["price"] = "100000000000"
// 		context["toFullShardKey"] = addr.FullShardKeyToHex()
// 		context["privateKey"] = common.Bytes2Hex(prvkey.D.Bytes())
	
// 		timeUnix[i] = time.Now().Unix()
// 		txid := sent(context)
// 		// context["txid"] = txid
// 		// getTransaction(context)
// 		// getReceipt(context)

// 		// sent(context)

// 		TransactionID[i] = txid

// 	}

// 	// runtime.Gosched()
// 	// quit <- 1

// 	// 消费完毕则调用 Done，减少需要等待的线程
// 	wg.Done()
// }



// func main() {

// 	// 读取及解析json文件
// 	json := make([]Account,0)
//     NewJson().Load("./loadtest.json", &json)
//     // fmt.Println(json[0].Address)

// 	fmt.Println("End reading json")

// 	// 参数设置
	
// 	runtime.GOMAXPROCS(CPUCount)

// 	// var accountBatch = 5000

// 	var startTime = time.Now()

// 	for iter := 0; iter < txCount/MaxThreadCount; iter++ {

// 		// 设置阻塞线程
// 		var wg sync.WaitGroup
// 		// 设置需要多少个线程阻塞
// 		wg.Add(MaxThreadCount)

// 		for i := iter*MaxThreadCount; i < (iter+1)*MaxThreadCount; i++{

// 			fmt.Println(i)

// 			// go sendinBatch(json[i%accountBatch])
// 			go sendinBatch(json[i], &wg, i)

// 			// time.Sleep(1 * time.Second)
		
// 		}

// 		// for i := 0; i < txCount; i++ {
// 		//     quit <- i
// 		//     // fmt.Println("sc:", sc, i)
// 		// }
// 		// close(quit)

// 		// 等待所有线程执行完毕的阻塞方法
// 		wg.Wait()

// 		// time.Sleep(10 * time.Second)

// 	}

// 	var endTime = time.Since(startTime)
// 	fmt.Println("Total running time: ", endTime)

// 	// 等待用户响应以后再获取每个transaction打包进块的时间
// 	fmt.Scanln()

// 	GetBlockTimeStamp(json, txCount)
// 	// fmt.Println(timeStampInt[99])
// 	// fmt.Println(timeUnix[99])

// 	// 计算平均打包时延
// 	var diff = int64(0)
// 	var count = 0
// 	for i := 0; i < txCount; i++ {

// 		_, err := crypto.ToECDSA(common.FromHex(json[i].Key))

// 		if err == nil {
// 			diff = timeStampInt[i] - timeUnix[i] + diff
// 			count++
// 		}
// 	}
// 	diff = diff/int64(count)
// 	fmt.Println(diff)

// }






//获取余额
func getBalance(addr *clt.QkcAddress) {
	//address := common.HexToAddress(ctx.FormValue("address"))
	balance, err := client.GetBalance(addr)
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
	fromFullShardKey := fullShardKey
	if _, ok := ctx["fromFullShardKey"]; ok {
		fromFullShardKey = uint32(new(big.Int).SetBytes(common.FromHex(ctx["fromFullShardKey"])).Uint64())
	}
	toFullShardKey := fullShardKey
	if _, ok := ctx["toFullShardKey"]; ok {
		toFullShardKey = uint32(new(big.Int).SetBytes(common.FromHex(ctx["toFullShardKey"])).Uint64())
	}
	// 我改了
	// 自己定gas limit和nonce
	// gas limit无data时42000(30000)，有data时105000(不一定，和data长度有关)
	tx, err := client.CreateTransaction(&clt.QkcAddress{Recipient: from, FullShardKey: fromFullShardKey}, &clt.QkcAddress{Recipient: to, FullShardKey: toFullShardKey}, amount, uint64(30000), gasPrice)
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

	// 我改了
	// 得到to shard 里交易的最终打包时间，以便于之后计算打包延迟
	var txidhex = common.Bytes2Hex(txid)
	var txidtoshard = txidhex[0:len(txidhex)-8]
	txidtoshard += ctx["toFullShardKey"][2:len(ctx["toFullShardKey"])]
	// fmt.Println(txidtoshard)
	// return common.Bytes2Hex(txid)
	return txidtoshard
}
