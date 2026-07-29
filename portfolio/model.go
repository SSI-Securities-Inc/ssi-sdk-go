package portfolio

import (
	"fmt"

	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/util"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/trading"
)

// AccountBalanceRequest is the account balance request.
type AccountBalanceRequest struct {
	ClientID  string `json:"clientId"`
	AccountNo string `json:"accountNo,omitempty"`
}

func (r *AccountBalanceRequest) ToMap() map[string]string {
	params := map[string]string{"clientId": r.ClientID}
	if r.AccountNo != "" {
		params["accountNo"] = r.AccountNo
	}
	return params
}

// EquityAccountBalance is equity account balance information.
type EquityAccountBalance struct {
	AccountNo        string  `json:"accountNo"`
	AccountBalance   float64 `json:"accountBalance"`
	TotalDebt        float64 `json:"totalDebt"`
	InterestLoan     float64 `json:"interestLoan"`
	OverdueFeeLoan   float64 `json:"overdueFeeLoan"`
	Withdrawable     float64 `json:"withdrawable"`
	OnHoldCash       float64 `json:"onHoldCash"`
	SellUnmatched    float64 `json:"sellUnmatched"`
	SellT0           float64 `json:"sellT0"`
	SellT1           float64 `json:"sellT1"`
	SellT2           float64 `json:"sellT2"`
	BuyUnmatched     float64 `json:"buyUnmatched"`
	BuyT0            float64 `json:"buyT0"`
	BuyT1            float64 `json:"buyT1"`
	BuyT2            float64 `json:"buyT2"`
	AdvanceCashT0    float64 `json:"advanceCashT0"`
	AdvanceCashT1    float64 `json:"advanceCashT1"`
	HoldSubscription float64 `json:"holdSubscription"`
}

func EquityAccountBalanceFromMap(data map[string]interface{}) *EquityAccountBalance {
	if data == nil {
		return &EquityAccountBalance{}
	}

	accBal := util.ToFloat64(data["accountBalance"])
	if accBal == 0 {
		accBal = util.ToFloat64(data["availableCash"])
	}

	withdrawable := util.ToFloat64(data["withdrawable"])
	if withdrawable == 0 {
		withdrawable = util.ToFloat64(data["withdrawal"])
	}

	return &EquityAccountBalance{
		AccountNo:        util.ToStr(data["accountNo"]),
		AccountBalance:   accBal,
		TotalDebt:        util.ToFloat64(data["totalDebt"]),
		InterestLoan:     util.ToFloat64(data["interestLoan"]),
		OverdueFeeLoan:   util.ToFloat64(data["overdueFeeLoan"]),
		Withdrawable:     withdrawable,
		OnHoldCash:       util.ToFloat64(data["onHoldCash"]),
		SellUnmatched:    util.ToFloat64(data["sellUnmatched"]),
		SellT0:           util.ToFloat64(data["sellT0"]),
		SellT1:           util.ToFloat64(data["sellT1"]),
		SellT2:           util.ToFloat64(data["sellT2"]),
		BuyUnmatched:     util.ToFloat64(data["buyUnmatched"]),
		BuyT0:            util.ToFloat64(data["buyT0"]),
		BuyT1:            util.ToFloat64(data["buyT1"]),
		BuyT2:            util.ToFloat64(data["buyT2"]),
		AdvanceCashT0:    util.ToFloat64(data["advanceCashT0"]),
		AdvanceCashT1:    util.ToFloat64(data["advanceCashT1"]),
		HoldSubscription: util.ToFloat64(data["holdSubscription"]),
	}
}


// DerivativeAccountBalance is derivative account balance information.
type DerivativeAccountBalance struct {
	AccountNo            string  `json:"accountNo"`
	AccountBalance       float64 `json:"accountBalance"`
	Fee                  float64 `json:"fee"`
	Commission           float64 `json:"commission"`
	Interest             float64 `json:"interest"`
	ExtInterest          float64 `json:"extInterest"`
	Loan                 float64 `json:"loan"`
	DeliveryAmount       float64 `json:"deliveryAmount"`
	FloatingPL           float64 `json:"floatingPL"`
	TradingPL            float64 `json:"tradingPL"`
	TotalPL              float64 `json:"totalPL"`
	Withdrawable         float64 `json:"withdrawable"`
	CashSSI              float64 `json:"cashSSI"`
	ValidNonCashSSI      float64 `json:"validNonCashSSI"`
	CashWithdrawableSSI  float64 `json:"cashWithdrawableSSI"`
	CashVSDC             float64 `json:"cashVSDC"`
	ValidNonCashVSDC     float64 `json:"validNonCashVSDC"`
	CashWithdrawableVSDC float64 `json:"cashWithdrawableVSDC"`
}

