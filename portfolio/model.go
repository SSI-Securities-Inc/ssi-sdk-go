package portfolio

import (
	"strconv"
	"strings"

	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/util"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/trading"
)

// num reads the first present key as a float64 (0 when none is present or numeric).
func num(m map[string]interface{}, keys ...string) float64 {
	return util.ToFloat64(util.Pick(m, keys...))
}

func whole(m map[string]interface{}, keys ...string) int {
	return util.ToInt(util.Pick(m, keys...))
}

func text(m map[string]interface{}, keys ...string) string {
	return util.ToStr(util.Pick(m, keys...))
}

func rowMaps(v interface{}) []map[string]interface{} {
	list, ok := v.([]interface{})
	if !ok {
		return nil
	}
	rows := make([]map[string]interface{}, 0, len(list))
	for _, item := range list {
		if m, ok := item.(map[string]interface{}); ok {
			rows = append(rows, m)
		}
	}
	return rows
}

// nonEmptyMap returns v as a map only when it holds at least one key: an empty section counts as
// absent.
func nonEmptyMap(v interface{}) (map[string]interface{}, bool) {
	m, ok := v.(map[string]interface{})
	if !ok || len(m) == 0 {
		return nil, false
	}
	return m, true
}

// AccountBalanceRequest is the account balance request.
type AccountBalanceRequest struct {
	ClientID  string
	AccountNo string
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
	AccountNo        string
	AccountBalance   float64
	TotalDebt        float64
	InterestLoan     float64
	OverdueFeeLoan   float64
	Withdrawable     float64
	OnHoldCash       float64
	SellUnmatched    float64
	SellT0           float64
	SellT1           float64
	SellT2           float64
	BuyUnmatched     float64
	BuyT0            float64
	BuyT1            float64
	BuyT2            float64
	AdvanceCashT0    float64
	AdvanceCashT1    float64
	HoldSubscription float64
	Dividend         float64
}

func EquityAccountBalanceFromMap(data map[string]interface{}) *EquityAccountBalance {
	return &EquityAccountBalance{
		AccountNo:        text(data, "accountNo"),
		AccountBalance:   num(data, "accountBalance"),
		TotalDebt:        num(data, "totalDebt"),
		InterestLoan:     num(data, "interestLoan"),
		OverdueFeeLoan:   num(data, "overdueFeeLoan"),
		Withdrawable:     num(data, "withdrawable"),
		OnHoldCash:       num(data, "onHoldCash"),
		SellUnmatched:    num(data, "sellUnmatched"),
		SellT0:           num(data, "sellT0"),
		SellT1:           num(data, "sellT1"),
		SellT2:           num(data, "sellT2"),
		BuyUnmatched:     num(data, "buyUnmatched"),
		BuyT0:            num(data, "buyT0"),
		BuyT1:            num(data, "buyT1"),
		BuyT2:            num(data, "buyT2"),
		AdvanceCashT0:    num(data, "advancedCashT0", "advanceCashT0"),
		AdvanceCashT1:    num(data, "advancedCashT1", "advanceCashT1"),
		HoldSubscription: num(data, "holdSubscription"),
		Dividend:         num(data, "dividend"),
	}
}

// DerivativeAccountBalance is derivative account balance information. The two withdrawable
// fields were named CashWithdrawableSSI/VSDC before 3.3.0; the server key is read under both
// names.
type DerivativeAccountBalance struct {
	AccountNo        string
	AccountBalance   float64
	Fee              float64
	Commission       float64
	Interest         float64
	ExtInterest      float64
	Loan             float64
	DeliveryAmount   float64
	FloatingPL       float64
	TradingPL        float64
	TotalPL          float64
	Withdrawable     float64
	CashSSI          float64
	ValidNonCashSSI  float64
	WithdrawableSSI  float64
	CashVSDC         float64
	ValidNonCashVSDC float64
	WithdrawableVSDC float64
}

// CashWithdrawableSSI is the pre-3.3.0 name of WithdrawableSSI.
//
// Deprecated: use WithdrawableSSI.
func (b *DerivativeAccountBalance) CashWithdrawableSSI() float64 { return b.WithdrawableSSI }

