# status

[![Run Tests](https://github.com/alarmistdev/status/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/alarmistdev/status/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/alarmistdev/status)](https://goreportcard.com/report/github.com/alarmistdev/status)
[![GoDoc](https://godoc.org/github.com/alarmistdev/status?status.svg)](https://godoc.org/github.com/alarmistdev/status)

A Go package for health checking and status page generation. It provides a simple way to monitor the health of various dependencies and services in your application, with a status page dashboard.

## Usage

```go
import (
    "time"

    "github.com/alarmistdev/status/check"
    dockercheck "github.com/alarmistdev/status/check/system/docker"
)

dockerCheck, err := dockercheck.Check(
    map[string]string{"app": "status", "env": "prod"},
    30*time.Second,           // background refresh interval
    check.Config{Timeout: 5 * time.Second},
)
if err != nil {
    log.Fatalf("docker check init failed: %v", err)
}
defer dockerCheck.Close()

healthChecker.WithTarget("Status containers", dockerCheck)
```

See [example/main.go](example/main.go).

