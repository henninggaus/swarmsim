package swarmscript

// --- AST types for SwarmScript ---

// ConditionOp is a comparison operator.
type ConditionOp int

const (
	OpGT ConditionOp = iota // >
	OpLT                    // <
	OpEQ                    // ==
)

// ConditionType identifies which sensor a condition checks.
type ConditionType int

const (
	CondNeighborsCount      ConditionType = iota // neighbors_count
	CondNearestDistance                          // nearest_distance
	CondState                                    // state
	CondCounter                                  // counter
	CondTimer                                    // timer
	CondOnEdge                                   // on_edge
	CondReceivedMessage                          // received_message
	CondLightValue                               // light_value
	CondRandom                                   // random (percentage chance)
	CondTrue                                     // always true
	CondHasLeader                                // has_leader
	CondHasFollower                              // has_follower
	CondChainLength                              // chain_length
	CondNearestLEDR                              // nearest_led_r
	CondNearestLEDG                              // nearest_led_g
	CondNearestLEDB                              // nearest_led_b
	CondTick                                     // tick
	CondMyState                                  // my_state (alias for state)
	CondObstacleAhead                            // obstacle_ahead
	CondObstacleDist                             // obstacle_distance
	CondValue1                                   // value1
	CondValue2                                   // value2
	CondCarrying                                 // carrying (true/false)
	CondCarryingColor                            // carrying_color (0-4)
	CondNearestPickupDist                        // nearest_pickup_dist
	CondNearestPickupColor                       // nearest_pickup_color
	CondNearestPickupHasPkg                      // nearest_pickup_has_package
	CondNearestDropoffDist                       // nearest_dropoff_dist
	CondNearestDropoffColor                      // nearest_dropoff_color
	CondDropoffMatch                             // dropoff_match
	CondHeardPickupColor                         // heard_pickup_color
	CondHeardDropoffColor                        // heard_dropoff_color
	CondNearestMatchLEDDist                      // nearest_matching_led_dist
	CondTruckHere                                // truck_here
	CondTruckPkgCount                            // truck_pkg_count
	CondOnRamp                                   // on_ramp
	CondNearestTruckPkgDist                      // nearest_truck_pkg
	CondHeardBeaconDropoff                       // heard_beacon (1 if heard, 0 if not)
	CondHeardBeaconDropoffDist                   // beacon_dist
	CondExploring                                // exploring (1 if lost carrier for >60 ticks)
)

// Condition represents a single boolean check in a rule.
type Condition struct {
	Type  ConditionType
	Op    ConditionOp
	Value int
}

// ActionType identifies what action a rule performs.
type ActionType int

const (
	ActMoveForward           ActionType = iota // MOVE_FORWARD
	ActTurnLeft                                // TURN_LEFT N
	ActTurnRight                               // TURN_RIGHT N
	ActTurnToNearest                           // TURN_TO_NEAREST
	ActTurnFromNearest                         // TURN_FROM_NEAREST
	ActTurnToCenter                            // TURN_TO_CENTER
	ActTurnToLight                             // TURN_TO_LIGHT
	ActTurnRandom                              // TURN_RANDOM
	ActStop                                    // STOP
	ActSetState                                // SET_STATE N
	ActSetCounter                              // SET_COUNTER N
	ActIncCounter                              // INC_COUNTER
	ActDecCounter                              // DEC_COUNTER
	ActSetLED                                  // SET_LED R G B
	ActSendMessage                             // SEND_MESSAGE N
	ActSetTimer                                // SET_TIMER N
	ActFollowNearest                           // FOLLOW_NEAREST
	ActUnfollow                                // UNFOLLOW
	ActTurnAwayObstacle                        // TURN_AWAY_OBSTACLE
	ActMoveForwardSlow                         // MOVE_FORWARD_SLOW
	ActSetValue1                               // SET_VALUE1 N
	ActSetValue2                               // SET_VALUE2 N
	ActCopyNearestLED                          // COPY_NEAREST_LED
	ActPickup                                  // PICKUP
	ActDrop                                    // DROP
	ActTurnToPickup                            // TURN_TO_PICKUP
	ActTurnToDropoff                           // TURN_TO_DROPOFF
	ActTurnToMatchingDropoff                   // TURN_TO_MATCHING_DROPOFF
	ActSendPickup                              // SEND_PICKUP N
	ActSendDropoff                             // SEND_DROPOFF N
	ActTurnToHeardPickup                       // TURN_TO_HEARD_PICKUP
	ActTurnToHeardDropoff                      // TURN_TO_HEARD_DROPOFF
	ActTurnToMatchingLED                       // TURN_TO_MATCHING_LED
	ActSetLEDPickupColor                       // SET_LED_PICKUP_COLOR
	ActSetLEDDropoffColor                      // SET_LED_DROPOFF_COLOR
	ActTurnToRamp                              // GOTO_RAMP
	ActTurnToTruckPkg                          // GOTO_TRUCK_PKG
	ActTurnToBeaconDropoff                     // GOTO_BEACON
	ActSpiralFwd                               // SPIRAL
)