// CashWithdrawableVSDC is the pre-3.3.0 name of WithdrawableVSDC.
//
// Deprecated: use WithdrawableVSDC.
func (b *DerivativeAccountBalance) CashWithdrawableVSDC() float64 { return b.WithdrawableVSDC }

func DerivativeAccountBalanceFromMap(data map[string]interface{}) *DerivativeAccountBalance {
	return &DerivativeAccountBalance{
		AccountNo:        text(data, "accountNo"),
		AccountBalance:   num(data, "accountBalance"),
		Fee:              num(data, "fee"),
		Commission:       num(data, "commission"),
		Interest:         num(data, "interest"),
		ExtInterest:      num(data, "extInterest"),
		Loan:             num(data, "loan"),
		DeliveryAmount:   num(data, "deliveryAmount"),
		FloatingPL:       num(data, "floatingPL"),
		TradingPL:        num(data, "tradingPL"),
		TotalPL:          num(data, "totalPL"),
		Withdrawable:     num(data, "withdrawable"),
		CashSSI:          num(data, "cashSSI"),
		ValidNonCashSSI:  num(data, "validNonCashSSI"),
		WithdrawableSSI:  num(data, "withdrawableSSI", "cashWithdrawableSSI"),
		CashVSDC:         num(data, "cashVSDC"),
		ValidNonCashVSDC: num(data, "validNonCashVSDC"),
		WithdrawableVSDC: num(data, "withdrawableVSDC", "cashWithdrawableVSDC"),
	}
}

// AccountBalance is account balance information. A section the account does not have is nil.
type AccountBalance struct {
	Equity     *EquityAccountBalance
	Derivative *DerivativeAccountBalance
}

func AccountBalanceFromMap(data map[string]interface{}) *AccountBalance {
	ab := &AccountBalance{}
	if eq, ok := nonEmptyMap(data["equity"]); ok {
		ab.Equity = EquityAccountBalanceFromMap(eq)
	}
	if der, ok := nonEmptyMap(data["derivative"]); ok {
		ab.Derivative = DerivativeAccountBalanceFromMap(der)
	}
	return ab
}

// PositionsRequest is the positions request. querySummary is always sent explicitly.
type PositionsRequest struct {
	ClientID     string
	AccountNo    string
	QuerySummary bool
}

func (r *PositionsRequest) ToMap() map[string]string {
	params := map[string]string{
		"clientId":     r.ClientID,
		"querySummary": strconv.FormatBool(r.QuerySummary),
	}
	if r.AccountNo != "" {
		params["accountNo"] = r.AccountNo
	}
	return params
}

// EquityPosition is equity position information.
type EquityPosition struct {
	AccountNo          string
	Symbol             string
	Quantity           int
	BlockQuantity      int
	DividendQuantity   int
	BuyingQuantity     int
	BoughtQuantity     int
	SellingQuantity    int
	SoldQuantity       int
	T1SellQuantity     int
	T2SellQuantity     int
	T1BuyQuantity      int
	T2BuyQuantity      int
	CostPrice          float64
	MortgageQuantity   int
	SellableQuantity   int
	RestrictedQuantity int
}

func EquityPositionFromList(data []interface{}) []EquityPosition {
	result := make([]EquityPosition, 0, len(data))
	for _, m := range rowMaps(data) {
		result = append(result, EquityPosition{
			AccountNo:          text(m, "accountNo"),
			Symbol:             text(m, "symbol"),
			Quantity:           whole(m, "quantity"),
			BlockQuantity:      whole(m, "blockQuantity"),
			DividendQuantity:   whole(m, "dividendQuantity"),
			BuyingQuantity:     whole(m, "buyingQuantity"),
			BoughtQuantity:     whole(m, "boughtQuantity"),
			SellingQuantity:    whole(m, "sellingQuantity"),
			SoldQuantity:       whole(m, "soldQuantity"),
			T1SellQuantity:     whole(m, "t1SellQuantity"),
			T2SellQuantity:     whole(m, "t2SellQuantity"),
			T1BuyQuantity:      whole(m, "t1BuyQuantity"),
			T2BuyQuantity:      whole(m, "t2BuyQuantity"),
			CostPrice:          num(m, "costPrice"),
			MortgageQuantity:   whole(m, "mortgageQuantity"),
			SellableQuantity:   whole(m, "sellableQuantity"),
			RestrictedQuantity: whole(m, "restrictedQuantity"),
		})
	}
	return result
}

