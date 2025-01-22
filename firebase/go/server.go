package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/prelude-so/go-sdk"
	preludeOption "github.com/prelude-so/go-sdk/option"
	googleOption "google.golang.org/api/option"
)

const (
	port                    = ":8080"                                    // Or any other HTTP port
	firebaseCredentialsFile = "YOUR_FIREBASE_CREDENTIALS_FILE_WITH_PATH" // Or use an env variable
	preludeApiKey           = "YOUR_PRELUDE_API_KEY"
)

func main() {
	ctx := context.Background()
	opt := googleOption.WithCredentialsFile(firebaseCredentialsFile)

	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}

	authClient, err := app.Auth(ctx)
	if err != nil {
		log.Fatalf("error initializing auth client: %s", err)
	}

	client := prelude.NewClient(
		preludeOption.WithAPIToken(preludeApiKey),
	)

	http.HandleFunc("/send_code", SendCode(client, ctx))
	http.HandleFunc("/verify", Check(client, ctx, authClient))

	log.Printf("Listening on port %s", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Start server error: %v", err)
	}
}

func SendCode(preludeClient *prelude.Client, ctx context.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		type request struct {
			PhoneNumber string `json:"phone_number"`
		}

		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		_, err := preludeClient.Verification.New(ctx, prelude.VerificationNewParams{
			Target: prelude.F(prelude.VerificationNewParamsTarget{
				Type:  prelude.F(prelude.VerificationNewParamsTargetTypePhoneNumber),
				Value: prelude.F(req.PhoneNumber),
			}),
		})
		if err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func Check(preludeClient *prelude.Client, ctx context.Context, firebaseAuthClient *auth.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		type request struct {
			PhoneNumber string `json:"phone_number"`
			Code        string `json:"code"`
		}

		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		checkResult, err := preludeClient.Verification.Check(ctx, prelude.VerificationCheckParams{
			Target: prelude.F(prelude.VerificationCheckParamsTarget{
				Type:  prelude.F(prelude.VerificationCheckParamsTargetTypePhoneNumber),
				Value: prelude.F(req.PhoneNumber),
			}),
			Code: prelude.F(req.Code),
		})
		switch {
		case err != nil:
			http.Error(w, err.Error(), http.StatusBadRequest)
		case checkResult.Status != prelude.VerificationCheckResponseStatusSuccess:
			http.Error(w, checkResult.JSON.RawJSON(), http.StatusBadRequest)
		default:
			firebaseToken, err := CreateFirebaseUser(r.Context(), firebaseAuthClient, req.PhoneNumber)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else {
				err := json.NewEncoder(w).Encode(struct {
					Token string `json:"token"`
				}{firebaseToken})
				if err != nil {
					return
				}
			}
		}
	}
}

func CreateFirebaseUser(ctx context.Context, firebaseAuth *auth.Client, phoneNumber string) (string, error) {
	user, err := firebaseAuth.GetUserByPhoneNumber(ctx, phoneNumber)
	if err != nil {
		newUser := (&auth.UserToCreate{}).PhoneNumber(phoneNumber)
		createdUser, err := firebaseAuth.CreateUser(ctx, newUser)
		if err != nil {
			return "", err
		}
		user = createdUser
	}

	customToken, err := firebaseAuth.CustomToken(ctx, user.UID)
	if err != nil {
		return "", err
	}
	return customToken, nil
}