// Action represents an action to execute when a rule matches.
type Action struct {
	Type   ActionType
	Param1 int // degrees, state value, R, message value, timer ticks
	Param2 int // G (for SET_LED)
	Param3 int // B (for SET_LED)
}

// Rule is a single IF-THEN statement in SwarmScript.
type Rule struct {
	Conditions []Condition
	Action     Action
	Line       int // source line number (1-based)
}

// SwarmProgram is a compiled SwarmScript program.
type SwarmProgram struct {
	Rules []Rule
}

// conditionNames maps sensor name strings to ConditionType.
var conditionNames = map[string]ConditionType{
	"neighbors_count":            CondNeighborsCount,
	"nearest_distance":           CondNearestDistance,
	"state":                      CondState,
	"counter":                    CondCounter,
	"timer":                      CondTimer,
	"on_edge":                    CondOnEdge,
	"received_message":           CondReceivedMessage,
	"light_value":                CondLightValue,
	"random":                     CondRandom,
	"has_leader":                 CondHasLeader,
	"has_follower":               CondHasFollower,
	"chain_length":               CondChainLength,
	"nearest_led_r":              CondNearestLEDR,
	"nearest_led_g":              CondNearestLEDG,
	"nearest_led_b":              CondNearestLEDB,
	"tick":                       CondTick,
	"my_state":                   CondMyState,
	"obstacle_ahead":             CondObstacleAhead,
	"obstacle_distance":          CondObstacleDist,
	"value1":                     CondValue1,
	"value2":                     CondValue2,
	"carrying":                   CondCarrying,
	"carrying_color":             CondCarryingColor,
	"nearest_pickup_dist":        CondNearestPickupDist,
	"nearest_pickup_color":       CondNearestPickupColor,
	"nearest_pickup_has_package": CondNearestPickupHasPkg,
	"nearest_dropoff_dist":       CondNearestDropoffDist,
	"nearest_dropoff_color":      CondNearestDropoffColor,
	"dropoff_match":              CondDropoffMatch,
	"heard_pickup_color":         CondHeardPickupColor,
	"heard_dropoff_color":        CondHeardDropoffColor,
	"nearest_matching_led_dist":  CondNearestMatchLEDDist,
	"truck_here":                CondTruckHere,
	"truck_pkg_count":           CondTruckPkgCount,
	"on_ramp":                   CondOnRamp,
	"nearest_truck_pkg":         CondNearestTruckPkgDist,
	// Short aliases for general conditions (keeps preset lines under 70 chars)
	"nearest_dist": CondNearestDistance,
	"nbr_count":    CondNeighborsCount,
	"leader":       CondHasLeader,
	"follower":     CondHasFollower,
	"chain_len":    CondChainLength,
	"rnd":          CondRandom,
	// Short aliases for delivery conditions
	"pickup_dist":    CondNearestPickupDist,
	"dropoff_dist":   CondNearestDropoffDist,
	"pickup_color":   CondNearestPickupColor,
	"dropoff_color":  CondNearestDropoffColor,
	"pickup_has_pkg": CondNearestPickupHasPkg,
	"led_match_dist": CondNearestMatchLEDDist,
	"heard_pickup":   CondHeardPickupColor,
	"heard_dropoff":  CondHeardDropoffColor,
	// Ultra-short aliases (keeps preset lines under 50 chars)
	"carry":     CondCarrying,
	"match":     CondDropoffMatch,
	"has_pkg":   CondNearestPickupHasPkg,
	"p_dist":    CondNearestPickupDist,
	"d_dist":    CondNearestDropoffDist,
	"led_dist":  CondNearestMatchLEDDist,
	"obs_ahead": CondObstacleAhead,
	"obs_dist":  CondObstacleDist,
	"near_dist": CondNearestDistance,
	"neighbors": CondNeighborsCount,
	"nbrs":      CondNeighborsCount,
	"msg":       CondReceivedMessage,
	"light":     CondLightValue,
	"edge":      CondOnEdge,
	// Short aliases for truck conditions
	"truck_pkg": CondTruckPkgCount,
	"t_pkg":     CondNearestTruckPkgDist,
	// Beacon conditions
	"heard_beacon": CondHeardBeaconDropoff,
	"beacon_dist":  CondHeardBeaconDropoffDist,
	"beacon":       CondHeardBeaconDropoff,
	// Exploration
	"exploring": CondExploring,
	"lost":      CondExploring,
	// Extra delivery alias
	"led_match": CondNearestMatchLEDDist,
}

