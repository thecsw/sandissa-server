//  ______________________
// < I'm a sandissa kitty >
//  ----------------------
//   \
//    \
//       /\_)o<
//      |      \
//      | O . O|
//      \_____/
package main

import (
	"fmt"
	"time"

	"github.com/pterm/pterm"
)

func main() {

	printSandissaBanner()

	// Initialize the database
	spinnerDB, _ := pterm.DefaultSpinner.WithRemoveWhenDone(false).Start("Initializing the database")
	err := initDB()
	if err != nil {
		panic(err)
	}
	// Give the DB arbitrary 3 seconds to turn on
	time.Sleep(3 * time.Second)
	defer closeDB()
	spinnerDB.Stop()

	// Initialize MQTT
	spinnerMQTT, _ := pterm.DefaultSpinner.WithRemoveWhenDone(false).Start("Initializing MQTT")
	clientMQTT, err = getClient()
	if err != nil {
		panic(err)

	}
	subscribe(1, topicTemperature)
	// Give MQTT 2 seconds to heat up
	time.Sleep(2 * time.Second)
	defer clientMQTT.Disconnect(250)
	spinnerMQTT.Stop()

	// Create sample users
	if err := addUser("sandy", "lily"); err != nil {
		pterm.Warning.Println("Failed adding sandy:", err.Error())
	}
	if err := addUser("anissa", "secret"); err != nil {
		pterm.Warning.Println("Failed adding anissa:", err.Error())
	}

	// Initialize REST
	fmt.Println()
	pterm.Info.Println("Started REST service")
	startRouter()
}

func printSandissaBanner() {
	s, _ := pterm.DefaultBigText.WithLetters(
		pterm.NewLettersFromStringWithStyle("Sand", pterm.NewStyle(pterm.FgRed)),
		pterm.NewLettersFromStringWithStyle("issa", pterm.NewStyle(pterm.FgMagenta)),
	).Srender()
	pterm.DefaultCenter.Print(s)
	pterm.DefaultCenter.WithCenterEachLineSeparately().Println(
		"Server for Sandissa\nHandle incoming REST, verify auth, and call MQTT")
}
