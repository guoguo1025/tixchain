package types

import (
	"bytes"
	"time"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
)

func CreateTxs(txs []*Transaction) (*Transactions, error) {
	if len(txs) < 2 {
		return nil, ErrTxGroupCountLessThanTwo
	}
	txgroup := &Transactions{}
	txgroup.Txs = txs
	var header []byte
	for i := len(txs) - 1; i >= 0; i-- {
		hash := txs[i].Hash()
		if i == 0 {
			header = hash
		} else {
			txs[i-1].Next = hash
		}
	}
	for i := 0; i < len(txs); i++ {
		txs[i].Header = header
		txs[i].GroupCount = int32(len(txs))
	}
	//修改txs[0] 的手续费
	totalfee := int64(0)
	minfee := int64(0)
	for i := 0; i < len(txs); i++ {
		totalfee += txs[i].GetFee()
		realfee, err := txs[i].GetRealFee(MinFee)
		if err != nil {
			return nil, err
		}
		minfee += realfee
		txs[i].Fee = 0
	}
	if totalfee < minfee {
		totalfee = minfee
	}
	//总的手续费 = 所有交易的手续费之和
	//除了txs[0], 其他交易手续费设置为0
	txs[0].Fee = totalfee
	return txgroup, nil
}

//这比用于检查的交易，包含了所有的交易。
//主要是为了兼容原来的设计
func (txgroup *Transactions) Tx() *Transaction {
	headtx := txgroup.GetTxs()[0]
	//不会影响原来的tx
	copytx := *headtx
	data := Encode(txgroup)
	//放到header中不影响交易的Hash
	copytx.Header = data
	return &copytx
}

func (txgroup *Transactions) GetTxGroup() *Transactions {
	return txgroup
}

func (txgroup *Transactions) CheckSign() bool {
	txs := txgroup.Txs
	for i := 0; i < len(txs); i++ {
		if !txs[i].checkSign() {
			return false
		}
	}
	return true
}

func (txgroup *Transactions) IsExpire(height, blocktime int64) bool {
	txs := txgroup.Txs
	for i := 0; i < len(txs); i++ {
		if txs[i].isExpire(height, blocktime) {
			return true
		}
	}
	return false
}

func (txgroup *Transactions) Check(minfee int64) error {
	txs := txgroup.Txs
	if len(txs) < 2 {
		return ErrTxGroupCountLessThanTwo
	}
	for i := 0; i < len(txs); i++ {
		if txs[i] == nil {
			return ErrTxGroupEmpty
		}
		err := txs[i].check(0)
		if err != nil {
			return err
		}
	}
	for i := 1; i < len(txs); i++ {
		if txs[i].Fee != 0 {
			return ErrTxGroupFeeNotZero
		}
	}
	//检查txs[0] 的费用是否满足要求
	totalfee := int64(0)
	for i := 0; i < len(txs); i++ {
		fee, err := txs[i].GetRealFee(minfee)
		if err != nil {
			return err
		}
		totalfee += fee
	}
	if txs[0].Fee < totalfee {
		return ErrTxFeeTooLow
	}
	//检查hash是否符合要求
	for i := 0; i < len(txs); i++ {
		//检查头部是否等于头部hash
		if i == 0 {
			if !bytes.Equal(txs[i].Hash(), txs[i].Header) {
				return ErrTxGroupHeader
			}
		} else {
			if !bytes.Equal(txs[0].Header, txs[i].Header) {
				return ErrTxGroupHeader
			}
		}
		//检查group count
		if txs[i].GroupCount > MaxTxGroupSize {
			return ErrTxGroupCountBigThanMaxSize
		}
		if txs[i].GroupCount != int32(len(txs)) {
			return ErrTxGroupCount
		}
		//检查next
		if i < len(txs)-1 {
			if !bytes.Equal(txs[i].Next, txs[i+1].Hash()) {
				return ErrTxGroupNext
			}
		} else {
			if txs[i].Next != nil {
				return ErrTxGroupNext
			}
		}
	}
	return nil
}

type TransactionCache struct {
	*Transaction
	txGroup *Transactions
	hash    []byte
	size    int
	signok  int   //init 0, ok 1, err 2
	checkok error //init 0, ok 1, err 2
	checked bool
}

func NewTransactionCache(tx *Transaction) *TransactionCache {
	return &TransactionCache{Transaction: tx}
}

