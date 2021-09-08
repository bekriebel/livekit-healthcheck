package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	lksdk "github.com/livekit/server-sdk-go"
	"github.com/thoas/go-funk"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)



func main() {
	// Get CLI parameters
	app := &cli.App{
		Name:        "livekit-healthcheck",
		Usage:       "Check the health of the livekit server by attempting to connect to a room",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "keys",
				Usage:   "api keys (key: secret\\n)",
				EnvVars: []string{"LIVEKIT_KEYS"},
			},
			&cli.StringFlag{
				Name:    "host",
				Usage:   "host (incl. port) of the livekit server to connect to (example: wss://livekit.example.com:7880)",
				EnvVars: []string{"LIVEKIT_HOST"},
			},
		},
		Action: healthcheck,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func healthcheck(c *cli.Context) error {
	// Check that host is set
	if !c.IsSet("host") {
		cli.ShowAppHelp(c)
		fmt.Printf("\n-----\n")
		return errors.New("error: host value not set")
	}
	host := c.String("host")

	// Check that keys are set
	if !c.IsSet("keys") {
		cli.ShowAppHelp(c)
		fmt.Printf("\n-----\n")
		return errors.New("error: keys not set")
	}
	apiKey, apiSecret, err := unmarshalKeys(c.String("keys"))
	if err != nil {
		return errors.New("Could not parse keys, it needs to be \"key: secret\", one per line")
	}

	// Create a random room name
  roomName := funk.RandomString(16)

	// Set identity
  identity := "livekit-healthcheck"

	// Create connection channel to watch for timeout
	connectChannel := make (chan lksdk.Room, 1)
	
	go func() {
		// Attempt to connect to the room
		room, err := lksdk.ConnectToRoom(host, lksdk.ConnectInfo{
			APIKey:              apiKey,
			APISecret:           apiSecret,
			RoomName:            roomName,
			ParticipantIdentity: identity,
		})
		if err != nil {
			fmt.Printf("failed to connect to host; %v\n", err)
			os.Exit(1)
		}
		connectChannel <- *room
		room.Disconnect()
	}()

	// Watch for timeout
	select {
	case room := <-connectChannel:
		if room.LocalParticipant.Identity() == identity {
			fmt.Println("successfully connected to host")
		} else {
			fmt.Println("failed to connect to host; identity did not match expected result")
			os.Exit(1)
		}
	case <-time.After(5 * time.Second):
		fmt.Println("failed to connect to host; timeout waiting for host")
		os.Exit(1)
	}

	return nil
}

func unmarshalKeys(keys string) (apiKey string, apiSecret string, err error) {
	// Get keys in standard livekit format. Use the last key that is set.
	temp := make(map[string]interface{})
	if err = yaml.Unmarshal([]byte(keys), temp); err != nil {
		return
	}

	for key, val := range temp {
		if secret, ok := val.(string); ok {
			apiKey = key
			apiSecret = secret
		}
	}
	return
}