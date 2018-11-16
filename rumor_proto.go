package main

import (
	"fmt"
	"net"
	"strings"
	"math/rand"
	"time"
    "sync/atomic"
    "crypto/sha256"
)

var timeout int = 1

func processRumorClient(gossiper *Gossiper, message string) {
	if message != "" {
    	fmt.Println("CLIENT MESSAGE", message)
	}
    gossiper.SafeMessageHistory.mux.Lock()
    gossiper.SafeMessageHistory.History[gossiper.Nodename] = append(gossiper.SafeMessageHistory.History[gossiper.Nodename], message)
    gossiper.SafeMessageHistory.mux.Unlock()
    if message != "" {
	    gossiper.SafeUpdateMessageList.mux.Lock()
	    gossiper.SafeUpdateMessageList.List = append(gossiper.SafeUpdateMessageList.List, gossiper.Nodename+": "+message)
	    gossiper.SafeUpdateMessageList.mux.Unlock()
	}
    mess := RumorMessage{gossiper.Nodename, atomic.AddUint32(&gossiper.id, 1), message}
    packet := GossipPacket{Rumor: &mess}
    gossiper.SafePeersList.mux.Lock()
    if message != "" {
    	fmt.Println("PEERS", strings.Join(gossiper.SafePeersList.PeersList, ","))
    }
    if len(gossiper.SafePeersList.PeersList) > 0 {
        rndPeer := gossiper.SafePeersList.PeersList[rand.Intn(len(gossiper.SafePeersList.PeersList))]
        fmt.Println("MONGERING with", rndPeer)
        sendMessage(&packet, rndPeer, gossiper.PeerConn)
        gossiper.SafeTimeoutMap.mux.Lock()
        timeoutCreate(gossiper, &packet, rndPeer, "")
        gossiper.SafeTimeoutMap.mux.Unlock()
    }
    gossiper.SafePeersList.mux.Unlock()
}

func listenRumorClient(gossiper *Gossiper) {
	for {
		dec, _, _ := listen(gossiper.ClientConn)
		if dec.Simple != nil {
    		processRumorClient(gossiper, dec.Simple.Contents)
    	}
    	if dec.Private != nil {
            fmt.Println("SENDING PRIVATE MESSAGE", dec.Private.Text, "TO", dec.Private.Destination)
    		dec.Private.Origin = gossiper.Nodename
            dec.Private.HopLimit -= 1
            gossiper.SafePrivateMessageHistory.mux.Lock()
            gossiper.SafePrivateMessageHistory.History[dec.Private.Destination] = append(gossiper.SafePrivateMessageHistory.History[dec.Private.Destination], 
                dec.Private.Origin+": "+dec.Private.Text)
            gossiper.SafePrivateMessageHistory.mux.Unlock()
            gossiper.RoutingTable.mux.Lock()
    		sendMessage(dec, gossiper.RoutingTable.RouteTable[dec.Private.Destination], gossiper.PeerConn)
            gossiper.RoutingTable.mux.Unlock()
    	}
    	if dec.DataRequest != nil {
    		if dec.DataRequest.Destination == "" {
    			go index_file(gossiper, dec.DataRequest.Origin)
			} else {
				go download_file(gossiper, dec.DataRequest.Destination, dec.DataRequest.HashValue, dec.DataRequest.Origin)
			}
    	}
	}
}

func createStatus(hist *map[string] []string, mess *StatusPacket) {
	for k, v := range *hist {
		mess.Want = append(mess.Want, PeerStatus{k, uint32(len(v)+1)})
	}
}

func flipCoinSend(gossiper *Gossiper, dec *GossipPacket, not_send string) {
	flag := rand.Int() % 2
    if flag == 1 {
        gossiper.SafePeersList.mux.Lock()
        if len(gossiper.SafePeersList.PeersList) > 0 {
        	if len(gossiper.SafePeersList.PeersList) == 1 && gossiper.SafePeersList.PeersList[0] == not_send {
        		gossiper.SafePeersList.mux.Unlock()
        		return
        	}
            rndPeer := not_send
            for rndPeer == not_send {
                rndPeer = gossiper.SafePeersList.PeersList[rand.Intn(len(gossiper.SafePeersList.PeersList))]
            }
            fmt.Println("FLIPPED COIN sending rumor to", rndPeer)
            sendMessage(dec, rndPeer, gossiper.PeerConn)
            gossiper.SafeTimeoutMap.mux.Lock()
	        timeoutCreate(gossiper, dec, rndPeer, "")
	        gossiper.SafeTimeoutMap.mux.Unlock()
        }
        gossiper.SafePeersList.mux.Unlock()
    }
}