// actionNames maps action name strings to (ActionType, paramCount).
var actionNames = map[string]struct {
	Type       ActionType
	ParamCount int
}{
	"MOVE_FORWARD":             {ActMoveForward, 0},
	"TURN_LEFT":                {ActTurnLeft, 1},
	"TURN_RIGHT":               {ActTurnRight, 1},
	"TURN_TO_NEAREST":          {ActTurnToNearest, 0},
	"TURN_FROM_NEAREST":        {ActTurnFromNearest, 0},
	"TURN_TO_CENTER":           {ActTurnToCenter, 0},
	"TURN_TO_LIGHT":            {ActTurnToLight, 0},
	"TURN_RANDOM":              {ActTurnRandom, 0},
	"STOP":                     {ActStop, 0},
	"SET_STATE":                {ActSetState, 1},
	"SET_COUNTER":              {ActSetCounter, 1},
	"INC_COUNTER":              {ActIncCounter, 0},
	"DEC_COUNTER":              {ActDecCounter, 0},
	"SET_LED":                  {ActSetLED, 3},
	"SEND_MESSAGE":             {ActSendMessage, 1},
	"SET_TIMER":                {ActSetTimer, 1},
	"FOLLOW_NEAREST":           {ActFollowNearest, 0},
	"UNFOLLOW":                 {ActUnfollow, 0},
	"TURN_AWAY_OBSTACLE":       {ActTurnAwayObstacle, 0},
	"MOVE_FORWARD_SLOW":        {ActMoveForwardSlow, 0},
	"SET_VALUE1":               {ActSetValue1, 1},
	"SET_VALUE2":               {ActSetValue2, 1},
	"COPY_NEAREST_LED":         {ActCopyNearestLED, 0},
	"PICKUP":                   {ActPickup, 0},
	"DROP":                     {ActDrop, 0},
	"TURN_TO_PICKUP":           {ActTurnToPickup, 0},
	"TURN_TO_DROPOFF":          {ActTurnToDropoff, 0},
	"TURN_TO_MATCHING_DROPOFF": {ActTurnToMatchingDropoff, 0},
	"SEND_PICKUP":              {ActSendPickup, 1},
	"SEND_DROPOFF":             {ActSendDropoff, 1},
	"TURN_TO_HEARD_PICKUP":     {ActTurnToHeardPickup, 0},
	"TURN_TO_HEARD_DROPOFF":    {ActTurnToHeardDropoff, 0},
	"TURN_TO_MATCHING_LED":     {ActTurnToMatchingLED, 0},
	"SET_LED_PICKUP_COLOR":     {ActSetLEDPickupColor, 0},
	"SET_LED_DROPOFF_COLOR":    {ActSetLEDDropoffColor, 0},
	// Truck actions
	"GOTO_RAMP":          {ActTurnToRamp, 0},
	"GOTO_TRUCK_PKG":     {ActTurnToTruckPkg, 0},
	// Beacon actions
	"GOTO_BEACON":        {ActTurnToBeaconDropoff, 0},
	"SPIRAL":             {ActSpiralFwd, 0},
	// Extra delivery alias
	"GOTO_LED_MATCH":     {ActTurnToMatchingLED, 0},
	// Short aliases for delivery actions (keeps preset lines under 70 chars)
	"GOTO_PICKUP":        {ActTurnToPickup, 0},
	"GOTO_DROPOFF":       {ActTurnToMatchingDropoff, 0},
	"GOTO_LED":           {ActTurnToMatchingLED, 0},
	"GOTO_HEARD_PICKUP":  {ActTurnToHeardPickup, 0},
	"GOTO_HEARD_DROPOFF": {ActTurnToHeardDropoff, 0},
	"LED_PICKUP":         {ActSetLEDPickupColor, 0},
	"LED_DROPOFF":        {ActSetLEDDropoffColor, 0},
	"COPY_LED":           {ActCopyNearestLED, 0},
	"AVOID_OBSTACLE":     {ActTurnAwayObstacle, 0},
	// Ultra-short aliases
	"FWD":        {ActMoveForward, 0},
	"FWD_SLOW":   {ActMoveForwardSlow, 0},
	"GOTO_MATCH": {ActTurnToMatchingDropoff, 0},
}

