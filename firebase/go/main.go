// Server verifies phone numbers with Prelude and mints Firebase Auth custom
// tokens for the verified users.
//
//	POST /send_code  {"phone_number": "+33..."}                  -> 204
//	POST /verify     {"phone_number": "+33...", "code": "1234"}  -> 200 {"token": "..."}
//
// See the README for the environment it expects.
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	firebase "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"
	prelude "github.com/prelude-so/go-sdk"
)

const addr = ":8080"

func main() {
	ctx := context.Background()

	app, err := firebase.NewApp(ctx, nil)
	if err != nil {
		log.Fatalf("initialize firebase: %v", err)
	}

	authClient, err := app.Auth(ctx)
	if err != nil {
		log.Fatalf("initialize firebase auth: %v", err)
	}

	// NewClient reads the API token from API_TOKEN.
	srv := &server{prelude: prelude.NewClient(), auth: authClient}

	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, srv.handler()))
}

// server wires the Prelude and Firebase clients into the HTTP handlers.
type server struct {
	prelude *prelude.Client
	auth    *firebaseauth.Client
}

// handler routes the two endpoints. The mux answers anything else, including a
// known path reached with the wrong method.
func (s *server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /send_code", s.sendCode)
	mux.HandleFunc("POST /verify", s.verify)
	return mux
}

func (s *server) sendCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PhoneNumber string `json:"phone_number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	_, err := s.prelude.Verification.New(r.Context(), prelude.VerificationNewParams{
		Target: prelude.F(prelude.VerificationNewParamsTarget{
			Type:  prelude.F(prelude.VerificationNewParamsTargetTypePhoneNumber),
			Value: prelude.F(req.PhoneNumber),
		}),
	})
	if err != nil {
		http.Error(w, "could not send code", http.StatusBadGateway)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *server) verify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PhoneNumber string `json:"phone_number"`
		Code        string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	check, err := s.prelude.Verification.Check(r.Context(), prelude.VerificationCheckParams{
		Target: prelude.F(prelude.VerificationCheckParamsTarget{
			Type:  prelude.F(prelude.VerificationCheckParamsTargetTypePhoneNumber),
			Value: prelude.F(req.PhoneNumber),
		}),
		Code: prelude.F(req.Code),
	})
	if err != nil {
		http.Error(w, "could not check code", http.StatusBadGateway)
		return
	}

	if check.Status != prelude.VerificationCheckResponseStatusSuccess {
		http.Error(w, "verification failed", http.StatusUnauthorized)
		return
	}

	token, err := s.customToken(r.Context(), req.PhoneNumber)
	if err != nil {
		http.Error(w, "could not issue token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"token": token})
}

// customToken returns a Firebase custom token for phoneNumber, creating the
// user on first sign-in. A lookup failure that is not "no such user" is
// reported rather than mistaken for a missing user, which would create a
// duplicate.
func (s *server) customToken(ctx context.Context, phoneNumber string) (string, error) {
	user, err := s.auth.GetUserByPhoneNumber(ctx, phoneNumber)
	if firebaseauth.IsUserNotFound(err) {
		user, err = s.auth.CreateUser(ctx, (&firebaseauth.UserToCreate{}).PhoneNumber(phoneNumber))
	}
	if err != nil {
		return "", err
	}

	return s.auth.CustomToken(ctx, user.UID)
}