func timeoutCreate(gossiper *Gossiper, dec *GossipPacket, peerAddr string, not_send string) {
	timer :=time.AfterFunc(time.Duration(timeout) * time.Second, func() {
		gossiper.SafeTimeoutMap.mux.Lock()
        index := -1;
        for i, v := range gossiper.SafeTimeoutMap.TimeoutMap[peerAddr] {
            if v.Origin == dec.Rumor.Origin && v.ID == dec.Rumor.ID {
                index = i
                break
            }
        }
        if index != -1 {
            copy(gossiper.SafeTimeoutMap.TimeoutMap[peerAddr][index:], gossiper.SafeTimeoutMap.TimeoutMap[peerAddr][index+1:])
            gossiper.SafeTimeoutMap.TimeoutMap[peerAddr][len(gossiper.SafeTimeoutMap.TimeoutMap[peerAddr])-1] = Timeout{}
            gossiper.SafeTimeoutMap.TimeoutMap[peerAddr] = gossiper.SafeTimeoutMap.TimeoutMap[peerAddr][:len(gossiper.SafeTimeoutMap.TimeoutMap[peerAddr])-1]
        }
	    gossiper.SafeTimeoutMap.mux.Unlock()
	    flipCoinSend(gossiper, dec, not_send)
	})
	gossiper.SafeTimeoutMap.TimeoutMap[peerAddr] = append(gossiper.SafeTimeoutMap.TimeoutMap[peerAddr], Timeout{timer, dec.Rumor.Origin, dec.Rumor.ID, dec.Rumor.Text})
}

func processRumorMessage(gossiper *Gossiper, dec *GossipPacket, addr *net.UDPAddr) {
	if dec.Rumor.Text != "" {
   		fmt.Println("RUMOR origin", dec.Rumor.Origin, "from", addr.String(), "ID", dec.Rumor.ID, "contents", dec.Rumor.Text)
	}
	if dec.Rumor.Origin != gossiper.Nodename {
        gossiper.RoutingTable.mux.Lock()
		if gossiper.RoutingTable.IdTable[dec.Rumor.Origin] < dec.Rumor.ID {
            gossiper.RoutingTable.IdTable[dec.Rumor.Origin] = dec.Rumor.ID
            if  gossiper.RoutingTable.RouteTable[dec.Rumor.Origin] != addr.String() {
                gossiper.RoutingTable.RouteTable[dec.Rumor.Origin] = addr.String()
                fmt.Println("DSDV", dec.Rumor.Origin, addr.String())
            } 
		}
        gossiper.RoutingTable.mux.Unlock()
	}
    gossiper.SafeMessageHistory.mux.Lock()
    if uint32(len(gossiper.SafeMessageHistory.History[dec.Rumor.Origin])+1) == dec.Rumor.ID {
        gossiper.SafeMessageHistory.History[dec.Rumor.Origin] = append(gossiper.SafeMessageHistory.History[dec.Rumor.Origin], dec.Rumor.Text)
        mess := StatusPacket{}
        createStatus(&gossiper.SafeMessageHistory.History, &mess)
        gossiper.SafeMessageHistory.mux.Unlock()
        if dec.Rumor.Text != "" {
	        gossiper.SafeUpdateMessageList.mux.Lock()
	        gossiper.SafeUpdateMessageList.List = append(gossiper.SafeUpdateMessageList.List, dec.Rumor.Origin+": "+dec.Rumor.Text)
	        gossiper.SafeUpdateMessageList.mux.Unlock()
    	}
        packet := GossipPacket{Status: &mess}
        sendMessage(&packet, addr.String(), gossiper.PeerConn)
    } else {
        mess := StatusPacket{}
        createStatus(&gossiper.SafeMessageHistory.History, &mess)
        packet := GossipPacket{Status: &mess}
        sendMessage(&packet, addr.String(), gossiper.PeerConn)
        gossiper.SafeMessageHistory.mux.Unlock()
        return
    }
    gossiper.SafePeersList.mux.Lock()
    if len(gossiper.SafePeersList.PeersList) > 1 {
        rndPeer := addr.String()
        for rndPeer == addr.String() {
            rndPeer = gossiper.SafePeersList.PeersList[rand.Intn(len(gossiper.SafePeersList.PeersList))]
        }
        fmt.Println("MONGERING with", rndPeer)
        sendMessage(dec, rndPeer, gossiper.PeerConn)
        gossiper.SafeTimeoutMap.mux.Lock()
        timeoutCreate(gossiper, dec, rndPeer, "")
        gossiper.SafeTimeoutMap.mux.Unlock()
    } 
    gossiper.SafePeersList.mux.Unlock()
}

