package swarm

import "math"

// BehaviorCluster represents a group of bots with similar behavior.
type BehaviorCluster struct {
	Centroid BehaviorDescriptor
	Members  []int     // bot indices
	AvgFit   float64   // average fitness of members
	Label    string    // auto-assigned role label
}

// ClusterResult holds the output of k-means behavior clustering.
type ClusterResult struct {
	Clusters []BehaviorCluster
	K        int
}

// ComputeBehaviorClusters runs k-means on bot behavior descriptors.
// k is the number of clusters (typically 3-5).
// Returns nil if not enough data.
func ComputeBehaviorClusters(ss *SwarmState, k int) *ClusterResult {
	n := len(ss.Bots)
	if n < k || k < 2 {
		return nil
	}

	// Collect behavior descriptors
	behaviors := make([]BehaviorDescriptor, n)
	for i := range ss.Bots {
		behaviors[i] = ComputeBehavior(&ss.Bots[i], ss)
	}

	// Initialize centroids from first k bots (spread evenly)
	centroids := make([]BehaviorDescriptor, k)
	for c := 0; c < k; c++ {
		idx := c * n / k
		centroids[c] = behaviors[idx]
	}

	// Run k-means for up to 20 iterations
	assignments := make([]int, n)
	for iter := 0; iter < 20; iter++ {
		changed := false

		// Assign each bot to nearest centroid
		for i := 0; i < n; i++ {
			bestC := 0
			bestDist := behaviorDistSq(behaviors[i], centroids[0])
			for c := 1; c < k; c++ {
				d := behaviorDistSq(behaviors[i], centroids[c])
				if d < bestDist {
					bestDist = d
					bestC = c
				}
			}
			if assignments[i] != bestC {
				assignments[i] = bestC
				changed = true
			}
		}

		if !changed {
			break
		}

		// Recompute centroids
		counts := make([]int, k)
		sums := make([]BehaviorDescriptor, k)
		for i := 0; i < n; i++ {
			c := assignments[i]
			counts[c]++
			for d := 0; d < BehaviorDims; d++ {
				sums[c][d] += behaviors[i][d]
			}
		}
		for c := 0; c < k; c++ {
			if counts[c] > 0 {
				for d := 0; d < BehaviorDims; d++ {
					centroids[c][d] = sums[c][d] / float64(counts[c])
				}
			}
		}
	}

	// Build result
	clusters := make([]BehaviorCluster, k)
	for c := 0; c < k; c++ {
		clusters[c].Centroid = centroids[c]
	}
	for i := 0; i < n; i++ {
		c := assignments[i]
		clusters[c].Members = append(clusters[c].Members, i)
	}

	// Compute average fitness per cluster and assign labels
	for c := 0; c < k; c++ {
		if len(clusters[c].Members) == 0 {
			clusters[c].Label = "Leer"
			continue
		}
		sumFit := 0.0
		for _, idx := range clusters[c].Members {
			sumFit += EvaluateGPFitness(&ss.Bots[idx])
		}
		clusters[c].AvgFit = sumFit / float64(len(clusters[c].Members))
		clusters[c].Label = classifyCluster(clusters[c].Centroid)
	}

	return &ClusterResult{Clusters: clusters, K: k}
}

// behaviorDistSq returns squared Euclidean distance between two behavior descriptors.
func behaviorDistSq(a, b BehaviorDescriptor) float64 {
	sum := 0.0
	for d := 0; d < BehaviorDims; d++ {
		diff := a[d] - b[d]
		sum += diff * diff
	}
	return sum
}

// classifyCluster assigns a human-readable role label based on centroid values.
func classifyCluster(c BehaviorDescriptor) string {
	// Behavior dims: 0=finalX, 1=finalY, 2=totalDist, 3=deliveries, 4=pickups,
	// 5=%carrying, 6=%idle, 7=avgNeighbors
	deliveries := c[3]
	idle := c[6]
	dist := c[2]
	neighbors := c[7]

	if deliveries > 0.5 {
		return "Lieferant"
	}
	if idle > 0.4 {
		return "Wachposten"
	}
	if dist > 0.6 && neighbors < 0.3 {
		return "Erkunder"
	}
	if neighbors > 0.5 {
		return "Schwarm"
	}
	if math.Abs(c[0]-0.5) < 0.2 && math.Abs(c[1]-0.5) < 0.2 {
		return "Zentrist"
	}
	return "Generalist"
}
