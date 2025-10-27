package main

import "flag"
import "log"
import "strconv"
import "strings"

func NodeFromCLI() (Address, *Node) {
	idFlag := flag.Int("id", 1, "numeric id for this node")
	addressFlag := flag.String("addr", "http://localhost:8001", "public address for this node, used by peers (include scheme and port)")
	peersFlag := flag.String("peers", "", "comma-separated list of peers as id=addr,id=addr")
	flag.Parse()

	peers := make(Peers)
	if *peersFlag != "" {
		items := strings.Split(*peersFlag, ",")
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			parts := strings.SplitN(item, "=", 2)
			if len(parts) != 2 {
				log.Fatalf("bad peer entry: %s", item)
			}
			pid, err := strconv.Atoi(parts[0])
			if err != nil {
				log.Fatalf("bad peer id: %s", parts[0])
			}
			peers[pid] = parts[1]
		}
	}

	peers[*idFlag] = *addressFlag

	node := NewNode(*idFlag, *addressFlag, peers)
	normalizedAddress := normalizeAddress(addressFlag)
	return normalizedAddress, node
}

/// server expects addr like http://host:port; extract host:port for Listen
func normalizeAddress(addressFlag *string) Address {
	listenAddr := ""
	if strings.HasPrefix(*addressFlag, "http://") {
		listenAddr = strings.TrimPrefix(*addressFlag, "http://")
	} else if strings.HasPrefix(*addressFlag, "https://") {
		listenAddr = strings.TrimPrefix(*addressFlag, "https://")
	} else {
		listenAddr = *addressFlag
	}
	return listenAddr
}