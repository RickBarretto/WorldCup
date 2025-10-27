package main

func main() {
	address, node := NodeFromCLI()

	node.StartLeaderLoop()
	node.AddRoutes()
	node.Serve(address)
}