func (tx *TransactionCache) Hash() []byte {
	if tx.hash == nil {
		tx.hash = tx.Transaction.Hash()
	}
	return tx.hash
}

func (tx *TransactionCache) Size() int {
	if tx.size == 0 {
		tx.size = Size(tx.Tx())
	}
	return tx.size
}

func (tx *TransactionCache) Tx() *Transaction {
	return tx.Transaction
}

func (tx *TransactionCache) Check(minfee int64) error {
	if !tx.checked {
		tx.checked = true
		txs, err := tx.GetTxGroup()
		if err != nil {
			tx.checkok = err
			return err
		}
		if txs == nil {
			tx.checkok = tx.check(minfee)
		} else {
			tx.checkok = txs.Check(minfee)
		}
	}
	return tx.checkok
}

func (tx *TransactionCache) GetTxGroup() (*Transactions, error) {
	var err error
	if tx.txGroup == nil {
		tx.txGroup, err = tx.Transaction.GetTxGroup()
		if err != nil {
			return nil, err
		}
	}
	return tx.txGroup, nil
}

func (tx *TransactionCache) CheckSign() bool {
	if tx.signok == 0 {
		tx.signok = 2
		group, err := tx.GetTxGroup()
		if err != nil {
			return false
		}
		if group == nil {
			//非group，简单校验签名
			if ok := tx.checkSign(); ok {
				tx.signok = 1
			}
		} else {
			if ok := group.CheckSign(); ok {
				tx.signok = 1
			}
		}
	}
	return tx.signok == 1
}

func TxsToCache(txs []*Transaction) (caches []*TransactionCache) {
	caches = make([]*TransactionCache, len(txs))
	for i := 0; i < len(caches); i++ {
		caches[i] = NewTransactionCache(txs[i])
	}
	return caches
}

func CacheToTxs(caches []*TransactionCache) (txs []*Transaction) {
	txs = make([]*Transaction, len(caches))
	for i := 0; i < len(caches); i++ {
		txs[i] = caches[i].Tx()
	}
	return txs
}

//hash 不包含签名，用户通过修改签名无法重新发送交易
func (tx *Transaction) HashSign() []byte {
	copytx := *tx
	copytx.Signature = nil
	data := Encode(&copytx)
	return common.Sha256(data)
}

func (tx *Transaction) Tx() *Transaction {
	return tx
}

func (tx *Transaction) GetTxGroup() (*Transactions, error) {
	if tx.GroupCount < 0 || tx.GroupCount == 1 || tx.GroupCount > 20 {
		return nil, ErrTxGroupCount
	}
	if tx.GroupCount > 0 {
		var txs Transactions
		err := Decode(tx.Header, &txs)
		if err != nil {
			return nil, err
		}
		return &txs, nil
	} else {
		if tx.Next != nil || tx.Header != nil {
			return nil, ErrNomalTx
		}
	}
	return nil, nil
}

//交易的hash不包含header的值，引入tx group的概念后，做了修改
func (tx *Transaction) Hash() []byte {
	copytx := *tx
	copytx.Signature = nil
	copytx.Header = nil
	data := Encode(&copytx)
	return common.Sha256(data)
}

func (tx *Transaction) Size() int {
	return Size(tx)
}

func (tx *Transaction) Sign(ty int32, priv crypto.PrivKey) {
	tx.Signature = nil
	data := Encode(tx)
	pub := priv.PubKey()
	sign := priv.Sign(data)
	tx.Signature = &Signature{ty, pub.Bytes(), sign.Bytes()}
}

//tx 有些时候是一个交易组
func (tx *Transaction) CheckSign() bool {
	group, err := tx.GetTxGroup()
	if err != nil {
		return false
	}
	if group == nil {
		return tx.checkSign()
	}
	return group.CheckSign()
}

//txgroup 的情况
func (tx *Transaction) checkSign() bool {
	copytx := *tx
	copytx.Signature = nil
	data := Encode(&copytx)
	if tx.GetSignature() == nil {
		return false
	}
	return CheckSign(data, tx.GetSignature())
}

func (tx *Transaction) Check(minfee int64) error {
	group, err := tx.GetTxGroup()
	if err != nil {
		return err
	}
	if group == nil {
		return tx.check(minfee)
	}
	return group.Check(minfee)
}

