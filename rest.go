package main

import (
	"crypto/sha512"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/patrickmn/go-cache"
	"github.com/rs/cors"
)

const (
	RESTPORT              = ":5000"
	RESTClientCert string = "./rest/cert1.pem"
	RESTClientKey  string = "./rest/privkey1.pem"
)

var (
	attemptCooldown  = 14 * time.Minute
	badLoginAttempts = cache.New(attemptCooldown, attemptCooldown)

	usernameRegexp = regexp.MustCompile(`^[-a-zA-Z0-9]{3,16}$`)
	passwordRegexp = regexp.MustCompile(`^[^ ]{2,32}$`)
)

// startRouter starts the mux router and blocks until a crash or
// a SIGINT signal.
func startRouter() {
	r := mux.NewRouter()
	// Standard GET methods to retrieve blogs and pages
	r.HandleFunc("/", rootPage).Methods(http.MethodGet)

	s := r.PathPrefix("/cmd").Subrouter()

	s.HandleFunc("/temp", tempHandler).Methods(http.MethodGet)
	s.HandleFunc("/temps", tempsHandler).Methods(http.MethodGet)
	s.HandleFunc("/led", ledHandler).Methods(http.MethodPost)
	s.HandleFunc("/auth", verifyAuth).Methods(http.MethodPost)

	s.Use(loggingMiddleware)

	// Declare and define our HTTP handler
	//handler := cors.Default().Handler(r)
	corsOptions := cors.New(cors.Options{
		AllowedOrigins:   []string{"https://sandyuraz.com"},
		AllowedMethods:   []string{http.MethodPost, http.MethodGet},
		AllowedHeaders:   []string{"Access-Control-Allow-Methods", "Authorization", "Content-Type"},
		AllowCredentials: true,
		Debug:            false,
	})
	handler := corsOptions.Handler(r)
	srv := &http.Server{
		Handler: handler,
		Addr:    RESTPORT,
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
		ipAddr, err := extractIP(r)
		if err != nil {
			httpJSON(w, nil, http.StatusBadRequest, errors.New("unknown origin"))
			return
		}
		if number, found := badLoginAttempts.Get(ipAddr); found && number.(uint) >= 4 {
			httpJSON(w, nil, http.StatusForbidden, errors.New("origin blocked"))
			return
		}
		tokens := strings.Split(r.Header.Get("Authorization"), "Basic ")
		if len(tokens) != 2 {
			httpJSON(w, nil, http.StatusBadRequest, errors.New("no basic auth provided"))
			return
		}
		decoded, err := base64.StdEncoding.DecodeString(tokens[1])
		if err != nil {
			httpJSON(w, nil, http.StatusBadRequest, errors.New("base64 decoding failed"))
			return
		}
		creds := strings.Split(string(decoded), ":")
		if len(creds) != 2 {
			httpJSON(w, nil, http.StatusBadRequest, errors.New("basic auth is malformed"))
			return
		}
		user, pass := creds[0], creds[1]
		if user == "" {
			httpJSON(w, nil, http.StatusBadRequest, errors.New("bad user credentials"))
			return
		}
		// Quickly sanitize the username and the password
		if !usernameRegexp.MatchString(user) || !passwordRegexp.MatchString(pass) {
			httpJSON(w, nil, http.StatusBadRequest, errors.New("bad user credentials"))
			return
		}
		foundUser, err := getUser(user)
		if err != nil || foundUser.Name == "" {
			httpJSON(w, nil, http.StatusForbidden, errors.New("bad user credentials"))
			return
		}
		if foundUser.Password != shaEncode(pass) {
			httpJSON(w, nil, http.StatusForbidden, errors.New("bad user credentials"))
			// Someone is maybe trying to guess the password
			badLoginAttempts.Add(ipAddr, uint(0), cache.DefaultExpiration)
			badLoginAttempts.IncrementUint(ipAddr, 1)
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

// verifyAuth verifies that the credentials are OK
func verifyAuth(w http.ResponseWriter, r *http.Request) {
	httpJSON(w, httpMessageReturn{Message: "OK"}, http.StatusOK, nil)
}

// httpJSON is a generic http object passer.
func httpJSON(w http.ResponseWriter, data interface{}, status int, err error) {
	w.Header().Set("Content-Type", "application/json")
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

// extractIP makes sure the request has a proper request IP
func extractIP(r *http.Request) (string, error) {
	// if not a proper remote addr, return empty
	if !strings.ContainsRune(r.RemoteAddr, ':') {
		return "", errors.New("lol")
	}
	ipAddr, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || ipAddr == "" {
		return "", errors.New("Request has failed origin validation. Please retry.")
	}
	return ipAddr, nil
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
