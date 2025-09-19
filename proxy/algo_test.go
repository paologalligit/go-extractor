package proxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProxyScoreAlgoNaive_SpeedAffectsScore(t *testing.T) {
	algo := &ProxyScoreAlgoNaive{}
	p1 := &Proxy{Speed: 10, Latency: 100}
	p2 := &Proxy{Speed: 1, Latency: 100}
	lastUsed := int64(3600)
	s1 := algo.CalculateScore(p1, lastUsed)
	s2 := algo.CalculateScore(p2, lastUsed)
	assert.Greater(t, s1, s2, "higher speed should yield higher score")
}

func TestProxyScoreAlgoNaive_LatencyAffectsScore(t *testing.T) {
	algo := &ProxyScoreAlgoNaive{}
	p1 := &Proxy{Speed: 5, Latency: 10}
	p2 := &Proxy{Speed: 5, Latency: 200}
	lastUsed := int64(3600)
	s1 := algo.CalculateScore(p1, lastUsed)
	s2 := algo.CalculateScore(p2, lastUsed)
	assert.Greater(t, s1, s2, "lower latency should yield higher score")
}

func TestProxyScoreAlgoNaive_LastUsedAffectsScore(t *testing.T) {
	algo := &ProxyScoreAlgoNaive{}
	p := &Proxy{Speed: 5, Latency: 100}
	old := algo.CalculateScore(p, 3600)
	recent := algo.CalculateScore(p, 10)
	assert.Greater(t, old, recent, "older last used should yield higher score")
}

func TestProxyScoreAlgoNaive_CombinedFactors(t *testing.T) {
	algo := &ProxyScoreAlgoNaive{}
	p1 := &Proxy{Speed: 10, Latency: 10}
	p2 := &Proxy{Speed: 1, Latency: 200}
	lastUsed := int64(3600)
	s1 := algo.CalculateScore(p1, lastUsed)
	s2 := algo.CalculateScore(p2, lastUsed)
	assert.Greater(t, s1, s2, "all factors combined should yield higher score for better proxy")
}
