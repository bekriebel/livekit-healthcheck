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

type ConnectResult struct {
	room *lksdk.Room
	err  error
}

func healthcheck(c *cli.Context) error {
	// Get host
	host := c.String("host")

	// Get API key/secret
	apiKey, apiSecret, err := unmarshalKeys(c.String("keys"))
	if err != nil {
		return errors.New("could not parse keys, it needs to be \"key: secret\", one per line")
	}

	// Create a random room name
	roomName := funk.RandomString(16)

	// Set identity
	identity := "livekit-healthcheck"

	// Create connection channel to watch for timeout
	connectChannel := make(chan ConnectResult, 1)

	go func() {
		// Attempt to connect to the room
		room, err := lksdk.ConnectToRoom(host, lksdk.ConnectInfo{
			APIKey:              apiKey,
			APISecret:           apiSecret,
			RoomName:            roomName,
			ParticipantIdentity: identity,
		})
		if err != nil {
			err = fmt.Errorf("failed to connect to host; %v", err)
		}
		connectChannel <- ConnectResult{room, err}
		room.Disconnect()
	}()

	// Watch for timeout
	select {
	case connectResult := <-connectChannel:
		if connectResult.err != nil {
			return connectResult.err
		}
		if connectResult.room.LocalParticipant.Identity() == identity {
			fmt.Println("successfully connected to host")
		} else {
			return errors.New("failed to connect to host; identity did not match expected result")
		}
	case <-time.After(c.Duration("timeout")):
		return errors.New("failed to connect to host; timeout waiting for host")
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

func main() {
	// Get CLI parameters
	app := &cli.App{
		Name:  "livekit-healthcheck",
		Usage: "Check the health of the livekit server by attempting to connect to a room",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "keys",
				Usage:    "api keys (key: secret\\n)",
				EnvVars:  []string{"LIVEKIT_KEYS"},
				Required: true,
			},
			&cli.StringFlag{
				Name:     "host",
				Usage:    "host (incl. port) of the livekit server to connect to (example: wss://livekit.example.com:7880)",
				EnvVars:  []string{"LIVEKIT_HOST"},
				Required: true,
			},
			&cli.DurationFlag{
				Name:    "timeout",
				Usage:   "time before giving up on connection to host",
				EnvVars: []string{"LIVEKIT_HEALTHCHECK_TIMEOUT"},
				Value:   5 * time.Second,
			},
		},
		Action: healthcheck,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
