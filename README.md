# ps [![Go Reference](https://pkg.go.dev/badge/github.com/peterbourgon/ps.svg)](https://pkg.go.dev/github.com/peterbourgon/ps) [![GitHub Release](https://img.shields.io/github/v/release/peterbourgon/ps?style=flat-square)](https://github.com/peterbourgon/ps/releases) [![Build Status](https://github.com/peterbourgon/ps/actions/workflows/test.yaml/badge.svg)](https://github.com/peterbourgon/ps/actions/workflows/test.yaml)

General purpose pub/sub for Go.

Create a [Broker](https://pkg.go.dev/github.com/peterbourgon/ps#Broker).
Callers can [Publish](https://pkg.go.dev/github.com/peterbourgon/ps#Broker.Publish) values to the broker,
and clients can [Subscribe](https://pkg.go.dev/github.com/peterbourgon/ps#Broker.Subscribe) with a
channel that receives all published values which pass the provided `allow` func.

Publishing is best-effort; if a subscriber is slow or non-responsive, published
values to that subscriber are dropped.

[package pshttp](https://pkg.go.dev/github.com/peterbourgon/ps/pshttp) provides
an HTTP interface over a pub/sub broker.