// DerivativePosition is derivative position information.
type DerivativePosition struct {
	AccountNo   string
	Symbol      string
	Long        int
	Short       int
	Net         int
	BidAvgPrice float64
	AskAvgPrice float64
	TradePrice  float64
	FloatingPL  float64
	TradingPL   float64
}

// DerivativePositionFromList parses position rows. accountNo is attached to rows that carry no
// accountNo of their own.
func DerivativePositionFromList(data []interface{}, accountNo string) []DerivativePosition {
	result := make([]DerivativePosition, 0, len(data))
	for _, m := range rowMaps(data) {
		acc := text(m, "accountNo")
		if acc == "" {
			acc = accountNo
		}
		result = append(result, DerivativePosition{
			AccountNo:   acc,
			Symbol:      text(m, "symbol"),
			Long:        whole(m, "long"),
			Short:       whole(m, "short"),
			Net:         whole(m, "net"),
			BidAvgPrice: num(m, "bidAvgPrice"),
			AskAvgPrice: num(m, "askAvgPrice"),
			TradePrice:  num(m, "tradePrice"),
			FloatingPL:  num(m, "floatingPL"),
			TradingPL:   num(m, "tradingPL"),
		})
	}
	return result
}

// AllDerivativePosition is derivatives position information.
type AllDerivativePosition struct {
	OpenPositions   []DerivativePosition
	ClosedPositions []DerivativePosition
	AccountNo       string
}

// AllDerivativePositionFromAny reads the server's {"derOpenPositions": [...],
// "derClosePositions": [...]} object; a bare list is read as the open positions.
func AllDerivativePositionFromAny(data interface{}, accountNo string) *AllDerivativePosition {
	result := &AllDerivativePosition{AccountNo: accountNo}
	switch v := data.(type) {
	case []interface{}:
		result.OpenPositions = DerivativePositionFromList(v, accountNo)
	case map[string]interface{}:
		if open, ok := v["derOpenPositions"].([]interface{}); ok {
			result.OpenPositions = DerivativePositionFromList(open, accountNo)
		}
		if closed, ok := v["derClosePositions"].([]interface{}); ok {
			result.ClosedPositions = DerivativePositionFromList(closed, accountNo)
		}
	}
	return result
}

// Position is position information. A section the account does not have is nil.
type Position struct {
	Equity     []EquityPosition
	Derivative *AllDerivativePosition
}

// PositionFromMap parses the positions payload. The server's {"code": 204} marker yields an empty
// Position. accountNo is attached to derivative positions that carry none.
func PositionFromMap(data map[string]interface{}, accountNo string) *Position {
	p := &Position{}
	if data == nil || util.IsNoContent(data) {
		return p
	}
	if eq, ok := data["equity"].([]interface{}); ok && len(eq) > 0 {
		p.Equity = EquityPositionFromList(eq)
	}
	switch der := data["derivative"].(type) {
	case []interface{}:
		if len(der) > 0 {
			p.Derivative = AllDerivativePositionFromAny(der, accountNo)
		}
	case map[string]interface{}:
		if len(der) > 0 {
			p.Derivative = AllDerivativePositionFromAny(der, accountNo)
		}
	}
	return p
}

// PPMMRRequest is PPMMR request.
type PPMMRRequest struct {
	AccountNo string
}

func (r *PPMMRRequest) ToMap() map[string]string {
	return map[string]string{"accountNo": r.AccountNo}
}

