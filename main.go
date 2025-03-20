package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/djotaku/go-mastodon"

	"github.com/adrg/xdg"
)

type lastfm struct {
	Key      string `json:"key"`
	Secret   string
	Username string
}

type mastodonConfig struct {
	Access_token string
	Api_base_url string
	ClientID     string
	ClientSecret string
}

type secrets struct {
	Lastfm   lastfm
	Mastodon mastodonConfig
}

func getSecrets() secrets {
	configFilePath, err := xdg.ConfigFile("lastfmmastodon/secrets.json")
	if err != nil {
		fmt.Println("error")
	}
	settingsJson, err := os.Open(configFilePath)
	// if os.Open returns an error then handle it
	if err != nil {
		fmt.Println("Unable to open the config file. Did you place it in the right spot?")

	}
	defer func(settingsJson *os.File) {
		err := settingsJson.Close()
		if err != nil {
			errorString := fmt.Sprintf("Couldn't close the settings file. Error: %s", err)
			fmt.Println(errorString)

		}
	}(settingsJson)
	byteValue, _ := io.ReadAll(settingsJson)
	var settings *secrets
	err = json.Unmarshal(byteValue, &settings)
	if err != nil {
		fmt.Println("Check that you do not have errors in your JSON file.")
		errorString := fmt.Sprintf("Could not unmashal json: %s\n", err)
		fmt.Println(errorString)
		panic("AAAAAAH!")
	}
	return *settings
}

type attribute struct {
	Rank string
}

type artist struct {
	Playcount string
	Attribute attribute `json:"@attr"`
	Name      string
}

type topArtists struct {
	Artist []artist
}

type topArtistsResult struct {
	Topartists topArtists
}

func submitLastfmCommand(period string, apiKey string, user string) (string, error) {
	apiURLBase := "https://ws.audioscrobbler.com/2.0/?"
	queryParameters := url.Values{}
	queryParameters.Set("method", "user.gettopartists")
	queryParameters.Set("user", user)
	if period == "weekly" {
		queryParameters.Set("period", "7day")
	} else {
		queryParameters.Set("period", "12month")
	}
	queryParameters.Set("api_key", apiKey)
	queryParameters.Set("format", "json")
	fullURL := apiURLBase + queryParameters.Encode()
	lastfmResponse, statusCode, err := WebGet(fullURL)
	if err != nil {
		fmt.Println(statusCode)
		return lastfmResponse, err
	}
	return lastfmResponse, err
}

// webGet handles contacting a URL
func WebGet(url string) (string, int, error) {
	response, err := http.Get(url)
	if err != nil {
		return "Error accessing URL", 0, err
	}
	result, err := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode > 299 {
		statusCodeString := fmt.Sprintf("Response failed with status code: %d and \nbody: %s\n", response.StatusCode, result)
		fmt.Println(statusCodeString)
		panic("Invalid status, data will be garbage")
	}
	if err != nil {
		return "Error reading response", 0, err
	}
	return string(result), response.StatusCode, err

}

func assembleTootString(artists topArtistsResult, period string) string {
	var tootString string
	if period == "weekly" {
		tootString = "My top #lastfm artists for the past week: "
	} else {
		tootString = "My top #lastfm artists for the past 12 months: "
	}
	for _, artist := range artists.Topartists.Artist {
		potentialString := fmt.Sprintf("%s.%s (%s), ", artist.Attribute.Rank, artist.Name, artist.Playcount)
		if len(tootString)+len(potentialString) < 500 {
			tootString += potentialString
		} else {
			return tootString
		}
	}
	return tootString
}

func registerClient(baseURL string) mastodonConfig {
	appConfig := &mastodon.AppConfig{
		Server:       baseURL,
		ClientName:   "lastfmmastodon",
		Scopes:       "read write follow",
		Website:      "https://github.com/mattn/go-mastodon",
		RedirectURIs: "urn:ietf:wg:oauth:2.0:oob",
	}
	app, err := mastodon.RegisterApp(context.Background(), appConfig)
	if err != nil {
		log.Fatal(err)
	}
	// Have the user manually get the token and send it back to us
	u, err := url.Parse(app.AuthURI)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Open your browser to \n%s\n and copy/paste the given token\n", u)
	var token string
	fmt.Print("Paste the token here:")
	fmt.Scanln(&token)
	// end of get access token
	config := &mastodon.Config{
		Server:       baseURL,
		ClientID:     app.ClientID,
		ClientSecret: app.ClientSecret,
		AccessToken:  token,
	}
	c := mastodon.NewClient(config)
	err = c.AuthenticateToken(context.Background(), config.AccessToken, "urn:ietf:wg:oauth:2.0:oob")
	if err != nil {
		fmt.Println("authentication token failed")
		log.Fatal(err)
	}

	var newMastodonConfig mastodonConfig
	newMastodonConfig.Access_token = config.AccessToken
	newMastodonConfig.Api_base_url = baseURL
	newMastodonConfig.ClientID = config.ClientID
	newMastodonConfig.ClientSecret = config.ClientSecret

	return newMastodonConfig
}

func main() {
	configFilePath, err := xdg.ConfigFile("lastfmmastodon/secrets.json")
	if err != nil {
		fmt.Println("error")
	}
	ourSecrets := getSecrets()
	// parse CLI flags
	register := flag.Bool("r", false, "register the client")
	period := flag.String("p", "weekly", "period to grab. Use: weekly, quarterly, or annual")
	flag.Parse()

	weeklyArtistsJSON, err := submitLastfmCommand(*period, ourSecrets.Lastfm.Key, ourSecrets.Lastfm.Username)
	if err != nil {
		fmt.Println(err)
	}
	var weeklyArtsts topArtistsResult
	err = json.Unmarshal([]byte(weeklyArtistsJSON), &weeklyArtsts)
	if err != nil {
		fmt.Printf("Unable to marshall. %s", err)
	}
	tootString := assembleTootString(weeklyArtsts, *period)
	fmt.Printf("Your toot will be: %s", tootString)

	if *register {

		newMastodonConfig := registerClient(ourSecrets.Mastodon.Api_base_url)
		var newConfig secrets
		newConfig.Lastfm = ourSecrets.Lastfm
		newConfig.Mastodon = newMastodonConfig
		jsonBytes, err := json.Marshal(newConfig)
		if err != nil {
			log.Fatal(err)
		}
		error := os.WriteFile(configFilePath, jsonBytes, 0666)
		if error != nil {
			log.Fatal(err)
		}

	} else {

		config := &mastodon.Config{
			ClientID:     ourSecrets.Mastodon.ClientID,
			ClientSecret: ourSecrets.Mastodon.ClientSecret,
			Server:       ourSecrets.Mastodon.Api_base_url,
			AccessToken:  ourSecrets.Mastodon.Access_token,
		}
		c := mastodon.NewClient(config)

		visibility := "public"

		// Post a toot
		toot := mastodon.Toot{
			Status:     tootString,
			Visibility: visibility,
		}
		post, err := c.PostStatus(context.Background(), &toot)

		if err != nil {
			log.Fatalf("%#v\n", err)
		}

		fmt.Printf("My new post is %v\n", post)
	}
}
