package services

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/sanda0/vps_pilot/internal/agentauth"
	"github.com/sanda0/vps_pilot/internal/db"
	"github.com/sanda0/vps_pilot/internal/dto"
)

// NodeAgents is what the node service needs from the live agent connections
// (implemented by tcpserver.Registry).
type NodeAgents interface {
	// Kick closes a node's live connection.
	Kick(nodeID int64) bool
	// Online reports whether a node's agent is connected.
	Online(nodeID int64) bool
}

// AgentEnrollment is what the node service needs to issue join tokens.
type AgentEnrollment struct {
	Fingerprint string // hub agent-channel TLS cert fingerprint
	AgentPort   string // agent channel port, for the install command
	Version     string // hub version; the install command installs the same agent release
	Agents      NodeAgents
}

type NodeService interface {
	CreateJoinToken(userID int32, nodeID *int64, hubHost string) (*dto.JoinTokenDto, error)
	RevokeNodeKey(nodeID int64) error
	GetNodesWithSysInfo(search string, limit int32, page int32) ([]db.GetNodesWithSysInfoRow, error)
	UpdateName(nodeId int32, name string) error
	GetNode(nodeId int32) (db.GetNodeWithSysInfoRow, error)
	GetSystemStat(queryParams chan dto.NodeSystemStatRequestDto, result chan dto.SystemStatResponseDto)
	Status(nodeID int64) dto.NodeStatusDto
}

type nodeService struct {
	repo   *db.Repo
	ctx    context.Context
	enroll AgentEnrollment
}

// GetSystemStat implements NodeService.
func (n *nodeService) GetSystemStat(queryParams chan dto.NodeSystemStatRequestDto, result chan dto.SystemStatResponseDto) {
	for query := range queryParams {
		fmt.Println("Query received", query)
		node, err := n.repo.Queries.GetNodeWithSysInfo(n.ctx, int64(query.ID))
		if err != nil {
			fmt.Println("Error getting node", err)
			continue
		}

		// Convert timeRange from string to seconds (assuming it's a duration like "3600" for 1 hour)
		timeRangeSeconds := parseTimeRangeToSeconds(query.TimeRange)

		memStat, err := n.repo.TimeseriesQueries.GetSystemStats(n.ctx, db.GetSystemStatsParams{
			NodeID:   int64(int64(query.ID)),
			StatType: "mem",
			CpuID:    sql.NullInt64{Int64: 0, Valid: true},
			Column4:  timeRangeSeconds,
		})

		if err != nil {
			fmt.Println("Error getting mem stats", err)
			continue
		}

		cpuStats, err := n.repo.TimeseriesQueries.GetCPUStats(n.ctx, db.GetCPUStatsParams{
			NodeID:    query.ID,
			TimeRange: timeRangeSeconds,
			CpuCount:  int32(node.Cpus.Int64),
		})

		if err != nil {
			fmt.Println("Error getting cpu stats", err)
			continue
		}

		netStat, err := n.repo.TimeseriesQueries.GetNetStats(n.ctx, db.GetNetStatsParams{
			NodeID:  int64(int64(query.ID)),
			Column2: timeRangeSeconds,
		})

		if err != nil {
			fmt.Println("Error getting net stats", err)
			continue
		}

		diskStat, err := n.repo.TimeseriesQueries.GetDiskStats(n.ctx, db.GetDiskStatsParams{
			NodeID:  int64(query.ID),
			Column2: timeRangeSeconds,
		})

		if err != nil {
			fmt.Println("Error getting disk stats", err)
			continue
		}

		result <- dto.SystemStatResponseDto{
			NodeID:    query.ID,
			TimeRange: query.TimeRange,
			Cpu:       cpuStats,
			Mem:       memStat,
			Net:       netStat,
			Disk:      diskStat,
		}
	}

	fmt.Println("Query processing done")
}

// Status implements NodeService.
func (n *nodeService) Status(nodeID int64) dto.NodeStatusDto {
	status := dto.NodeStatusDto{}
	if n.enroll.Agents != nil {
		status.Online = n.enroll.Agents.Online(nodeID)
	}
	lastSeen, err := n.repo.TimeseriesQueries.GetNodeLastSeen(n.ctx, nodeID)
	if err != nil {
		fmt.Println("Error getting node last seen", err)
	}
	status.LastSeen = lastSeen
	return status
}