func DerivativeAccountBalanceFromMap(data map[string]interface{}) *DerivativeAccountBalance {
	return &DerivativeAccountBalance{
		AccountNo:            util.ToStr(data["accountNo"]),
		AccountBalance:       util.ToFloat64(data["accountBalance"]),
		Fee:                  util.ToFloat64(data["fee"]),
		Commission:           util.ToFloat64(data["commission"]),
		Interest:             util.ToFloat64(data["interest"]),
		ExtInterest:          util.ToFloat64(data["extInterest"]),
		Loan:                 util.ToFloat64(data["loan"]),
		DeliveryAmount:       util.ToFloat64(data["deliveryAmount"]),
		FloatingPL:           util.ToFloat64(data["floatingPL"]),
		TradingPL:            util.ToFloat64(data["tradingPL"]),
		TotalPL:              util.ToFloat64(data["totalPL"]),
		Withdrawable:         util.ToFloat64(data["withdrawable"]),
		CashSSI:              util.ToFloat64(data["cashSSI"]),
		ValidNonCashSSI:      util.ToFloat64(data["validNonCashSSI"]),
		CashWithdrawableSSI:  util.ToFloat64(data["cashWithdrawableSSI"]),
		CashVSDC:             util.ToFloat64(data["cashVSDC"]),
		ValidNonCashVSDC:     util.ToFloat64(data["validNonCashVSDC"]),
		CashWithdrawableVSDC: util.ToFloat64(data["cashWithdrawableVSDC"]),
	}
}

// AccountBalance is account balance information.
type AccountBalance struct {
	Equity     *EquityAccountBalance     `json:"equity,omitempty"`
	Derivative *DerivativeAccountBalance `json:"derivative,omitempty"`
}

func AccountBalanceFromMap(data map[string]interface{}) *AccountBalance {
	ab := &AccountBalance{}
	if eq, ok := data["equity"].(map[string]interface{}); ok {
		ab.Equity = EquityAccountBalanceFromMap(eq)
	}
	if der, ok := data["derivative"].(map[string]interface{}); ok {
		ab.Derivative = DerivativeAccountBalanceFromMap(der)
	}
	return ab
}

// PositionsRequest is positions request.
type PositionsRequest struct {
	ClientID  string `json:"clientId"`
	AccountNo string `json:"accountNo,omitempty"`
}

func (r *PositionsRequest) ToMap() map[string]string {
	params := map[string]string{"clientId": r.ClientID}
	if r.AccountNo != "" {
		params["accountNo"] = r.AccountNo
	}
	return params
}

// EquityPosition is equity position information.
type EquityPosition struct {
	AccountNo          string  `json:"accountNo"`
	Symbol             string  `json:"symbol"`
	Quantity           int     `json:"quantity"`
	BlockQuantity      int     `json:"blockQuantity"`
	DividendQuantity   int     `json:"dividendQuantity"`
	BuyingQuantity     int     `json:"buyingQuantity"`
	BoughtQuantity     int     `json:"boughtQuantity"`
	SellingQuantity    int     `json:"sellingQuantity"`
	SoldQuantity       int     `json:"soldQuantity"`
	T1SellQuantity     int     `json:"t1SellQuantity"`
	T2SellQuantity     int     `json:"t2SellQuantity"`
	CostPrice          float64 `json:"costPrice"`
	MortgageQuantity   int     `json:"mortgageQuantity"`
	SellableQuantity   int     `json:"sellableQuantity"`
	RestrictedQuantity int     `json:"restrictedQuantity"`
}