func (tx *Transaction) check(minfee int64) error {
	if !isAllowExecName(tx.Execer) {
		return ErrExecNameNotAllow
	}
	txSize := Size(tx)
	if txSize > int(MaxTxSize) {
		return ErrTxMsgSizeTooBig
	}
	if minfee == 0 {
		return nil
	}
	// 检查交易费是否小于最低值
	realFee := int64(txSize/1000+1) * minfee
	if tx.Fee < realFee {
		return ErrTxFeeTooLow
	}
	return nil
}

func (tx *Transaction) SetExpire(expire time.Duration) {
	if int64(expire) > expireBound {
		if expire < time.Second*120 {
			expire = time.Second * 120
		}
		//用秒数来表示的时间
		tx.Expire = time.Now().Unix() + int64(expire/time.Second)
	} else {
		tx.Expire = int64(expire)
	}
}

func (tx *Transaction) GetRealFee(minFee int64) (int64, error) {
	txSize := Size(tx)
	//如果签名为空，那么加上签名的空间
	if tx.Signature == nil {
		txSize += 300
	}
	if txSize > int(MaxTxSize) {
		return 0, ErrTxMsgSizeTooBig
	}
	// 检查交易费是否小于最低值
	realFee := int64(txSize/1000+1) * minFee
	return realFee, nil
}

var expireBound int64 = 1000000000 // 交易过期分界线，小于expireBound比较height，大于expireBound比较blockTime

func (tx *Transaction) IsExpire(height, blocktime int64) bool {
	group, err := tx.GetTxGroup()
	if err != nil {
		return true
	}
	if group == nil {
		return tx.isExpire(height, blocktime)
	}
	return group.IsExpire(height, blocktime)
}

//检查交易是否过期，过期返回true，未过期返回false
func (tx *Transaction) isExpire(height, blocktime int64) bool {
	valid := tx.Expire
	// Expire为0，返回false
	if valid == 0 {
		return false
	}

	if valid <= expireBound {
		//Expire小于1e9，为height
		if valid > height { // 未过期
			return false
		} else { // 过期
			return true
		}
	} else {
		// Expire大于1e9，为blockTime
		if valid > blocktime { // 未过期
			return false
		} else { // 过期
			return true
		}
	}
}

//解析tx的payload获取amount值
func (tx *Transaction) Amount() (int64, error) {

	if "coins" == string(tx.Execer) {
		var action CoinsAction
		err := Decode(tx.GetPayload(), &action)
		if err != nil {
			return 0, ErrDecode
		}
		if action.Ty == CoinsActionTransfer && action.GetTransfer() != nil {
			transfer := action.GetTransfer()
			return transfer.Amount, nil
		} else if action.Ty == CoinsActionGenesis && action.GetGenesis() != nil {
			gen := action.GetGenesis()
			return gen.Amount, nil
		} else if action.Ty == CoinsActionWithdraw && action.GetWithdraw() != nil {
			transfer := action.GetWithdraw()
			return transfer.Amount, nil
		}
	} else if "ticket" == string(tx.Execer) {
		var action TicketAction
		err := Decode(tx.GetPayload(), &action)
		if err != nil {
			return 0, ErrDecode
		}
		if action.Ty == TicketActionMiner && action.GetMiner() != nil {
			ticketMiner := action.GetMiner()
			return ticketMiner.Reward, nil
		}
	} else if "token" == string(tx.Execer) { //TODO: 补充和完善token和trade分支的amount的计算, added by hzj
		var action TokenAction
		err := Decode(tx.GetPayload(), &action)
		if err != nil {
			return 0, ErrDecode
		}

		if TokenActionPreCreate == action.Ty && action.GetTokenprecreate() != nil {
			precreate := action.GetTokenprecreate()
			return precreate.Price, nil
		} else if TokenActionFinishCreate == action.Ty && action.GetTokenfinishcreate() != nil {
			return 0, nil
		} else if TokenActionRevokeCreate == action.Ty && action.GetTokenrevokecreate() != nil {
			return 0, nil
		} else if ActionTransfer == action.Ty && action.GetTransfer() != nil {
			return 0, nil
		} else if ActionWithdraw == action.Ty && action.GetWithdraw() != nil {
			return 0, nil
		}

	} else if "trade" == string(tx.Execer) {
		var trade Trade
		err := Decode(tx.GetPayload(), &trade)
		if err != nil {
			return 0, ErrDecode
		}

		if TradeSellLimit == trade.Ty && trade.GetTokensell() != nil {
			return 0, nil
		} else if TradeBuyMarket == trade.Ty && trade.GetTokenbuy() != nil {
			return 0, nil
		} else if TradeRevokeSell == trade.Ty && trade.GetTokenrevokesell() != nil {
			return 0, nil
		}
	}
	return 0, nil
}

