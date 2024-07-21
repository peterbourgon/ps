// Package pshttp provides an HTTP interface to a ps.Broker.
//
// [Handler] wraps a [ps.Broker] and implements [http.Handler]. It accepts POST
// requests for publishing events, and GET requests for subscribing to events.
// Subscriptions are implemented via SSE, so GET requests must accept:
// text/event-stream.
//
// [Client] wraps a remote URI, which is assumed to be handled by a [Handler].
// It provides publish and subscribe methods similar to a [ps.Broker].
package pshttp
