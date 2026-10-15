package trading

import (
	"time"

	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/device"
)

// PayloadBuilder builds the validated JSON text of each order command without sending it. The
// experimental trading WebSocket signs and sends these texts verbatim; the REST Service uses the
// same builders, so both transports enforce identical rules (device id, idempotency key, plain
// decimal prices, batch and FCO window limits).
type PayloadBuilder struct{ cfg *ssi.Config }

// NewPayloadBuilder returns a builder bound to cfg (read-only).
func NewPayloadBuilder(cfg *ssi.Config) *PayloadBuilder { return &PayloadBuilder{cfg: cfg} }

// PlaceOrder is the body of a new order. price 0 is valid only for market-priced types.
func (b *PayloadBuilder) PlaceOrder(accountNo, symbol string, side OrderSide, quantity int, price float64, orderType OrderType, clientRequestID string) (string, error) {
	if err := ssi.RequireNonNegative(price, "price"); err != nil {
		return "", err
	}
	req, err := buildPlaceOrder(b.cfg, accountNo, symbol, side, quantity, price, orderType, clientRequestID)
	if err != nil {
		return "", err
	}
	return req.ToJSON()
}

// ModifyOrder changes either the price or the quantity of one order.
func (b *PayloadBuilder) ModifyOrder(accountNo, orderID, clientRequestID string, price *float64, quantity *int) (string, error) {
	req, err := buildModifyOrder(b.cfg, accountNo, orderID, clientRequestID, price, quantity)
	if err != nil {
		return "", err
	}
	return req.ToJSON()
}

// CancelOrder cancels one order identified by exactly one of orderID / clientRequestID.
func (b *PayloadBuilder) CancelOrder(accountNo, orderID, clientRequestID string) (string, error) {
	req, err := buildCancelOrder(b.cfg, accountNo, orderID, clientRequestID)
	if err != nil {
		return "", err
	}
	return req.ToJSON()
}

// BatchPlace validates every order, then builds the batch body.
func (b *PayloadBuilder) BatchPlace(items []BatchPlaceOrderItem) (string, error) {
	req, err := buildBatchPlace(b.cfg, items, time.Now())
	if err != nil {
		return "", err
	}
	return req.ToJSON()
}

// BatchCancel validates every item, then builds the batch body.
func (b *PayloadBuilder) BatchCancel(items []BatchCancelOrderItem) (string, error) {
	req, err := buildBatchCancel(b.cfg, items, time.Now())
	if err != nil {
		return "", err
	}
	return req.ToJSON()
}

// FCOPlace validates the validity window, fills the caller identity and builds the body. p is
// modified (identity fields only).
func (b *PayloadBuilder) FCOPlace(p FCOParamsRequest) (string, error) {
	if err := prepareFCO(b.cfg, p); err != nil {
		return "", err
	}
	return p.ToJSON()
}

// FCOCancel builds the cancel body; code is the optional one-time code.
func (b *PayloadBuilder) FCOCancel(fcoID, code string) (string, error) {
	if err := ssi.RequireNonEmpty(fcoID, "fcoId"); err != nil {
		return "", err
	}
	// userAgent is sent only when Config.UserAgent is set (unlike the place payloads).
	return (&FCOCancelRequest{FCOID: fcoID, DeviceID: device.ID(), UserAgent: b.cfg.UserAgent, Code: code}).ToJSON()
}

// FCOListParams validates the filters and returns them as command params (a single
// processStatus; page numbers only when > 0).
func (b *PayloadBuilder) FCOListParams(req *FCOListRequest) (map[string]interface{}, error) {
	if err := ssi.RequireNonEmpty(req.AccountNo, "accountNo"); err != nil {
		return nil, err
	}
	if len(req.ProcessStatus) > 1 {
		return nil, ssi.NewValidationError("fco.list takes a single process_status, not a list")
	}
	if err := checkFCOListDates(req); err != nil {
		return nil, err
	}
	params := map[string]interface{}{}
	for key, values := range req.ToQuery() {
		params[key] = values[0]
	}
	if req.PageIndex > 0 {
		params["pageIndex"] = req.PageIndex
	}
	if req.PageSize > 0 {
		params["pageSize"] = req.PageSize
	}
	return params, nil
}

func checkFCOListDates(req *FCOListRequest) error {
	if req.FromDate != "" && req.ToDate != "" {
		return ssi.RequireDateRange(req.FromDate, req.ToDate, true, false, 0)
	}
	for name, v := range map[string]string{"from": req.FromDate, "to": req.ToDate} {
		if v == "" {
			continue
		}
		if _, err := ssi.ParseDateArg(v, name, true, false); err != nil {
			return err
		}
	}
	return nil
}
