package swarm

import (
	"swarmsim/engine/swarmscript"
)

// ASTNodeViz represents a node in the AST visualization tree.
type ASTNodeViz struct {
	Label    string
	Type     string // "rule", "condition", "action", "operator", "value"
	X, Y     float64
	Width    float64
	Height   float64
	Children []*ASTNodeViz
	Depth    int
	Active   bool // true if this node was last matched
}

// ASTLayout holds the computed layout for an AST visualization.
type ASTLayout struct {
	Root       *ASTNodeViz
	TotalWidth float64
	TotalHeight float64
	NodeCount  int
	MaxDepth   int
}

const (
	astNodeW     = 120.0
	astNodeH     = 30.0
	astNodeGapX  = 20.0
	astNodeGapY  = 50.0
)

// BuildASTLayout converts a SwarmProgram into a visual AST tree.
func BuildASTLayout(prog *swarmscript.SwarmProgram) *ASTLayout {
	if prog == nil || len(prog.Rules) == 0 {
		return nil
	}

	layout := &ASTLayout{}
	root := &ASTNodeViz{
		Label: "PROGRAMM",
		Type:  "rule",
		Depth: 0,
	}

	for i, rule := range prog.Rules {
		ruleNode := buildRuleNode(&rule, i, 1)
		root.Children = append(root.Children, ruleNode)
	}

	layout.Root = root
	layout.NodeCount = astCountNodes(root)
	layout.MaxDepth = astMaxDepth(root)

	// Compute positions
	astLayoutPositions(root, 0, 0)
	layout.TotalWidth = astSubtreeWidth(root)
	layout.TotalHeight = float64(layout.MaxDepth+1) * (astNodeH + astNodeGapY)

	return layout
}

func buildRuleNode(rule *swarmscript.Rule, idx int, depth int) *ASTNodeViz {
	node := &ASTNodeViz{
		Label: ruleLabel(idx),
		Type:  "rule",
		Depth: depth,
	}

	// Conditions
	for _, cond := range rule.Conditions {
		condNode := &ASTNodeViz{
			Label: conditionLabel(cond),
			Type:  "condition",
			Depth: depth + 1,
		}
		node.Children = append(node.Children, condNode)
	}

	// Action
	actionNode := &ASTNodeViz{
		Label: actionLabel(rule.Action),
		Type:  "action",
		Depth: depth + 1,
	}
	node.Children = append(node.Children, actionNode)

	return node
}

func ruleLabel(idx int) string {
	return "Regel " + itoa(idx+1)
}

func conditionLabel(cond swarmscript.Condition) string {
	sensor := swarmscript.ConditionTypeName(cond.Type)
	opStr := "?"
	switch cond.Op {
	case swarmscript.OpGT:
		opStr = ">"
	case swarmscript.OpLT:
		opStr = "<"
	case swarmscript.OpEQ:
		opStr = "=="
	}
	return sensor + " " + opStr + " " + itoa(cond.Value)
}

func actionLabel(action swarmscript.Action) string {
	name := swarmscript.ActionTypeName(action.Type)
	if action.Param1 > 0 {
		return name + "(" + itoa(action.Param1) + ")"
	}
	return name
}

// itoa simple int to string without fmt dependency.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	buf := make([]byte, 0, 10)
	for n > 0 {
		buf = append(buf, byte('0'+n%10))
		n /= 10
	}
	if neg {
		buf = append(buf, '-')
	}
	// reverse
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}

// astLayoutPositions assigns X,Y coordinates to each node.
func astLayoutPositions(node *ASTNodeViz, startX, startY float64) float64 {
	if node == nil {
		return startX
	}

	node.Y = startY
	node.Width = astNodeW
	node.Height = astNodeH

	if len(node.Children) == 0 {
		node.X = startX
		return startX + astNodeW + astNodeGapX
	}

	// Layout children first
	childX := startX
	childY := startY + astNodeH + astNodeGapY
	for _, child := range node.Children {
		childX = astLayoutPositions(child, childX, childY)
	}

	// Center parent over children
	firstChild := node.Children[0]
	lastChild := node.Children[len(node.Children)-1]
	node.X = (firstChild.X + lastChild.X + lastChild.Width) / 2 - node.Width/2

	return childX
}

// astSubtreeWidth returns the total width of a subtree.
func astSubtreeWidth(node *ASTNodeViz) float64 {
	if node == nil {
		return 0
	}
	if len(node.Children) == 0 {
		return node.Width
	}
	w := 0.0
	for _, child := range node.Children {
		w += astSubtreeWidth(child) + astNodeGapX
	}
	return w - astNodeGapX // remove trailing gap
}

// astCountNodes counts all nodes in the tree.
func astCountNodes(node *ASTNodeViz) int {
	if node == nil {
		return 0
	}
	count := 1
	for _, child := range node.Children {
		count += astCountNodes(child)
	}
	return count
}

// astMaxDepth returns the maximum depth of the tree.
func astMaxDepth(node *ASTNodeViz) int {
	if node == nil {
		return -1
	}
	maxD := node.Depth
	for _, child := range node.Children {
		d := astMaxDepth(child)
		if d > maxD {
			maxD = d
		}
	}
	return maxD
}

// MarkActiveRules marks which AST nodes correspond to active rules.
func MarkActiveRules(layout *ASTLayout, matchedRules []int) {
	if layout == nil || layout.Root == nil {
		return
	}
	// Reset all
	astClearActive(layout.Root)
	// Mark matched
	for _, ruleIdx := range matchedRules {
		if ruleIdx >= 0 && ruleIdx < len(layout.Root.Children) {
			astSetActive(layout.Root.Children[ruleIdx])
		}
	}
}

func astClearActive(node *ASTNodeViz) {
	if node == nil {
		return
	}
	node.Active = false
	for _, child := range node.Children {
		astClearActive(child)
	}
}

func astSetActive(node *ASTNodeViz) {
	if node == nil {
		return
	}
	node.Active = true
	for _, child := range node.Children {
		astSetActive(child)
	}
}

// CollectASTNodes returns all nodes as a flat list (for rendering).
func CollectASTNodes(layout *ASTLayout) []*ASTNodeViz {
	if layout == nil || layout.Root == nil {
		return nil
	}
	var nodes []*ASTNodeViz
	astCollectFlat(layout.Root, &nodes)
	return nodes
}

func astCollectFlat(node *ASTNodeViz, out *[]*ASTNodeViz) {
	if node == nil {
		return
	}
	*out = append(*out, node)
	for _, child := range node.Children {
		astCollectFlat(child, out)
	}
}
