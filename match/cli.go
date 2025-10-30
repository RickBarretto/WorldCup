package main

import (
	"flag"
	"fmt"
)

type CliArguments struct {
	address Address
	peers   []Address
}

func parseCli() CliArguments {
	var port string
	var rawPeers string

	flag.StringVar(&port, "port", "8081", "server listen port")
	flag.StringVar(&rawPeers, "peers", "", "comma-separated peer host:port list")
	flag.Parse()

	address := fmt.Sprintf("0.0.0.0:%s", port)
	peers := []Address{}

	if rawPeers != "" {
		peers = append(peers, listPeers(rawPeers)...)
	}
	return CliArguments{address, peers}
}


func listPeers(peers string) []Address {
	out := []Address{}
	cur := ""

	for _, ch := range peers {
		if ch == ',' {
			if cur != "" { out = append(out, cur) }
			cur = ""
			continue
		}
		cur += string(ch)
	}

	if cur != "" {
		out = append(out, cur)
	}

	return out
}