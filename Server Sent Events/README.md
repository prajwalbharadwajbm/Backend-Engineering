# Server Sent Events 

This Project is a simple learning to understand how SSE works. Server written in Golang and Client in NodeJS with features on close connection and re-establish connection. 

## How to run the Project?
1. Clone the repo
2. Make sure to have go installed on the machine you want to run. 
3. Run `go run main.go` under Server Sent Events repo
4. Check the magic in localhost:8080 to see 10 messages sent as response events using SSE

A Server Push is a technology that allows a server to push data/events to the client over a single, persistent HTTP connection, enabling real-time data streaming on the web.

## Limitations of Request/Response Model
1. Vanilla Request/Response is not ideal for notification backends. *For example: Whenever a client wants notifications/events like **"Client just logged-in" or "A message was just received"**.*
2. **Push** works, but is restrictive. Yes, we can set up a WebSocket and push notifications to the client, but it has limitations.
3. Server Sent Events work with any HTTP connection (Request/Response Model), so we don't need a WebSocket connection. SSE is specifically designed for HTTP.

## How Does it Work?
1. In Server Sent Events, responses contain markers to indicate when a message starts and ends.
2. Chronologically: Client sends a request, and the server sends back logical events as part of the response. The server never actually sends an end event in the response. Instead, in the ongoing response stream, whenever the client receives a mini-response, it parses it and understands that "hey, somebody just logged in."
3. It's still a request but with a never-ending response.
4. The client parses streams of data, looking for these events.

**Note**: Server must be at least HTTP 1.1, as anything below that will not support streaming.

## Pros and Cons of SSE

### Pros
1. **Real-time**: It's an open channel where the server delivers information as soon as it's available.
2. **Compatible** with the Request/Response model.

### Cons
1. **Client must be online** - As we're sending a response, the client has to be there to receive it.
2. **Client handling challenges** - You may need to handle scenarios by saving the client state on the server and resuming when it's back online, or storing responses that weren't received by the client. This can put unnecessary heavy load on the backend.
3. **Polling is preferred** for lightweight clients.
4. **HTTP 1.1 limitation** of having only 6 open connections per domain.

**Implementation Note**: Start with `data: xxxx \n\n` for the browser to parse it, and remember to set the header as `text/event-stream`.