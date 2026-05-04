package state

import (
	"sort"
	"sync"
	"time"

	"netweaver-backend/pkg/protocol"
)

type Registry struct {
	mu    sync.RWMutex
	nodes map[string]protocol.NodeInfo
}

func NewRegistry() *Registry {
	return &Registry{
		nodes: make(map[string]protocol.NodeInfo),
	}
}

func (r *Registry) Upsert(node protocol.NodeInfo) protocol.NodeInfo {
	r.mu.Lock()
	defer r.mu.Unlock()

	if node.LastSeen.IsZero() {
		node.LastSeen = time.Now()
	}
	r.nodes[node.ID] = node
	return node
}

func (r *Registry) List() []protocol.NodeInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	nodes := make([]protocol.NodeInfo, 0, len(r.nodes))
	for _, node := range r.nodes {
		nodes = append(nodes, node)
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})
	return nodes
}

func (r *Registry) Snapshot() map[string]protocol.NodeInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	snapshot := make(map[string]protocol.NodeInfo, len(r.nodes))
	for id, node := range r.nodes {
		snapshot[id] = node
	}
	return snapshot
}
