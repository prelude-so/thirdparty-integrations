package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	firebase "firebase.google.com/go/v4"
	prelude "github.com/prelude-so/go-sdk"
	preludeoption "github.com/prelude-so/go-sdk/option"
)

const (
	testPhoneNumber = "+33123456789"
	sendBody        = `{"phone_number":"` + testPhoneNumber + `"}`
	verifyBody      = `{"phone_number":"` + testPhoneNumber + `","code":"12345678"}`
)

// fakePrelude stands in for the Prelude API. The SDK talks to it over HTTP, so
// requests and responses go through the real marshaling code.
type fakePrelude struct {
	// checkStatus is the "status" field of the check response.
	checkStatus string

	newTarget map[string]any // target the New call sent, for assertions
}

func (f *fakePrelude) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.URL.Path {
	case "/v2/verification":
		var body struct {
			Target map[string]any `json:"target"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		f.newTarget = body.Target

		json.NewEncoder(w).Encode(map[string]any{
			"id": "vrf_1", "method": "message", "status": "success",
		})

	case "/v2/verification/check":
		json.NewEncoder(w).Encode(map[string]any{
			"id": "vrf_1", "status": f.checkStatus,
		})

	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// fakeFirebase stands in for the Identity Toolkit API. Setting
// FIREBASE_AUTH_EMULATOR_HOST makes the Firebase SDK address it over plain HTTP
// with no credentials, and sign custom tokens locally.
type fakeFirebase struct {
	// existingUID, when set, is the account already registered for the number.
	existingUID string

	createdUID string // UID of the account created during the test, if any
}

func (f *fakeFirebase) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case strings.HasSuffix(r.URL.Path, "/accounts:lookup"):
		var body struct {
			LocalID []string `json:"localId"`
		}
		json.NewDecoder(r.Body).Decode(&body)

		switch {
		case len(body.LocalID) > 0:
			// CreateUser looks the new account up by UID once it exists.
			f.writeUser(w, body.LocalID[0])
		case f.existingUID != "":
			f.writeUser(w, f.existingUID)
		default:
			// No account for this number. The SDK turns an empty list into an
			// error that IsUserNotFound recognizes.
			json.NewEncoder(w).Encode(map[string]any{"users": []any{}})
		}

	case strings.HasSuffix(r.URL.Path, "/accounts"):
		f.createdUID = "created-uid"
		json.NewEncoder(w).Encode(map[string]any{"localId": f.createdUID})

	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (f *fakeFirebase) writeUser(w http.ResponseWriter, uid string) {
	json.NewEncoder(w).Encode(map[string]any{
		"users": []any{map[string]any{
			"localId":     uid,
			"phoneNumber": testPhoneNumber,
		}},
	})
}

// newTestHandler builds the real router against the two fakes.
func newTestHandler(t *testing.T, fp *fakePrelude, ff *fakeFirebase) http.Handler {
	t.Helper()

	preludeSrv := httptest.NewServer(fp)
	t.Cleanup(preludeSrv.Close)

	firebaseSrv := httptest.NewServer(ff)
	t.Cleanup(firebaseSrv.Close)

	// Must be set before the Firebase client is built.
	t.Setenv("FIREBASE_AUTH_EMULATOR_HOST", strings.TrimPrefix(firebaseSrv.URL, "http://"))
	t.Setenv("GOOGLE_CLOUD_PROJECT", "demo-test")

	ctx := t.Context()

	app, err := firebase.NewApp(ctx, nil)
	if err != nil {
		t.Fatalf("initialize firebase: %v", err)
	}

	authClient, err := app.Auth(ctx)
	if err != nil {
		t.Fatalf("initialize firebase auth: %v", err)
	}

	srv := &server{
		prelude: prelude.NewClient(
			preludeoption.WithAPIToken("test-token"),
			preludeoption.WithBaseURL(preludeSrv.URL+"/"),
		),
		auth: authClient,
	}
	return srv.handler()
}

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

// TestSendCode checks that a request reaches Prelude with the phone number
// intact.
func TestSendCode(t *testing.T) {
	fp := &fakePrelude{}

	rec := do(newTestHandler(t, fp, &fakeFirebase{}), "POST", "/send_code", sendBody)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusNoContent, rec.Body)
	}
	if got, want := fp.newTarget["value"], testPhoneNumber; got != want {
		t.Errorf("target value = %v, want %v", got, want)
	}
	if got, want := fp.newTarget["type"], "phone_number"; got != want {
		t.Errorf("target type = %v, want %v", got, want)
	}
}

// TestVerify checks that a valid code yields a Firebase custom token for the
// phone number, whether or not the user already exists.
func TestVerify(t *testing.T) {
	tests := []struct {
		name     string
		firebase fakeFirebase
		wantUID  string
	}{
		{"first sign-in creates the user", fakeFirebase{}, "created-uid"},
		{"returning user", fakeFirebase{existingUID: "existing-uid"}, "existing-uid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ff := tt.firebase
			h := newTestHandler(t, &fakePrelude{checkStatus: "success"}, &ff)

			rec := do(h, "POST", "/verify", verifyBody)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d (body %q)", rec.Code, http.StatusOK, rec.Body)
			}
			if got := tokenUID(t, rec.Body.Bytes()); got != tt.wantUID {
				t.Errorf("token uid = %q, want %q", got, tt.wantUID)
			}
		})
	}
}

// TestVerifyRejectsWrongCode is the one failure path worth pinning: a code that
// does not check out must not produce a token.
func TestVerifyRejectsWrongCode(t *testing.T) {
	h := newTestHandler(t, &fakePrelude{checkStatus: "failure"}, &fakeFirebase{})

	rec := do(h, "POST", "/verify", verifyBody)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if strings.Contains(rec.Body.String(), "token") {
		t.Errorf("body %q leaks a token", rec.Body)
	}
}

// tokenUID pulls the uid claim out of the custom token in a /verify response.
func tokenUID(t *testing.T, body []byte) string {
	t.Helper()

	var resp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	parts := strings.Split(resp.Token, ".")
	if len(parts) != 3 {
		t.Fatalf("token %q is not a JWT", resp.Token)
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode token payload %q: %v", parts[1], err)
	}

	var claims struct {
		UID string `json:"uid"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("decode token claims %s: %v", payload, err)
	}
	return claims.UID
}
