package swarm

import "math/rand"

// BTStatus is the result of a behavior tree node tick.
type BTStatus int

const (
	BTSuccess BTStatus = iota
	BTFailure
	BTRunning
)

// BTNodeType identifies the kind of BT node.
type BTNodeType int

const (
	BTSequence BTNodeType = iota // runs children left-to-right, fails on first failure
	BTSelector                   // runs children left-to-right, succeeds on first success
	BTCondition                  // leaf: checks a sensor condition
	BTAction                     // leaf: performs an action
	BTInverter                   // decorator: inverts child result
	BTRepeater                   // decorator: repeats child N times
)

// BTNode is a single node in a behavior tree.
type BTNode struct {
	Type      BTNodeType
	Children  []*BTNode
	CondFunc  func(bot *SwarmBot, ss *SwarmState) bool // for BTCondition
	ActFunc   func(bot *SwarmBot, ss *SwarmState)      // for BTAction
	Label     string                                    // human-readable name
	RepeatN   int                                       // for BTRepeater
	repeatCur int                                       // current repeat count
}

// BTBrain holds a behavior tree for a bot.
type BTBrain struct {
	Root     *BTNode
	ActionID int // last action selected (for compatibility with neuro actions)
}

// BTTick evaluates the behavior tree for one tick.
func BTTick(node *BTNode, bot *SwarmBot, ss *SwarmState) BTStatus {
	if node == nil {
		return BTFailure
	}

	switch node.Type {
	case BTSequence:
		for _, child := range node.Children {
			status := BTTick(child, bot, ss)
			if status != BTSuccess {
				return status
			}
		}
		return BTSuccess

	case BTSelector:
		for _, child := range node.Children {
			status := BTTick(child, bot, ss)
			if status != BTFailure {
				return status
			}
		}
		return BTFailure

	case BTCondition:
		if node.CondFunc != nil && node.CondFunc(bot, ss) {
			return BTSuccess
		}
		return BTFailure

	case BTAction:
		if node.ActFunc != nil {
			node.ActFunc(bot, ss)
		}
		return BTSuccess

	case BTInverter:
		if len(node.Children) == 0 {
			return BTFailure
		}
		status := BTTick(node.Children[0], bot, ss)
		if status == BTSuccess {
			return BTFailure
		}
		if status == BTFailure {
			return BTSuccess
		}
		return BTRunning

	case BTRepeater:
		if len(node.Children) == 0 {
			return BTFailure
		}
		for i := 0; i < node.RepeatN; i++ {
			status := BTTick(node.Children[0], bot, ss)
			if status == BTFailure {
				return BTFailure
			}
		}
		return BTSuccess
	}

	return BTFailure
}

// BTNodeCount returns total number of nodes in the tree.
func BTNodeCount(node *BTNode) int {
	if node == nil {
		return 0
	}
	count := 1
	for _, child := range node.Children {
		count += BTNodeCount(child)
	}
	return count
}

// BTDepth returns the maximum depth of the tree.
func BTDepth(node *BTNode) int {
	if node == nil {
		return 0
	}
	maxChildDepth := 0
	for _, child := range node.Children {
		d := BTDepth(child)
		if d > maxChildDepth {
			maxChildDepth = d
		}
	}
	return 1 + maxChildDepth
}

// BTMutate applies random structural mutations to a behavior tree.
func BTMutate(rng *rand.Rand, node *BTNode, mutRate float64) {
	if node == nil {
		return
	}

	// Chance to swap node type (for composites only)
	if rng.Float64() < mutRate {
		if node.Type == BTSequence {
			node.Type = BTSelector
		} else if node.Type == BTSelector {
			node.Type = BTSequence
		}
	}

	// Chance to swap children order
	if len(node.Children) > 1 && rng.Float64() < mutRate {
		i := rng.Intn(len(node.Children))
		j := rng.Intn(len(node.Children))
		node.Children[i], node.Children[j] = node.Children[j], node.Children[i]
	}

	// Recurse
	for _, child := range node.Children {
		BTMutate(rng, child, mutRate)
	}
}

// BTCrossover creates a child tree by combining subtrees from two parents.
func BTCrossover(rng *rand.Rand, a, b *BTNode) *BTNode {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}

	// Deep copy of a
	child := btDeepCopy(a)

	// Replace a random subtree with one from b
	aNodes := btCollectNodes(child)
	bNodes := btCollectNodes(b)
	if len(aNodes) > 1 && len(bNodes) > 0 {
		targetIdx := 1 + rng.Intn(len(aNodes)-1) // skip root
		sourceIdx := rng.Intn(len(bNodes))
		target := aNodes[targetIdx]
		source := bNodes[sourceIdx]

		// Replace target's content with source's
		target.Type = source.Type
		target.Label = source.Label
		target.CondFunc = source.CondFunc
		target.ActFunc = source.ActFunc
		target.RepeatN = source.RepeatN
		if len(source.Children) > 0 {
			target.Children = make([]*BTNode, len(source.Children))
			for i, sc := range source.Children {
				target.Children[i] = btDeepCopy(sc)
			}
		} else {
			target.Children = nil
		}
	}

	return child
}

func btDeepCopy(node *BTNode) *BTNode {
	if node == nil {
		return nil
	}
	cp := &BTNode{
		Type:     node.Type,
		Label:    node.Label,
		CondFunc: node.CondFunc,
		ActFunc:  node.ActFunc,
		RepeatN:  node.RepeatN,
	}
	for _, child := range node.Children {
		cp.Children = append(cp.Children, btDeepCopy(child))
	}
	return cp
}

func btCollectNodes(node *BTNode) []*BTNode {
	if node == nil {
		return nil
	}
	result := []*BTNode{node}
	for _, child := range node.Children {
		result = append(result, btCollectNodes(child)...)
	}
	return result
}
