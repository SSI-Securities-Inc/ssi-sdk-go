package ssisdk

import "testing"

func TestServerErrorCodeLookup(t *testing.T) {
	if len(knownServerErrorCodes) != 29 {
		t.Fatalf("want 29 distinct codes, got %d", len(knownServerErrorCodes))
	}
	if ErrOTPSendFailed != ErrSymbolRequired || ErrSignatureMismatch != ErrSignatureInvalid ||
		ErrPushApprovalPending != ErrSmartOTPPending || ErrInvalidAPICredential != ErrTokenWithoutOTP {
		t.Fatal("alias codes must share the number")
	}
	if c, ok := LookupServerErrorCode(401114); !ok || c != ErrSmartOTPPending {
		t.Fatalf("401114 -> %v %v", c, ok)
	}
	if _, ok := LookupServerErrorCode(999999); ok {
		t.Fatal("unknown code must not be known")
	}
	if ServerErrorCode(504209).IsKnown() != true || ServerErrorCode(1).IsKnown() {
		t.Fatal("IsKnown")
	}
}
