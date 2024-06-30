package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

func main() {
	godotenv.Load()

	// uncomment this to send sample slack alert
	// valFields := []slack.AttachmentField{
	// 	{Title: "welcome", Value: "welcome to slack bot"},
	// 	{Title: "date", Value: time.Now().UTC().String()},
	// }
	// if err := sendSlackAlert("Super Bot Text", "Slack Bot", valFields); err != nil {
	// 	log.Fatal("error sending slack alert")
	// }

	// now using slack bot to handle events
	handleEvents()
}

func sendSlackAlert(pretext, text string, valFields []slack.AttachmentField) error {

	//  get these token from web UI
	token := os.Getenv("SLACK_AUTH_TOKEN")
	channelID := os.Getenv("SLACK_CHANNEL_ID")

	// slack client to send messages
	client := slack.New(token, slack.OptionDebug(true))

	// message which is supposed to be sent
	attachment := slack.Attachment{
		Pretext: pretext,
		Text:    text,
		Color:   "4af030",
		Fields:  valFields,
	}

	// send message with post request
	_, timeStamp, err := client.PostMessage(channelID, slack.MsgOptionAttachments(attachment))
	if err != nil {
		return err
	}

	fmt.Println("message sent at ", timeStamp)
	return nil
}

func handleEvents() {
	token := os.Getenv("SLACK_AUTH_TOKEN")
	appToken := os.Getenv("SLACK_APP_TOKEN")

	client := slack.New(token, slack.OptionDebug(true), slack.OptionAppLevelToken(appToken))

	socketClient := socketmode.New(
		client,
		socketmode.OptionDebug(true),
		socketmode.OptionLog(log.New(os.Stdout, "socketmode: ", log.Lshortfile|log.LstdFlags)),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func(ctx context.Context, client *slack.Client, socketClient *socketmode.Client) {
		for {
			select {
			case <-ctx.Done():
				fmt.Println("shutting down socketmode listener")
				return
			case event := <-socketClient.Events:
				switch event.Type {
				case socketmode.EventTypeEventsAPI:
					eventsAPI, ok := event.Data.(slackevents.EventsAPIEvent)
					if !ok {
						log.Printf("Could not type cast the event to the EventsAPI: %v\n", event)
						continue
					}

					socketClient.Ack(*event.Request)
					err := handleEventMessage(eventsAPI, client)
					if err != nil {
						log.Fatal(err)
					}
				}
			}
		}
	}(ctx, client, socketClient)

	socketClient.Run()
}

func handleEventMessage(event slackevents.EventsAPIEvent, client *slack.Client) error {
	switch event.Type {
	case slackevents.CallbackEvent:
		innerEvent := event.InnerEvent
		switch evnt := innerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			err := handleBotMentionEvent(evnt, client)
			if err != nil {
				return err
			}
		default:
			return errors.New("unsupported inner event type")
		}
	default:
		return errors.New("invalid event received")
	}

	return nil
}

func handleBotMentionEvent(event *slackevents.AppMentionEvent, client *slack.Client) error {

	user, err := client.GetUserInfo(event.User)
	if err != nil {
		return err
	}

	text := strings.ToLower(event.Text)
	attachment := slack.Attachment{}

	if strings.Contains(text, "hello") || strings.Contains(text, "hi") {
		attachment.Text = fmt.Sprintf("Hello :) %s", user.Name)
		attachment.Color = "#4af030"
	} else if strings.Contains(text, "weather") {
		attachment.Text = fmt.Sprintf("Weather is sunny today. %s ", user.Name)
		attachment.Color = "#4af030"
	} else {
		attachment.Text = fmt.Sprintf("I am good. How are you %s?", user.Name)
		attachment.Color = "#4af030"
	}

	_, _, err = client.PostMessage(event.Channel, slack.MsgOptionAttachments(attachment))
	if err != nil {
		return nil
	}
	return nil
}
