package main

import "go.uber.org/zap"

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// SQS Receiver
	receiver *SQSReceiver

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

func newHub(receiver *SQSReceiver) *Hub {
	return &Hub{
		receiver:   receiver,
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			logger.Debug("client register")
			h.clients[client] = true
		case client := <-h.unregister:
			logger.Debug("client unregister")
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case msg, ok := <-h.receiver.RecvCh:
			if !ok {
				logger.Debug("recv channel closed")
				return
			}

			msgCounter.Add(float64(len(msg.Messages)))
			for _, m := range msg.Messages {
				body := []byte(*m.Body)
				logger.Debug("sqs msg", zap.String("body", string(body)))

				for client := range h.clients {
					select {
					case client.send <- body:
					default:
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
			h.receiver.DeleteCh <- msg
		}
	}
}