func EquityPositionFromList(data []interface{}) []EquityPosition {
	var result []EquityPosition
	for _, item := range data {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		result = append(result, EquityPosition{
			AccountNo:          util.ToStr(m["accountNo"]),
			Symbol:             util.ToStr(m["symbol"]),
			Quantity:           util.ToInt(m["quantity"]),
			BlockQuantity:      util.ToInt(m["blockQuantity"]),
			DividendQuantity:   util.ToInt(m["dividendQuantity"]),
			BuyingQuantity:     util.ToInt(m["buyingQuantity"]),
			BoughtQuantity:     util.ToInt(m["boughtQuantity"]),
			SellingQuantity:    util.ToInt(m["sellingQuantity"]),
			SoldQuantity:       util.ToInt(m["soldQuantity"]),
			T1SellQuantity:     util.ToInt(m["t1SellQuantity"]),
			T2SellQuantity:     util.ToInt(m["t2SellQuantity"]),
			CostPrice:          util.ToFloat64(m["costPrice"]),
			MortgageQuantity:   util.ToInt(m["mortgageQuantity"]),
			SellableQuantity:   util.ToInt(m["sellableQuantity"]),
			RestrictedQuantity: util.ToInt(m["restrictedQuantity"]),
		})
	}
	return result
}

// DerivativePosition is derivative position information.
type DerivativePosition struct {
	AccountNo   string  `json:"accountNo"`
	Symbol      string  `json:"symbol"`
	Long        int     `json:"long"`
	Short       int     `json:"short"`
	Net         int     `json:"net"`
	BidAvgPrice float64 `json:"bidAvgPrice"`
	AskAvgPrice float64 `json:"askAvgPrice"`
	TradePrice  float64 `json:"tradePrice"`
	FloatingPL  float64 `json:"floatingPL"`
	TradingPL   float64 `json:"tradingPL"`
}

func DerivativePositionFromList(data []interface{}) []DerivativePosition {
	var result []DerivativePosition
	for _, item := range data {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		result = append(result, DerivativePosition{
			AccountNo:   util.ToStr(m["accountNo"]),
			Symbol:      util.ToStr(m["symbol"]),
			Long:        util.ToInt(m["long"]),
			Short:       util.ToInt(m["short"]),
			Net:         util.ToInt(m["net"]),
			BidAvgPrice: util.ToFloat64(m["bidAvgPrice"]),
			AskAvgPrice: util.ToFloat64(m["askAvgPrice"]),
			TradePrice:  util.ToFloat64(m["tradePrice"]),
			FloatingPL:  util.ToFloat64(m["floatingPL"]),
			TradingPL:   util.ToFloat64(m["tradingPL"]),
		})
	}
	return result
}

// AllDerivativePosition is derivatives position information.
type AllDerivativePosition struct {
	OpenPositions   []DerivativePosition `json:"derOpenPositions"`
	ClosedPositions []DerivativePosition `json:"derClosePositions"`
}

func AllDerivativePositionFromMap(data map[string]interface{}) *AllDerivativePosition {
	result := &AllDerivativePosition{}
	if open, ok := data["derOpenPositions"].([]interface{}); ok {
		result.OpenPositions = DerivativePositionFromList(open)
	}
	if closed, ok := data["derClosePositions"].([]interface{}); ok {
		result.ClosedPositions = DerivativePositionFromList(closed)
	}
	return result
}

// Position is position information.
type Position struct {
	Equity     []EquityPosition       `json:"equity,omitempty"`
	Derivative *AllDerivativePosition `json:"derivative,omitempty"`
}

func PositionFromMap(data map[string]interface{}) *Position {
	p := &Position{}
	if eq, ok := data["equity"].([]interface{}); ok {
		p.Equity = EquityPositionFromList(eq)
	}
	if der, ok := data["derivative"].(map[string]interface{}); ok {
		p.Derivative = AllDerivativePositionFromMap(der)
	}
	return p
}

// PPMMRRequest is PPMMR request.
type PPMMRRequest struct {
	AccountNo string `json:"accountNo"`
}

func (r *PPMMRRequest) ToMap() map[string]string {
	return map[string]string{"accountNo": r.AccountNo}
}