// EquityPPMMR is equity purchasing power / maintenance margin information.
type EquityPPMMR struct {
	AccountNo            string
	Dividend             float64
	LoanValue            float64
	TotalDebt            float64
	Debt                 float64
	Liability            float64
	LiabilitySSI         float64
	NetLiability         float64
	Fees                 float64
	InterestSSI          float64
	InterestSPV          float64
	Withdrawable         float64
	EE                   float64
	EE50                 float64
	EE60                 float64
	EE70                 float64
	EE80                 float64
	EE90                 float64
	Action               float64
	ActionSSI            float64
	Equity               float64
	EquitySSI            float64
	EECash               float64
	HoldSubscription     float64
	BankBalance          float64
	OnHoldCash           float64
	Doverdue             float64
	DoverdueSSI          float64
	AccountBalance       float64
	D                    float64
	DSPV                 float64
	DSSI                 float64
	CIA                  float64
	CollateralAsset      float64
	CollateralAssetSSI   float64
	TotalAssets          float64
	TotalEquity          float64
	TotalEquitySSI       float64
	LMV                  float64
	LMVMargin            float64
	LMVMarginSSI         float64
	CallLMV              float64
	ForceLMV             float64
	CallLMVSSI           float64
	ForceLMVSSI          float64
	LMVNonMarginable     float64
	LMVNonMarginableSSI  float64
	PreLoan              float64
	MarginRatio          float64
	MarginRatioSSI       float64
	PurchasingPower      float64
	EEOrigin             float64
	BuyUnmatched         float64
	SellUnmatched        float64
	BuyT0                float64
	SellT0               float64
	SellT1               float64
	SellT2               float64
	BuyT1                float64
	BuyT2                float64
	CreditLimit          float64
	MarginCallLMVSold    float64
	MarginCallLMVSoldSSI float64
	MarginCall           float64
	MarginCallSSI        float64
	CollateralA          float64
	CollateralNon        float64
	CollateralASSI       float64
	CollateralNonSSI     float64
	CallMargin           float64
	CallForceSell        float64
	CallMarginSSI        float64
	CallForceSellSSI     float64
	AR                   float64
}

// EquityPPMMRFromMap reads the server key first and the alternative spelling second (the server
// spells some keys with different casing between environments).
func EquityPPMMRFromMap(data map[string]interface{}) *EquityPPMMR {
	return &EquityPPMMR{
		AccountNo:            text(data, "accountNo"),
		Dividend:             num(data, "dividend"),
		LoanValue:            num(data, "loanValue"),
		TotalDebt:            num(data, "totalDebt"),
		Debt:                 num(data, "debt"),
		Liability:            num(data, "liability"),
		LiabilitySSI:         num(data, "liabilitySSI"),
		NetLiability:         num(data, "netLiability"),
		Fees:                 num(data, "fees"),
		InterestSSI:          num(data, "interestSSI"),
		InterestSPV:          num(data, "interestSPV"),
		Withdrawable:         num(data, "withdrawable"),
		EE:                   num(data, "ee"),
		EE50:                 num(data, "ee50"),
		EE60:                 num(data, "ee60"),
		EE70:                 num(data, "ee70"),
		EE80:                 num(data, "ee80"),
		EE90:                 num(data, "ee90"),
		Action:               num(data, "action"),
		ActionSSI:            num(data, "actionSSI"),
		Equity:               num(data, "equity"),
		EquitySSI:            num(data, "equitySSI"),
		EECash:               num(data, "eeCash"),
		HoldSubscription:     num(data, "holdSubscription"),
		BankBalance:          num(data, "bankBalance"),
		OnHoldCash:           num(data, "onHoldCash"),
		Doverdue:             num(data, "doverdue"),
		DoverdueSSI:          num(data, "doverdue_ssi", "doverdueSSI"),
		AccountBalance:       num(data, "accountBalance"),
		D:                    num(data, "D", "d"),
		DSPV:                 num(data, "dSPV", "dspv"),
		DSSI:                 num(data, "dSSI", "dssi"),
		CIA:                  num(data, "cia"),
		CollateralAsset:      num(data, "collateralAsset"),
		CollateralAssetSSI:   num(data, "collateralAssetSSI"),
		TotalAssets:          num(data, "totalAssets"),
		TotalEquity:          num(data, "totalEquity"),
		TotalEquitySSI:       num(data, "totalEquitySSI"),
		LMV:                  num(data, "lmv"),
		LMVMargin:            num(data, "lmvMargin"),
		LMVMarginSSI:         num(data, "lmvMarginSSI"),
		CallLMV:              num(data, "callLMV", "callLmv"),
		ForceLMV:             num(data, "forceLMV", "forceLmv"),
		CallLMVSSI:           num(data, "callLMVSSI", "callLmvSSI"),
		ForceLMVSSI:          num(data, "forceLMVSSI", "forceLmvSSI"),
		LMVNonMarginable:     num(data, "lmvNonMarginable"),
		LMVNonMarginableSSI:  num(data, "lmvNonMarginableSSI"),
		PreLoan:              num(data, "preLoan"),
		MarginRatio:          num(data, "marginRatio"),
		MarginRatioSSI:       num(data, "marginRatioSSI"),
		PurchasingPower:      num(data, "purchasingPower"),
		EEOrigin:             num(data, "eeOrigin"),
		BuyUnmatched:         num(data, "buyUnmatched"),
		SellUnmatched:        num(data, "sellUnmatched"),
		BuyT0:                num(data, "buyT0"),
		BuyT1:                num(data, "buyT1"),
		BuyT2:                num(data, "buyT2"),
		SellT0:               num(data, "sellT0"),
		SellT1:               num(data, "sellT1"),
		SellT2:               num(data, "sellT2"),
		CreditLimit:          num(data, "creditLimit"),
		MarginCallLMVSold:    num(data, "marginCallLMVSold", "marginCallLmvSold"),
		MarginCallLMVSoldSSI: num(data, "marginCallLMVSoldSSI", "marginCallLmvSoldSSI"),
		MarginCall:           num(data, "marginCall"),
		MarginCallSSI:        num(data, "marginCallSSI"),
		CollateralA:          num(data, "collateralA"),
		CollateralNon:        num(data, "collateralNon"),
		CollateralASSI:       num(data, "collateralASSI"),
		CollateralNonSSI:     num(data, "collateralNonSSI"),
		CallMargin:           num(data, "callMargin"),
		CallForceSell:        num(data, "callForcesell", "callForceSell"),
		CallMarginSSI:        num(data, "callMarginSSI"),
		CallForceSellSSI:     num(data, "callForcesellSSI", "callForceSellSSI"),
		AR:                   num(data, "ar"),
	}
}

