package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/bjarneh/bloomfilter"
	"github.com/jteeuwen/go-pkg-xmlx"
	"log"
	"net/http"
	"strings"
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
var wg sync.WaitGroup // For determining when we're done fetching and eliminating all of the games a user owns
var err error

func init() {
	flag.StringVar(&config.Host, "host", "localhost", "host to listen on for the webserver")
	flag.StringVar(&config.Port, "port", "8000", "port to listen on for the webserver")
	flag.BoolVar(&config.EnableTLS, "tls", false, "enable serving with TLS (https)")
	flag.StringVar(&config.TLSCertPath, "tls-cert", "/cert.pem", "path to certificate file")
	flag.StringVar(&config.TLSKeyPath, "tls-key", "/key.pem", "path to private key for certificate")
	flag.StringVar(&config.APIKey, "api-key", "XXXXXXXXXXXXXXXXX", "steam api key") // If not specified the API will return a 403 (XXXXXXXXXXXXXXXXX is not a valid API key, obviously)
	flag.Parse()                                                                    // Parse the rest of the flags
}

func main() {
	http.Handle("/", reqHandler(handler))
	bind := fmt.Sprintf("%s:%s", config.Host, config.Port)
	if !config.EnableTLS {
		err = http.ListenAndServe(bind, nil)
		if err != nil {
			panic(err)
		}
	} else if config.EnableTLS {
		log.Println("Serving with TLS...")
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

	var players []string // For storing the split query parameter
	if reqHttp.URL.Query().Get("players") != "" {
		players = strings.Split(reqHttp.URL.Query().Get("players"), "|")
	} else {
		return &reqError{errors.New("Invalid query parameter"), "Invalid query parameter", 400}
	}

	var games gameList                 // Holds the mutex and map of games
	games.games = make(map[int]string) // Initialize the map
	for i := range players {
		wg.Add(1) // Add for each goroutine made
		go func(steamId string, glist gameList) {
			defer wg.Done() // Make sure we always say we're done when we are

			doc := xmlx.New()                                                                                                                         // Make a new document to store xml data in
			doc.SetUserAgent("steamownedbot/1.0 (+https://github.com/pietroglyph/steamowned)")                                                        // Be polite
			doc.LoadUri("https://api.steampowered.com/IPlayerService/GetOwnedGames/v0001/?key="+config.APIKey+"&steamid="+steamId+"&format=xml", nil) // Get xml data from the Steam API
			gamenodes := doc.SelectNodes("", "appid")                                                                                                 // gamenodes is *all* nodes with appid as their key

			if len(gamenodes) == 0 { // Probably an invalid SteamID64 parameter
				log.Println("Couldn't extract appids using SteamID", steamId)
				return
			}

			glist.mux.Lock() // We're going to start using the map of games
			defer glist.mux.Unlock()
			if len(glist.games) == 0 {
				for i := range gamenodes {
					glist.games[gamenodes[i].I("", "appid")] = gamenodes[i].S("", "appid")
				}
			} else {
				filter := bloomfilter.New() // This is because gamenodes is not indexed by game-name, the bloom filter might be slower, but I wanted to use it
				for i := range gamenodes {
					filter.Add(gamenodes[i].S("", "appid"))
				}
				for k := range glist.games {
					if !filter.Marked(glist.games[k]) { // Game games[k] is not owned (does not exist) in gamenodes, so we delete it
						delete(glist.games, k)
					}
				}
			}
		}(players[i], games)
	}
	wg.Wait()
	// This isn't the prettiest or most configurable/extendable thing, and I should use another web page and wrap this in json and make ajax calls or something
	// But I really don't feel like doing JavaScript today.
	fmt.Fprint(resWriter, "<!DOCTYPE html><html lang='en'><head><title>steamowned</title><link rel='stylesheet' href='https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css' integrity='sha384-BVYiiSIFeK1dGmJRAkycuHAHRg32OmUcww7on3RYdg4Va+PmSTsz/K68vbdEjh4u' crossorigin='anonymous'></head><body><h1>steamowned</h1><br>")
	for i := range games.games {
		fmt.Fprint(resWriter, "<img src='http://cdn.akamai.steamstatic.com/steam/apps/", games.games[i], "/header.jpg'><img/><br>")
	}
	fmt.Fprint(resWriter, "</body></html>")
	return nil // Nothing went wrong
}

func (fn reqHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e := fn(w, r); e != nil { // e is *appError, not os.Error
		http.Error(w, e.Message, e.Code) // Serve an error message
		log.Println("HTTP", e.Code, e.Message)
	}
}