// EquityPPMMR is equity PPMMR information.
type EquityPPMMR struct {
	AccountNo            string  `json:"accountNo"`
	Dividend             float64 `json:"dividend"`
	LoanValue            float64 `json:"loanValue"`
	TotalDebt            float64 `json:"totalDebt"`
	Debt                 float64 `json:"debt"`
	Liability            float64 `json:"liability"`
	LiabilitySSI         float64 `json:"liabilitySSI"`
	NetLiability         float64 `json:"netLiability"`
	Fees                 float64 `json:"fees"`
	InterestSSI          float64 `json:"interestSSI"`
	InterestSPV          float64 `json:"interestSPV"`
	Withdrawable         float64 `json:"withdrawable"`
	EE                   float64 `json:"ee"`
	EE50                 float64 `json:"ee50"`
	EE60                 float64 `json:"ee60"`
	EE70                 float64 `json:"ee70"`
	EE80                 float64 `json:"ee80"`
	EE90                 float64 `json:"ee90"`
	Action               float64 `json:"action"`
	ActionSSI            float64 `json:"actionSSI"`
	Equity               float64 `json:"equity"`
	EquitySSI            float64 `json:"equitySSI"`
	EECash               float64 `json:"eeCash"`
	HoldSubscription     float64 `json:"holdSubscription"`
	BankBalance          float64 `json:"bankBalance"`
	OnHoldCash           float64 `json:"onHoldCash"`
	Doverdue             float64 `json:"doverdue"`
	DoverdueSSI          float64 `json:"doverdueSSI"`
	AccountBalance       float64 `json:"accountBalance"`
	D                    float64 `json:"D"`
	DSPV                 float64 `json:"dSPV"`
	DSSI                 float64 `json:"dSSI"`
	CIA                  float64 `json:"cia"`
	CollateralAsset      float64 `json:"collateralAsset"`
	CollateralAssetSSI   float64 `json:"collateralAssetSSI"`
	TotalAssets          float64 `json:"totalAssets"`
	TotalEquity          float64 `json:"totalEquity"`
	TotalEquitySSI       float64 `json:"totalEquitySSI"`
	LMV                  float64 `json:"lmv"`
	LMVMargin            float64 `json:"lmvMargin"`
	LMVMarginSSI         float64 `json:"lmvMarginSSI"`
	CallLMV              float64 `json:"callLmv"`
	ForceLMV             float64 `json:"forceLmv"`
	CallLMVSSI           float64 `json:"callLmvSSI"`
	ForceLMVSSI          float64 `json:"forceLmvSSI"`
	LMVNonMarginable     float64 `json:"lmvNonMarginable"`
	LMVNonMarginableSSI  float64 `json:"lmvNonMarginableSSI"`
	PreLoan              float64 `json:"preLoan"`
	MarginRatio          float64 `json:"marginRatio"`
	MarginRatioSSI       float64 `json:"marginRatioSSI"`
	PurchasingPower      float64 `json:"purchasingPower"`
	EEOrigin             float64 `json:"eeOrigin"`
	BuyUnmatched         float64 `json:"buyUnmatched"`
	SellUnmatched        float64 `json:"sellUnmatched"`
	BuyT0                float64 `json:"buyT0"`
	SellT0               float64 `json:"sellT0"`
	SellT1               float64 `json:"sellT1"`
	SellT2               float64 `json:"sellT2"`
	BuyT1                float64 `json:"buyT1"`
	BuyT2                float64 `json:"buyT2"`
	CreditLimit          float64 `json:"creditLimit"`
	MarginCallLMVSold    float64 `json:"marginCallLmvSold"`
	MarginCallLMVSoldSSI float64 `json:"marginCallLmvSoldSSI"`
	MarginCall           float64 `json:"marginCall"`
	MarginCallSSI        float64 `json:"marginCallSSI"`
	CollateralA          float64 `json:"collateralA"`
	CollateralNon        float64 `json:"collateralNon"`
	CollateralASSI       float64 `json:"collateralASSI"`
	CollateralNonSSI     float64 `json:"collateralNonSSI"`
	CallMargin           float64 `json:"callMargin"`
	CallForceSell        float64 `json:"callForceSell"`
	CallMarginSSI        float64 `json:"callMarginSSI"`
	CallForceSellSSI     float64 `json:"callForceSellSSI"`
	AR                   float64 `json:"ar"`
}

