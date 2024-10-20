
[<img src="https://github.com/user-attachments/assets/e273ed0c-240b-4498-ac75-edbaaac734c4" width="250"/>](https://github.com/user-attachments/assets/e273ed0c-240b-4498-ac75-edbaaac734c4)
# socky.go - A simple WebSocket library for Go.
## Overview
socky.go is designed to simplify WebSocket server management in Go applications. This library abstracts common WebSocket operations through a system of Events. This system is both easy to understand and work with on the backend, and also easy to integrate within the frontend. Here is a quick look at what an Event looks like:
```go
type Event struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
	From    string          `json:"from"`
}
```
Pretty much, every event contains 3 key items; a type, a payload, and who it is from.
**Type**: The type of message which is being sent: this allows the server to route the message to its corresponding event handler.

**Payload**: The meaty part of the message: this is what is actually being sent, for example: in a chat application, the payload of a message of type ```send_message``` would contain the payload as a string of the actual message sent by the user. Or, in a game, a message of type ```update_state``` would contain the payload as a JSON Object containing the player that has moved and the new coordinates of the player in something like this:
```json
{
	"type": "update_state",
	"payload": {
		"player": "bob123",
		"position": {
			"x": 220.710,
			"y": -310.03
		}
	},
	"from": "bob123"
}
```
The ability to use JSON Objects on the front-end and Event structs on the back-end greatly eases the process of using these custom objects and allows for a simple integration on the full stack of development.

## Features
- **Simplified Event Management**: socky.go uses an `Event` struct to streamline event handling and communcation.
- **Connection Health**: Built in ping/pong mechanisms ensure that connections remain alive and healthy.
-  **Easy Integration**: Designed to integrate seamlessly into Go applications with minimal configuration. 
- **Room Support**: Built in support for rooms, allowing the management of private communication within groups of (or single) connections. 

## Installation
To install the library, run:
```go
go get github.com/smarbo/socky
```
## Usage
### Importing the Package
```go
import "github.com/smarbo/socky"
```
### Creating a new Socky Instance
```go
sc := socky.Socky()
```
To run your socky instance, use it's `Serve` function as a handler function for a HTTP route, for example:
```go
http.HandleFunc("/socky_ws", sc.Serve)
```
###  Defining Custom Event Handlers
Socky comes with a built-in function to allow the creation of custom Event Handlers.
The following example defines an Event Handler which handles events of type `ping`, and sends back an Event of type `pong` with the payload of the incoming event, and from `SOCKY_SERVER`. These are fully configurable and allow you to create any type of Event Handler needed through a simple interface.
```go
sc.AddEventHandler("ping", func(event socky.Event, c *socky.Client) error {
    c.SendEvent(socky.Event{ Type: "pong", Payload: event.Payload, From: "SOCKY_SERVER" });
    return nil
})
```
The `sc.AddEventListener` function takes in 2 parameters, `msgType` and `handler`, where `msgType` is a string representing the message type to match when routing incoming events, and where `handler` is a function of type `EventHandler`, which takes in an event of type `Event`, and a `Client` object.

In your custom event handlers, JSON data can be extracted from the `event.Payload json.rawMessage` object via the following steps:
1. Define your struct, which will be used for extracting data.
```go
type RequestData struct {
    Title   string `json:"title"`
    Content string `json:"content"`
    Age     int    `json:"age"`
}
```
2. Then, create your `socky` event handler as shown below, where you can unmarshal the payload into an instance of your struct. This instance can then be used to access the individual fields of the JSON object.
```go
sc.AddEventHandler("request", func(event socky.Event, c *socky.Client) error {
        var reqData RequestData
        if err := json.Unmarshal(event.Payload, &reqData); err != nil {
        return err
    }
    fmt.Printf("Request Title: %s\n", reqData.Title)
    fmt.Printf("Request Content: %s\n", reqData.Content)
    fmt.Printf("Request Age: %d\n", reqData.Age)
    return nil
})
```

### Sending Events
Socky comes with multiple ways to send Events:
```go
client.SendEvent(event socky.Event)
client.BroadcastEvent(event socky.Event)
client.RoomcastEvent(event socky.Event)
```
Each one of these are pretty self-explanatory, but for those who want to know more, here is an explanation for each:
- **SendEvent**: Sends an event to the client on which it is being called to.
- **BroadcastEvent**: Sends an event to every client (except the current) currently connected to the Socky instance.
- **RoomcastEvent**: Sends an event to every client (except the current) that shares the same `room` property in the Socky instance.

## API Reference
`Socky() *Manager`
Returns a new Socky Manager instance.

`(*Manager) AddEventHandler(msgType string, handler EventHandler)`
Adds a new `EventHandler` to the Event router.

`(*Manager) OnConnect(func(*Client) error)`
Run when a new connection is created.

`(*Manager) OnDisconnect(func(*Client) error)`
Run when a connection is closed.

`(*Manager) Serve(w http.ResponseWriter, r *http.Request)`
Function used for serving the WebSocket server.

## Contributing
Contributions are welcome! Please submit issues or pull requests on the [GitHub Repository](https://github.com/smarbo/socky)
## License
This library is licensed under the [MIT License](https://opensource.org/license/mit).
