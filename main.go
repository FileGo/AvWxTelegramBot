package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/yanzay/tbot/v2"
)

const dbPath = "airports.json"

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
	Airports []Airport
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
func GetNOAAinterval() (int, error) {
	if os.Getenv("NOAA_INTERVAL") == "" {
		return 12, nil
	}

	NOAAinterval, err := strconv.Atoi(os.Getenv("NOAA_INTERVAL"))
	if err != nil {
		return 0, err
	}

	// Return error for non-positive interval
	if NOAAinterval <= 0 {
		return 0, errors.New("interval should be a positive number")
	}

	return int(NOAAinterval), nil
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
	NOAAinterval, err := GetNOAAinterval()
	if err != nil {
		log.Fatalf("unable to read NOAA interval: %v", err)
	}

	// Fail if TELEGRAM_TOKEN is not set
	if os.Getenv("TELEGRAM_TOKEN") == "" {
		log.Fatal("TELEGRAM_TOKEN not set. Unable to start the bot.")
	}

	// Check if using webhook or not
	var bot *tbot.Server
	if os.Getenv("WEBHOOK_URL") != "" && os.Getenv("WEBHOOK_PORT") != "" {
		bot = tbot.New(os.Getenv("TELEGRAM_TOKEN"),
			tbot.WithWebhook(os.Getenv("WEBHOOK_URL"), fmt.Sprintf(":%s", os.Getenv("WEBHOOK_PORT"))))
		fmt.Printf("Starting the bot with webhook: %s:%s\n", os.Getenv("WEBHOOK_URL"), os.Getenv("WEBHOOK_PORT"))
	} else {
		bot = tbot.New(os.Getenv("TELEGRAM_TOKEN"))
		fmt.Println("Starting the bot with the updates method")
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

	bot.HandleMessage(".*", func(m *tbot.Message) {
		// Send "typing..." to client
		c.SendChatAction(m.Chat.ID, tbot.ActionTyping)

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
						tafCh := make(chan string, 1)
						metarCh := make(chan string, 1)

						wg.Add(2)
						go GetTafNOAA(arpt.ICAO, tafCh, NOAAinterval, &wg)
						go GetMetarNOAA(arpt.ICAO, metarCh, NOAAinterval, &wg)

						wg.Wait()
						taf := <-tafCh
						metar := <-metarCh

						message := fmt.Sprintf("<b>%s/%s\nMETAR</b>\n<code>%s</code>\n<b>TAF</b>\n<code>%s</code>", strings.ToUpper(arpt.ICAO), strings.ToUpper(arpt.IATA), metar, taf)
						messages = append(messages, message)
					}
				}(airport, &wgMain)
				wgMain.Wait()

			}

			// Send messages once we have all data
			for _, message := range messages {
				// we need HTML parse mode to enable <code>, which disables displaying numbers as URLs on mobile devices
				c.SendMessage(m.Chat.ID, message, tbot.OptParseModeHTML)
			}
		} else {
			c.SendMessage(m.Chat.ID,
				"Incorrect format.\nExample usage: KLAX JFK LHR or KLAX,JFK,LHR")
		}
	})

	log.Fatal(bot.Start())
}
