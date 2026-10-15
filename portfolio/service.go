package portfolio

import (
	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/logger"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/util"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/transport"
)

var portfolioLog = logger.New("ssi_sdk.services.portfolio")

const (
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
	return &Service{rest: rest, clientID: clientID}
}

func (s *Service) getBalance(accountNo string) (*AccountBalance, error) {
	params := (&AccountBalanceRequest{ClientID: s.clientID, AccountNo: accountNo}).ToMap()
	data, err := s.rest.Get(epAccountBalance, params, nil)
	if err != nil {
		portfolioLog.Error("Failed to fetch balance: %v", err)
		return nil, err
	}
	return AccountBalanceFromMap(data), nil
}

// GetEquityBalance returns the equity balance, or nil when the account has no equity section.
func (s *Service) GetEquityBalance(accountNo string) (*EquityAccountBalance, error) {
	if err := ssi.RequireNonEmpty(accountNo, "accountNo"); err != nil {
		return nil, err
	}
	balance, err := s.getBalance(accountNo)
	if err != nil {
		return nil, err
	}
	return balance.Equity, nil
}

// GetDerivativeBalance returns the derivative balance, or nil when the account has none.
func (s *Service) GetDerivativeBalance(accountNo string) (*DerivativeAccountBalance, error) {
	if err := ssi.RequireNonEmpty(accountNo, "accountNo"); err != nil {
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
		Page:      ssi.DefaultPage,
		Size:      ssi.DefaultSize,
	}).ToMap()
	data, err := s.rest.Get(epOrderHistory, params, nil)
	if err != nil {
		portfolioLog.Error("Failed to fetch order book: %v", err)
		return nil, err
	}
	return OrderBookFromMap(data), nil
}

// GetTodayOrders lists today's orders.
func (s *Service) GetTodayOrders(accountNo string) ([]Order, error) {
	if err := ssi.RequireNonEmpty(accountNo, "accountNo"); err != nil {
		return nil, err
	}
	today := util.TodayDateStr()
	ob, err := s.getOrderBook(accountNo, today, today)
	if err != nil {
		return nil, err
	}
	return ob.Orders, nil
}

// GetHistoricalOrders lists orders within a date range ("YYYY/MM/DD").
func (s *Service) GetHistoricalOrders(accountNo, fromDate, toDate string) ([]Order, error) {
	if err := ssi.RequireNonEmpty(accountNo, "accountNo"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(fromDate, "fromDate"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(toDate, "toDate"); err != nil {
		return nil, err
	}
	ob, err := s.getOrderBook(accountNo, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	return ob.Orders, nil
}

func (s *Service) getPositions(accountNo string) (*Position, error) {
	params := (&PositionsRequest{ClientID: s.clientID, AccountNo: accountNo, QuerySummary: true}).ToMap()
	data, err := s.rest.Get(epPositions, params, nil)
	if err != nil {
		portfolioLog.Error("Failed to fetch positions: %v", err)
		return nil, err
	}
	return PositionFromMap(data, accountNo), nil
}

// GetEquityPositions lists equity positions (nil when the account has none).
func (s *Service) GetEquityPositions(accountNo string) ([]EquityPosition, error) {
	if err := ssi.RequireNonEmpty(accountNo, "accountNo"); err != nil {
		return nil, err
	}
	position, err := s.getPositions(accountNo)
	if err != nil {
		return nil, err
	}
	return position.Equity, nil
}

// GetDerivativePositions returns open and closed derivative positions (nil when the account has
// none).
func (s *Service) GetDerivativePositions(accountNo string) (*AllDerivativePosition, error) {
	if err := ssi.RequireNonEmpty(accountNo, "accountNo"); err != nil {
		return nil, err
	}
	position, err := s.getPositions(accountNo)
	if err != nil {
		return nil, err
	}
	return position.Derivative, nil
}

func (s *Service) GetOpenDerivativePositions(accountNo string) ([]DerivativePosition, error) {
	all, err := s.GetDerivativePositions(accountNo)
	if err != nil || all == nil {
		return nil, err
	}
	return all.OpenPositions, nil
}

func (s *Service) GetClosedDerivativePositions(accountNo string) ([]DerivativePosition, error) {
	all, err := s.GetDerivativePositions(accountNo)
	if err != nil || all == nil {
		return nil, err
	}
	return all.ClosedPositions, nil
}

func (s *Service) getPPMMR(accountNo string) (*PPMMR, error) {
	params := (&PPMMRRequest{AccountNo: accountNo}).ToMap()
	data, err := s.rest.Get(epAccountPPMMR, params, nil)
	if err != nil {
		portfolioLog.Error("Failed to fetch PPMMR: %v", err)
		return nil, err
	}
	return PPMMRFromMap(data), nil
}

// GetEquityPPMMR returns the equity purchasing power data (nil when the account has none).
func (s *Service) GetEquityPPMMR(accountNo string) (*EquityPPMMR, error) {
	if err := ssi.RequireNonEmpty(accountNo, "accountNo"); err != nil {
		return nil, err
	}
	ppmmr, err := s.getPPMMR(accountNo)
	if err != nil {
		return nil, err
	}
	return ppmmr.Equity, nil
}

// GetDerivativePPMMR returns the derivative purchasing power data (nil when the account has none).
func (s *Service) GetDerivativePPMMR(accountNo string) (*DerivativePPMMR, error) {
	if err := ssi.RequireNonEmpty(accountNo, "accountNo"); err != nil {
		return nil, err
	}
	ppmmr, err := s.getPPMMR(accountNo)
	if err != nil {
		return nil, err
	}
	return ppmmr.Derivative, nil
}