func processStatusMessage(gossiper *Gossiper, dec *GossipPacket, addr *net.UDPAddr) {
    status_map := map[string]uint32{}
    status_message := "STATUS from " + addr.String()
    for _, peerstatus := range dec.Status.Want {
        status_map[peerstatus.Identifier] = peerstatus.NextID
        status_message = status_message + " peer " + peerstatus.Identifier + " nextID " + fmt.Sprint(peerstatus.NextID)
    }
    fmt.Println(status_message)
    ask_was_received := false
    var flipmessages []GossipPacket
    gossiper.SafeTimeoutMap.mux.Lock()
    indexes := []int{};
    for i, v := range gossiper.SafeTimeoutMap.TimeoutMap[addr.String()] {
        newid, ok := status_map[v.Origin] 
        if ok == true {
            if newid-1 >= v.ID {
                indexes = append(indexes, i)
            }
        }
    }
    if len(indexes) > 0 {
        for i, v :=range indexes {
            index_to_delete := v-i
            timeout := gossiper.SafeTimeoutMap.TimeoutMap[addr.String()][index_to_delete]
            timeout.Timer.Stop()
            copy(gossiper.SafeTimeoutMap.TimeoutMap[addr.String()][index_to_delete:], gossiper.SafeTimeoutMap.TimeoutMap[addr.String()][index_to_delete+1:])
            gossiper.SafeTimeoutMap.TimeoutMap[addr.String()][len(gossiper.SafeTimeoutMap.TimeoutMap[addr.String()])-1] = Timeout{}
            gossiper.SafeTimeoutMap.TimeoutMap[addr.String()] = gossiper.SafeTimeoutMap.TimeoutMap[addr.String()][:len(gossiper.SafeTimeoutMap.TimeoutMap[addr.String()])-1]
            ask_was_received = true
            flipmessages = append(flipmessages, GossipPacket{Rumor: &RumorMessage{timeout.Origin, timeout.ID, timeout.Text}})
        }
    }
    gossiper.SafeTimeoutMap.mux.Unlock()
    gossiper.SafeMessageHistory.mux.Lock()
    was_found := false
    var message RumorMessage;
    for k, v := range gossiper.SafeMessageHistory.History {
        newid, ok := status_map[k]
        if ok == false {
            message = RumorMessage{k, 1, v[0]}       
            was_found = true
            break
        }
        if newid <= uint32(len(v)) && newid > 0 {
            message = RumorMessage{k, newid, v[newid-1]}
            was_found = true
            break
        }
    }
    if was_found {
        gossiper.SafeMessageHistory.mux.Unlock()
        packet := GossipPacket{Rumor: &message}
        fmt.Println("MONGERING with", addr.String())
        sendMessage(&packet, addr.String(), gossiper.PeerConn)
        gossiper.SafeTimeoutMap.mux.Lock()
	    timeoutCreate(gossiper, &packet, addr.String(), "")
	    gossiper.SafeTimeoutMap.mux.Unlock()
        return
    }
    for k, newid := range status_map {
        list, ok := gossiper.SafeMessageHistory.History[k]
        if ok == false {
            was_found = true
            break
        }
        if newid-1 > uint32(len(list)) {
            was_found = true
            break
        }
    }
    if was_found {
        mess := StatusPacket{}
        createStatus(&gossiper.SafeMessageHistory.History, &mess)
        gossiper.SafeMessageHistory.mux.Unlock()
        packet := GossipPacket{Status: &mess}
        sendMessage(&packet, addr.String(), gossiper.PeerConn)
        return
    }
    gossiper.SafeMessageHistory.mux.Unlock()
    fmt.Println("IN SYNC WITH", addr.String())
    if ask_was_received {
        for _, v := range flipmessages {
    	    flipCoinSend(gossiper, &v, addr.String())
        }
    }
}

func processPrivateMessage(gossiper *Gossiper, dec *GossipPacket, addr *net.UDPAddr) {
	if dec.Private.Destination == gossiper.Nodename {
		fmt.Println("PRIVATE origin", dec.Private.Origin, "hop-limit", dec.Private.HopLimit, "contents", dec.Private.Text)
        gossiper.SafePrivateMessageHistory.mux.Lock()
        gossiper.SafePrivateMessageHistory.History[dec.Private.Origin] = append(gossiper.SafePrivateMessageHistory.History[dec.Private.Origin], 
            dec.Private.Origin+": "+dec.Private.Text)
        gossiper.SafePrivateMessageHistory.mux.Unlock()
	} else {
		dec.Private.HopLimit -= 1
		if dec.Private.HopLimit > 0 {
            gossiper.RoutingTable.mux.Lock()
			sendMessage(dec, gossiper.RoutingTable.RouteTable[dec.Private.Destination], gossiper.PeerConn)
            gossiper.RoutingTable.mux.Unlock()
		}
	}
}

