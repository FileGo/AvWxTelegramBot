package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/yanzay/tbot/v2"
)

const dbPath = "airports.db3"

// GetICAOs returns array of ICAO codes from a message string
func GetICAOs(input string) (output []string) {
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

func main() {
	// Check that DB file exists and is readable
	_, err := os.Stat(dbPath)
	if os.IsNotExist(err) {
		log.Fatalf("%s does not exist.\n", dbPath)
	}

	if os.IsPermission(err) {
		log.Fatalf("Unable to read from %s.\n", dbPath)
	}

	// Open SQLite connection
	dbDSN := fmt.Sprintf("file:%s?mode=ro", dbPath)
	db, err := sql.Open("sqlite3", dbDSN)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Set default NOAA interval if not set
	var NOAAinterval int
	if os.Getenv("NOAA_INTERVAL") == "" {
		NOAAinterval = 12
	} else {
		NOAAinterval, err = strconv.Atoi(os.Getenv("NOAA_INTERVAL"))
		if err != nil {
			log.Fatal(err)
		}
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

	// Start message
	bot.HandleMessage("/start", func(m *tbot.Message) {
		c.SendMessage(m.Chat.ID, "Welcome to METAR/TAF bot!")
	})

	// Handle /help
	bot.HandleMessage("/help", func(m *tbot.Message) {
		c.SendMessage(m.Chat.ID,
			`This bot quickly retrieves METAR and TAF for multiple airports.
To use it, simply type one or more IATA or ICAO airport codes seperated by either a space or a comma, e.g.
KLAX JFK LHR or KLAX,JFK,LHR`)
	})

	bot.HandleMessage(".*", func(m *tbot.Message) {
		// Send "typing..." to client
		c.SendChatAction(m.Chat.ID, tbot.ActionTyping)

		var messages []string

		// Get airports from received message
		airports := GetICAOs(m.Text)

		if len(airports) > 0 {
			var wgMain sync.WaitGroup
			for _, airport := range airports {

				wgMain.Add(1)
				go func(arpt string, wgMain *sync.WaitGroup) {
					defer wgMain.Done()
					var icao string
					var iata string
					success := true

					if len(arpt) == 4 {
						// ICAO
						icao = arpt

						// Get IATA code from MYSQL
						row := db.QueryRow("SELECT iata_code FROM airports WHERE ident=?", arpt)
						switch err := row.Scan(&iata); err {
						case sql.ErrNoRows:
							iata = ""
						default:
						}
					} else if len(arpt) == 3 {
						// IATA code
						iata = arpt

						// Get ICAO code from MYSQL
						row := db.QueryRow("SELECT ident FROM airports WHERE iata_code=?", arpt)
						switch err := row.Scan(&icao); err {
						case sql.ErrNoRows:
							iata = ""
						default:

						}
					} else {
						message := fmt.Sprintf("Airport %s not found.", arpt)
						messages = append(messages, message)
						success = false
					}

					if success {
						// Get NOAA data
						var wg sync.WaitGroup
						tafCh := make(chan string, 1)
						metarCh := make(chan string, 1)

						wg.Add(2)
						go GetTafNOAA(icao, tafCh, NOAAinterval, &wg)
						go GetMetarNOAA(icao, metarCh, NOAAinterval, &wg)

						wg.Wait()
						taf := <-tafCh
						metar := <-metarCh

						message := fmt.Sprintf("<b>%s/%s\nMETAR</b>\n<code>%s</code>\n<b>TAF</b>\n<code>%s</code>", strings.ToUpper(icao), strings.ToUpper(iata), metar, taf)
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
