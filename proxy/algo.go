package proxy

import "time"

type ProxyScoreAlgo interface {
	CalculateScore(proxy *Proxy, lastUsedAgoSeconds int64) int
}

type ProxyScoreAlgoNaive struct{}

func (p *ProxyScoreAlgoNaive) CalculateScore(proxy *Proxy, lastUsedAgoSeconds int64) int {
	now := time.Now().Unix()
	// Avoid division by zero
	successRate := 0.0
	if proxy.UpTimeTryCount > 0 {
		successRate = float64(proxy.UpTimeSuccessCount) / float64(proxy.UpTimeTryCount)
	}
	// Normalize values (tune divisors as needed for your data)
	latencyScore := 1.0 / (1.0 + proxy.Latency) // lower latency = higher score
	uptimeScore := proxy.UpTime / 100.0         // assuming UpTime is a percentage (0-100)
	successScore := successRate                 // already 0-1
	lastCheckedScore := 0.0
	if now > 0 {
		lastCheckedScore = float64(proxy.LastChecked) / float64(now) // more recent = closer to 1
	}
	responseTimeScore := 1.0 / (1.0 + float64(proxy.ResponseTime))
	speedScore := float64(proxy.Speed) / 10.0                // assuming speed is 0-10
	lastUsedScore := float64(lastUsedAgoSeconds) / (60 * 60) // hours since last used

	// Weights (tune as needed)
	score := 0.25*latencyScore +
		0.20*uptimeScore +
		0.20*successScore +
		0.10*lastCheckedScore +
		0.10*responseTimeScore +
		0.10*lastUsedScore +
		0.05*speedScore

	// Scale to int for heap
	return int(score * 1000)
}
