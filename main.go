package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yanzay/tbot/v2"
)

const dbPath = "airports.json"

const urlPrefix = "https://aviationweather.gov/cgi-bin/data/dataserver.php"

// GetAirportCodes returns array of ICAO codes from a message string
func GetAirportCodes(input string) (output []string) {
	output = []string{}
	var airports []string

	// Trim input first
	input = strings.TrimSpace(input)
	textLen := len(input)
	if textLen != 3 && textLen != 4 {
		// Multiple airports or wrong input
		// Try comma
		airports = strings.Split(input, ",")
		if len(airports) <= 1 {
			// Try spaces
			airports = strings.Fields(input)
		}

	} else {
		airports = append(airports, input)
	}

	// Trim results and make it uppercase
	for _, airport := range airports {
		str := strings.ToUpper(strings.TrimSpace(airport))

		// Only add non-empty strings to output
		if len(str) > 0 {
			output = append(output, str)
		}
	}

	return
}

// Airport represents an airport entry
type Airport struct {
	ICAO string
	IATA string
	Name string
}

// Env stores global variables
type Env struct {
	Airports     []Airport
	httpClient   *http.Client
	NOAAinterval int
	logRequests  bool
}

// FindAirport returns an airport
func (env *Env) FindAirport(code string) (Airport, error) {
	if len(code) == 4 { // ICAO
		for i := range env.Airports {
			if env.Airports[i].ICAO == code {
				return env.Airports[i], nil
			}
		}
	} else if len(code) == 3 { // IATA
		for i := range env.Airports {
			if env.Airports[i].IATA == code {
				return env.Airports[i], nil
			}
		}
	} else {
		return Airport{}, errors.New("airport code should be in IATA (3 letters) or ICAO (4 letters) form")
	}

	return Airport{}, errors.New("no airport found")
}

// LoadAirports Loads the airports into memory
func (env *Env) LoadAirports(r io.Reader) error {
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	err = json.Unmarshal(buf, &env.Airports)
	if err != nil {
		return err
	}

	return nil
}

// GetNOAAinterval retrieves interval for checking the METER/TAF data
func (env *Env) GetNOAAinterval() error {
	if os.Getenv("NOAA_INTERVAL") == "" {
		env.NOAAinterval = 12
		return nil
	}

	var err error
	env.NOAAinterval, err = strconv.Atoi(os.Getenv("NOAA_INTERVAL"))
	if err != nil {
		return err
	}

	// Return error for non-positive interval
	if env.NOAAinterval <= 0 {
		return errors.New("interval should be a positive number")
	}

	return nil
}

