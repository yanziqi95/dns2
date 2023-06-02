package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"github.com/boltdb/bolt"
	"golang.org/x/crypto/ripemd160"
	"log"
	"math/big"
	"os"
)

func getBalance(address string) int {
	if !ValidateAddress(address) {
		log.Panic("ERROR: 地址非法")
	}
	bc := NewBlockchain()
	defer bc.Db.Close()

	balance := 0
	pubKeyHash := Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	UTXOs := bc.FindUTXO(pubKeyHash)

	for _, out := range UTXOs {
		balance += out.Value
	}

	fmt.Printf("'%s'的账号余额是: %d\n", address, balance)
	return balance
}

func ValidateAddress(address string) bool {
	pubKeyHash := Base58Decode([]byte(address))
	actualChecksum := pubKeyHash[len(pubKeyHash)-addressChecksumLen:]
	version := pubKeyHash[0]
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-addressChecksumLen]
	targetChecksum := checksum(append([]byte{version}, pubKeyHash...))

	return bytes.Compare(actualChecksum, targetChecksum) == 0
}

func checksum(payload []byte) []byte {
	firstSHA := sha256.Sum256(payload)
	secondSHA := sha256.Sum256(firstSHA[:])

	return secondSHA[:addressChecksumLen]
}

var b58Alphabet = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")

func Base58Decode(input []byte) []byte {
	result := big.NewInt(0)
	zeroBytes := 0

	for b := range input {
		if b == 0x00 {
			zeroBytes++
		}
	}

	payload := input[zeroBytes:]
	for _, b := range payload {
		charIndex := bytes.IndexByte(b58Alphabet, b)
		result.Mul(result, big.NewInt(58))
		result.Add(result, big.NewInt(int64(charIndex)))
	}

	decoded := result.Bytes()
	decoded = append(bytes.Repeat([]byte{byte(0x00)}, zeroBytes), decoded...)

	return decoded
}

func NewBlockchain() *Blockchain {
	if dbExist() == false {
		fmt.Println("区块链不存在，请首先创建一个新的.")
		os.Exit(1)
	}

	var tip []byte
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket)) //通过名称获得bucket
		tip = b.Get([]byte("1"))             //获得最后区块的哈希

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	bc := Blockchain{Tip: tip, Db: db}

	return &bc
}

func dbExist() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}
	return true
}

func (bc *Blockchain) FindUTXO(pubKeyHash []byte) []TxOutput {
	var UTXOs []TxOutput
	unspentTransactions := bc.FindUnspentTransaction(pubKeyHash)
	for _, tx := range unspentTransactions {
		for _, out := range tx.Vout {
			//FindUnspentTransaction已经做了检查，保险起见，这里做一个冗余检查
			if out.IsLockedWithKey(pubKeyHash) {
				UTXOs = append(UTXOs, out)
			}
		}
	}

	return UTXOs
}

func (out *TxOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Compare(out.PubKeyHash, pubKeyHash) == 0
}

