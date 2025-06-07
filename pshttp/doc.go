// Package pshttp provides an HTTP interface to a [ps.Broker].
//
// [NewHandler] wraps a [ps.Broker] and returns an [http.Handler]. The handler
// accepts POST requests for publishing events, and GET requests for subscribing
// to events. Subscriptions are implemented via server-sent events, or SSE, so
// GET requests must accept: text/event-stream.
//
// [Client] wraps a remote URI, which is assumed to be handled by a handler
// returned by [NewHandler]. It provides publish and subscribe methods similar
// to a [ps.Broker].
package pshttp
