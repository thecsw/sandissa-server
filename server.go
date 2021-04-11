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
	"github.com/pterm/pterm"
)

func main() {

	// Print the big banner
	s, _ := pterm.DefaultBigText.WithLetters(
		pterm.NewLettersFromStringWithStyle("Sand", pterm.NewStyle(pterm.FgRed)),
		pterm.NewLettersFromStringWithStyle("issa", pterm.NewStyle(pterm.FgMagenta)),
	).Srender()
	pterm.DefaultCenter.Print(s)
	pterm.DefaultCenter.WithCenterEachLineSeparately().Println(
		"Server for Sandissa\nHandle incoming REST+MQTT for IoT Security")

	var err error

	// Initialize the database
	spinnerDB, _ := pterm.DefaultSpinner.WithRemoveWhenDone(true).Start("Initializing the database\n")
	err = initDB()
	if err != nil {
		panic(err)
	}
	defer closeDB()
	spinnerDB.Stop()

	// Initialize MQTT
	spinnerMQTT, _ := pterm.DefaultSpinner.WithRemoveWhenDone(true).Start("Initializing MQTT\n")
	clientMQTT, err = getClient()
	if err != nil {
		panic(err)

	}
	subscribe(1, topicTemperature)
	defer clientMQTT.Disconnect(250)
	spinnerMQTT.Stop()

	// Create sample users
	addUser("sandy", "lily")

	// Initialize REST
	pterm.Info.Println("Started REST service")
	startRouter()
}
