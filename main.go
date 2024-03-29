package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	. "github.com/asymmetricia/vator/log"
	"github.com/asymmetricia/vator/models"
	"github.com/asymmetricia/withings"
	"go.etcd.io/bbolt"
	"golang.org/x/crypto/acme/autocert"
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

func RequireLink(db *bbolt.DB, handler func(w http.ResponseWriter, r *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		user, err := models.LoadUserRequest(db, req)
		if err != nil {
			Bail(rw, req, fmt.Errorf("should be logged in, but: %s", err), http.StatusInternalServerError)
			return
		}
		if user.RefreshSecret == "" {
			http.Redirect(rw, req, "/", http.StatusFound)
			return
		}
		handler(rw, req)
	}
}

func callbackUrl(proto string, domain string, port int, path string) string {
	if proto == "https" && port == 443 || proto == "http" && port == 80 {
		return proto + "://" + domain + "/" + path
	}
	return fmt.Sprintf("%s://%s:%d/%s", proto, domain, port, path)
}

func main() {
	consumerKey := flag.String("consumer-key", "", "oauth consumer key")
	consumerSecret := flag.String("consumer-secret", "", "oauth consumer secret")
	port := flag.Int("port", 0, "port to listen on (if 0, actual port will depend on whether TLS is enabled or not)")
	callbackDomain := flag.String("callback-domain", "localhost", "fqdn for oauth callbacks")
	callbackPort := flag.Int("callback-port", 0, "callback port; if zero, same as -port")
	callbackProto := flag.String("callback-proto", "http", "protocol to use in requesting callbacks")
	dbFile := flag.String("db-file", "vator.db", "path to the bolt database file used to persist state")

	twilioSid := flag.String("twilio-sid", "", "twilio account SID")
	twilioToken := flag.String("twilio-token", "", "twilio auth token")

	tlsEnabled := flag.Bool("tls", false, "if true, will configure TLS using a certificate from letsencrypt")

	flag.Parse()

	if *port == 0 {
		if *tlsEnabled {
			*port = 443
		} else {
			*port = 80
		}
	}

	for n, f := range map[string]*string{
		"-consumer-key":    consumerKey,
		"-consumer-secret": consumerSecret,
	} {
		if *f == "" {
			log.Fatalf("%s must be provided", n)
		}
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

	db, err := bbolt.Open(*dbFile, 0600, nil)
	if err != nil {
		Log.Fatalf("opening bolt db file %q: %s", *dbFile, err)
	}
	defer db.Close()
	models.TidyUsers(db)

	cbUrl := callbackUrl(*callbackProto, *callbackDomain, *callbackPort, "callback")
	Log.Infof("using callback URL %q", cbUrl)
	withingsClient := new(withings.Client)
	*withingsClient = withings.NewClient(*consumerKey, *consumerSecret, cbUrl)

	minutely := time.NewTicker(time.Minute)
	go func() {
		for {
			go func() {
				time.Sleep(30 * time.Second)
				BackfillMeasures(db, withingsClient)
			}()
			ScanMeasures(db, withingsClient, twilio)
			<-minutely.C
		}
	}()

	withings := WithingsClient{
		Db:       db,
		Withings: withingsClient,
	}

	sessionizer := http.NewServeMux()
	sessionizer.HandleFunc("/", models.WithSession(db, http.DefaultServeMux.ServeHTTP))
	sessionizer.HandleFunc("/login", models.WithNewSession(db, RequireNotAuth(db, LoginHandler(db))))

	http.HandleFunc("/", RequireAuth(db, IndexHandler(db, withingsClient)))

	http.HandleFunc("/withings/begin", RequireAuth(db, withings.Begin))
	http.HandleFunc("/callback", RequireAuth(db, withings.Complete))

	http.HandleFunc("/signup", RequireNotAuth(db, SignupHandler(db)))
	http.HandleFunc("/logout", RequireAuth(db, LogoutHandler(db)))
	http.HandleFunc("/measures", RequireAuth(db, MeasuresHandler(db)))

	http.HandleFunc("/phone", RequireAuth(db, PhoneHandler(db)))
	http.HandleFunc("/kgs", RequireAuth(db, KgsHandler(db)))
	http.HandleFunc("/rename", RequireAuth(db, RenameHandler(db)))
	http.HandleFunc("/share", RequireAuth(db, ShareHandler(db)))
	http.HandleFunc("/summary", RequireAuth(db, RequireLink(db, SummaryHandler(db, twilio))))

	http.Handle("/static/", http.FileServer(http.FS(static)))
	http.HandleFunc("/graph", Graph(db))
	http.HandleFunc("/data", Data(db))

	Log.Infof("Listening on port %d", *port)

	if *tlsEnabled {
		certmgr := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(*callbackDomain),
		}

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
	} else {
		server := &http.Server{
			Addr:    fmt.Sprintf(":%d", *port),
			Handler: sessionizer,
		}

		Log.Fatal(server.ListenAndServe())
	}
}
