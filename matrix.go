/*
midgaard_matrix_bot, a Matrix bot which sets a bridge to MUD

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package main

import (
	"context"
	"fmt"
	"html"
	"log"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type MatrixConfig struct {
	HomeServerUrl string `short:"s" long:"homeserver" description:"URL of the Matrix home server" required:"true"`
	UserID string `short:"u" long:"user" description:"ID of the Matrix user" required:"true"`
	AccessToken string `short:"t" long:"token" description:"Access token of the Matrix user" required:"true"`
}

type MatrixSendable struct {
	Message string
	RoomID id.RoomID
}

type MatrixEvent struct {
	Event *event.Event
	IsDirect bool
}

var sendChannel chan *MatrixSendable

var directRooms map[id.RoomID] struct{}

func initMatrixWorkers(config MatrixConfig, ctx context.Context) error {
	client, err := mautrix.NewClient(config.HomeServerUrl, id.UserID(config.UserID), config.AccessToken)
	if err != nil {
		return err
	}

	log.Printf("Authorized on account %s", client.UserID)
	directRooms = make(map[id.RoomID]struct{})


	syncer := client.Syncer.(*mautrix.DefaultSyncer)
	syncer.OnEventType(event.EventMessage, func(ctx context.Context, evt *event.Event) {
		if evt.Sender != client.UserID {
			if evt.Content.AsMessage().MsgType == event.MsgText {
				if evt.Timestamp > time.Now().Add(-time.Minute).UnixMilli() {
					_, isDirect := directRooms[evt.RoomID]
					session := getSession(evt.RoomID)
					sendToSession(session, &MatrixEvent{
						evt,
						isDirect,
					})
				}
			}
		}
	})
	syncer.OnEventType(event.StateMember, func(ctx context.Context, evt *event.Event) {
		if evt.GetStateKey() == client.UserID.String() && evt.Content.AsMember().Membership == event.MembershipInvite {
			_, err := client.JoinRoomByID(ctx, evt.RoomID)
			if err == nil {
				log.Printf("Joined room %s after invite by %s", evt.RoomID.String(), evt.Sender.String())
				if evt.Content.AsMember().IsDirect {
					log.Printf("Room %s is a DM!", evt.RoomID)
					directRooms[evt.RoomID] = struct{}{}
				}
			} else {
				log.Printf("Failed to join room %s after invite by %s", evt.RoomID.String(), evt.Sender.String())
				log.Print(err)
			}
		}
	})
	syncer.OnEventType(event.AccountDataDirectChats, func(ctx context.Context, evt *event.Event) {
		log.Printf("DM data received: %s", evt.Content.Raw)
		chatMap := evt.Content.AsDirectChats()
		for _, rooms := range *chatMap {
			for _, room := range rooms {
				directRooms[room] = struct{}{}
			}
		}
	})



	sendChannel = make(chan *MatrixSendable)
	go sendWorker(client, sendChannel, ctx)
	go recvWorker(client, ctx)

	return nil
}

func recvWorker(client *mautrix.Client, ctx context.Context) {
	err := client.SyncWithContext(ctx)
	if err != nil {
		log.Panic(err)
	}
}

func sendWorker(client *mautrix.Client, sendChannel chan *MatrixSendable, ctx context.Context) {
	for {
		select {
		case msg := <-sendChannel:
			client.SendMessageEvent(ctx, msg.RoomID, event.EventMessage, &event.MessageEventContent{
				MsgType: event.MsgText,
				Body:    formatStringForSending(msg.Message),
				Format: event.FormatHTML,
				FormattedBody: formatStringForHTML(msg.Message),
			})
		case <-ctx.Done():
			return
		}
	}
}

func sendToMatrix(room_id id.RoomID, body string) {
	msg := MatrixSendable{
		Message: body,
		RoomID: room_id,
	}
	sendChannel <- &msg
}

func formatStringForSending(raw string) string {
	return fmt.Sprintf("```\n%s\n```", raw)
}
func formatStringForHTML(raw string) string {
	return fmt.Sprintf("<pre>%s</pre>", html.EscapeString(raw))
}