// --- SwarmScript syntax highlighting support ---

// SwarmTokenType for syntax highlighting.
type SwarmTokenType int

const (
	TokKeyword   SwarmTokenType = iota // IF, THEN, AND
	TokCondition                       // sensor names
	TokAction                          // action names
	TokNumber                          // numeric values
	TokOperator                        // >, <, ==
	TokComment                         // # comments
	TokText                            // other text
)

// SwarmToken represents a highlighted text segment.
type SwarmToken struct {
	Text string
	Type SwarmTokenType
	Col  int // starting column in line
}

// keywords for syntax highlighting
var highlightKeywords = map[string]bool{
	"IF": true, "THEN": true, "AND": true,
}

var highlightConditions = map[string]bool{
	"neighbors_count": true, "nearest_distance": true,
	"state": true, "counter": true, "timer": true,
	"on_edge": true, "received_message": true,
	"light_value": true, "random": true, "true": true,
	"has_leader": true, "has_follower": true, "chain_length": true,
	"nearest_led_r": true, "nearest_led_g": true, "nearest_led_b": true,
	"tick": true, "my_state": true, "obstacle_ahead": true,
	"obstacle_distance": true, "value1": true, "value2": true,
	"carrying": true, "carrying_color": true,
	"nearest_pickup_dist": true, "nearest_pickup_color": true,
	"nearest_pickup_has_package": true,
	"nearest_dropoff_dist":       true, "nearest_dropoff_color": true,
	"dropoff_match":      true,
	"heard_pickup_color": true, "heard_dropoff_color": true,
	"nearest_matching_led_dist": true,
	// Short aliases (general)
	"nearest_dist": true, "nbr_count": true,
	"leader": true, "follower": true, "chain_len": true, "rnd": true,
	// Short aliases (delivery)
	"pickup_dist": true, "dropoff_dist": true,
	"pickup_color": true, "dropoff_color": true,
	"pickup_has_pkg": true, "led_match_dist": true,
	"heard_pickup": true, "heard_dropoff": true,
	// Ultra-short aliases
	"carry": true, "match": true, "has_pkg": true,
	"p_dist": true, "d_dist": true, "led_dist": true,
	"obs_ahead": true, "obs_dist": true, "near_dist": true,
	"neighbors": true, "nbrs": true, "msg": true, "light": true, "edge": true,
	// Truck sensors
	"truck_here": true, "truck_pkg_count": true, "on_ramp": true,
	"nearest_truck_pkg": true, "truck_pkg": true, "t_pkg": true,
	// Beacon sensors
	"heard_beacon": true, "beacon_dist": true, "beacon": true, "led_match": true,
	"exploring": true, "lost": true,
}

var highlightActions = map[string]bool{
	"MOVE_FORWARD": true, "TURN_LEFT": true, "TURN_RIGHT": true,
	"TURN_TO_NEAREST": true, "TURN_FROM_NEAREST": true,
	"TURN_TO_CENTER": true, "TURN_TO_LIGHT": true,
	"TURN_RANDOM": true, "STOP": true,
	"SET_STATE": true, "SET_COUNTER": true,
	"INC_COUNTER": true, "DEC_COUNTER": true,
	"SET_LED": true, "SEND_MESSAGE": true, "SET_TIMER": true,
	"FOLLOW_NEAREST": true, "UNFOLLOW": true,
	"TURN_AWAY_OBSTACLE": true, "MOVE_FORWARD_SLOW": true,
	"SET_VALUE1": true, "SET_VALUE2": true,
	"COPY_NEAREST_LED": true,
	"PICKUP":           true, "DROP": true,
	"TURN_TO_PICKUP": true, "TURN_TO_DROPOFF": true,
	"TURN_TO_MATCHING_DROPOFF": true,
	"SEND_PICKUP":              true, "SEND_DROPOFF": true,
	"TURN_TO_HEARD_PICKUP": true, "TURN_TO_HEARD_DROPOFF": true,
	"TURN_TO_MATCHING_LED": true,
	"SET_LED_PICKUP_COLOR": true, "SET_LED_DROPOFF_COLOR": true,
	// Short aliases
	"GOTO_PICKUP": true, "GOTO_DROPOFF": true, "GOTO_LED": true,
	"GOTO_HEARD_PICKUP": true, "GOTO_HEARD_DROPOFF": true,
	"LED_PICKUP": true, "LED_DROPOFF": true,
	"COPY_LED": true, "AVOID_OBSTACLE": true,
	"FWD": true, "FWD_SLOW": true, "GOTO_MATCH": true,
	// Truck actions
	"GOTO_RAMP": true, "GOTO_TRUCK_PKG": true,
	// Beacon actions
	"GOTO_BEACON": true, "GOTO_LED_MATCH": true,
	"SPIRAL": true,
}

