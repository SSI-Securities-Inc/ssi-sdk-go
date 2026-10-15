package account

import (
	"strings"

	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/util"
)

// AccountType is the account type as the server names it. An unknown type from the server is
// kept as its raw string.
type AccountType string

const (
	AccountTypeEquity       AccountType = "Cash"
	AccountTypeEquityMargin AccountType = "Margin"
	AccountTypeDerivative   AccountType = "Derivative"
)

var knownAccountTypes = []AccountType{AccountTypeEquity, AccountTypeEquityMargin, AccountTypeDerivative}

// AccountTypeFromValue maps a value (case-insensitive) to a known AccountType.
func AccountTypeFromValue(raw string) (AccountType, bool) {
	for _, t := range knownAccountTypes {
		if strings.EqualFold(string(t), strings.TrimSpace(raw)) {
			return t, true
		}
	}
	return "", false
}

// Account represents an accessible trading account.
type Account struct {
	AccountNo string `json:"accountNo"`
	// AccountType is the matching AccountType, the raw value when the server sends one this
	// SDK does not know (e.g. "2"), and nil when the key is missing.
	AccountType *AccountType `json:"accountType,omitempty"`
}

// AccountsFromList builds accounts from the raw rows of the account/info response.
func AccountsFromList(data []interface{}) []Account {
	accounts := make([]Account, 0, len(data))
	for _, item := range data {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		acc := Account{AccountNo: util.ToStr(m["accountNo"])}
		if raw, present := m["accountType"]; present && raw != nil {
			text := util.ToStr(raw)
			t, known := AccountTypeFromValue(text)
			if !known {
				t = AccountType(text)
			}
			acc.AccountType = &t
		}
		accounts = append(accounts, acc)
	}
	return accounts
}
