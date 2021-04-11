package main

import (
	"crypto/sha512"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

var (
	RESTClientCert string = "./rest/cert1.pem"
	RESTClientKey  string = "./rest/privkey1.pem"
)

// startRouter starts the mux router and blocks until a crash or
// a SIGINT signal.
func startRouter() {
	r := mux.NewRouter()
	// Standard GET methods to retrieve blogs and pages
	r.HandleFunc("/", rootPage).Methods(http.MethodGet)
	r.HandleFunc("/temp", tempHandler).Methods(http.MethodGet)
	r.HandleFunc("/temps", tempsHandler).Methods(http.MethodGet)
	r.HandleFunc("/led", ledHandler).Methods(http.MethodPost)

	r.Use(loggingMiddleware)

	// Declare and define our HTTP handler
	handler := cors.Default().Handler(r)
	srv := &http.Server{
		Handler: handler,
		Addr:    ":5000",
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	// Fire up the router
	go func() {
		if err := srv.ListenAndServeTLS(RESTClientCert, RESTClientKey); err != nil {
			lerr("Failed to fire up the router", err, params{})
		}
	}()
	// Listen to SIGINT and other shutdown signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		tokens := strings.Split(header, "Basic ")
		if len(tokens) != 2 {
			httpJSON(w, nil, http.StatusBadRequest, errors.New("no auth provided"))
			return
		}
		decoded, err := base64.StdEncoding.DecodeString(tokens[1])
		if err != nil {
			httpJSON(w, nil, http.StatusBadRequest, errors.New("failed auth decoding"))
			return
		}
		creds := strings.Split(string(decoded), ":")
		if len(creds) != 2 {
			httpJSON(w, nil, http.StatusBadRequest, errors.New("malformed auth"))
			return
		}
		user, pass := creds[0], creds[1]
		if user == "" {
			httpJSON(w, nil, http.StatusBadRequest, errors.New("no user provided"))
			return
		}
		foundUser, err := getUser(user)
		if err != nil || foundUser.Name == "" {
			httpJSON(w, nil, http.StatusForbidden, errors.New("no user found"))
			return
		}
		if foundUser.Password != shaEncode(pass) {
			httpJSON(w, nil, http.StatusForbidden, errors.New("bad login"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// rootPage is a generic placeholder HTTP handler
func rootPage(w http.ResponseWriter, r *http.Request) {
	httpHTML(w, "hello, world")
}

// tempHandler returns the last temperature reading.
func tempHandler(w http.ResponseWriter, r *http.Request) {
	temp, err := getTempDB()
	if err != nil {
		httpJSON(w, nil, http.StatusInternalServerError, err)
		return
	}
	httpJSON(w, httpMessageReturn{Message: temp.Value}, http.StatusOK, nil)
}

// tempsHandler returns the last 60 recorded temperature values.
func tempsHandler(w http.ResponseWriter, r *http.Request) {
	temp, err := getTempsDB()
	if err != nil {
		httpJSON(w, nil, http.StatusInternalServerError, err)
		return
	}
	toReturn := make([]float64, len(temp))
	for i, v := range temp {
		toReturn[i] = v.Value
	}
	httpJSON(w, httpMessageReturn{Message: toReturn}, http.StatusOK, nil)
}

// ledHandler handles POST to control lights.
func ledHandler(w http.ResponseWriter, r *http.Request) {
	request := &ledStatusRequset{}
	json.NewDecoder(r.Body).Decode(request)
	toSend := "off"
	if request.Status {
		toSend = "on"
	}
	publish(1, topicLED, toSend)
	httpJSON(w, httpMessageReturn{Message: "OK"}, http.StatusOK, nil)
}

// httpJSON is a generic http object passer.
func httpJSON(w http.ResponseWriter, data interface{}, status int, err error) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	if err != nil && status >= 400 && status < 600 {
		json.NewEncoder(w).Encode(httpErrorReturn{Error: err.Error()})
		return
	}
	json.NewEncoder(w).Encode(data)
}

// httpHTML sends a good HTML response.
func httpHTML(w http.ResponseWriter, data interface{}) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, data)
}

// shaEncode return SHA512 sum of a string.
func shaEncode(input string) string {
	sha := sha512.Sum512([]byte(input))
	return hex.EncodeToString(sha[:])
}

// UserCredentials unparses POST request.
type UserCredentials struct {
	Name     string `json:"username"`
	Password string `json:"password"`
}

// ledStatusRequset is the POST body for LED.
type ledStatusRequset struct {
	Status bool `json:"status"`
}

// httpMessageReturn defines a generic HTTP return message.
type httpMessageReturn struct {
	Message interface{} `json:"message"`
}

// httpErrorReturn defines a generic HTTP error message.
type httpErrorReturn struct {
	Error string `json:"error"`
}
