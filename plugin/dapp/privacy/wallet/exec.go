package wallet

import (
	"gitlab.33.cn/chain33/chain33/types"
)

func (policy *privacyPolicy) On_ShowPrivacyAccountSpend(req *types.ReqPrivBal4AddrToken) (interface{}, error) {
	policy.getWalletOperate().GetMutex().Lock()
	defer policy.getWalletOperate().GetMutex().Unlock()
	reply, err := policy.showPrivacyAccountsSpend(req)
	if err != nil {
		bizlog.Error("showPrivacyAccountsSpend", "err", err.Error())
	}
	return reply, err
}

func (policy *privacyPolicy) On_ShowPrivacyPK(req *types.ReqString) (interface{}, error) {
	policy.getWalletOperate().GetMutex().Lock()
	defer policy.getWalletOperate().GetMutex().Unlock()
	reply, err := policy.showPrivacyKeyPair(req)
	if err != nil {
		bizlog.Error("showPrivacyKeyPair", "err", err.Error())
	}
	return reply, err
}

func (policy *privacyPolicy) On_Public2Privacy(req *types.ReqPub2Pri) (interface{}, error) {
	policy.getWalletOperate().GetMutex().Lock()
	defer policy.getWalletOperate().GetMutex().Unlock()
	reply, err := policy.sendPublic2PrivacyTransaction(req)
	if err != nil {
		bizlog.Error("sendPublic2PrivacyTransaction", "err", err.Error())
	}
	return reply, err
}

func (policy *privacyPolicy) On_Privacy2Privacy(req *types.ReqPri2Pri) (interface{}, error) {
	policy.getWalletOperate().GetMutex().Lock()
	defer policy.getWalletOperate().GetMutex().Unlock()
	reply, err := policy.sendPrivacy2PrivacyTransaction(req)
	if err != nil {
		bizlog.Error("sendPrivacy2PrivacyTransaction", "err", err.Error())
	}
	return reply, err
}

func (policy *privacyPolicy) On_Privacy2Public(req *types.ReqPri2Pub) (interface{}, error) {
	policy.getWalletOperate().GetMutex().Lock()
	defer policy.getWalletOperate().GetMutex().Unlock()
	reply, err := policy.sendPrivacy2PublicTransaction(req)
	if err != nil {
		bizlog.Error("sendPrivacy2PublicTransaction", "err", err.Error())
	}
	return reply, err
}

func (policy *privacyPolicy) On_CreateUTXOs(req *types.ReqCreateUTXOs) (interface{}, error) {
	policy.getWalletOperate().GetMutex().Lock()
	defer policy.getWalletOperate().GetMutex().Unlock()
	reply, err := policy.createUTXOs(req)
	if err != nil {
		bizlog.Error("createUTXOs", "err", err.Error())
	}
	return reply, err
}

func (policy *privacyPolicy) On_CreateTransaction(req *types.ReqCreateTransaction) (interface{}, error) {
	ok, err := policy.getWalletOperate().CheckWalletStatus()
	if !ok {
		bizlog.Error("createTransaction", "CheckWalletStatus cause error.", err)
		return nil, err
	}
	if ok, err := policy.isRescanUtxosFlagScaning(); ok {
		bizlog.Error("createTransaction", "isRescanUtxosFlagScaning cause error.", err)
		return nil, err
	}
	if !checkAmountValid(req.Amount) {
		err = types.ErrAmount
		bizlog.Error("createTransaction", "isRescanUtxosFlagScaning cause error.", err)
		return nil, err
	}
	policy.getWalletOperate().GetMutex().Lock()
	defer policy.getWalletOperate().GetMutex().Unlock()

	reply, err := policy.createTransaction(req)
	if err != nil {
		bizlog.Error("createTransaction", "err", err.Error())
	}
	return reply, err
}

func (policy *privacyPolicy) On_PrivacyAccountInfo(req *types.ReqPPrivacyAccount) (interface{}, error) {
	policy.getWalletOperate().GetMutex().Lock()
	defer policy.getWalletOperate().GetMutex().Unlock()

	reply, err := policy.getPrivacyAccountInfo(req)
	if err != nil {
		bizlog.Error("getPrivacyAccountInfo", "err", err.Error())
	}
	return reply, err
}

func (policy *privacyPolicy) On_PrivacyTransactionList(req *types.ReqPrivacyTransactionList) (interface{}, error) {
	if req.Direction != 0 && req.Direction != 1 {
		bizlog.Error("getPrivacyTransactionList", "invalid direction ", req.Direction)
		return nil, types.ErrInvalidParam
	}
	// convert to sendTx / recvTx
	sendRecvFlag := req.SendRecvFlag + sendTx
	if sendRecvFlag != sendTx && sendRecvFlag != recvTx {
		bizlog.Error("getPrivacyTransactionList", "invalid sendrecvflag ", req.SendRecvFlag)
		return nil, types.ErrInvalidParam
	}
	req.SendRecvFlag = sendRecvFlag

	policy.getWalletOperate().GetMutex().Lock()
	defer policy.getWalletOperate().GetMutex().Unlock()

	reply, err := policy.store.getWalletPrivacyTxDetails(req)
	if err != nil {
		bizlog.Error("getWalletPrivacyTxDetails", "err", err.Error())
	}
	return reply, err
}

func (policy *privacyPolicy) On_RescanUtxos(req *types.ReqRescanUtxos) (interface{}, error) {
	policy.getWalletOperate().GetMutex().Lock()
	defer policy.getWalletOperate().GetMutex().Unlock()
	reply, err := policy.rescanUTXOs(req)
	if err != nil {
		bizlog.Error("rescanUTXOs", "err", err.Error())
	}
	return reply, err
}

func (policy *privacyPolicy) On_EnablePrivacy(req *types.ReqEnablePrivacy) (interface{}, error) {
	operater := policy.getWalletOperate()
	operater.GetMutex().Lock()
	defer operater.GetMutex().Unlock()
	reply, err := policy.enablePrivacy(req)
	if err != nil {
		bizlog.Error("enablePrivacy", "err", err.Error())
	}
	return reply, err
}
