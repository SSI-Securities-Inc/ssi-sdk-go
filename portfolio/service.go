package portfolio

import (
	"fmt"

	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/logger"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/util"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/transport"
)

var portfolioLog = logger.New("ssi_sdk.services.portfolio")

const (
	defaultSize = 1000
	defaultPage = 1

	epAccountBalance = "/api/v3/trading/accountBalance"
	epAccountPPMMR   = "/api/v3/trading/ppmmrAccount"
	epPositions      = "/api/v3/trading/position"
	epOrderHistory   = "/api/v3/trading/orderBook"
)

// Service provides portfolio operations.
type Service struct {
	rest     *transport.RestClient
	clientID string
}

func NewService(rest *transport.RestClient, clientID string) *Service {
	return &Service{
		rest:     rest,
		clientID: clientID,
	}
}

func (s *Service) getBalance(accountNo string) (*AccountBalance, error) {
	params := (&AccountBalanceRequest{
		ClientID:  s.clientID,
		AccountNo: accountNo,
	}).ToMap()

	data, err := s.rest.Get(epAccountBalance, params, nil)
	if err != nil {
		portfolioLog.Error("Failed to fetch balance for account %s: %v", accountNo, err)
		return nil, err
	}
	return AccountBalanceFromMap(data), nil
}

func (s *Service) GetEquityBalance(accountNo string) (*EquityAccountBalance, error) {
	if err := ssi.RequireNonEmpty(accountNo, "Account number must be provided"); err != nil {
		return nil, err
	}
	balance, err := s.getBalance(accountNo)
	if err != nil {
		return nil, err
	}
	return balance.Equity, nil
}

func (s *Service) GetDerivativeBalance(accountNo string) (*DerivativeAccountBalance, error) {
	if err := ssi.RequireNonEmpty(accountNo, "Account number must be provided"); err != nil {
		return nil, err
	}
	balance, err := s.getBalance(accountNo)
	if err != nil {
		return nil, err
	}
	return balance.Derivative, nil
}

func (s *Service) getOrderBook(accountNo, fromDate, toDate string) (*OrderBook, error) {
	params := (&OrderBookRequest{
		AccountNo: accountNo,
		FromDate:  fromDate,
		ToDate:    toDate,
		Page:      defaultPage,
		Size:      defaultSize,
	}).ToMap()

	data, err := s.rest.Get(epOrderHistory, params, nil)
	if err != nil {
		portfolioLog.Error("Failed to fetch order book for account %s: %v", accountNo, err)
		return nil, err
	}
	return OrderBookFromMap(data), nil
}

func (s *Service) GetTodayOrders(accountNo string) ([]Order, error) {
	if err := ssi.RequireNonEmpty(accountNo, "Account number must be provided"); err != nil {
		return nil, err
	}
	today := util.TodayDateStr()
	ob, err := s.getOrderBook(accountNo, today, today)
	if err != nil {
		return nil, err
	}
	return ob.Orders, nil
}

func (s *Service) GetHistoricalOrders(accountNo, fromDate, toDate string) ([]Order, error) {
	if err := ssi.RequireNonEmpty(accountNo, "Account number must be provided"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(fromDate, "From date (yyyy/mm/dd) must be provided"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(toDate, "To date (yyyy/mm/dd) must be provided"); err != nil {
		return nil, err
	}
	ob, err := s.getOrderBook(accountNo, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	return ob.Orders, nil
}

func (s *Service) getPositions(clientID, accountNo string) (*Position, error) {
	params := (&PositionsRequest{
		ClientID:  clientID,
		AccountNo: accountNo,
	}).ToMap()

	data, err := s.rest.Get(epPositions, params, nil)
	if err != nil {
		portfolioLog.Error("Failed to fetch positions for account %s: %v", accountNo, err)
		return nil, err
	}
	return PositionFromMap(data), nil
}

func (s *Service) GetEquityPositions(accountNo string) ([]EquityPosition, error) {
	if err := ssi.RequireNonEmpty(accountNo, "Account number must be provided"); err != nil {
		return nil, err
	}
	position, err := s.getPositions(s.clientID, accountNo)
	if err != nil {
		return nil, err
	}
	return position.Equity, nil
}

func (s *Service) GetDerivativePositions(accountNo string) (*AllDerivativePosition, error) {
	if err := ssi.RequireNonEmpty(accountNo, "Account number must be provided"); err != nil {
		return nil, err
	}
	position, err := s.getPositions(s.clientID, accountNo)
	if err != nil {
		return nil, err
	}
	return position.Derivative, nil
}

func (s *Service) GetOpenDerivativePositions(accountNo string) ([]DerivativePosition, error) {
	if err := ssi.RequireNonEmpty(accountNo, "Account number must be provided"); err != nil {
		return nil, err
	}
	position, err := s.getPositions(s.clientID, accountNo)
	if err != nil {
		return nil, err
	}
	if position.Derivative == nil {
		return nil, nil
	}
	return position.Derivative.OpenPositions, nil
}

func (s *Service) GetClosedDerivativePositions(accountNo string) ([]DerivativePosition, error) {
	if err := ssi.RequireNonEmpty(accountNo, "Account number must be provided"); err != nil {
		return nil, err
	}
	position, err := s.getPositions(s.clientID, accountNo)
	if err != nil {
		return nil, err
	}
	if position.Derivative == nil {
		return nil, nil
	}
	return position.Derivative.ClosedPositions, nil
}

func (s *Service) getPPMMR(accountNo string) (*PPMMR, error) {
	params := (&PPMMRRequest{AccountNo: accountNo}).ToMap()

	data, err := s.rest.Get(epAccountPPMMR, params, nil)
	if err != nil {
		portfolioLog.Error("Failed to fetch PPMMR for account %s: %v", accountNo, err)
		return nil, err
	}
	return PPMMRFromMap(data), nil
}

func (s *Service) GetEquityPPMMR(accountNo string) (*EquityPPMMR, error) {
	if err := ssi.RequireNonEmpty(accountNo, "Account number must be provided"); err != nil {
		return nil, err
	}
	ppmmr, err := s.getPPMMR(accountNo)
	if err != nil {
		return nil, err
	}
	return ppmmr.Equity, nil
}

func (s *Service) GetDerivativePPMMR(accountNo string) (*DerivativePPMMR, error) {
	if err := ssi.RequireNonEmpty(accountNo, "Account number must be provided"); err != nil {
		return nil, err
	}
	ppmmr, err := s.getPPMMR(accountNo)
	if err != nil {
		return nil, err
	}
	if ppmmr == nil {
		return nil, fmt.Errorf("no PPMMR data found for account %s", accountNo)
	}
	return ppmmr.Derivative, nil
}
