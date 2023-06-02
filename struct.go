package main

import "github.com/boltdb/bolt"

const dbFile = "blockchain.db"
const blocksBucket = "blocks"
const addressChecksumLen = 4

type Block struct {
	Timestamp     int64
	Transactions  []*Transaction //存储交易数据，不再是字符串数据了
	PrevBlockHash []byte
	Nonce         int
	Hash          []byte
	Ip            []string
}

type Blockchain struct {
	Tip []byte   //区块链最后一块的哈希值
	Db  *bolt.DB //数据库
}

type Transaction struct {
	ID   []byte     //交易ID
	Vin  []TxInput  //交易输入，由上次交易输入（可能多个）
	Vout []TxOutput //交易输出，由本次交易产生（可能多个）
}

type TxInput struct {
	Txid []byte //前一笔交易的ID
	Vout int    //前一笔交易在该笔交易所有输出中的索引（一笔交易可能有多个输出，需要有信息指明具体是哪一个）

	Signature []byte //输入数据签名

	//PubKey公钥，是发送者的钱包的公钥，用于解锁输出
	//如果PubKey与所引用的锁定输出的PubKey相同，那么引用的输出就会被解锁，然后被解锁的值就可以被用于产生新的输出
	//如果不正确，前一笔交易的输出就无法被引用在输入中，或者说，也就无法使用这个输出
	//这种机制，保证了用户无法花费其他人的币
	PubKey []byte
}

type TxOutput struct {
	Value int //输出里面存储的“币”

	//锁定输出的公钥（比特币里面是一个脚本，这里是公钥）
	PubKeyHash []byte
}

type BlockchainIterator struct {
	currentHash []byte
	db          *bolt.DB
}
