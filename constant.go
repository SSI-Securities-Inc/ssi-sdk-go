package ssisdk

// Protocol constants fixed by the server/spec (not user-configurable).

const (
	// DefaultSize is the page size SDK requests are sent with. The server defaults to 10 and has
	// no upper bound, so a large explicit size cuts round trips.
	DefaultSize = 1000
	DefaultPage = 1
)

const (
	DefaultAPIURL             = "https://api.ssi.com.vn"
	DefaultStreamingURL       = "wss://stream.ssi.com.vn/ws/v3"
	DefaultTimeoutMs          = 60000 // milliseconds
	DefaultMaxRetries         = 5
	DefaultRetryDelayMs       = 2000 // milliseconds
	DefaultRateLimitPerSecond = 10

	// DefaultTradingAPIDomain follows APIURL while left at this default.
	DefaultTradingAPIDomain = DefaultAPIURL
	// TradingWSPath is the path of the trading WebSocket.
	TradingWSPath            = "/ws/v3/trading"
	DefaultTradingWSDomain   = "wss://api.ssi.com.vn" + TradingWSPath
	DefaultOTPPollIntervalMs = 5000  // milliseconds
	DefaultOTPPollMaxWaitMs  = 60000 // milliseconds
	MaxBatchOrders           = 20
	MaxClientRequestIDLength = 20
	HTTPStatusConflict       = 409
	TokenExpirySkewMs        = 30000 // a token this close to expiry counts as expired
	SmartOTPPendingStatus    = 202
	SmartOTPPendingCode      = 401114
	RateLimitMaxRetries      = 3
	RateLimitDefaultWaitMs   = 1000  // ms when a 429 carries no Retry-After (doubles per retry)
	RateLimitMaxWaitMs       = 30000 // ms; a longer wait is not taken silently: RateLimitError is returned
	WSReconnectMaxDelayMs    = 60000 // ms; upper bound of the exponential backoff
	WSRateLimitBackoffMs     = 30000 // ms; minimum wait after the server answers 429 on connect
	WSTokenRefreshLeadMs     = 60000 // ms; reconnect with a fresh token this long before expiry
	WSServerMaxConnections   = 10
	WSMaxFrameBytes          = 4000 // server MaxMessageSize is 4096; keep frames under it
	TopicMarketFlag          = "market.flag"
)

// Paths served by the trading API host (orders, FCO, account, portfolio).
var TradingAPIPathPrefixes = []string{"/api/v3/trading", "/api/v3/account"}

// HTTP headers.
const (
	HeaderContentType        = "Content-Type"
	HeaderAccept             = "Accept"
	HeaderAuthorization      = "Authorization"
	HeaderRetryAfter         = "Retry-After"
	HeaderRateLimitLimit     = "X-RateLimit-Limit"
	HeaderRateLimitRemaining = "X-RateLimit-Remaining"
	HeaderSignature          = "X-Signature"
	HeaderUserAgent          = "User-Agent"
	ContentTypeJSON          = "application/json"
	AuthSchemeBearer         = "Bearer "
)

// SmartOTPTerminalCodes: codes after which Smart OTP polling can never succeed.
var SmartOTPTerminalCodes = map[int]bool{401113: true, 401115: true, 401116: true}

// RefreshReauthCodes: refresh failures meaning the refresh token is unusable; a new OTP is required.
var RefreshReauthCodes = map[int]bool{
	401101: true, 401102: true, 401103: true, 401104: true,
	401106: true, 401107: true, 401108: true,
}

// Synthetic dispatch routes for WebSocket frames that carry no "channel".
const (
	WSRouteAck              = "@ack"
	WSRouteError            = "@error"
	WSRouteListSubscription = "@list_subscription"
	WSRouteID               = "@id"
)

// DefaultUserAgent is sent when Config.UserAgent is empty.
const DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// ServerErrorCode is an application error code the server returns in {"code","msg"}.
// A few numbers are reused in different places; the later names are aliases of the first
// (400104 is both "OTP could not be sent" and "symbol is required"). The signature codes
// (4010xx) do not line up with the server's own error-code-mapping.json: trust the number.
type ServerErrorCode int

