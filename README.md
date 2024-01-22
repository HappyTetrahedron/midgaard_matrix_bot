# midgaard_matrix_bot

## tl;dr;

`midgaard_matrix_bot` is a Matrix bot designed to connect to the Midgaard MUD (but it can be used to connect to other MUDs or telnet servers).

## Midgaard Matrix Bot, conneting to a MUD with Matrix

This bot is based on the [Telegram version](https://github.com/HappyTetrahedron/midgaard_bot).

The original author "had the idea of writing a bot to connect to a MUD (Multi User Dungeon) using Telegram".
He "decided to use Golang to practice a little with this language and its goroutines".

This bot is available under a GPLv3 License.

When the bot starts, it launchs a new goroutine that listens messages that arrive from Matrix. When a message arrives, it checks if there is an open session for the chat owner of the message. If it's the first message from that chat, propably there won't be an open session.

In that case, the bot launches a new goroutine for connecting with the MUD by telnet and serving as a connector between Matrix and the server.

For checking in which time the telnet connection has finished sending a complete message, and then send it to Matrix, a buffer was implemented to receive data from the MUD and use a 500 milliseconds timeout. After the timeout, in theory the message is complete so it can be sent: it isn't a magical solution, but for local connections it works well.

When the message is complete, we send it to a specific goroutine that receives messages from open sessions and forwards them to Matrix.

The original author created this bot as a PoC, but further development has been [happening by the community](https://github.com/Jereviendrai/midgaard_bot).