func EquityPPMMRFromMap(data map[string]interface{}) *EquityPPMMR {
	return &EquityPPMMR{
		AccountNo:            util.ToStr(data["accountNo"]),
		Dividend:             util.ToFloat64(data["dividend"]),
		LoanValue:            util.ToFloat64(data["loanValue"]),
		TotalDebt:            util.ToFloat64(data["totalDebt"]),
		Debt:                 util.ToFloat64(data["debt"]),
		Liability:            util.ToFloat64(data["liability"]),
		LiabilitySSI:         util.ToFloat64(data["liabilitySSI"]),
		NetLiability:         util.ToFloat64(data["netLiability"]),
		Fees:                 util.ToFloat64(data["fees"]),
		InterestSSI:          util.ToFloat64(data["interestSSI"]),
		InterestSPV:          util.ToFloat64(data["interestSPV"]),
		Withdrawable:         util.ToFloat64(data["withdrawable"]),
		EE:                   util.ToFloat64(data["ee"]),
		EE50:                 util.ToFloat64(data["ee50"]),
		EE60:                 util.ToFloat64(data["ee60"]),
		EE70:                 util.ToFloat64(data["ee70"]),
		EE80:                 util.ToFloat64(data["ee80"]),
		EE90:                 util.ToFloat64(data["ee90"]),
		Action:               util.ToFloat64(data["action"]),
		ActionSSI:            util.ToFloat64(data["actionSSI"]),
		Equity:               util.ToFloat64(data["equity"]),
		EquitySSI:            util.ToFloat64(data["equitySSI"]),
		EECash:               util.ToFloat64(data["eeCash"]),
		HoldSubscription:     util.ToFloat64(data["holdSubscription"]),
		BankBalance:          util.ToFloat64(data["bankBalance"]),
		OnHoldCash:           util.ToFloat64(data["onHoldCash"]),
		Doverdue:             util.ToFloat64(data["doverdue"]),
		DoverdueSSI:          util.ToFloat64(data["doverdueSSI"]),
		AccountBalance:       util.ToFloat64(data["accountBalance"]),
		D:                    util.ToFloat64(data["D"]),
		DSPV:                 util.ToFloat64(data["dSPV"]),
		DSSI:                 util.ToFloat64(data["dSSI"]),
		CIA:                  util.ToFloat64(data["cia"]),
		CollateralAsset:      util.ToFloat64(data["collateralAsset"]),
		CollateralAssetSSI:   util.ToFloat64(data["collateralAssetSSI"]),
		TotalAssets:          util.ToFloat64(data["totalAssets"]),
		TotalEquity:          util.ToFloat64(data["totalEquity"]),
		TotalEquitySSI:       util.ToFloat64(data["totalEquitySSI"]),
		LMV:                  util.ToFloat64(data["lmv"]),
		LMVMargin:            util.ToFloat64(data["lmvMargin"]),
		LMVMarginSSI:         util.ToFloat64(data["lmvMarginSSI"]),
		CallLMV:              util.ToFloat64(data["callLmv"]),
		ForceLMV:             util.ToFloat64(data["forceLmv"]),
		CallLMVSSI:           util.ToFloat64(data["callLmvSSI"]),
		ForceLMVSSI:          util.ToFloat64(data["forceLmvSSI"]),
		LMVNonMarginable:     util.ToFloat64(data["lmvNonMarginable"]),
		LMVNonMarginableSSI:  util.ToFloat64(data["lmvNonMarginableSSI"]),
		PreLoan:              util.ToFloat64(data["preLoan"]),
		MarginRatio:          util.ToFloat64(data["marginRatio"]),
		MarginRatioSSI:       util.ToFloat64(data["marginRatioSSI"]),
		PurchasingPower:      util.ToFloat64(data["purchasingPower"]),
		EEOrigin:             util.ToFloat64(data["eeOrigin"]),
		BuyUnmatched:         util.ToFloat64(data["buyUnmatched"]),
		SellUnmatched:        util.ToFloat64(data["sellUnmatched"]),
		BuyT0:                util.ToFloat64(data["buyT0"]),
		BuyT1:                util.ToFloat64(data["buyT1"]),
		BuyT2:                util.ToFloat64(data["buyT2"]),
		SellT0:               util.ToFloat64(data["sellT0"]),
		SellT1:               util.ToFloat64(data["sellT1"]),
		SellT2:               util.ToFloat64(data["sellT2"]),
		CreditLimit:          util.ToFloat64(data["creditLimit"]),
		MarginCallLMVSold:    util.ToFloat64(data["marginCallLmvSold"]),
		MarginCallLMVSoldSSI: util.ToFloat64(data["marginCallLmvSoldSSI"]),
		MarginCall:           util.ToFloat64(data["marginCall"]),
		MarginCallSSI:        util.ToFloat64(data["marginCallSSI"]),
		CollateralA:          util.ToFloat64(data["collateralA"]),
		CollateralNon:        util.ToFloat64(data["collateralNon"]),
		CollateralASSI:       util.ToFloat64(data["collateralASSI"]),
		CollateralNonSSI:     util.ToFloat64(data["collateralNonSSI"]),
		CallMargin:           util.ToFloat64(data["callMargin"]),
		CallForceSell:        util.ToFloat64(data["callForceSell"]),
		CallMarginSSI:        util.ToFloat64(data["callMarginSSI"]),
		CallForceSellSSI:     util.ToFloat64(data["callForceSellSSI"]),
		AR:                   util.ToFloat64(data["ar"]),
	}
}