// GetNode implements NodeService.
func (n *nodeService) GetNode(nodeId int32) (db.GetNodeWithSysInfoRow, error) {
	node, err := n.repo.Queries.GetNodeWithSysInfo(n.ctx, int64(nodeId))
	if err != nil {
		return db.GetNodeWithSysInfoRow{}, err
	}
	return node, nil
}

// UpdateName implements NodeService.
func (n *nodeService) UpdateName(nodeId int32, name string) error {
	err := n.repo.Queries.UpdateNodeName(n.ctx, db.UpdateNodeNameParams{
		ID: int64(nodeId),
		Name: sql.NullString{
			String: name,
			Valid:  true,
		},
	})
	if err != nil {
		return err
	}
	return nil
}

// GetNodesWithSysInfo implements NodeService.
func (n *nodeService) GetNodesWithSysInfo(search string, limit int32, page int32) ([]db.GetNodesWithSysInfoRow, error) {
	offset := (page - 1) * limit
	nodes, err := n.repo.Queries.GetNodesWithSysInfo(n.ctx, db.GetNodesWithSysInfoParams{
		Column1: sql.NullString{
			String: search,
			Valid:  true,
		},
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	fmt.Println(nodes)
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

// CreateJoinToken implements NodeService. With nodeID set, the token
// re-enrolls that node (new agent install, rotated key) instead of adding one.
func (n *nodeService) CreateJoinToken(userID int32, nodeID *int64, hubHost string) (*dto.JoinTokenDto, error) {
	nodeRef := sql.NullInt64{}
	if nodeID != nil {
		if _, err := n.repo.Queries.GetNode(n.ctx, *nodeID); err != nil {
			return nil, fmt.Errorf("node %d not found", *nodeID)
		}
		nodeRef = sql.NullInt64{Int64: *nodeID, Valid: true}
	}

	if _, err := n.repo.Queries.DeleteExpiredJoinTokens(n.ctx); err != nil {
		fmt.Println("Error deleting expired join tokens", err)
	}

	token, secretHash, err := agentauth.NewJoinToken(n.enroll.Fingerprint)
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(agentauth.JoinTokenTTL).Unix()
	if _, err := n.repo.Queries.CreateJoinToken(n.ctx, db.CreateJoinTokenParams{
		TokenHash: secretHash,
		NodeID:    nodeRef,
		CreatedBy: int64(userID),
		ExpiresAt: expiresAt,
	}); err != nil {
		return nil, err
	}

	hub := net.JoinHostPort(hubHost, n.enroll.AgentPort)
	return &dto.JoinTokenDto{
		Token:          token,
		ExpiresAt:      expiresAt,
		Command:        fmt.Sprintf("VPS_PILOT_TOKEN=%s vps-pilot-agent --hub %s", token, hub),
		InstallCommand: installCommand(token, hub, n.enroll.Version),
	}, nil
}

// installCommand runs the website's install script, which sets the agent up
// as a systemd service. A release hub pins the agent to its own version; a
// dev build gets the latest release.
func installCommand(token, hub, version string) string {
	cmd := fmt.Sprintf("curl -fsSL https://vpspilot.dev/install-agent.sh | sudo VPS_PILOT_TOKEN=%s sh -s -- --hub %s", token, hub)
	if strings.HasPrefix(version, "v") {
		cmd += " --version " + version
	}
	return cmd
}

// RevokeNodeKey implements NodeService. The agent is disconnected and can
// only come back through a new join token.
func (n *nodeService) RevokeNodeKey(nodeID int64) error {
	rows, err := n.repo.Queries.ClearNodeKey(n.ctx, nodeID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("node %d not found", nodeID)
	}
	if n.enroll.Agents != nil {
		n.enroll.Agents.Kick(nodeID)
	}
	return nil
}

func NewNodeService(ctx context.Context, repo *db.Repo, enroll AgentEnrollment) NodeService {
	return &nodeService{
		repo:   repo,
		ctx:    ctx,
		enroll: enroll,
	}
}

// Helper function to convert a time range like "5 minutes" or "2 days" to seconds
func parseTimeRangeToSeconds(timeRange string) int64 {
	var n int64
	var unit string
	fmt.Sscanf(timeRange, "%d %s", &n, &unit)
	switch strings.TrimSuffix(unit, "s") {
	case "hour":
		return n * 3600
	case "day":
		return n * 86400
	case "week":
		return n * 7 * 86400
	default:
		return n * 60
	}
}