const (
	// auth / account / OTP
	ErrCredentialNotLinkedToClient    ServerErrorCode = 400101
	ErrTwoFAInfoError                 ServerErrorCode = 400102
	ErrTwoFAInfoMissing               ServerErrorCode = 400103
	ErrOTPSendFailed                  ServerErrorCode = 400104
	ErrSymbolRequired                 ServerErrorCode = 400104
	ErrPushApprovalCreateFailed       ServerErrorCode = 400171
	ErrOTPVerificationFailed          ServerErrorCode = 401100
	ErrRefreshTokenInvalid            ServerErrorCode = 401101
	ErrRefreshClientMissing           ServerErrorCode = 401102
	ErrRefreshTokenRevokedOrExpired   ServerErrorCode = 401103
	ErrRefreshAPIKeyGone              ServerErrorCode = 401104
	ErrInvalidAPICredential           ServerErrorCode = 401105
	ErrTokenWithoutOTP                ServerErrorCode = 401105
	ErrPushApprovalNotFound           ServerErrorCode = 401113
	ErrPushApprovalPending            ServerErrorCode = 401114
	ErrSmartOTPPending                ServerErrorCode = 401114
	ErrPushApprovalRejected           ServerErrorCode = 401115
	ErrPushApprovalForeignTransaction ServerErrorCode = 401116
	ErrNoScopeGranted                 ServerErrorCode = 403100

	// request signature
	ErrSignatureMissing   ServerErrorCode = 401006
	ErrSignatureBodyEmpty ServerErrorCode = 401008
	ErrSignatureMismatch  ServerErrorCode = 401013
	ErrSignatureInvalid   ServerErrorCode = 401013

	// trading
	ErrModifyPriceAndQuantity ServerErrorCode = 400107
	ErrReplayWindowExceeded   ServerErrorCode = 400111

	// market data validators
	ErrSymbolInvalidCharacters              ServerErrorCode = 400006
	ErrFromAfterTo                          ServerErrorCode = 400202
	ErrBoardNotAllowed                      ServerErrorCode = 400208
	ErrTimeframeNotSupported                ServerErrorCode = 400210
	ErrDateFormatInvalid                    ServerErrorCode = 400213
	ErrIndexSummaryNeedsOneOf               ServerErrorCode = 504207
	ErrSecuritiesInfoNeedsOneOf             ServerErrorCode = 504208
	ErrSecuritiesSummaryNeedsSymbolXorIndex ServerErrorCode = 504209
)

var knownServerErrorCodes = map[ServerErrorCode]bool{
	ErrCredentialNotLinkedToClient: true, ErrTwoFAInfoError: true, ErrTwoFAInfoMissing: true,
	ErrOTPSendFailed: true, ErrPushApprovalCreateFailed: true, ErrOTPVerificationFailed: true,
	ErrRefreshTokenInvalid: true, ErrRefreshClientMissing: true, ErrRefreshTokenRevokedOrExpired: true,
	ErrRefreshAPIKeyGone: true, ErrInvalidAPICredential: true, ErrPushApprovalNotFound: true,
	ErrPushApprovalPending: true, ErrPushApprovalRejected: true, ErrPushApprovalForeignTransaction: true,
	ErrNoScopeGranted: true, ErrSignatureMissing: true, ErrSignatureBodyEmpty: true,
	ErrSignatureMismatch: true, ErrModifyPriceAndQuantity: true, ErrReplayWindowExceeded: true,
	ErrSymbolInvalidCharacters: true, ErrFromAfterTo: true, ErrBoardNotAllowed: true,
	ErrTimeframeNotSupported: true, ErrDateFormatInvalid: true, ErrIndexSummaryNeedsOneOf: true,
	ErrSecuritiesInfoNeedsOneOf: true, ErrSecuritiesSummaryNeedsSymbolXorIndex: true,
}

// IsKnown reports whether the code is one the server documents. An unknown code never errors.
func (c ServerErrorCode) IsKnown() bool { return knownServerErrorCodes[c] }

// LookupServerErrorCode returns the code and true when it is a documented server code.
func LookupServerErrorCode(code int) (ServerErrorCode, bool) {
	c := ServerErrorCode(code)
	return c, c.IsKnown()
}