func main() {
	// Check that DB file exists and is readable
	f, err := os.Open(dbPath)
	if err != nil {
		log.Fatalf("unable to open %s: %v\n", dbPath, err)
	}
	defer f.Close()

	env := Env{}
	err = env.LoadAirports(f)
	if err != nil {
		log.Fatalf("unable to load airports: %v", err)
	}

	// Set default NOAA interval if not set
	err = env.GetNOAAinterval()
	if err != nil {
		log.Fatalf("unable to read NOAA interval: %v", err)
	}

	// Fail if TELEGRAM_TOKEN is not set
	if os.Getenv("TELEGRAM_TOKEN") == "" {
		log.Fatal("TELEGRAM_TOKEN not set. Unable to start the bot.")
	}

	// Logging
	if os.Getenv("LOG_REQUESTS") != "" {
		env.logRequests = true
	}

	// Set httpClient
	env.httpClient = &http.Client{
		Timeout: 10 * time.Second,
	}

	// Check if using webhook or not
	var bot *tbot.Server
	if os.Getenv("WEBHOOK_URL") != "" && os.Getenv("WEBHOOK_PORT") != "" {
		bot = tbot.New(os.Getenv("TELEGRAM_TOKEN"),
			tbot.WithWebhook(os.Getenv("WEBHOOK_URL"), fmt.Sprintf(":%s", os.Getenv("WEBHOOK_PORT"))))
		fmt.Printf("Starting the bot with webhook: %s:%s\n", os.Getenv("WEBHOOK_URL"), os.Getenv("WEBHOOK_PORT"))
	} else {
		bot = tbot.New(os.Getenv("TELEGRAM_TOKEN"))
		log.Println("Starting the bot with the updates method")
	}
	c := bot.Client()

	// Handle / messages
	bot.HandleMessage("/start", func(m *tbot.Message) {
		switch strings.TrimLeft(m.Text, "/") {
		case "start":
			c.SendMessage(m.Chat.ID, "Welcome to METAR/TAF bot!")
		case "help":
			c.SendMessage(m.Chat.ID,
				`This bot quickly retrieves METAR and TAF for multiple airports.
To use it, simply type one or more IATA or ICAO airport codes seperated by either a space or a comma, e.g.
KLAX JFK LHR or KLAX,JFK,LHR`)
		case "?":
			c.SendMessage(m.Chat.ID,
				`Available commands:
/start
/help`)
		default:
			c.SendMessage(m.Chat.ID, fmt.Sprintf("Unknown command: %s", m.Text))
		}
	})

	getSWC := func(m *tbot.Message) {
		resp, err := http.Get("https://www.aviationweather.gov/data/iffdp/2104.gif")
		if err != nil {
			log.Printf("error getting sigwx chart: %v", err)
			c.SendMessage(m.Chat.ID, "Cannot retrieve SigWX chart.")
			return
		}
		defer resp.Body.Close()

		// Create temp file
		f, err := os.CreateTemp(os.TempDir(), "avwxbot")
		if err != nil {
			log.Printf("error creating temp file: %v", err)
			c.SendMessage(m.Chat.ID, "Cannot retrieve SigWX chart.")
			return
		}
		defer os.Remove(f.Name())

		// Write to temp file
		_, err = io.Copy(f, resp.Body)
		if err != nil {
			log.Printf("error writing to temp file: %v", err)
			c.SendMessage(m.Chat.ID, "Cannot retrieve SigWX chart.")
			return
		}

		_, err = c.SendPhotoFile(m.Chat.ID, f.Name())
		if err != nil {
			log.Printf("error sending message: %v", err)
		}
	}

	bot.HandleMessage("/swc", getSWC)
	bot.HandleMessage("/sigwx", getSWC)

	bot.HandleMessage(".*", func(m *tbot.Message) {
		// Send "typing..." to client
		c.SendChatAction(m.Chat.ID, tbot.ActionTyping)

		if env.logRequests {
			log.Printf("Received request from %s %s (%s): %s", m.From.FirstName, m.From.LastName, m.Chat.ID, m.Text)
		}

		var messages []string

		// Get airports from received message
		airports := GetAirportCodes(m.Text)

		if len(airports) > 0 {
			var wgMain sync.WaitGroup
			for _, airport := range airports {

				wgMain.Add(1)
				go func(code string, wgMain *sync.WaitGroup) {
					defer wgMain.Done()
					success := true

					arpt, err := env.FindAirport(code)
					if err != nil {
						message := fmt.Sprintf("Airport %s not found.", code)
						messages = append(messages, message)
						success = false
					}

					if success {
						// Get NOAA data
						var wg sync.WaitGroup
						tafCh := make(chan outputData, 1)
						metarCh := make(chan outputData, 1)

						wg.Add(2)
						go env.getData("tafs", arpt.ICAO, tafCh, &wg)
						go env.getData("metars", arpt.ICAO, metarCh, &wg)

						wg.Wait()
						taf := <-tafCh
						metar := <-metarCh

						metarString, err := ParseMetarNOAA(metar.data)
						if err != nil {
							log.Printf("error decoding metar: %v", err)
						}

						tafString, err := ParseTafNOAA(taf.data)
						if err != nil {
							log.Printf("error decoding taf: %v", err)
						}

						message := fmt.Sprintf("<b>%s/%s\nMETAR</b>\n<code>%s</code>\n<b>TAF</b>\n<code>%s</code>", strings.ToUpper(arpt.ICAO), strings.ToUpper(arpt.IATA), metarString, tafString)
						messages = append(messages, message)
					}
				}(airport, &wgMain)
				wgMain.Wait()

			}

			// Send messages once we have all data
			for _, message := range messages {
				// we need HTML parse mode to enable <code>, which disables displaying numbers as URLs on mobile devices
				_, err = c.SendMessage(m.Chat.ID, message, tbot.OptParseModeHTML)
				if err != nil {
					log.Printf("error sending message: %v\n", err)
				}

			}
		} else {
			c.SendMessage(m.Chat.ID,
				"Incorrect format.\nExample usage: KLAX JFK LHR or KLAX,JFK,LHR")
		}
	})

	log.Fatal(bot.Start())
}
