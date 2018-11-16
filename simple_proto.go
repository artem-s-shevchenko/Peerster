package main

import (
    "fmt"
    "strings"
)

func processSimpleClient(gossiper *Gossiper) {
	for {
		dec, _, _ := listen(gossiper.ClientConn)
		dec.Simple.OriginalName = gossiper.Nodename
		dec.Simple.RelayPeerAddr = gossiper.GossipAddress
    	fmt.Println("CLIENT MESSAGE", dec.Simple.Contents)
    	gossiper.SafePeersList.mux.Lock()
    	fmt.Println("PEERS", strings.Join(gossiper.SafePeersList.PeersList, ","))
    	for _, v := range gossiper.SafePeersList.PeersList {
    		sendMessage(dec, v, gossiper.PeerConn)
    	}
    	gossiper.SafePeersList.mux.Unlock()
	}
}

func processSimplePeer(gossiper *Gossiper) {
	for {
		dec, _, err:= listen(gossiper.PeerConn)
        if err == nil {
        	sender := dec.Simple.RelayPeerAddr
        	dec.Simple.RelayPeerAddr = gossiper.GossipAddress
        	fmt.Println("SIMPLE MESSAGE origin", dec.Simple.OriginalName, "from", sender, "contents", dec.Simple.Contents)
        	gossiper.SafePeersList.mux.Lock()
        	checkAndAppend(&gossiper.SafePeersList.PeersList, sender)
        	fmt.Println("PEERS", strings.Join(gossiper.SafePeersList.PeersList, ","))
        	for _, v := range gossiper.SafePeersList.PeersList {
        		if v != sender {
        			sendMessage(dec, v, gossiper.PeerConn)
        		}
        	}
        	gossiper.SafePeersList.mux.Unlock()
        }
	}
}
