package account

import (
	"encoding/json"
	"fmt"

	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/logger"
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

func (s *Service) GetAccountInfo() ([]Account, error) {
	data, err := s.rest.Get(epAccountInfo, nil, nil)
	if err != nil {
		accountLog.Error("Failed to fetch account info for endpoint %s: %v", epAccountInfo, err)
		return nil, err
	}

	raw, err := json.Marshal(data["data"])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal account data: %w", err)
	}

	var accounts []Account
	if err := json.Unmarshal(raw, &accounts); err != nil {
		return nil, fmt.Errorf("failed to parse account list: %w", err)
	}
	return accounts, nil
}