// DerivativePPMMR is derivative PPMMR information.
type DerivativePPMMR struct {
	AccountNo                  string  `json:"accountNo"`
	AccountBalance             float64 `json:"accountBalance"`
	Fee                        float64 `json:"fee"`
	Commission                 float64 `json:"commission"`
	Interest                   float64 `json:"interest"`
	Loan                       float64 `json:"loan"`
	DeliveryAmount             float64 `json:"deliveryAmount"`
	FloatingPL                 float64 `json:"floatingPL"`
	TradingPL                  float64 `json:"tradingPL"`
	TotalPL                    float64 `json:"totalPL"`
	Marginable                 float64 `json:"marginable"`
	Depositable                float64 `json:"depositable"`
	RCCall                     float64 `json:"rcCall"`
	Withdrawable               float64 `json:"withdrawable"`
	NonCashDrawableRCCall      float64 `json:"nonCashDrawableRcCall"`
	CashSSI                    float64 `json:"cashSSI"`
	ValidNonCashSSI            float64 `json:"validNonCashSSI"`
	TotalAssetSSI              float64 `json:"totalAssetSSI"`
	WithdrawableSSI            float64 `json:"withdrawableSSI"`
	EESSI                      float64 `json:"eeSSI"`
	CashVSDC                   float64 `json:"cashVSDC"`
	ValidNonCashVSDC           float64 `json:"validNonCashVSDC"`
	TotalAssetVSDC             float64 `json:"totalAssetVSDC"`
	WithdrawableVSDC           float64 `json:"withdrawableVSDC"`
	EEVSDC                     float64 `json:"eeVSDC"`
	SpreadMarginSSI            float64 `json:"spreadMarginSSI"`
	DeliveryMarginSSI          float64 `json:"deliveryMarginSSI"`
	MarginReqSSI               float64 `json:"marginReqSSI"`
	AccountRatioSSI            float64 `json:"accountRatioSSI"`
	UsedLimitWarningLevel1SSI  float64 `json:"usedLimitWarningLevel1SSI"`
	UsedLimitWarningLevel2SSI  float64 `json:"usedLimitWarningLevel2SSI"`
	UsedLimitWarningLevel3SSI  float64 `json:"usedLimitWarningLevel3SSI"`
	MarginCallSSI              float64 `json:"marginCallSSI"`
	SpreadMarginVSDC           float64 `json:"spreadMarginVSDC"`
	DeliveryMarginVSDC         float64 `json:"deliveryMarginVSDC"`
	MarginReqVSDC              float64 `json:"marginReqVSDC"`
	AccountRatioVSDC           float64 `json:"accountRatioVSDC"`
	UsedLimitWarningLevel1VSDC float64 `json:"usedLimitWarningLevel1VSDC"`
	UsedLimitWarningLevel2VSDC float64 `json:"usedLimitWarningLevel2VSDC"`
	UsedLimitWarningLevel3VSDC float64 `json:"usedLimitWarningLevel3VSDC"`
	MarginCallVSDC             float64 `json:"marginCallVSDC"`
	TotalEquity                float64 `json:"totalEquity"`
	ExtInterest                float64 `json:"extInterest"`
}

