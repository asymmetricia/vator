package main

import (
	"flag"
	"fmt"
	bolt "github.com/coreos/bbolt"
	"github.com/jrmycanady/nokiahealth"
	"net/http"
	"strconv"
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
				log.Debugf("Missing parameter: %s", str)
				http.Error(rw, fmt.Sprintf("Your request was missing required parameters."), http.StatusBadRequest)
				return
			}
		}
		handler(rw, req)
	}
}

func IndexHandler(nokia nokiahealth.Client) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		ar, err := nokia.CreateAccessRequest()
		if err != nil {
			http.Error(rw, fmt.Sprintf("I had trouble starting the sign-up process: %s", err), http.StatusInternalServerError)
			return
		}
		rw.Header().Add("content-type", "text/html; charset=utf-8")
		fmt.Fprintf(rw, "Welcome to vator! Click <a href='%s'>here</a> to link up to your Nokia Health account.", ar.AuthorizationURL)
	}
}

func OauthHandler(nokia nokiahealth.Client, consumerSecret string) func(http.ResponseWriter, *http.Request) {
	return RequireForm([]string{"oauth_token", "oauth_verifier", "userid"}, func(rw http.ResponseWriter, req *http.Request) {
		userid, err := strconv.Atoi(req.Form.Get("userid"))
		verifier := req.Form.Get("oauth_verifier")
		if err != nil {
			http.Error(rw, "I had trouble parsing that userid.", http.StatusBadRequest)
			log.Errorf("failed to parse userid %q: %s", req.Form.Get("userid"), err)
			return
		}
		ar := nokia.RebuildAccessRequest(req.Form.Get("oauth_token"), consumerSecret)
		fmt.Println(req.Form)
		//		user
		_, err = ar.GenerateUser(verifier, userid)
		if err != nil {
			http.Error(rw, "Failed to obtain user for oauth verifier.", http.StatusInternalServerError)
			log.Errorf("obtaining user for id %d and verifier %q: %s", userid, verifier, err)
			return
		}
	})
}

func main() {
	consumerKey := flag.String("consumer-key", "", "oauth consumer key")
	consumerSecret := flag.String("consumer-secret", "", "oauth consumer secret")
	port := flag.Int("port", 8080, "port to listen on")
	callbackDomain := flag.String("callback-domain", "localhost", "fqdn for oauth callbacks")
	callbackPort := flag.Int("callback-port", 0, "callback port; if zero, same as --port")
	callbackProto := flag.String("callback-proto", "http", "protocol to use in requesting callbacks")
	dbFile := flag.String("db-file", "vator.db", "path to the bolt database file used to persist state")

	flag.Parse()

	if *consumerKey == "" {
		log.Fatalf("consumer key must be provided")
	}
	if *consumerSecret == "" {
		log.Fatalf("consumer secret must be provided")
	}

	if *callbackPort == 0 {
		*callbackPort = *port
	}

	db, err := bolt.Open(*dbFile, 0600, nil)
	if err != nil {
		log.Fatalf("opening bolt db file %q: %s", *dbFile, err)
	}
	defer db.Close()

	client := nokiahealth.NewClient(*consumerKey, *consumerSecret, fmt.Sprintf("%s://%s:%d/callback", *callbackProto, *callbackDomain, *callbackPort))

	sessionizer := http.NewServeMux()
	sessionizer.HandleFunc("/", WithSession(db, http.DefaultServeMux.ServeHTTP))

	http.HandleFunc("/", RequireAuth(db, IndexHandler(client)))
	http.HandleFunc("/callback", RequireAuth(db, OauthHandler(client, *consumerSecret)))
	http.HandleFunc("/login", RequireNotAuth(db, LoginHandler(db)))
	log.Infof("Listening on port %d", *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), sessionizer))
}
