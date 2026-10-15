package account

import (
	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/logger"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/util"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/transport"
)

var accountLog = logger.New("ssi_sdk.services.account")

const epAccountInfo = "/api/v3/account/info"

// Service provides account operations.
type Service struct {
	rest *transport.RestClient
}

func NewService(rest *transport.RestClient) *Service {
	return &Service{rest: rest}
}

// GetAccountInfo lists the accessible trading accounts. The server answers {"code": 204}
// instead of an empty array when there is nothing to list; that yields an empty list.
func (s *Service) GetAccountInfo() ([]Account, error) {
	data, err := s.rest.Get(epAccountInfo, nil, nil)
	if err != nil {
		accountLog.Error("Failed to fetch account info for endpoint %s: %v", epAccountInfo, err)
		return nil, err
	}
	if data == nil || util.IsNoContent(data) {
		return []Account{}, nil
	}
	raw, present := data["data"]
	if !present || raw == nil {
		return []Account{}, nil
	}
	rows, ok := raw.([]interface{})
	if !ok {
		return nil, ssi.NewAPIError("Unexpected response format while listing accounts", "", 0, nil)
	}
	return AccountsFromList(rows), nil
}