// --- Reverse mapping functions (for block editor / serialization) ---

// ConditionTypeName returns the canonical short sensor name for a ConditionType.
func ConditionTypeName(ct ConditionType) string {
	switch ct {
	case CondNeighborsCount:
		return "neighbors"
	case CondNearestDistance:
		return "near_dist"
	case CondState, CondMyState:
		return "state"
	case CondCounter:
		return "counter"
	case CondTimer:
		return "timer"
	case CondOnEdge:
		return "edge"
	case CondReceivedMessage:
		return "msg"
	case CondLightValue:
		return "light"
	case CondRandom:
		return "rnd"
	case CondTrue:
		return "true"
	case CondHasLeader:
		return "leader"
	case CondHasFollower:
		return "follower"
	case CondChainLength:
		return "chain_len"
	case CondNearestLEDR:
		return "nearest_led_r"
	case CondNearestLEDG:
		return "nearest_led_g"
	case CondNearestLEDB:
		return "nearest_led_b"
	case CondTick:
		return "tick"
	case CondObstacleAhead:
		return "obs_ahead"
	case CondObstacleDist:
		return "obs_dist"
	case CondValue1:
		return "value1"
	case CondValue2:
		return "value2"
	case CondCarrying:
		return "carry"
	case CondCarryingColor:
		return "carrying_color"
	case CondNearestPickupDist:
		return "p_dist"
	case CondNearestPickupColor:
		return "pickup_color"
	case CondNearestPickupHasPkg:
		return "has_pkg"
	case CondNearestDropoffDist:
		return "d_dist"
	case CondNearestDropoffColor:
		return "dropoff_color"
	case CondDropoffMatch:
		return "match"
	case CondHeardPickupColor:
		return "heard_pickup"
	case CondHeardDropoffColor:
		return "heard_dropoff"
	case CondNearestMatchLEDDist:
		return "led_dist"
	case CondTruckHere:
		return "truck_here"
	case CondTruckPkgCount:
		return "truck_pkg_count"
	case CondOnRamp:
		return "on_ramp"
	case CondNearestTruckPkgDist:
		return "nearest_truck_pkg"
	case CondHeardBeaconDropoff:
		return "heard_beacon"
	case CondHeardBeaconDropoffDist:
		return "beacon_dist"
	case CondExploring:
		return "exploring"
	}
	return "unknown"
}

// OpString returns the string representation of a ConditionOp.
func OpString(op ConditionOp) string {
	switch op {
	case OpGT:
		return ">"
	case OpLT:
		return "<"
	case OpEQ:
		return "=="
	}
	return "=="
}