func processDataRequestMessage(gossiper *Gossiper, dec *GossipPacket, addr *net.UDPAddr) {
    if dec.DataRequest.Destination == gossiper.Nodename {
        hashvalue := [32]byte{}
        copy(hashvalue[:], dec.DataRequest.HashValue)
        gossiper.SafeFileIndex.mux.Lock()
        data, ok := gossiper.SafeFileIndex.FileIndex[hashvalue]
        gossiper.SafeFileIndex.mux.Unlock()
        if ok == true {
        	data_reply := DataReply{gossiper.Nodename, dec.DataRequest.Origin, 9, dec.DataRequest.HashValue, data}
	        packet := GossipPacket{DataReply: &data_reply}
	        gossiper.RoutingTable.mux.Lock()
	        sendMessage(&packet, gossiper.RoutingTable.RouteTable[dec.DataRequest.Origin], gossiper.PeerConn)
	        gossiper.RoutingTable.mux.Unlock()
	    }
    } else {
        dec.DataRequest.HopLimit -= 1
        if dec.DataRequest.HopLimit > 0 {
            gossiper.RoutingTable.mux.Lock()
            sendMessage(dec, gossiper.RoutingTable.RouteTable[dec.DataRequest.Destination], gossiper.PeerConn)
            gossiper.RoutingTable.mux.Unlock()
        }
    }
}

func processDataReplyMessage(gossiper *Gossiper, dec *GossipPacket, addr *net.UDPAddr) {
    if dec.DataReply.Destination == gossiper.Nodename {
    	hashvalue := [32]byte{}
        copy(hashvalue[:], dec.DataReply.HashValue)
        hash_of_data := sha256.Sum256(dec.DataReply.Data)
        if hashvalue == hash_of_data {
        	gossiper.SafeFileIndex.mux.Lock()
            _, ok := gossiper.SafeFileIndex.FileIndex[hashvalue]
            if ok == false {
	           gossiper.SafeFileIndex.FileIndex[hashvalue] = dec.DataReply.Data
            }
	        gossiper.SafeFileIndex.mux.Unlock()
        }
    } else {
        dec.DataReply.HopLimit -= 1
        if dec.DataReply.HopLimit > 0 {
            gossiper.RoutingTable.mux.Lock()
            sendMessage(dec, gossiper.RoutingTable.RouteTable[dec.DataReply.Destination], gossiper.PeerConn)
            gossiper.RoutingTable.mux.Unlock()
        }
    }
}

func processRumorPeer(gossiper *Gossiper) {
	for {
		dec, addr, err := listen(gossiper.PeerConn)
        if err == nil {
    		if dec.Rumor != nil || dec.Status != nil || dec.Private != nil || dec.DataRequest != nil || dec.DataReply != nil {
    			gossiper.SafePeersList.mux.Lock()
    	    	checkAndAppend(&gossiper.SafePeersList.PeersList, addr.String())
    	    	fmt.Println("PEERS", strings.Join(gossiper.SafePeersList.PeersList, ","))
    	    	gossiper.SafePeersList.mux.Unlock()
        	}
        	if dec.Rumor != nil {
        		processRumorMessage(gossiper, dec, addr)
        	}
        	if dec.Status != nil {
        		processStatusMessage(gossiper, dec, addr)
        	}
        	if dec.Private != nil {
				processPrivateMessage(gossiper, dec, addr)
        	}
            if dec.DataRequest != nil {
                processDataRequestMessage(gossiper, dec, addr)
            }
            if dec.DataReply != nil {
                processDataReplyMessage(gossiper, dec, addr)
            }
        }
	}
}

func antiEntropy(gossiper *Gossiper) {
	for {
		time.Sleep(time.Duration(timeout) * time.Second)
		mess := StatusPacket{}
        gossiper.SafeMessageHistory.mux.Lock()
		createStatus(&gossiper.SafeMessageHistory.History, &mess)
        gossiper.SafeMessageHistory.mux.Unlock()
    	packet := GossipPacket{Status: &mess}
		gossiper.SafePeersList.mux.Lock()
    	if len(gossiper.SafePeersList.PeersList) > 0 {
    		rndPeer := gossiper.SafePeersList.PeersList[rand.Intn(len(gossiper.SafePeersList.PeersList))]
    		sendMessage(&packet, rndPeer, gossiper.PeerConn)
    	}
    	gossiper.SafePeersList.mux.Unlock()
	}
}