// DerivativePPMMR is derivative purchasing power / maintenance margin information.
type DerivativePPMMR struct {
	AccountNo                  string
	AccountBalance             float64
	Fee                        float64
	Commission                 float64
	Interest                   float64
	Loan                       float64
	DeliveryAmount             float64
	FloatingPL                 float64
	TradingPL                  float64
	TotalPL                    float64
	Marginable                 float64
	Depositable                float64
	RCCall                     float64
	Withdrawable               float64
	NonCashDrawableRCCall      float64
	CashSSI                    float64
	ValidNonCashSSI            float64
	TotalAssetSSI              float64
	WithdrawableSSI            float64
	EESSI                      float64
	CashVSDC                   float64
	ValidNonCashVSDC           float64
	TotalAssetVSDC             float64
	WithdrawableVSDC           float64
	EEVSDC                     float64
	SpreadMarginSSI            float64
	DeliveryMarginSSI          float64
	MarginReqSSI               float64
	AccountRatioSSI            float64
	UsedLimitWarningLevel1SSI  float64
	UsedLimitWarningLevel2SSI  float64
	UsedLimitWarningLevel3SSI  float64
	MarginCallSSI              float64
	SpreadMarginVSDC           float64
	DeliveryMarginVSDC         float64
	MarginReqVSDC              float64
	AccountRatioVSDC           float64
	UsedLimitWarningLevel1VSDC float64
	UsedLimitWarningLevel2VSDC float64
	UsedLimitWarningLevel3VSDC float64
	MarginCallVSDC             float64
	TotalEquity                float64
	ExtInterest                float64
}

