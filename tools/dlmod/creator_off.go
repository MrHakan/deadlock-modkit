//go:build nomeshy

package main

import "net/http"

// dlmod without Meshy: the studio has no Create tab (creator.go is left out).

type creatorSession struct{}

func (s *studio) creatorRoutes(api func(string, func(*http.Request) (any, error))) {}
