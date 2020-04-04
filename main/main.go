package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"runtime"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	clt "github.com/wdwd5wd/AllocationClient/client"

	"encoding/csv"
	"encoding/json"
	"io/ioutil"
	"log"

	"strconv"
	"sync"
	"time"
)

var (
	client       = clt.NewClient("http://13.56.95.27:38391")
	fullShardKey = uint32(0)

	txCount        = 720000
	CPUCount       = 4
	MaxThreadCount = 50
	ShardNum       = 4
	SendingTPS     = ShardNum * 50

	// 负责给账户搬移签名的账户
	ShardTx = Account{
		Address: "0x5dd8509d4f619f126273092308ce36335854ead2",
		Key:     "cc9f0a764f6d42b198c0f316670013b8081b25687364cd327955f08a59cee071",
	}
)

type Account struct {
	Address string `json:"address"`
	Key     string `json:"key"`
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

// 得到交易打包进块的时间，如果是用shard node签名的话此处的account就是shard node
func GetBlockTimeStamp(sendingTx Account, txCount int) {

	for i := 0; i < txCount; i++ {
		_, err := crypto.ToECDSA(common.FromHex(sendingTx.Key))

		if err == nil {
			var timeStampResultTemp []interface{}
			var timeStampIntTemp []int64
			for j := 0; j < ShardNum; j++ {
				txidval, err := clt.ByteToTransactionId(common.FromHex(TransactionID[i][j]))
				if err != nil {
					fmt.Println(err.Error())
				}
				result, err := client.GetTransactionReceipt(txidval)
				if err != nil {
					fmt.Println(err.Error())
				}
				timeStampResultTemp = append(timeStampResultTemp, result.Result.(map[string]interface{})["timestamp"])
				timeStampIntTempTemp, _ := strconv.ParseInt(result.Result.(map[string]interface{})["timestamp"].(string), 0, 64)
				timeStampIntTemp = append(timeStampIntTemp, timeStampIntTempTemp)

			}
			timeStampResult[i] = timeStampResultTemp
			timeStampInt[i] = timeStampIntTemp

			// txidval, err := clt.ByteToTransactionId(common.FromHex(TransactionID[i]))
			// if err != nil {
			// 	fmt.Println(err.Error())
			// }
			// result, err := client.GetTransactionReceipt(txidval)
			// if err != nil {
			// 	fmt.Println(err.Error())
			// }
			// timeStampResult[i] = result.Result.(map[string]interface{})["timestamp"]

			// timeStampInt[i], _ = strconv.ParseInt(timeStampResult[i].(string), 0, 64)
		}
	}

}

var ReturnTime = make([]time.Duration, txCount)

var TransactionID = make([][]string, txCount)

var timeUnix = make([]int64, txCount)

var timeStampResult = make([][]interface{}, txCount)
var timeStampInt = make([][]int64, txCount)

////////////////////////////////////////////////////////////////////////////////////////////////////

// // 用于在多shard上测试load unbalance

// 并行发送交易函数
func sendinBatch(sendingTx Account, receivingTx string, TxNonce int, TxPrice string, wg *sync.WaitGroup, RandNum string, RandNumTo string, i int) {

	// time.Sleep(15 * time.Millisecond)
	// fmt.Println(sendingNum)

	Sendaddress, _ := hexutil.Decode(sendingTx.Address)
	Recaddress, _ := hexutil.Decode(receivingTx)

	// TODO: 所有交易都由node签名？能方便做实验
	prvkey, err := crypto.ToECDSA(common.FromHex(sendingTx.Key))

	if err == nil {
		context := make(map[string]string)
		// addr := account.NewAddress(common.BytesToAddress(address[:20]), binary.BigEndian.Uint32(address[20:]))
		addr := clt.QkcAddress{Recipient: common.BytesToAddress(Sendaddress[:20]), FullShardKey: binary.BigEndian.Uint32(Sendaddress[20:])}
		recaddr := clt.QkcAddress{Recipient: common.BytesToAddress(Recaddress[:20]), FullShardKey: uint32(1)}
		context["address"] = addr.Recipient.Hex()

		RandNumInt, _ := strconv.ParseInt(RandNum, 10, 64)
		RandNumHex := strconv.FormatInt(RandNumInt, 16)
		context["fromFullShardKey"] = "0x" + RandNumHex + "0001"

		RandNumIntTo, _ := strconv.ParseInt(RandNumTo, 10, 64)
		RandNumHexTo := strconv.FormatInt(RandNumIntTo, 16)
		context["toFullShardKey"] = "0x" + RandNumHexTo + "0001"

		// context["fromFullShardKey"] = addr.FullShardKeyToHex()

		// getBalance(&addr)
		// _, qkcToAddr, err := clt.NewAddress(0)
		// if err != nil {
		// 	fmt.Println("NewAddress error: ", err.Error())
		// }

		context["from"] = addr.Recipient.Hex()
		context["to"] = recaddr.Recipient.Hex()
		context["amount"] = "0"
		// context["price"] = "100000000000"
		context["price"] = TxPrice
		// context["toFullShardKey"] = addr.FullShardKeyToHex()

		// 如果是账户搬移操作，则让sharding node签名
		if context["from"] == context["to"] {
			fmt.Println("account migrate tx")
			prvkey, err = crypto.ToECDSA(common.FromHex(ShardTx.Key))

			ShardPubkey, _ := hexutil.Decode(ShardTx.Address)
			fmt.Println("sharding node pubkey (common.Address):", common.BytesToAddress(ShardPubkey))
		}

		context["privateKey"] = common.Bytes2Hex(prvkey.D.Bytes())

		timeUnix[i] = time.Now().Unix()
		txid := SendTx(context, TxNonce)
		// context["txid"] = txid
		// getTransaction(context)
		// getReceipt(context)

		// sent(context)

		TransactionID[i] = txid

	} else {
		fmt.Println(err)
	}

	// runtime.Gosched()
	// quit <- 1

	// 消费完毕则调用 Done，减少需要等待的线程
	wg.Done()
}

func ReadAddLoc(fileName string) [][]string {

	var AddLoc_tocsv [][]string
	// 读取文件
	fs, err := os.Open(fileName)
	if err != nil {
		log.Fatalf("can not open the file, err is %+v", err)
	}
	defer fs.Close()

	r := csv.NewReader(fs)
	//针对大文件，一行一行的读取文件
	for {
		row, err := r.Read()
		if err != nil && err != io.EOF {
			log.Fatalf("can not read, err is %+v", err)
		}
		if err == io.EOF {
			break
		}
		// fmt.Println(row)

		AddLoc_tocsv = append(AddLoc_tocsv, row)

	}

	return AddLoc_tocsv

}

// GenesisTransfer 创世账户为其余普通账户分钱
func GenesisTransfer(Genesis Account, FromAdd string, TxNonce int) {

	// time.Sleep(15 * time.Millisecond)
	// fmt.Println(sendingNum)

	Sendaddress, _ := hexutil.Decode(Genesis.Address)
	Recaddress, _ := hexutil.Decode(FromAdd)

	// TODO: 所有交易都由node签名？能方便做实验
	prvkey, err := crypto.ToECDSA(common.FromHex(Genesis.Key))

	if err == nil {
		context := make(map[string]string)
		// addr := account.NewAddress(common.BytesToAddress(address[:20]), binary.BigEndian.Uint32(address[20:]))
		addr := clt.QkcAddress{Recipient: common.BytesToAddress(Sendaddress[:20]), FullShardKey: binary.BigEndian.Uint32(Sendaddress[20:])}
		recaddr := clt.QkcAddress{Recipient: common.BytesToAddress(Recaddress[:20]), FullShardKey: binary.BigEndian.Uint32(Recaddress[20:])}
		context["address"] = addr.Recipient.Hex()

		context["from"] = addr.Recipient.Hex()
		context["to"] = recaddr.Recipient.Hex()
		context["amount"] = "100000000000000000000"
		// context["price"] = "100000000000"
		context["price"] = "1"
		// context["toFullShardKey"] = addr.FullShardKeyToHex()

		context["privateKey"] = common.Bytes2Hex(prvkey.D.Bytes())

		for index := 0; index < ShardNum; index++ {
			RandNumHex := strconv.FormatInt(int64(index), 16)
			context["fromFullShardKey"] = "0x" + RandNumHex + "0001"
			context["toFullShardKey"] = "0x" + RandNumHex + "0001"

			SendTx(context, TxNonce)
			// context["txid"] = txid
			// getTransaction(context)
			// getReceipt(context)

			// sent(context)
		}

	} else {
		fmt.Println(err)
	}

	// runtime.Gosched()
	// quit <- 1

	// // 消费完毕则调用 Done，减少需要等待的线程
	// wg.Done()

}

// AccountMigration 模拟账户搬移操作
func AccountMigration(sendingTx Account, receivingTx Account, RandNum string, RandNumTo string) {

	// time.Sleep(15 * time.Millisecond)
	// fmt.Println(sendingNum)

	Sendaddress, _ := hexutil.Decode(sendingTx.Address)
	Recaddress, _ := hexutil.Decode(receivingTx.Address)

	// TODO: 所有交易都由node签名？能方便做实验
	prvkey, err := crypto.ToECDSA(common.FromHex(ShardTx.Key))

	if err == nil {
		context := make(map[string]string)
		// addr := account.NewAddress(common.BytesToAddress(address[:20]), binary.BigEndian.Uint32(address[20:]))
		addr := clt.QkcAddress{Recipient: common.BytesToAddress(Sendaddress[:20]), FullShardKey: binary.BigEndian.Uint32(Sendaddress[20:])}
		recaddr := clt.QkcAddress{Recipient: common.BytesToAddress(Recaddress[:20]), FullShardKey: binary.BigEndian.Uint32(Recaddress[20:])}
		context["address"] = addr.Recipient.Hex()

		RandNumInt, _ := strconv.ParseInt(RandNum, 10, 64)
		RandNumHex := strconv.FormatInt(RandNumInt, 16)
		context["fromFullShardKey"] = "0x" + RandNumHex + "0001"

		RandNumIntTo, _ := strconv.ParseInt(RandNumTo, 10, 64)
		RandNumHexTo := strconv.FormatInt(RandNumIntTo, 16)
		context["toFullShardKey"] = "0x" + RandNumHexTo + "0001"

		// context["fromFullShardKey"] = addr.FullShardKeyToHex()

		// getBalance(&addr)
		// _, qkcToAddr, err := clt.NewAddress(0)
		// if err != nil {
		// 	fmt.Println("NewAddress error: ", err.Error())
		// }

		context["from"] = addr.Recipient.Hex()
		context["to"] = recaddr.Recipient.Hex()
		context["amount"] = "0"
		// context["price"] = "100000000000"
		context["price"] = "1000000"
		// context["toFullShardKey"] = addr.FullShardKeyToHex()

		// 如果是账户搬移操作，则让sharding node签名
		if context["from"] == context["to"] {
			fmt.Println("account migration tx")
			prvkey, err = crypto.ToECDSA(common.FromHex(ShardTx.Key))

			// ShardPubkey, _ := hexutil.Decode(ShardTx.Address)
			// fmt.Println("sharding node pubkey (common.Address):", common.BytesToAddress(ShardPubkey))
		}

		context["privateKey"] = common.Bytes2Hex(prvkey.D.Bytes())

		// timeUnix[i] = time.Now().Unix()
		SendMigTx(context)
		// context["txid"] = txid
		// getTransaction(context)
		// getReceipt(context)

		// sent(context)

		// TransactionID[i] = txid

	} else {
		fmt.Println(err)
	}

	// runtime.Gosched()
	// quit <- 1

	// // 消费完毕则调用 Done，减少需要等待的线程
	// wg.Done()
}

// CalculateNonceAll 为所有sender计算每次发送交易的nonce
func CalculateNonceAll(FromAdd []string) []int {

	var NonceAll []int

	for _, value := range FromAdd {

		_, ok := FromAddNonceMap[value]
		if !ok {
			FromAddNonceMap[value] = 0
		}

		NonceAll = append(NonceAll, FromAddNonceMap[value])
		FromAddNonceMap[value] = FromAddNonceMap[value] + 1
	}

	return NonceAll
}

// CalculatePriceAll 为所有sender计算每个交易的gasprice（其实是关乎到其交易被打包进块的时间）
func CalculatePriceAll(FromAdd []string) []string {

	var PriceString []string
	for index := len(FromAdd); index > 0; index-- {
		PriceString = append(PriceString, strconv.Itoa(index))
	}

	return PriceString
}

// 以下是服务器接收相关

func connHandler(c net.Conn) {
	//1.conn是否有效
	if c == nil {
		log.Panic("无效的 socket 连接")
	}

	//2.新建网络数据流存储结构
	buf := make([]byte, 4096)
	//3.循环读取网络数据流
	for {
		//3.1 网络数据流读入 buffer
		cnt, err := c.Read(buf)
		//3.2 数据读尽、读取错误 关闭 socket 连接
		if cnt == 0 || err != nil {
			c.Close()
			break
		}

		//3.3 根据输入流进行逻辑处理
		//buf数据 -> 去两端空格的string
		inStr := strings.TrimSpace(string(buf[0:cnt]))
		//去除 string 内部空格
		cInputs := strings.Split(inStr, " ")
		//获取 客户端输入第一条命令
		NewEPOCH, _ = strconv.Atoi(cInputs[0])

		fmt.Println("客户端传输->", NewEPOCH)

		c.Write([]byte("服务器端回复" + cInputs[0] + "\n"))

		//c.Close() //关闭client端的连接，telnet 被强制关闭

		fmt.Printf("来自 %v 的连接关闭\n", c.RemoteAddr())
	}
}

//开启serverSocket
func ServerSocket() {
	//1.监听端口
	server, err := net.Listen("tcp", ":8087")

	if err != nil {
		fmt.Println("开启socket服务失败")
	}

	fmt.Println("正在开启 Server ...")

	// for {
	// 	//2.接收来自 client 的连接,会阻塞
	// 	conn, err := server.Accept()

	// 	if err != nil {
	// 		fmt.Println("连接出错")
	// 	}

	// 	fmt.Println("caonimabi")

	// 	//并发模式 接收来自客户端的连接请求，一个连接 建立一个 conn，服务器资源有可能耗尽 BIO模式
	// 	go connHandler(conn)

	// }

	go OpenService(server)

}

func OpenService(server net.Listener) {
	for {
		//2.接收来自 client 的连接,会阻塞
		conn, err := server.Accept()

		if err != nil {
			fmt.Println("连接出错")
		}

		fmt.Println("allocation node连接了")

		connHandler(conn)

	}
}

// 批量生成账户
func GenerateAccounts() {

	number := 20000

	account := []map[string]string{}

	for i := 1; i <= number; i++ {
		PrivKey, qkcAddr, err := clt.NewAddress(1)
		if err != nil {
			fmt.Println("NewAddress error: ", err.Error())
		}

		qkcAddrHex := qkcAddr.ToHex()
		PrivKeyHex := common.Bytes2Hex(PrivKey.D.Bytes())

		// fmt.Println(qkcAddrHex)
		// fmt.Println(PrivKeyHex)

		_, err = crypto.ToECDSA(common.FromHex(PrivKeyHex))
		if err == nil {
			accounttemp := map[string]string{
				"address": qkcAddrHex,
				"key":     PrivKeyHex,
			}

			fmt.Println("Adding account: ", i)
			account = append(account, accounttemp)
		} else {
			fmt.Println(err)
		}

	}

	// accounttest := map[string][]map[string]string{
	// 	"nimabi": account,
	// }

	b, err := json.Marshal(account)
	if err != nil {
		fmt.Println("Marshal error:", err)
	}

	err = ioutil.WriteFile("loadtest.json", b, os.ModeAppend)
	if err != nil {
		fmt.Println("Write error:", err)
	}

}

////////////////////////////////////////////////////////////////////////////////////////////////////

// EPOCH 从allocation node处得知的EPOCH轮数
var EPOCH = 1
var NewEPOCH = 1

var FromAddLocMap = make(map[string]string)
var ToAddLocMap = make(map[string]string)

var FromAddNonceMap = make(map[string]int)

// AccountMap 映射以太坊账户到我自己生成的账户
var AccountMap = make(map[string]Account)

func main() {

	// 测试用
	// TestMain()

	// 预先生成账户，只用一次
	// GenerateAccounts()

	ServerSocket()

	// socket.ClientSocket()

	// 读取及解析genesis account json文件
	json := make([]Account, 0)
	NewJson().Load("./loadtest.json", &json)
	// fmt.Println(json[0].Address)

	fmt.Println("End reading genesis account json file")

	// 读取及解析自己生成的account json文件
	GeneratedAccountJSON := make([]Account, 0)
	NewJson().Load("./accounts.json", &GeneratedAccountJSON)

	fmt.Println("End reading generated account json file")

	i := 0
	var from_address []string
	var to_address []string

	// 读取交易信息文件
	fileName := "bq-results-20190905-154154-u51yqnfufcbn.csv"
	fs, err := os.Open(fileName)
	if err != nil {
		log.Fatalf("can not open the file, err is %+v", err)
	}
	defer fs.Close()

	r := csv.NewReader(fs)
	//针对大文件，一行一行的读取文件
	for {
		row, err := r.Read()
		if err != nil && err != io.EOF {
			log.Fatalf("can not read, err is %+v", err)
		}
		if err == io.EOF {
			break
		}
		// fmt.Println(row)

		if i != 0 {
			from_address = append(from_address, row[1])
			to_address = append(to_address, row[2])
		}

		i++

	}

	// 读取address location文件
	FromAddLoc := ReadAddLoc("FromAddLoc_all_mig_Mar29.csv")
	ToAddLoc := ReadAddLoc("ToAddLoc_all_mig_Mar29.csv")

	for index, value := range FromAddLoc {
		FromAddLocMap[value[0]] = value[EPOCH+1]

		FromAddNonceMap[value[0]] = 0

		AccountMap[value[0]] = GeneratedAccountJSON[index]

	}

	for _, value := range ToAddLoc {
		ToAddLocMap[value[0]] = value[EPOCH+1]
	}

	// 计算nonce和price
	NonceAll := CalculateNonceAll(from_address)
	PriceAll := CalculatePriceAll(from_address)

	// 参数设置

	runtime.GOMAXPROCS(CPUCount)

	TPSCount := 0
	tic := time.Now()

	// var accountBatch = 5000

	// 初始化，为每个账户分一些钱
	fmt.Println("创世账户开始分钱")

	// for iter := 0; iter < len(FromAddLoc)/MaxThreadCount; iter++ {

	// // 设置阻塞线程
	// var wg sync.WaitGroup
	// // 设置需要多少个线程阻塞
	// wg.Add(MaxThreadCount)

	for i := 0; i < len(FromAddLoc); i++ {

		fmt.Println(i)

		go GenesisTransfer(json[i/1000], AccountMap[FromAddLoc[i][0]].Address, i%1000)

		// 限制每秒交易次数
		TPSCount++
		if TPSCount == SendingTPS/ShardNum {
			toc := time.Since(tic)
			if toc < 1000*1000*1000 {
				fmt.Println("sleeping")
				time.Sleep((1000*1000*1000 - toc) * time.Nanosecond)
			}

			TPSCount = 0
			tic = time.Now()
		}

	}

	// // 等待所有线程执行完毕的阻塞方法
	// wg.Wait()

	// }
	fmt.Println("创世账户分钱结束")

	// 等待用户响应
	fmt.Scanln()

	// 以下为真正开始发送交易

	fmt.Println("真正交易开始")

	TPSCount = 0
	tic = time.Now()
	var startTime = time.Now()

	for iter := 0; iter < txCount/MaxThreadCount; iter++ {

		// 如果epoch变了，则开始搬移账户
		if EPOCH != NewEPOCH {

			fmt.Println("账户搬移开始")

			// // 设置阻塞线程
			// var wg sync.WaitGroup
			// // 设置需要多少个线程阻塞
			// wg.Add(MaxThreadCount)

			for _, value := range FromAddLoc {
				// 如果epoch变了，则该账户放置位置变化
				if value[EPOCH+1] != value[NewEPOCH+1] {
					go AccountMigration(AccountMap[value[0]], AccountMap[value[0]], value[EPOCH+1], value[NewEPOCH+1])
					FromAddLocMap[value[0]] = value[NewEPOCH+1]

					// // 限制每秒交易次数
					// TPSCount++
					// if TPSCount == SendingTPS {
					// 	toc := time.Since(tic)
					// 	if toc < 1000*1000*1000 {
					// 		fmt.Println("sleeping")
					// 		time.Sleep((1000*1000*1000 - toc) * time.Nanosecond)
					// 	}

					// 	TPSCount = 0
					// 	tic = time.Now()
					// }

					fmt.Println("Migrated account:", AccountMap[value[0]].Address)

				}

			}

			// // 等待所有线程执行完毕的阻塞方法
			// wg.Wait()

			// 更新epoch
			EPOCH = NewEPOCH

			fmt.Println("账户搬移结束")

			// // 等待用户响应
			// fmt.Scanln()

		}

		// 设置阻塞线程
		var wg sync.WaitGroup
		// 设置需要多少个线程阻塞
		wg.Add(MaxThreadCount)

		fmt.Println("iteration:", iter)

		for i := iter * MaxThreadCount; i < (iter+1)*MaxThreadCount; i++ {

			// fmt.Println(i)

			// TODO: to_address的位置也要发生变化

			// 代表from, to账户的位置
			var FromRandNum = FromAddLocMap[from_address[i]]
			var ToRandNum = ToAddLocMap[to_address[i]]

			// go sendinBatch(json[i%accountBatch])
			go sendinBatch(AccountMap[from_address[i]], to_address[i], NonceAll[i], PriceAll[i], &wg, FromRandNum, ToRandNum, i)

			// 限制每秒交易次数
			TPSCount++
			if TPSCount == SendingTPS {
				toc := time.Since(tic)
				if toc < 1000*1000*1000 {
					fmt.Println("sleeping")
					time.Sleep((1000*1000*1000 - toc) * time.Nanosecond)
				}

				TPSCount = 0
				tic = time.Now()
			}

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

	// TODO: 有些交易的toshard不对（在队列里的交易会被搬移）

	GetBlockTimeStamp(ShardTx, txCount)
	// fmt.Println(timeStampInt[99])
	// fmt.Println(timeUnix[99])

	// 计算平均打包时延
	fmt.Println("Start calculating packing delay...")
	var diff = int64(0)
	var count = 0
	for i := 0; i < txCount; i++ {

		_, err := crypto.ToECDSA(common.FromHex(ShardTx.Key))
		for j := 0; j < ShardNum; j++ {
			if err == nil && timeStampInt[i][j]-timeUnix[i] >= 0 {
				diff = timeStampInt[i][j] - timeUnix[i] + diff
				count++
				// fmt.Println(i)
			}
		}
		// if err == nil && timeStampInt[i]-timeUnix[i] >= 0 {
		// 	diff = timeStampInt[i] - timeUnix[i] + diff
		// 	count++
		// 	// fmt.Println(i)
		// }
	}
	diff = diff / int64(count)
	fmt.Println("Count:", count)
	fmt.Println("Average delay:", diff)

}

////////////////////////////////////////////////////////////////////////////////////////////////////

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

	fmt.Println("tx hash:", txidhex)

	var txidtoshard = txidhex[0 : len(txidhex)-8]
	txidtoshard += ctx["toFullShardKey"][2:len(ctx["toFullShardKey"])]
	// fmt.Println(txidtoshard)
	// return common.Bytes2Hex(txid)
	return txidtoshard
}

// SendTx 普通交易调用，创世账户分钱调用
func SendTx(ctx map[string]string, nonce int) []string {
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
	tx, err := client.CreateTransactionWithNonce(nonce, &clt.QkcAddress{Recipient: from, FullShardKey: fromFullShardKey}, &clt.QkcAddress{Recipient: to, FullShardKey: toFullShardKey}, amount, uint64(30000), gasPrice)
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

	txidtoshardSlice := []string{}
	// 由于交易的搬移，交易的hash产生了变化，故自己计算txhash
	for i := 0; i < ShardNum; i++ {
		RandNumHex := strconv.FormatInt(int64(i), 16)
		ctx["fromFullShardKey"] = "0x" + RandNumHex + "0001"
		if _, ok := ctx["fromFullShardKey"]; ok {
			fromFullShardKey = uint32(new(big.Int).SetBytes(common.FromHex(ctx["fromFullShardKey"])).Uint64())
		}
		tx, err := client.CreateTransactionWithNonce(nonce, &clt.QkcAddress{Recipient: from, FullShardKey: fromFullShardKey}, &clt.QkcAddress{Recipient: to, FullShardKey: toFullShardKey}, amount, uint64(30000), gasPrice)
		if err != nil {
			fmt.Println(err.Error())
		}
		tx, err = clt.SignTx(tx, prvkey)
		if err != nil {
			fmt.Println(err.Error())
		}

		TX := &clt.Transaction{
			EvmTx:  tx,
			TxType: clt.EvmTx,
		}

		txid = TX.Hash().Bytes()
		var txidhex = common.Bytes2Hex(txid)
		var txidtoshard = txidhex

		zero := "0"
		for zeroCounter := 1; zeroCounter < 8-len(ctx["toFullShardKey"][2:len(ctx["toFullShardKey"])]); zeroCounter++ {
			zero += "0"
		}
		txidtoshard += zero
		txidtoshard += ctx["toFullShardKey"][2:len(ctx["toFullShardKey"])]
		// fmt.Println(txidtoshard)
		// return common.Bytes2Hex(txid)
		txidtoshardSlice = append(txidtoshardSlice, txidtoshard)

	}
	return txidtoshardSlice

	// // 我改了
	// // 得到to shard 里交易的最终打包时间，以便于之后计算打包延迟
	// var txidhex = common.Bytes2Hex(txid)

	// // fmt.Println("tx hash:", txidhex)

	// var txidtoshard = txidhex[0 : len(txidhex)-8]

	// zero := "0"
	// for zeroCounter := 1; zeroCounter < 8-len(ctx["toFullShardKey"][2:len(ctx["toFullShardKey"])]); zeroCounter++ {
	// 	zero += "0"
	// }
	// txidtoshard += zero
	// txidtoshard += ctx["toFullShardKey"][2:len(ctx["toFullShardKey"])]
	// fmt.Println(txidtoshard)
	// // return common.Bytes2Hex(txid)
	// return txidtoshard
}

// SendMigTx 账户搬移时调用
func SendMigTx(ctx map[string]string) string {
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
	tx, err := client.CreateTransactionGetNonceFromShard(&clt.QkcAddress{Recipient: from, FullShardKey: fromFullShardKey}, &clt.QkcAddress{Recipient: to, FullShardKey: toFullShardKey}, amount, uint64(30000), gasPrice)
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

	fmt.Println("tx hash:", txidhex)

	var txidtoshard = txidhex[0 : len(txidhex)-8]
	txidtoshard += ctx["toFullShardKey"][2:len(ctx["toFullShardKey"])]
	// fmt.Println(txidtoshard)
	// return common.Bytes2Hex(txid)
	return txidtoshard
}

func TestMain() {

	MaxThreadCount = 5
	txCountTest := 15

	// 读取及解析genesis account json文件
	json := make([]Account, 0)
	NewJson().Load("./loadtest.json", &json)
	// fmt.Println(json[0].Address)

	fmt.Println("End reading genesis account json file")

	// 读取及解析自己生成的account json文件
	GeneratedAccountJSON := make([]Account, 0)
	NewJson().Load("./accounts.json", &GeneratedAccountJSON)

	fmt.Println("End reading generated account json file")

	runtime.GOMAXPROCS(CPUCount)

	TPSCount := 0
	tic := time.Now()

	// 初始化，为每个账户分一些钱
	fmt.Println("创世账户开始分钱")
	for iter := 0; iter < 100/MaxThreadCount; iter++ {

		// // 设置阻塞线程
		// var wg sync.WaitGroup
		// // 设置需要多少个线程阻塞
		// wg.Add(MaxThreadCount)

		for i := iter * MaxThreadCount; i < (iter+1)*MaxThreadCount; i++ {
			fmt.Println(i)

			go GenesisTransfer(json[i/1000], GeneratedAccountJSON[i].Address, i%1000)

			// 限制每秒交易次数
			TPSCount++
			if TPSCount == SendingTPS/ShardNum {
				toc := time.Since(tic)
				if toc < 1000*1000*1000 {
					fmt.Println("sleeping")
					time.Sleep((1000*1000*1000 - toc) * time.Nanosecond)
				}

				TPSCount = 0
				tic = time.Now()
			}

		}

		// // 等待所有线程执行完毕的阻塞方法
		// wg.Wait()

	}
	fmt.Println("创世账户分钱结束")

	// // 等待用户响应
	// fmt.Scanln()

	time.Sleep(30 * time.Second)

	// 以下为真正开始发送交易

	fmt.Println("真正交易开始")

	TPSCount = 0
	tic = time.Now()
	var startTime = time.Now()
	migrated := false

	for iter := 0; iter < txCountTest/MaxThreadCount; iter++ {

		// 如果epoch变了，则开始搬移账户
		if iter == 2 {

			// time.Sleep(20 * time.Second)

			fmt.Println("账户搬移开始")

			// // 设置阻塞线程
			// var wg sync.WaitGroup
			// // 设置需要多少个线程阻塞
			// wg.Add(MaxThreadCount)

			for index := 0; index < 4; index++ {

				go AccountMigration(GeneratedAccountJSON[index], GeneratedAccountJSON[index], "0", "2")

				// // 限制每秒交易次数
				// TPSCount++
				// if TPSCount == SendingTPS {
				// 	toc := time.Since(tic)
				// 	if toc < 1000*1000*1000 {
				// 		fmt.Println("sleeping")
				// 		time.Sleep((1000*1000*1000 - toc) * time.Nanosecond)
				// 	}

				// 	TPSCount = 0
				// 	tic = time.Now()
				// }

			}

			// // 等待所有线程执行完毕的阻塞方法
			// wg.Wait()

			// 更新epoch
			EPOCH = NewEPOCH

			migrated = true

			fmt.Println("账户搬移结束")

			// // 等待用户响应
			// fmt.Scanln()

		}

		// 设置阻塞线程
		var wg sync.WaitGroup
		// 设置需要多少个线程阻塞
		wg.Add(MaxThreadCount)

		fmt.Println("iteration:", iter)

		for i := iter * MaxThreadCount; i < (iter+1)*MaxThreadCount; i++ {

			// fmt.Println(i)

			// TODO: to_address的位置也要发生变化

			// 代表from, to账户的位置
			var FromRandNum = "0"
			var ToRandNum = "1"

			if migrated && i%MaxThreadCount < 4 {
				FromRandNum = "2"
			}

			// go sendinBatch(json[i%accountBatch])
			go sendinBatch(GeneratedAccountJSON[i%MaxThreadCount], GeneratedAccountJSON[i%MaxThreadCount+MaxThreadCount].Address, iter, "1", &wg, FromRandNum, ToRandNum, i)

			// 限制每秒交易次数
			TPSCount++
			if TPSCount == SendingTPS {
				toc := time.Since(tic)
				if toc < 1000*1000*1000 {
					fmt.Println("sleeping")
					time.Sleep((1000*1000*1000 - toc) * time.Nanosecond)
				}

				TPSCount = 0
				tic = time.Now()
			}

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

	// TODO: 有些交易的toshard不对（在队列里的交易会被搬移）

	GetBlockTimeStamp(ShardTx, txCountTest)
	// fmt.Println(timeStampInt[99])
	// fmt.Println(timeUnix[99])

	// 计算平均打包时延
	fmt.Println("Start calculating packing delay...")
	var diff = int64(0)
	var count = 0
	for i := 0; i < txCountTest; i++ {

		_, err := crypto.ToECDSA(common.FromHex(ShardTx.Key))
		for j := 0; j < ShardNum; j++ {
			if err == nil && timeStampInt[i][j]-timeUnix[i] >= 0 {
				diff = timeStampInt[i][j] - timeUnix[i] + diff
				count++
				// fmt.Println(i)
			}
		}
		// if err == nil && timeStampInt[i]-timeUnix[i] >= 0 {
		// 	diff = timeStampInt[i] - timeUnix[i] + diff
		// 	count++
		// 	// fmt.Println(i)
		// }
	}
	diff = diff / int64(count)
	fmt.Println("Count:", count)
	fmt.Println("Average delay:", diff)

	fmt.Scanln()

}