//获取tx交易的Actionname
func (tx *Transaction) ActionName() string {
	if bytes.Equal(tx.Execer, []byte("coins")) {
		var action CoinsAction
		err := Decode(tx.Payload, &action)
		if err != nil {
			return "unknow-err"
		}
		if action.Ty == CoinsActionTransfer && action.GetTransfer() != nil {
			return "transfer"
		} else if action.Ty == CoinsActionWithdraw && action.GetWithdraw() != nil {
			return "withdraw"
		} else if action.Ty == CoinsActionGenesis && action.GetGenesis() != nil {
			return "genesis"
		}
	} else if bytes.Equal(tx.Execer, []byte("ticket")) {
		var action TicketAction
		err := Decode(tx.Payload, &action)
		if err != nil {
			return "unknow-err"
		}
		if action.Ty == TicketActionGenesis && action.GetGenesis() != nil {
			return "genesis"
		} else if action.Ty == TicketActionOpen && action.GetTopen() != nil {
			return "open"
		} else if action.Ty == TicketActionClose && action.GetTclose() != nil {
			return "close"
		} else if action.Ty == TicketActionMiner && action.GetMiner() != nil {
			return "miner"
		} else if action.Ty == TicketActionBind && action.GetTbind() != nil {
			return "bindminer"
		}
	} else if bytes.Equal(tx.Execer, []byte("none")) {
		return "none"
	} else if bytes.Equal(tx.Execer, []byte("hashlock")) {
		var action HashlockAction
		err := Decode(tx.Payload, &action)
		if err != nil {
			return "unknow-err"
		}
		if action.Ty == HashlockActionLock && action.GetHlock() != nil {
			return "lock"
		} else if action.Ty == HashlockActionUnlock && action.GetHunlock() != nil {
			return "unlock"
		} else if action.Ty == HashlockActionSend && action.GetHsend() != nil {
			return "send"
		}
	} else if bytes.Equal(tx.Execer, []byte("retrieve")) {
		var action RetrieveAction
		err := Decode(tx.Payload, &action)
		if err != nil {
			return "unknow-err"
		}
		if action.Ty == RetrievePre && action.GetPreRet() != nil {
			return "prepare"
		} else if action.Ty == RetrievePerf && action.GetPerfRet() != nil {
			return "perform"
		} else if action.Ty == RetrieveBackup && action.GetBackup() != nil {
			return "backup"
		} else if action.Ty == RetrieveCancel && action.GetCancel() != nil {
			return "cancel"
		}
	} else if bytes.Equal(tx.Execer, []byte("token")) {
		var action TokenAction
		err := Decode(tx.Payload, &action)
		if err != nil {
			return "unknow-err"
		}

		if action.Ty == TokenActionPreCreate && action.GetTokenprecreate() != nil {
			return "preCreate"
		} else if action.Ty == TokenActionFinishCreate && action.GetTokenfinishcreate() != nil {
			return "finishCreate"
		} else if action.Ty == TokenActionRevokeCreate && action.GetTokenrevokecreate() != nil {
			return "revokeCreate"
		} else if action.Ty == ActionTransfer && action.GetTransfer() != nil {
			return "transferToken"
		} else if action.Ty == ActionWithdraw && action.GetWithdraw() != nil {
			return "withdrawToken"
		}
	} else if bytes.Equal(tx.Execer, []byte("trade")) {
		var trade Trade
		err := Decode(tx.Payload, &trade)
		if err != nil {
			return "unknow-err"
		}

		if trade.Ty == TradeSellLimit && trade.GetTokensell() != nil {
			return "selltoken"
		} else if trade.Ty == TradeBuyMarket && trade.GetTokenbuy() != nil {
			return "buytoken"
		} else if trade.Ty == TradeRevokeSell && trade.GetTokenrevokesell() != nil {
			return "revokeselltoken"
		} else if trade.Ty == TradeBuyLimit && trade.GetTokenbuylimit() != nil {
			return "buylimittoken"
		} else if trade.Ty == TradeSellMarket && trade.GetTokensellmarket() != nil {
			return "sellmarkettoken"
		} else if trade.Ty == TradeRevokeBuy && trade.GetTokenrevokebuy() != nil {
			return "revokebuytoken"
		}
	}

	return "unknow"
}