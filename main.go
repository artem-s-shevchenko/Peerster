package main

import (
	//"fmt"
	"flag"
	"strings"
	"math/rand"
	"time"
)

func main() {
	uiport := flag.String("UIPort", "8080", "port for UI client")
	guiport := flag.String("GUIPort", "-1", "port for gui")
	gossipaddr := flag.String("gossipAddr", "127.0.0.1:5000", "ip:port for the gossiper")
	nodename := flag.String("name", "", "name of the gossiper")
	peers := flag.String("peers", "", "peers list")
	simple := flag.Bool("simple", false, "simple mode")
	rtimer := flag.Int("rtimer", 0, "route rumors sending period in seconds, 0 to disable")
	flag.Parse()
	peers_list := strings.Split(*peers, ",")
	if *peers == "" {
		peers_list = []string{}
	}
	rand.Seed(time.Now().UnixNano())
	goss := NewGossiper("127.0.0.1:"+*uiport, *gossipaddr, *nodename, peers_list)
	defer goss.ClientConn.Close()
	defer goss.PeerConn.Close()
	if (*simple == true) {
		go processSimpleClient(goss)
		processSimplePeer(goss)
	} else {
		go listenRumorClient(goss)
		go antiEntropy(goss)
		go mine(goss)
		if *guiport != "-1" {
			go runGui(goss, *guiport)
		}
		if *rtimer != 0 {
			processRumorClient(goss, "")
			go routeRumor(goss, *rtimer)
		}
		processRumorPeer(goss)
	}
}
