package proxy

import (
	"container/heap"
	"fmt"
	"time"
)

// ProxyElement is already defined in manager.go, but we need to add an Index field for heap.Fix
// If you want to keep the definition in one place, you can add Index there. For now, we redefine it here for clarity.
type ProxyElement struct {
	Proxy      *Proxy
	LastUsedAt time.Time
	Score      int
	Index      int // Index in the heap
}

func (p *ProxyElement) UpdateScore(algo ProxyScoreAlgo) error {
	now := time.Now().Unix()
	lastUsedAgo := now - p.LastUsedAt.Unix()
	p.Score = algo.CalculateScore(p.Proxy, lastUsedAgo)
	return nil
}

type ProxyHeap struct {
	elements []*ProxyElement
	algo     ProxyScoreAlgo
}

func NewProxyHeap(elements []*ProxyElement, algo ProxyScoreAlgo) *ProxyHeap {
	h := &ProxyHeap{elements: elements, algo: algo}
	heap.Init(h)
	return h
}

func (h ProxyHeap) Len() int { return len(h.elements) }

// Max-heap: higher Score is better
func (h ProxyHeap) Less(i, j int) bool {
	return h.elements[i].Score > h.elements[j].Score
}

func (h ProxyHeap) Swap(i, j int) {
	h.elements[i], h.elements[j] = h.elements[j], h.elements[i]
	h.elements[i].Index = i
	h.elements[j].Index = j
}

func (h *ProxyHeap) Push(x any) {
	n := len(h.elements)
	item := x.(*ProxyElement)
	item.Index = n
	h.elements = append(h.elements, item)
}
func (h *ProxyHeap) Pop() any {
	n := len(h.elements)
	if n == 0 {
		return nil
	}
	item := h.elements[n-1]
	h.elements = h.elements[:n-1]
	return item
}

// GetElements returns the top n elements from the heap, in order, without removing them.
func (h *ProxyHeap) GetElements(n uint16) []*ProxyElement {
	if n > uint16(len(h.elements)) {
		n = uint16(len(h.elements))
	}
	picked := h.elements[:n]
	go h.reorderElements(picked)

	return picked
}

func (h *ProxyHeap) reorderElements(elements []*ProxyElement) {
	for _, elem := range elements {
		if err := elem.UpdateScore(h.algo); err != nil {
			fmt.Println("failed to update score: %w", err)
			continue
		}
		heap.Fix(h, elem.Index)
	}
}
