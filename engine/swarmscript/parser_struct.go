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
}

// wordPos tracks a word and its column position in a line.
type wordPos struct {
	text string
	col  int
}
