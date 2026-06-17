package google

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"google.golang.org/api/googleapi"

	docalendar "github.com/luckyrogue/lead-cat/internal/domain/calendar"
)

func TestIsGoogleAPIDisabled(t *testing.T) {
	disabled := &googleapi.Error{Code: 403, Errors: []googleapi.ErrorItem{{Reason: "accessNotConfigured"}}}
	if !isGoogleAPIDisabled(disabled) {
		t.Fatal("403 accessNotConfigured should be disabled")
	}
	msg := &googleapi.Error{Code: 403, Message: "Calendar API has not been used in project ... is disabled"}
	if !isGoogleAPIDisabled(msg) {
		t.Fatal("403 'has not been used' should be disabled")
	}
	if isGoogleAPIDisabled(&googleapi.Error{Code: 404}) {
		t.Fatal("404 is not disabled")
	}
	if isGoogleAPIDisabled(errors.New("plain")) {
		t.Fatal("non-googleapi error is not disabled")
	}
}

func TestIsImpersonationFail(t *testing.T) {
	if !isImpersonationFail(errors.New("oauth2: unauthorized_client blah")) {
		t.Fatal("unauthorized_client")
	}
	if !isImpersonationFail(errors.New("...Not Authorized to access this resource...")) {
		t.Fatal("not authorized message")
	}
	if !isImpersonationFail(&googleapi.Error{Code: 401}) {
		t.Fatal("401 googleapi")
	}
	if isImpersonationFail(errors.New("something else")) {
		t.Fatal("unrelated should be false")
	}
}

func TestIsJSONParseErr(t *testing.T) {
	if !isJSONParseErr(errors.New("private key should be a PEM or plain PKCS1 or PKCS8")) {
		t.Fatal("PEM message")
	}
	if isJSONParseErr(errors.New("totally unrelated")) {
		t.Fatal("unrelated should be false")
	}
}

func TestMapProbeError(t *testing.T) {
	cases := map[error]error{
		nil:                                 nil,
		fmt.Errorf("%w: x", ErrJSONParse):   docalendar.ErrProbeSAInvalid,
		fmt.Errorf("%w: x", ErrAPIDisabled): docalendar.ErrProbeAPIDisabled,
		fmt.Errorf("%w: x", ErrSubject):     docalendar.ErrProbeSubject,
		fmt.Errorf("%w: x", ErrCalendar):    docalendar.ErrProbeCalendar,
		errors.New("unknown"):               docalendar.ErrProbeCalendar,
	}
	for in, want := range cases {
		if got := mapProbeError(in); got != want && !errors.Is(got, want) {
			t.Fatalf("mapProbeError(%v) = %v, want %v", in, got, want)
		}
	}
}

func TestProbe_BadSAJSON_NoNetwork(t *testing.T) {
	_, err := Probe(context.Background(), "not json", "s@x.io", "primary")
	if !errors.Is(err, ErrJSONParse) {
		t.Fatalf("want ErrJSONParse, got %v", err)
	}
	if _, perr := (Prober{}).Probe(context.Background(), "not json", "s@x.io", "primary"); !errors.Is(perr, docalendar.ErrProbeSAInvalid) {
		t.Fatalf("Prober want ErrProbeSAInvalid, got %v", perr)
	}
}
