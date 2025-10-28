package main





func main() {
	cli := parseCli()
	StartServer(cli.address, cli.peers)
}