// ActionTypeName returns the canonical short action name for an ActionType.
func ActionTypeName(at ActionType) string {
	switch at {
	case ActMoveForward:
		return "FWD"
	case ActTurnLeft:
		return "TURN_LEFT"
	case ActTurnRight:
		return "TURN_RIGHT"
	case ActTurnToNearest:
		return "TURN_TO_NEAREST"
	case ActTurnFromNearest:
		return "TURN_FROM_NEAREST"
	case ActTurnToCenter:
		return "TURN_TO_CENTER"
	case ActTurnToLight:
		return "TURN_TO_LIGHT"
	case ActTurnRandom:
		return "TURN_RANDOM"
	case ActStop:
		return "STOP"
	case ActSetState:
		return "SET_STATE"
	case ActSetCounter:
		return "SET_COUNTER"
	case ActIncCounter:
		return "INC_COUNTER"
	case ActDecCounter:
		return "DEC_COUNTER"
	case ActSetLED:
		return "SET_LED"
	case ActSendMessage:
		return "SEND_MESSAGE"
	case ActSetTimer:
		return "SET_TIMER"
	case ActFollowNearest:
		return "FOLLOW_NEAREST"
	case ActUnfollow:
		return "UNFOLLOW"
	case ActTurnAwayObstacle:
		return "AVOID_OBSTACLE"
	case ActMoveForwardSlow:
		return "FWD_SLOW"
	case ActSetValue1:
		return "SET_VALUE1"
	case ActSetValue2:
		return "SET_VALUE2"
	case ActCopyNearestLED:
		return "COPY_LED"
	case ActPickup:
		return "PICKUP"
	case ActDrop:
		return "DROP"
	case ActTurnToPickup:
		return "GOTO_PICKUP"
	case ActTurnToDropoff:
		return "TURN_TO_DROPOFF"
	case ActTurnToMatchingDropoff:
		return "GOTO_DROPOFF"
	case ActSendPickup:
		return "SEND_PICKUP"
	case ActSendDropoff:
		return "SEND_DROPOFF"
	case ActTurnToHeardPickup:
		return "GOTO_HEARD_PICKUP"
	case ActTurnToHeardDropoff:
		return "GOTO_HEARD_DROPOFF"
	case ActTurnToMatchingLED:
		return "GOTO_LED"
	case ActSetLEDPickupColor:
		return "LED_PICKUP"
	case ActSetLEDDropoffColor:
		return "LED_DROPOFF"
	case ActTurnToRamp:
		return "GOTO_RAMP"
	case ActTurnToTruckPkg:
		return "GOTO_TRUCK_PKG"
	case ActTurnToBeaconDropoff:
		return "GOTO_BEACON"
	case ActSpiralFwd:
		return "SPIRAL"
	}
	return "UNKNOWN"
}

// ActionParamCount returns the number of parameters an ActionType expects.
func ActionParamCount(at ActionType) int {
	for _, info := range actionNames {
		if info.Type == at {
			return info.ParamCount
		}
	}
	return 0
}

// ActionParamCountByName returns the param count for a named action.
func ActionParamCountByName(name string) int {
	if info, ok := actionNames[name]; ok {
		return info.ParamCount
	}
	return 0
}

// SensorGrouped returns sensor names organized in groups for dropdown display.
var SensorGrouped = [][]string{
	{"-- Nachbarn --", "neighbors", "near_dist", "leader", "follower", "chain_len"},
	{"-- Navigation --", "edge", "obs_ahead", "obs_dist", "light"},
	{"-- Zufall --", "rnd", "true"},
	{"-- Delivery --", "carry", "match", "has_pkg", "p_dist", "d_dist", "pickup_color", "dropoff_color", "heard_beacon", "beacon_dist", "exploring"},
	{"-- Kommunikation --", "msg", "heard_pickup", "heard_dropoff", "led_dist"},
	{"-- LED --", "nearest_led_r", "nearest_led_g", "nearest_led_b"},
	{"-- Intern --", "state", "counter", "value1", "value2", "timer", "tick"},
	{"-- Truck --", "truck_here", "truck_pkg_count", "on_ramp", "nearest_truck_pkg"},
}

// ActionGrouped returns action names organized in groups for dropdown display.
var ActionGrouped = [][]string{
	{"-- Bewegung --", "FWD", "FWD_SLOW", "STOP", "TURN_LEFT", "TURN_RIGHT", "TURN_RANDOM"},
	{"-- Navigation --", "TURN_TO_NEAREST", "TURN_FROM_NEAREST", "TURN_TO_CENTER", "TURN_TO_LIGHT", "AVOID_OBSTACLE"},
	{"-- Delivery --", "PICKUP", "DROP", "GOTO_PICKUP", "GOTO_DROPOFF", "GOTO_LED", "GOTO_BEACON", "SPIRAL"},
	{"-- Kommunikation --", "SEND_MESSAGE", "SEND_PICKUP", "SEND_DROPOFF", "GOTO_HEARD_PICKUP", "GOTO_HEARD_DROPOFF"},
	{"-- LED --", "SET_LED", "LED_PICKUP", "LED_DROPOFF", "COPY_LED"},
	{"-- Intern --", "SET_STATE", "SET_COUNTER", "INC_COUNTER", "DEC_COUNTER", "SET_TIMER", "SET_VALUE1", "SET_VALUE2"},
	{"-- Follow --", "FOLLOW_NEAREST", "UNFOLLOW"},
	{"-- Truck --", "GOTO_RAMP", "GOTO_TRUCK_PKG"},
}

// wordPos tracks a word and its column position in a line.
type wordPos struct {
	text string
	col  int
}
