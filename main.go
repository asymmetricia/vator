package main

import (
	"flag"
	"fmt"
	"github.com/coreos/bbolt"
	"github.com/pdbogen/nokiahealth"
	. "github.com/pdbogen/vator/log"

	"crypto/tls"
	"github.com/pdbogen/vator/models"
	"golang.org/x/crypto/acme/autocert"
	"log"
	"net/http"
	"time"
)

// RequireForm returns an func(http.ResponseWriter,*http.Request) that wraps the given `handler`, ensuring
// it only receives requests that have form data, and requiring parameters with
// the named given by `required`.
func RequireForm(required []string, handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		if err := req.ParseForm(); err != nil {
			http.Error(rw, fmt.Sprintf("An error occurred parsing your request: %s", err), http.StatusBadRequest)
			return
		}

		for _, str := range required {
			if req.Form.Get(str) == "" {
				Log.Debugf("Missing parameter: %s", str)
				http.Error(rw, fmt.Sprintf("Your request was missing required parameters."), http.StatusBadRequest)
				return
			}
		}
		handler(rw, req)
	}
}

func RequireLink(db *bolt.DB, handler func(w http.ResponseWriter, r *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		user, err := models.LoadUserRequest(db, req)
		if err != nil {
			Bail(rw, req, fmt.Errorf("should be logged in, but: %s", err), http.StatusInternalServerError)
			return
		}
		if user.Id == 0 {
			http.Redirect(rw, req, "/", http.StatusFound)
			return
		}
		handler(rw, req)
	}
}

func IndexHandler(db *bolt.DB, nokia nokiahealth.Client) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		user, err := models.LoadUserRequest(db, req)
		if err != nil {
			Bail(rw, req, fmt.Errorf("should be logged in, but: %s", err), http.StatusInternalServerError)
			return
		}
		if user.Id == 0 {
			BeginOauth(db, nokia, rw, req)
		} else {
			ctx, err := notifications(db, req)
			if err != nil {
				Bail(rw, req, err, http.StatusInternalServerError)
				return
			}
			ctx["phone"] = user.Phone

			TemplateGet(rw, req, indexTemplate, ctx)
		}
	}
}

func main() {
	consumerKey := flag.String("consumer-key", "", "oauth consumer key")
	consumerSecret := flag.String("consumer-secret", "", "oauth consumer secret")
	port := flag.Int("port", 8080, "port to listen on")
	callbackDomain := flag.String("callback-domain", "localhost", "fqdn for oauth callbacks")
	callbackPort := flag.Int("callback-port", 0, "callback port; if zero, same as --port")
	callbackProto := flag.String("callback-proto", "http", "protocol to use in requesting callbacks")
	dbFile := flag.String("db-file", "vator.db", "path to the bolt database file used to persist state")

	twilioSid := flag.String("twilio-sid", "", "twilio account SID")
	twilioToken := flag.String("twilio-token", "", "twilio auth token")

	flag.Parse()

	if *consumerKey == "" {
		Log.Fatal("consumer key must be provided")
	}
	if *consumerSecret == "" {
		Log.Fatal("consumer secret must be provided")
	}

	var twilio *models.Twilio
	if *twilioSid == "" || *twilioToken == "" {
		Log.Warning("missing twilio-sid and/or twilio-token, toasts via SMS will not function")
	} else {
		var err error
		twilio, err = models.NewTwilio(*twilioSid, *twilioToken)
		if err != nil {
			log.Fatalf("error connecting to twilio: %s", err)
		}
	}

	if *callbackPort == 0 {
		*callbackPort = *port
	}

	db, err := bolt.Open(*dbFile, 0600, nil)
	if err != nil {
		Log.Fatalf("opening bolt db file %q: %s", *dbFile, err)
	}
	defer db.Close()

	client := nokiahealth.NewClient(*consumerKey, *consumerSecret, fmt.Sprintf("%s://%s:%d/callback", *callbackProto, *callbackDomain, *callbackPort))

	go func() {
		for {
			ScanMeasures(db, client, twilio)
			time.Sleep(time.Minute)
		}
	}()

	certmgr := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(*callbackDomain),
	}

	sessionizer := http.NewServeMux()
	sessionizer.HandleFunc("/", models.WithSession(db, http.DefaultServeMux.ServeHTTP))

	http.HandleFunc("/", RequireAuth(db, IndexHandler(db, client)))
	http.HandleFunc("/callback", RequireAuth(db, OauthHandler(db, client, *consumerSecret)))
	http.HandleFunc("/login", RequireNotAuth(db, LoginHandler(db)))
	http.HandleFunc("/signup", RequireNotAuth(db, SignupHandler(db)))
	http.HandleFunc("/logout", RequireAuth(db, LogoutHandler(db)))
	http.HandleFunc("/measures", RequireAuth(db, RequireLink(db, MeasuresHandler(db, client))))
	http.HandleFunc("/phone", RequireAuth(db, PhoneHandler(db)))
	Log.Infof("Listening on port %d", *port)

	server := &http.Server{
		Addr:      fmt.Sprintf(":%d", *port),
		Handler:   sessionizer,
		TLSConfig: &tls.Config{GetCertificate: certmgr.GetCertificate},
	}

	challengeServer := &http.Server{
		Addr:    ":80",
		Handler: certmgr.HTTPHandler(http.RedirectHandler(fmt.Sprintf("https://%s", *callbackDomain), http.StatusMovedPermanently)),
	}

	go func() { Log.Fatal(challengeServer.ListenAndServe()) }()

	Log.Fatal(server.ListenAndServeTLS("", ""))
}