func DerivativePPMMRFromMap(data map[string]interface{}) *DerivativePPMMR {
	return &DerivativePPMMR{
		AccountNo:                  text(data, "accountNo"),
		AccountBalance:             num(data, "accountBalance"),
		Fee:                        num(data, "fee"),
		Commission:                 num(data, "commission"),
		Interest:                   num(data, "interest"),
		Loan:                       num(data, "loan"),
		DeliveryAmount:             num(data, "deliveryAmount"),
		FloatingPL:                 num(data, "floatingPL"),
		TradingPL:                  num(data, "tradingPL"),
		TotalPL:                    num(data, "totalPL"),
		Marginable:                 num(data, "marginable"),
		Depositable:                num(data, "depositable"),
		RCCall:                     num(data, "rcCall"),
		Withdrawable:               num(data, "withdrawable"),
		NonCashDrawableRCCall:      num(data, "nonCashDrawableRCCall", "nonCashDrawableRcCall"),
		CashSSI:                    num(data, "cashSSI"),
		ValidNonCashSSI:            num(data, "validNonCashSSI"),
		TotalAssetSSI:              num(data, "totalAssetSSI"),
		WithdrawableSSI:            num(data, "withdrawableSSI"),
		EESSI:                      num(data, "eeSSI"),
		CashVSDC:                   num(data, "cashVSDC"),
		ValidNonCashVSDC:           num(data, "validNonCashVSDC"),
		TotalAssetVSDC:             num(data, "totalAssetVSDC"),
		WithdrawableVSDC:           num(data, "withdrawableVSDC"),
		EEVSDC:                     num(data, "eeVSDC"),
		SpreadMarginSSI:            num(data, "spreadMarginSSI"),
		DeliveryMarginSSI:          num(data, "deliveryMarginSSI"),
		MarginReqSSI:               num(data, "marginReqSSI"),
		AccountRatioSSI:            num(data, "accountRatioSSI"),
		UsedLimitWarningLevel1SSI:  num(data, "usedLimitWarningLevel1SSI"),
		UsedLimitWarningLevel2SSI:  num(data, "usedLimitWarningLevel2SSI"),
		UsedLimitWarningLevel3SSI:  num(data, "usedLimitWarningLevel3SSI"),
		MarginCallSSI:              num(data, "marginCallSSI"),
		SpreadMarginVSDC:           num(data, "spreadMarginVSDC"),
		DeliveryMarginVSDC:         num(data, "deliveryMarginVSDC"),
		MarginReqVSDC:              num(data, "marginReqVSDC"),
		AccountRatioVSDC:           num(data, "accountRatioVSDC"),
		UsedLimitWarningLevel1VSDC: num(data, "usedLimitWarningLevel1VSDC"),
		UsedLimitWarningLevel2VSDC: num(data, "usedLimitWarningLevel2VSDC"),
		UsedLimitWarningLevel3VSDC: num(data, "usedLimitWarningLevel3VSDC"),
		MarginCallVSDC:             num(data, "marginCallVSDC"),
		TotalEquity:                num(data, "totalEquity"),
		ExtInterest:                num(data, "extInterest"),
	}
}

// PPMMR is PPMMR information. A section the account does not have is nil.
type PPMMR struct {
	Equity     *EquityPPMMR
	Derivative *DerivativePPMMR
}

func PPMMRFromMap(data map[string]interface{}) *PPMMR {
	p := &PPMMR{}
	if eq, ok := nonEmptyMap(data["equity"]); ok {
		p.Equity = EquityPPMMRFromMap(eq)
	}
	if der, ok := nonEmptyMap(data["derivative"]); ok {
		p.Derivative = DerivativePPMMRFromMap(der)
	}
	return p
}

// OrderBookRequest is the order book request. Empty strings are not sent.
type OrderBookRequest struct {
	AccountNo   string
	FromDate    string
	ToDate      string
	Page        int
	Size        int
	Symbol      string
	OrderStatus string
}

func (r *OrderBookRequest) ToMap() map[string]string {
	params := map[string]string{"accountNo": r.AccountNo}
	if r.FromDate != "" {
		params["from"] = r.FromDate
	}
	if r.ToDate != "" {
		params["to"] = r.ToDate
	}
	if r.Symbol != "" {
		params["symbol"] = r.Symbol
	}
	if r.OrderStatus != "" {
		params["orderStatus"] = r.OrderStatus
	}
	params["pageIndex"] = strconv.Itoa(r.Page)
	params["pageSize"] = strconv.Itoa(r.Size)
	return params
}

