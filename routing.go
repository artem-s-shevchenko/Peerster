package main

import(
	"time"
)

func routeRumor(gossiper *Gossiper, period int) {
	for {
		time.Sleep(time.Duration(period) * time.Second)
		processRumorClient(gossiper, "")
	}
}
