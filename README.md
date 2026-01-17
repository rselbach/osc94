# osc94

[![tests](https://github.com/rselbach/osc94/actions/workflows/test.yml/badge.svg)](https://github.com/rselbach/osc94/actions/workflows/test.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/rselbach/osc94.svg)](https://pkg.go.dev/github.com/rselbach/osc94)
[![Go Report Card](https://goreportcard.com/badge/github.com/rselbach/osc94)](https://goreportcard.com/report/github.com/rselbach/osc94)
[![License](https://img.shields.io/github/license/rselbach/osc94)](LICENSE)

OSC 9;4 progress helpers for terminal tabs/taskbars.

## Install

```sh
go get github.com/rselbach/osc94
```

## Usage

```go
package main

import (
  "os"

  "github.com/rselbach/osc94"
)

func main() {
  progress := osc94.New(os.Stderr, osc94.WithAutoEnable())

  _ = progress.SetPercent(10)
  _ = progress.Warning(50)
  _ = progress.Indeterminate()
  _ = progress.Clear()
}
```

## Env overrides

- `OSC94_DISABLE=1` disables output.
- `OSC94_FORCE=1` enables output.

## Low-level escape

```go
seq, _ := osc94.Escape(osc94.StateNormal, 25)
```
