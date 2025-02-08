package handlers

import (
	"log"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sanda0/vps_pilot/internal/dto"
	"github.com/sanda0/vps_pilot/internal/services"
)

type NodeHandler interface {
	GetNodes(c *gin.Context)
	UpdateName(c *gin.Context)
	GetNode(c *gin.Context)
	SystemStatWSHandler(c *gin.Context)
}

type nodeHandler struct {
	nodeService services.NodeService
}

var systemStatUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// SystemStatWSHandler implements NodeHandler.
func (n *nodeHandler) SystemStatWSHandler(c *gin.Context) {
	conn, err := systemStatUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		log.Printf("recv: %s", message)
		err = conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Println(err)
			return
		}
	}
}

// GetNode implements NodeHandler.
func (n *nodeHandler) GetNode(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	node, err := n.nodeService.GetNode(int32(id))
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": dto.NodeDto{
		ID:   node.ID,
		Name: node.Name.String,
		Ip:   node.Ip,
	}})
}

// UpdateName implements NodeHandler.
func (n *nodeHandler) UpdateName(c *gin.Context) {
	form := dto.NodeNameUpdateDto{}
	if err := c.ShouldBindJSON(&form); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	err := n.nodeService.UpdateName(form.NodeId, form.Name)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": "Node name updated"})
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

	nodesRows, err := n.nodeService.GetNodesWithSysInfo(searchQuery, int32(limit), int32(page))

	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	nodes := []dto.NodeWithSysInfoDto{}
	for _, row := range nodesRows {
		node := dto.NodeWithSysInfoDto{}
		node.Convert(&row)
		nodes = append(nodes, node)
	}

	c.JSON(200, gin.H{"data": nodes})

}

func NewNodeHandler(nodeService services.NodeService) NodeHandler {
	return &nodeHandler{
		nodeService: nodeService,
	}
}
