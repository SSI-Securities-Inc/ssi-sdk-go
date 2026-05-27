package account

type AccountType string

const (
	AccountTypeEquity       AccountType = "Cash"
	AccountTypeEquityMargin AccountType = "Margin"
	AccountTypeDerivative   AccountType = "Derivative"
)

// Account represents an accessible trading account.
type Account struct {
	AccountNo   string      `json:"accountNo"`
	AccountType AccountType `json:"accountType"`
}
