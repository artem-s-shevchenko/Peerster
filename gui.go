package main

import (
	"net"
	"net/http"
	"github.com/gorilla/mux"
	"log"
	"fmt"
	"encoding/json"
	"io/ioutil"
)

type HandlerData struct {
	gossiper *Gossiper
}

func (handler_data *HandlerData) messageHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
        fmt.Println(err)
    }
	mess := MessageMessage{}
	json.Unmarshal(body, &mess)
	processRumorClient(handler_data.gossiper, mess.Message)
	w.WriteHeader(http.StatusOK)
}

func (handler_data *HandlerData) privateMessageHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
        fmt.Println(err)
    }
	mess := PrivateMessage{}
	json.Unmarshal(body, &mess)
	mess.Origin = handler_data.gossiper.Nodename
	mess.HopLimit -= 1
	packet := GossipPacket{Private: &mess}
	handler_data.gossiper.SafePrivateMessageHistory.mux.Lock()
    handler_data.gossiper.SafePrivateMessageHistory.History[mess.Destination] = append(handler_data.gossiper.SafePrivateMessageHistory.History[mess.Destination], 
        mess.Origin+": "+mess.Text)
    handler_data.gossiper.SafePrivateMessageHistory.mux.Unlock()
    handler_data.gossiper.RoutingTable.mux.Lock()
   	sendMessage(&packet, handler_data.gossiper.RoutingTable.RouteTable[mess.Destination], handler_data.gossiper.PeerConn)
    handler_data.gossiper.RoutingTable.mux.Unlock()
	w.WriteHeader(http.StatusOK)
}

func (handler_data *HandlerData) nodeHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
        fmt.Println(err)
    }
	ip := IPMessage{}
	json.Unmarshal(body, &ip)
	_, err = net.ResolveUDPAddr("udp4", ip.IP)
	if err == nil {
		handler_data.gossiper.SafePeersList.mux.Lock()
		checkAndAppend(&handler_data.gossiper.SafePeersList.PeersList, ip.IP)
		handler_data.gossiper.SafePeersList.mux.Unlock()
	}
	w.WriteHeader(http.StatusOK)
}

func (handler_data *HandlerData) getUpdateHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	handler_data.gossiper.SafeUpdateMessageList.mux.Lock()
	handler_data.gossiper.SafePeersList.mux.Lock()
	update := Update{handler_data.gossiper.SafeUpdateMessageList.List, handler_data.gossiper.SafePeersList.PeersList, handler_data.gossiper.Nodename}
	jsonUpdate, err := json.Marshal(update)
	if err != nil {
        fmt.Println(err)
    }
    handler_data.gossiper.SafeUpdateMessageList.mux.Unlock()
	handler_data.gossiper.SafePeersList.mux.Unlock()
	w.Write(jsonUpdate)
}

func (handler_data *HandlerData) getPrivateUpdateHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	dest := r.URL.Query()["node"][0]
	nodelist := []string{}
	handler_data.gossiper.RoutingTable.mux.Lock()
	for k, _ := range handler_data.gossiper.RoutingTable.RouteTable {
		nodelist = append(nodelist, k)
	}
    handler_data.gossiper.RoutingTable.mux.Unlock()
	update := Update{Peerlist: nodelist}
	handler_data.gossiper.SafePrivateMessageHistory.mux.Lock()
	if dest != "" && len(handler_data.gossiper.SafePrivateMessageHistory.History[dest]) > 0 {
		update.Newmessages = handler_data.gossiper.SafePrivateMessageHistory.History[dest]
	} else {
		update.Newmessages = []string{}
	}
	jsonUpdate, err := json.Marshal(update)
	if err != nil {
        fmt.Println(err)
    }
    handler_data.gossiper.SafePrivateMessageHistory.mux.Unlock()
	w.Write(jsonUpdate)
}

func (handler_data *HandlerData) fileSharingHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
        fmt.Println(err)
    }
    mess := DataRequest{}
    json.Unmarshal(body, &mess)
    if mess.Destination == "" {
		go index_file(handler_data.gossiper, mess.Origin)
	} else {
		go download_file(handler_data.gossiper, mess.Destination, mess.HashValue, mess.Origin)
	}
	w.WriteHeader(http.StatusOK)
}

func (handler_data *HandlerData) getAvailableRoutes(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	nodelist := []string{}
	handler_data.gossiper.RoutingTable.mux.Lock()
	for k, _ := range handler_data.gossiper.RoutingTable.RouteTable {
		nodelist = append(nodelist, k)
	}
    handler_data.gossiper.RoutingTable.mux.Unlock()
	update := Update{Peerlist: nodelist}
	jsonUpdate, err := json.Marshal(update)
	if err != nil {
        fmt.Println(err)
    }
	w.Write(jsonUpdate)
}

func runGui(goss *Gossiper, guiport string) {
	handler_data := &HandlerData{goss}
	r := mux.NewRouter()
	r.HandleFunc("/message", handler_data.messageHandler).Methods("POST")
	r.HandleFunc("/privatemessage", handler_data.privateMessageHandler).Methods("POST")
	r.HandleFunc("/node", handler_data.nodeHandler).Methods("POST")
	r.HandleFunc("/getupdate", handler_data.getUpdateHandler).Methods("GET")
	r.HandleFunc("/getprivateupdate", handler_data.getPrivateUpdateHandler).Methods("GET")
	r.HandleFunc("/filesharing", handler_data.fileSharingHandler).Methods("POST")
	r.HandleFunc("/availableroutes", handler_data.getAvailableRoutes).Methods("GET")
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("./static"))))
	log.Fatal(http.ListenAndServe("127.0.0.1:"+guiport,r))
}
