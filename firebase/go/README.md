# Firebase Auth and Prelude Go SDK

Integrating Firebase Auth with Prelude's Go SDK is a pretty straightforward
task. Here we present a simple server exposing two API endpoints that allow you
to verify a phone number and then create (or authenticate) the corresponding
user in Firebase Auth.

You can read a full explanation of this example in our
[documentation website][1].

## Running the example

Requires Go 1.26 or later.

The server reads its configuration from the environment:

- `API_TOKEN`: your Prelude API token, read by the Go SDK itself. Without it,
  requests to Prelude fail with `502`.
- `GOOGLE_APPLICATION_CREDENTIALS`: path to a Firebase service account key file.
  Can be left unset when running on Google Cloud, where the credentials come
  from the metadata server.

```sh
export API_TOKEN=...
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json

go run .
```

It listens on `:8080` and exposes two endpoints:

```sh
# Send a code to the phone number. Responds 204.
curl -i -X POST localhost:8080/send_code \
  -d '{"phone_number": "+33xxxxxxxxx"}'

# Check the code and get a Firebase custom token back. Responds 200.
curl -i -X POST localhost:8080/verify \
  -d '{"phone_number": "+33xxxxxxxxx", "code": "12345678"}'
# {"token":"eyJhbGci..."}
```

Your client then signs in with that token through
[`signInWithCustomToken`][2].

Errors come back as a status code and a short plain-text message: `400` for a
malformed body, `401` for a code that did not check out, `502` when the Prelude
API call failed.

To build a binary, name it explicitly. This directory is called `go`, so a bare
`go build` would produce an executable named `go`:

```sh
go build -o server .
```

## Tests

```sh
go test ./...
```

The tests need no credentials and make no network calls. They run the real
router against two `httptest` servers: one standing in for the Prelude API, and
one for Firebase, reached by pointing `FIREBASE_AUTH_EMULATOR_HOST` at it. In
emulator mode the Firebase SDK signs custom tokens locally, so no service
account key or emulator install is required.

[1]: https://docs.prelude.so/verify/v2/documentation/integrations/firebase
[2]: https://firebase.google.com/docs/auth/web/custom-auth
