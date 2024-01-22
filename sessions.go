/*
midgaard_bot, a Telegram bot which sets a bridge to Midgaard Merc MUD
Copyright (C) 2017 by Javier Sancho Fernandez <jsf at jsancho dot org>

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
	"strings"

	"github.com/reiver/go-telnet"
	"maunium.net/go/mautrix/id"
)

type Session struct {
	RoomID id.RoomID
	Input chan *MatrixEvent
}

var sessions map[id.RoomID]*Session
var mercHost string
var sessionsCtx context.Context

func initSessions(host string, ctx context.Context) error {
	sessions = make(map[id.RoomID]*Session)
	mercHost = host
	sessionsCtx = ctx
	return nil
}

func getSession(chat id.RoomID) *Session {
	session, ok := sessions[chat]
	if !ok {
		session = newSession(chat)
	}
	return session
}

func newSession(room id.RoomID) *Session {
	session := Session{room, make(chan *MatrixEvent)}
	sessions[room] = &session
	startSession(&session)
	return &session
}

func startSession(session *Session) {
	ctx, cancel := context.WithCancel(sessionsCtx)

	go func() {
		telnetInput, telnetOutput, telnetError := make(chan string), make(chan string), make(chan string)
		caller := TelnetCaller{
			Input:  telnetInput,
			Output: telnetOutput,
			Error:  telnetError,
		}

		go func() {
			for {
				select {
				case evt := <-session.Input:
					msg := evt.Event.Content.AsMessage()
					text := msg.Body
					if msg.NewContent != nil {
						text = msg.NewContent.Body
					}
					if evt.IsDirect {
						telnetInput <- text
					} else {
						if strings.HasPrefix(text, "$") {
							telnetInput <- strings.Trim(text[1:], " ")
						}
					}
				case body := <-telnetOutput:
					sendToMatrix(session.RoomID, body)
				case <-telnetError:
					cancel()
					delete(sessions, session.RoomID)
					return
				case <-ctx.Done():
					return
				}
			}
		}()

		telnet.DialToAndCall(mercHost, caller)
	}()
}

func sendToSession(session *Session, message *MatrixEvent) {
	session.Input <- message
}
