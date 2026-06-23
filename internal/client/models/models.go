package models

type SingleReturnResultObject struct {
	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
	Result  string `json:"result,omitempty"`
}

type GetAPILimitResult struct {
	CreditsUsed      int    `json:"creditsUsed"`
	CreditsAvailable int    `json:"creditsAvailable"`
	CreditLimit      int    `json:"creditLimit"`
	LimitInterval    string `json:"limitInterval"`
}

type AccountBalance struct {
	Account string `json:"account,omitempty"`
	Balance string `json:"balance"`
}

type GasTrackerOracle struct {
	LastBlock          string `json:"LastBlock"`
	SafeGasPrice       string `json:"SafeGasPrice"`
	ProposeGasPrice    string `json:"ProposeGasPrice,omitempty"`
	StandardGasPrice   string `json:"StandardGasPrice,omitempty"`
	FastGasPrice       string `json:"FastGasPrice"`
	SuggestBaseFee     string `json:"suggestBaseFee"`
	SuggestPriorityFee string `json:"suggestPriorityFee"`
	GasUsedRatio       string `json:"gasUsedRatio"`
}

type EtherPrice struct {
	ETHBTC          string `json:"ethbtc"`
	ETHBTCTimestamp string `json:"ethbtc_timestamp"`
	ETHUSD          string `json:"ethusd"`
	ETHUSDTimestamp string `json:"ethusd_timestamp"`
}

type TransactionStatus struct {
	IsError        string `json:"isError,omitempty"`
	ErrDescription string `json:"errDescription,omitempty"`
	Status         string `json:"status,omitempty"`
}
