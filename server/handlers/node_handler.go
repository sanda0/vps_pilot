package handlers

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sanda0/vps_pilot/services"
)

type NodeHandler interface {
	GetNodes(c *gin.Context)
}

type nodeHandler struct {
	nodeService services.NodeService
}

// GetNodes implements NodeHandler.
func (n *nodeHandler) GetNodes(c *gin.Context) {
	searchQuery := c.Query("search")
	pageQuery := c.Query("page")
	limitQuery := c.Query("limit")

	page, err := strconv.Atoi(pageQuery)
	if err != nil {
		page = 1 // default value
	}

	limit, err := strconv.Atoi(limitQuery)
	if err != nil {
		limit = 10 // default value
	}

	nodes, err := n.nodeService.GetNodesWithSysInfo(searchQuery, int32(limit), int32(page))
	fmt.Println(nodes)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"data": nodes})

}

func NewNodeHandler(nodeService services.NodeService) NodeHandler {
	return &nodeHandler{
		nodeService: nodeService,
	}
}