func (bc *Blockchain) FindUnspentTransaction(pubKeyHash []byte) []Transaction {
	var unspentTXs []Transaction //未花费交易

	//已花费输出，key是转化为字符串的当前交易的ID
	//value是该交易包含的引用输出的所有已花费输出值数组
	//一个交易可能有多个输出，在这种情况下，该交易将引用所有的输出：输出不可分规则，无法引用它的一部分，要么不用，要么一次性用完
	//在go中，映射的值可以是数组，所以这里创建一个映射来存储未花费输出
	spentTXOs := make(map[string][]int)

	//从区块链中取得所有已花费输出
	bci := bc.Iterator()
	for { //第一层循环，对区块链中的所有区块进行迭代查询
		block := bci.Next()

		for _, tx := range block.Transactions { //第二层循环，对单个区块中的所有交易进行循环：一个区块可能打包了多个交易
			//检查交易的输入，将所有可以解锁的引用的输出加入到已花费输出map中
			if tx.IsCoinbase() == false { //不适用于创始区块的交易，因为它没有引用输出
				for _, in := range tx.Vin { //第三层循环，对单个交易中的所有输入进行循环（一个交易可能有多个输入）
					if in.UsesKey(pubKeyHash) { //可以被pubKeyHash解锁，即属于pubKeyHash发起的交易（sender）
						inTxID := hex.EncodeToString(in.Txid)
						//in.Vout为引用输出在该交易所有输出中的一个索引
						spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Vout) //加入到已花费映射之中
					}
				}
			}
		}
		if len(block.PrevBlockHash) == 0 { //创始区块都检查完了，退出最外层循环
			break
		}
	}

	//获得未花费交易
	bci = bc.Iterator()
	for { //第一层循环，对区块链中的所有区块进行迭代查询
		block := bci.Next()

		for _, tx := range block.Transactions { //第二层循环，对单个区块中的所有交易进行循环：一个区块可能打包了多个交易
			txID := hex.EncodeToString(tx.ID) //交易ID转为字符串，便于比较

		Outputs:
			for outIdx, out := range tx.Vout { //第三层循环，对单个交易中的所有输出进行循环（一个交易可能有多个输出）
				//检查交易的输出，OutIdx为数组序号，实际上也是某个TxOutput的索引，out为TxOutput
				//一个交易，可能会有多个输出
				//输出是否已经花费了？
				if spentTXOs[txID] != nil {
					for _, spentOut := range spentTXOs[txID] { //第四层循环，对前面获得的所有未花费输出进行循环，spentOut是value
						//根据输出引用不可再分规则，
						//只要有一个输出值被引用，那么该输出的所有值都被引用了
						//所以通过比较索引值，只要发现一个输出值被引用了，就不必查询下一个输出值了
						//说明该输出已经被引用（被包含在其它交易的输入之中，即被花费掉了）
						if spentOut == outIdx {
							continue Outputs //在 continue 语句后添加标签Outputs时，表示开始标签Outputs对应的循环
						}
					}
				}

				//输出没有被花费，且由pubKeyHash锁定（即归pubKeyHash用户所有）
				if out.IsLockedWithKey(pubKeyHash) {
					unspentTXs = append(unspentTXs, *tx) //将tx值加入到已花费交易数组中
				}
			}

		}
		if len(block.PrevBlockHash) == 0 { //创始区块都检查完了，退出最外层循环
			break
		}
	}
	return unspentTXs
}

func (bc *Blockchain) Iterator() *BlockchainIterator {
	bci := &BlockchainIterator{bc.Tip, bc.Db}
	return bci
}

func (i *BlockchainIterator) Next() *Block {
	var block *Block

	err := i.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		encodeBlock := b.Get(i.currentHash)
		block = DeserializeBlock(encodeBlock)

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	i.currentHash = block.PrevBlockHash

	return block
}

func DeserializeBlock(d []byte) *Block {
	var block Block //一般都不会通过指针来创建一个struct。记住struct是一个值类型

	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&block)
	if err != nil {
		log.Panic(err)
	}

	return &block //返回block的引用
}

func (tx Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vin[0].Vout == -1
}

func (in *TxInput) UsesKey(pubKeyHash []byte) bool {
	//注意输入中的公钥是来自于钱包中的公钥，是原生的公钥
	//而引用的输出中的公钥是哈希后的公钥
	lockingHash := HashPubKey(in.PubKey)

	return bytes.Compare(lockingHash, pubKeyHash) == 0
}

func HashPubKey(pubKey []byte) []byte {
	publicSHA256 := sha256.Sum256(pubKey)

	RIPEMD160Hasher := ripemd160.New()
	_, err := RIPEMD160Hasher.Write(publicSHA256[:])
	if err != nil {
		log.Panic(err)
	}
	publicRIPEMD160 := RIPEMD160Hasher.Sum(nil)

	return publicRIPEMD160
}
