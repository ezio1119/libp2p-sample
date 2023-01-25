package main

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
)

const (
	pingProtocol     = protocol.ID("/ping/1.0.0")
	chatProtocol     = protocol.ID("/chat/1.0.0")
	discoverySVCName = "ping"
)

func readData(rw *bufio.ReadWriter) {
	for {
		str, err := rw.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from buffer")
			panic(err)
		}

		if str == "" {
			return
		}
		if str != "\n" {
			// Green console colour: 	\x1b[32m
			// Reset console colour: 	\x1b[0m
			fmt.Printf("\x1b[32m%s\x1b[0m> ", str)
		}

	}
}

func writeData(rw *bufio.ReadWriter) {
	stdReader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		sendData, err := stdReader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from stdin")
			panic(err)
		}

		_, err = rw.WriteString(fmt.Sprintf("%s\n", sendData))
		if err != nil {
			fmt.Println("Error writing to buffer")
			panic(err)
		}
		err = rw.Flush()
		if err != nil {
			fmt.Println("Error flushing buffer")
			panic(err)
		}
	}
}

func handleStream(stream network.Stream) {
	fmt.Println("Got a new stream!")

	// Create a buffer stream for non blocking read and write.
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	go readData(rw)
	go writeData(rw)

}

type discoveryNotifee struct {
	PeerChan chan peer.AddrInfo
}

func (n *discoveryNotifee) HandlePeerFound(peerInfo peer.AddrInfo) {
	n.PeerChan <- peerInfo
}

func main() {
	ctx := context.Background()

	host, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	if err != nil {
		panic(err)
	}
	defer host.Close()

	fmt.Printf("Addresses: %s: ID: %s\n ", host.Addrs(), host.ID())

	pingSVC := ping.NewPingService(host)
	host.SetStreamHandler(pingProtocol, pingSVC.PingHandler)
	host.SetStreamHandler(chatProtocol, handleStream)

	peerChan := make(chan peer.AddrInfo)
	notifee := &discoveryNotifee{PeerChan: peerChan}

	discoverySVC := mdns.NewMdnsService(
		host,
		discoverySVCName,
		notifee,
	)
	defer discoverySVC.Close()

	if err := discoverySVC.Start(); err != nil {
		panic(err)
	}

	for {
		peer := <-peerChan
		fmt.Printf("Found peer: %s: connecting...\n", peer)

		if err := host.Connect(ctx, peer); err != nil {
			fmt.Println("Connection failed:", err)
			continue
		}

		ch := pingSVC.Ping(ctx, peer.ID)
		res := <-ch
		fmt.Printf("Pinged %s in %s", peer.Addrs[1], res.RTT)

		stream, err := host.NewStream(ctx, peer.ID, chatProtocol)
		if err != nil {
			fmt.Println("Connection failed:", err)
			continue
		}

		rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

		go readData(rw)
		go writeData(rw)
	}
}