func DerivativePPMMRFromMap(data map[string]interface{}) *DerivativePPMMR {
	return &DerivativePPMMR{
		AccountNo:                  util.ToStr(data["accountNo"]),
		AccountBalance:             util.ToFloat64(data["accountBalance"]),
		Fee:                        util.ToFloat64(data["fee"]),
		Commission:                 util.ToFloat64(data["commission"]),
		Interest:                   util.ToFloat64(data["interest"]),
		Loan:                       util.ToFloat64(data["loan"]),
		DeliveryAmount:             util.ToFloat64(data["deliveryAmount"]),
		FloatingPL:                 util.ToFloat64(data["floatingPL"]),
		TradingPL:                  util.ToFloat64(data["tradingPL"]),
		TotalPL:                    util.ToFloat64(data["totalPL"]),
		Marginable:                 util.ToFloat64(data["marginable"]),
		Depositable:                util.ToFloat64(data["depositable"]),
		RCCall:                     util.ToFloat64(data["rcCall"]),
		Withdrawable:               util.ToFloat64(data["withdrawable"]),
		NonCashDrawableRCCall:      util.ToFloat64(data["nonCashDrawableRcCall"]),
		CashSSI:                    util.ToFloat64(data["cashSSI"]),
		ValidNonCashSSI:            util.ToFloat64(data["validNonCashSSI"]),
		TotalAssetSSI:              util.ToFloat64(data["totalAssetSSI"]),
		WithdrawableSSI:            util.ToFloat64(data["withdrawableSSI"]),
		EESSI:                      util.ToFloat64(data["eeSSI"]),
		CashVSDC:                   util.ToFloat64(data["cashVSDC"]),
		ValidNonCashVSDC:           util.ToFloat64(data["validNonCashVSDC"]),
		TotalAssetVSDC:             util.ToFloat64(data["totalAssetVSDC"]),
		WithdrawableVSDC:           util.ToFloat64(data["withdrawableVSDC"]),
		EEVSDC:                     util.ToFloat64(data["eeVSDC"]),
		SpreadMarginSSI:            util.ToFloat64(data["spreadMarginSSI"]),
		DeliveryMarginSSI:          util.ToFloat64(data["deliveryMarginSSI"]),
		MarginReqSSI:               util.ToFloat64(data["marginReqSSI"]),
		AccountRatioSSI:            util.ToFloat64(data["accountRatioSSI"]),
		UsedLimitWarningLevel1SSI:  util.ToFloat64(data["usedLimitWarningLevel1SSI"]),
		UsedLimitWarningLevel2SSI:  util.ToFloat64(data["usedLimitWarningLevel2SSI"]),
		UsedLimitWarningLevel3SSI:  util.ToFloat64(data["usedLimitWarningLevel3SSI"]),
		MarginCallSSI:              util.ToFloat64(data["marginCallSSI"]),
		SpreadMarginVSDC:           util.ToFloat64(data["spreadMarginVSDC"]),
		DeliveryMarginVSDC:         util.ToFloat64(data["deliveryMarginVSDC"]),
		MarginReqVSDC:              util.ToFloat64(data["marginReqVSDC"]),
		AccountRatioVSDC:           util.ToFloat64(data["accountRatioVSDC"]),
		UsedLimitWarningLevel1VSDC: util.ToFloat64(data["usedLimitWarningLevel1VSDC"]),
		UsedLimitWarningLevel2VSDC: util.ToFloat64(data["usedLimitWarningLevel2VSDC"]),
		UsedLimitWarningLevel3VSDC: util.ToFloat64(data["usedLimitWarningLevel3VSDC"]),
		MarginCallVSDC:             util.ToFloat64(data["marginCallVSDC"]),
		TotalEquity:                util.ToFloat64(data["totalEquity"]),
		ExtInterest:                util.ToFloat64(data["extInterest"]),
	}
}