// Order is order information. Side, OrderType and Status are the typed value when the server sent
// a known member and the raw string otherwise (the types are string-based, so nothing is lost).
type Order struct {
	AccountNo       string
	ClientRequestID string
	OrderID         string
	// Symbol is read from "symbol", falling back to "instrumentId" (order history).
	Symbol    string
	Side      trading.OrderSide
	OrderType trading.OrderType
	// Price is the numeric price. For a market-priced order the server sends a word (ATO, ATC,
	// MP...) instead: Price is then 0 and PriceText holds the word.
	Price     float64
	PriceText string
	AvgPrice  float64
	Quantity  int
	// OSQuantity is nil when the server did not send it (the REST order book never does).
	OSQuantity     *int
	FilledQuantity int
	CancelQuantity int
	Status         trading.OrderStatus
	// InputTime and ModifyTime are "yyyy/MM/dd HH:mm:ss".
	InputTime  string
	ModifyTime string
	// Message is the server message (the reason for a rejection).
	Message string
}

// OrderFromMap parses one order row. The server names are filledQty/cancelQty (filledQuantity /
// cancelQuantity are read as a fallback) and buySell for side in the order history.
func OrderFromMap(data map[string]interface{}, accountNo string) *Order {
	o := &Order{
		AccountNo:       accountNo,
		ClientRequestID: text(data, "clientRequestId"),
		OrderID:         text(data, "orderId"),
		Symbol:          text(data, "symbol", "instrumentId"),
		Side:            normSide(text(data, "side", "buySell")),
		OrderType:       normOrderType(text(data, "orderType")),
		AvgPrice:        num(data, "avgPrice"),
		Quantity:        whole(data, "quantity"),
		OSQuantity:      util.ToOptInt(data["osQuantity"]),
		FilledQuantity:  whole(data, "filledQty", "filledQuantity"),
		CancelQuantity:  whole(data, "cancelQty", "cancelQuantity"),
		Status:          normStatus(text(data, "orderStatus")),
		InputTime:       text(data, "inputTime"),
		ModifyTime:      text(data, "modifiedTime"),
		Message:         text(data, "message"),
	}
	o.Price, o.PriceText = parsePrice(data["price"])
	return o
}

// normSide, normOrderType and normStatus match case-insensitively and keep an unknown value as sent.
func normSide(raw string) trading.OrderSide {
	if v, ok := trading.OrderSideFromValue(raw); ok {
		return v
	}
	return trading.OrderSide(raw)
}

func normOrderType(raw string) trading.OrderType {
	if v, ok := trading.OrderTypeFromValue(raw); ok {
		return v
	}
	return trading.OrderType(raw)
}

func normStatus(raw string) trading.OrderStatus {
	if v, ok := trading.OrderStatusFromValue(raw); ok {
		return v
	}
	return trading.OrderStatus(raw)
}

// parsePrice returns the numeric price, or 0 plus the word the server sent for a market-priced
// order. nil and blank are 0.
func parsePrice(v interface{}) (float64, string) {
	if v == nil {
		return 0, ""
	}
	if s, ok := v.(string); ok {
		s = strings.TrimSpace(s)
		if s == "" {
			return 0, ""
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f, ""
		}
		if t, ok := trading.OrderTypeFromValue(s); ok {
			return 0, string(t)
		}
		return 0, s
	}
	return util.ToFloat64(v), ""
}

// OrderBook is order book information. TotalOrders is informational: do not derive a page count
// from it (the server's paging can be off), page until a page comes back empty instead.
type OrderBook struct {
	Orders      []Order
	TotalOrders int
}

// OrderBookFromMap parses the order book payload. The server's {"code": 204} marker and an absent
// orderList both yield an empty book.
func OrderBookFromMap(data map[string]interface{}) *OrderBook {
	ob := &OrderBook{Orders: []Order{}}
	if data == nil || util.IsNoContent(data) {
		return ob
	}
	ob.TotalOrders = whole(data, "totalRecord")
	accountNo := text(data, "accountNo")
	for _, m := range rowMaps(data["orderList"]) {
		ob.Orders = append(ob.Orders, *OrderFromMap(m, accountNo))
	}
	return ob
}
