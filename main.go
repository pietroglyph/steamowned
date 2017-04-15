package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"strings"
	"encoding/json"
	"sync"
)

type configuration struct {
	Host        string
	Port        string
	EnableTLS   bool
	TLSCertPath string
	TLSKeyPath  string
	APIKey      string
}

type reqError struct {
	Error   error
	Message string
	Code    int
}

type gameList struct {
	games map[int]string
	mux   sync.Mutex
}

type reqHandler func(http.ResponseWriter, *http.Request) *reqError

var config configuration
var err error

func init() {
	flag.StringVar(&config.Host, "host", "localhost", "host to listen on for the webserver")
	flag.StringVar(&config.Port, "port", "8000", "port to listen on for the webserver")
	flag.BoolVar(&config.EnableTLS, "tls", false, "enable serving with TLS (https)")
	flag.StringVar(&config.TLSCertPath, "tls-cert", folderPath+"/cert.pem", "path to certificate file")
	flag.StringVar(&config.TLSKeyPath, "tls-key", folderPath+"/key.pem", "path to private key for certificate")
	flag.StringVar(&config.APIKey, "api-key", "XXXXXXXXXXXXXXXXX", "steam api key")
	flag.Parse() // Parse the rest of the flags
}

func main() {
	http.HandleFunc("/", handler)
	if !config.EnableTLS {
		err = http.ListenAndServe(bind, nil)
		if err != nil {
			panic(err)
		}
	} else if config.EnableTLS {
		fmt.Println("Serving with TLS...")
		err = http.ListenAndServeTLS(bind, config.TLSCertPath, config.TLSKeyPath, nil)
		if err != nil {
			panic(err)
		}
	}
}

func handler(resWriter http.ResponseWriter, reqHttp *http.Request) *reqError {
	defer func() { // Recover from a panic if one occurred
		if err := recover(); err != nil {
			fmt.Println(err)
			fmt.Fprint(resWriter, err)
		}
	}()

	if reqHttp.URL.Query().Get("players") != "" {
		players := strings.Split(reqHttp.URL.Query().Get("players"))
	} else {
		return &reqError{errors.New("Invalid query parameter"), "Invalid query paramter", 400}
	}

	for i := range players {
		go func(steamId string) {
			resp, err := http.Get("https://api.steampowered.com/IPlayerService/GetOwnedGames/v0001/?key=" + config.APIKey + "&steamid=" + steamId + "&format=json")
			if err != nil {
				log.Println(err.Error())
				return
			}
			json.
		}(players[i])
	}
}

func (fn reqHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e := fn(w, r); e != nil { // e is *appError, not os.Error
		http.Error(w, e.Message, e.Code) // Serve an error message
		log.Println("HTTP", e.Code, e.Message)
	}
}