// PPMMR is PPMMR information.
type PPMMR struct {
	Equity     *EquityPPMMR     `json:"equity,omitempty"`
	Derivative *DerivativePPMMR `json:"derivative,omitempty"`
}

func PPMMRFromMap(data map[string]interface{}) *PPMMR {
	p := &PPMMR{}
	if eq, ok := data["equity"].(map[string]interface{}); ok {
		p.Equity = EquityPPMMRFromMap(eq)
	}
	if der, ok := data["derivative"].(map[string]interface{}); ok {
		p.Derivative = DerivativePPMMRFromMap(der)
	}
	return p
}

// OrderBookRequest is order book request.
type OrderBookRequest struct {
	AccountNo string `json:"accountNo"`
	FromDate  string `json:"from,omitempty"`
	ToDate    string `json:"to,omitempty"`
	Page      int    `json:"pageIndex"`
	Size      int    `json:"pageSize"`
}

func (r *OrderBookRequest) ToMap() map[string]string {
	return map[string]string{
		"accountNo": r.AccountNo,
		"from":      r.FromDate,
		"to":        r.ToDate,
		"pageIndex": fmt.Sprintf("%d", r.Page),
		"pageSize":  fmt.Sprintf("%d", r.Size),
	}
}

// Order is order information.
type Order struct {
	AccountNo       string              `json:"accountNo"`
	ClientRequestID string              `json:"clientRequestId"`
	OrderID         string              `json:"orderId"`
	Symbol          string              `json:"symbol"`
	Side            trading.OrderSide   `json:"side,omitempty"`
	OrderType       trading.OrderType   `json:"orderType,omitempty"`
	Price           float64             `json:"price"`
	AvgPrice        float64             `json:"avgPrice"`
	Quantity        int                 `json:"quantity"`
	OSQuantity      int                 `json:"osQuantity"`
	FilledQuantity  int                 `json:"filledQuantity"`
	CancelQuantity  int                 `json:"cancelQuantity"`
	Status          trading.OrderStatus `json:"orderStatus,omitempty"`
	InputTime       string              `json:"inputTime"`
	ModifyTime      string              `json:"modifiedTime"`
	Message         string              `json:"message"`
}

func OrderFromMap(data map[string]interface{}, accountNo string) *Order {
	o := &Order{
		AccountNo:       accountNo,
		ClientRequestID: util.ToStr(data["clientRequestId"]),
		OrderID:         util.ToStr(data["orderId"]),
		Symbol:          util.ToStr(data["symbol"]),
		Price:           util.ToFloat64(data["price"]),
		AvgPrice:        util.ToFloat64(data["avgPrice"]),
		Quantity:        util.ToInt(data["quantity"]),
		OSQuantity:      util.ToInt(data["osQuantity"]),
		FilledQuantity:  util.ToInt(data["filledQuantity"]),
		CancelQuantity:  util.ToInt(data["cancelQuantity"]),
		InputTime:       util.ToStr(data["inputTime"]),
		ModifyTime:      util.ToStr(data["modifiedTime"]),
		Message:         util.ToStr(data["message"]),
	}
	if s := util.ToStr(data["side"]); s != "" {
		o.Side = trading.OrderSide(s)
	}
	if ot := util.ToStr(data["orderType"]); ot != "" {
		o.OrderType = trading.OrderType(ot)
	}
	if os := util.ToStr(data["orderStatus"]); os != "" {
		o.Status = trading.OrderStatus(os)
	}
	return o
}

// OrderBook is order book information.
type OrderBook struct {
	Orders      []Order `json:"orders"`
	TotalOrders int     `json:"totalRecord"`
}

func OrderBookFromMap(data map[string]interface{}) *OrderBook {
	ob := &OrderBook{
		TotalOrders: util.ToInt(data["totalRecord"]),
	}
	accountNo := util.ToStr(data["accountNo"])
	if orderList, ok := data["orderList"].([]interface{}); ok {
		for _, item := range orderList {
			if m, ok := item.(map[string]interface{}); ok {
				ob.Orders = append(ob.Orders, *OrderFromMap(m, accountNo))
			}
		}
	}
	return ob
}
