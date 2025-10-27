package main

import "github.com/gin-gonic/gin"


/// Add Routes to a Node
func (node *Node) AddRoutes(router *gin.Engine) {
	// Domain endpoints
	router.GET("/cards", gin.WrapF(node.handleGetCards))
	router.POST("/cards", gin.WrapF(node.handlePostCard))
	router.DELETE("/cards/:id", gin.WrapF(node.handleDeleteCard))

	// Peer endpoints
	router.GET("/status", gin.WrapF(node.handleStatus))

	// Private endpoints
	router.POST("/replicate", gin.WrapF(node.handleReplicate))
}
