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
	CondWallRight                                // wall_right (wall within 25px to the right)
	CondWallLeft                                 // wall_left (wall within 25px to the left)
	CondPherAhead                                // pheromone intensity ahead (0-100)
	CondTeam                                     // team (1=A, 2=B, 0=none)
	CondTeamScore                                // team_score (own team's score)
	CondEnemyScore                               // enemy_score (opponent's score)
	CondBotAhead                                 // bot_ahead (neighbors in front 90° cone)
	CondBotBehind                                // bot_behind (neighbors behind 90° cone)
	CondBotLeft                                  // bot_left (neighbors left 90° cone)
	CondBotRight                                 // bot_right (neighbors right 90° cone)
	CondHeading                                  // heading (0-359 degrees)
	CondSpeed                                    // speed (current speed * 100)
	CondVisitedHere                              // visited_here (times visited current cell)
	CondVisitedAhead                             // visited_ahead (times visited cell ahead)
	CondExplored                                 // explored (% of arena explored by this bot)
	CondGroupCarry                               // group_carry (% of neighbors carrying)
	CondGroupSpeed                               // group_speed (avg speed of neighbors * 100)
	CondGroupSize                                // group_size (connected cluster size)
	CondSwarmCenterDist                          // swarm_center_dist (distance to swarm center of mass)
	CondSwarmSpread                              // swarm_spread (overall swarm spread)
	CondIsolationLevel                           // isolation_level (0=close, >0=isolated)
	CondResourceGradientX                        // resource_gradient_x (direction to resources, 0-359)
	CondResourceGradientY                        // resource_gradient_y (resource proximity, 0-100)
	CondEnergy                                   // energy (0-100)
	CondBotCarrying                              // bot_carrying (count of neighbors carrying)
	CondTimeSinceDelivery                        // time_since_delivery (ticks since last delivery)
	CondRecentCollision                          // recent_collision (1 if collided recently)
	CondNeighborMinDist                          // neighbor_min_dist (distance to closest neighbor)
	CondPathDist                                 // path_dist (remaining A* path distance, 0 if no path)
	CondPathAngle                                // path_angle (angle to next waypoint, -180..180)
	CondFlockAlign                               // flock_align (angle diff to neighbor avg heading, -180..180)
	CondFlockCohesion                            // flock_cohesion (distance to neighbor center of mass)
	CondFlockSeparation                          // flock_separation (separation urgency 0-100)
	CondRole                                     // role (0=none, 1=scout, 2=worker, 3=guard)
	CondRoleDemand                               // role_demand (most needed role 1-3)
	CondVote                                     // vote (current vote value)
	CondQuorumCount                              // quorum_count (nearby bots with same vote)
	CondQuorumReached                            // quorum_reached (1 if threshold met)
	CondReputation                               // reputation (0-100, trust level)
	CondSuspectNearby                            // suspect_nearby (1 if anomalous neighbor)
	CondLevyPhase                                // levy_phase (0=idle, 1=short walk, 2=long jump)
	CondLevyStep                                 // levy_step (remaining step distance)
	CondFlashPhase                               // flash_phase (oscillator phase 0-255)
	CondFlashSync                                // flash_sync (1 if currently flashing)
	CondTransportNearby                          // transport_nearby (heavy objects in range)
	CondTransportCount                           // transport_count (bots assisting nearest task)
	CondVortexStrength                           // vortex_strength (local rotation strength 0-100)
)

// Condition represents a single boolean check in a rule.
type Condition struct {
	Type       ConditionType
	Op         ConditionOp
	Value      int
	IsParamRef bool    // true if value is $A-$Z reference
	ParamIdx   int     // 0-25 for A-Z
	ParamHint  float64 // default/initial value from $X:NNN
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
	ActWallFollowRight                         // WALL_FOLLOW_RIGHT (right-hand rule)
	ActWallFollowLeft                          // WALL_FOLLOW_LEFT (left-hand rule)
	ActFollowPheromone                         // FOLLOW_PHER (follow pheromone gradient)
	ActDash                                    // DASH (double-speed burst for 10 ticks)
	ActEmergencyBroadcast                      // EMERGENCY_BROADCAST N (3x range broadcast)
	ActReverse                                 // REVERSE (turn 180° and move forward)
	ActBrake                                   // BRAKE (reduce speed to 0 over 3 ticks)
	ActScatterRandom                           // SCATTER_RANDOM (scatter away from neighbors)
	ActFollowPath                              // FOLLOW_PATH (steer toward next A* waypoint)
	ActFlock                                   // FLOCK (apply all Reynolds rules: separation+alignment+cohesion)
	ActAlign                                   // ALIGN (align heading with neighbors)
	ActCohere                                  // COHERE (steer toward neighbor center of mass)
	ActBecomeScout                             // BECOME_SCOUT (switch role to scout)
	ActBecomeWorker                            // BECOME_WORKER (switch role to worker)
	ActBecomeGuard                             // BECOME_GUARD (switch role to guard)
	ActVote                                    // VOTE N (cast a vote with value N)
	ActFlagRogue                               // FLAG_ROGUE (decrease nearest neighbor's reputation)
	ActLevyWalk                                // LEVY_WALK (Lévy flight: short walks + rare long jumps)
	ActFlash                                   // FLASH (trigger immediate firefly flash)
	ActAssistTransport                         // ASSIST_TRANSPORT (join nearest transport task)
	ActVortex                                  // VORTEX (join/maintain vortex rotation)
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
	// Wall sensors
	"wall_right": CondWallRight,
	"wall_left":  CondWallLeft,
	"wall_front": CondObstacleAhead, // alias for obs_ahead
	// Pheromone sensors
	"pheromone":  CondPherAhead,
	"pher":       CondPherAhead,
	"pher_ahead": CondPherAhead,
	// Team sensors
	"team":        CondTeam,
	"team_score":  CondTeamScore,
	"enemy_score": CondEnemyScore,
	// Directional sensors (90° cones)
	"bot_ahead":  CondBotAhead,
	"bot_behind": CondBotBehind,
	"bot_left":   CondBotLeft,
	"bot_right":  CondBotRight,
	"ahead":      CondBotAhead,
	"behind":     CondBotBehind,
	// Heading & speed
	"heading": CondHeading,
	"speed":   CondSpeed,
	// Memory sensors
	"visited_here":  CondVisitedHere,
	"visited_ahead": CondVisitedAhead,
	"explored":      CondExplored,
	"visited":       CondVisitedHere,
	// Cooperative sensors (sensor fusion)
	"group_carry": CondGroupCarry,
	"group_speed": CondGroupSpeed,
	"group_size":  CondGroupSize,
	// Swarm awareness sensors
	"swarm_center_dist":   CondSwarmCenterDist,
	"swarm_spread":        CondSwarmSpread,
	"isolation_level":     CondIsolationLevel,
	"resource_gradient_x": CondResourceGradientX,
	"resource_gradient_y": CondResourceGradientY,
	"center_dist":         CondSwarmCenterDist,
	"isolation":           CondIsolationLevel,
	"res_grad_x":          CondResourceGradientX,
	"res_grad_y":          CondResourceGradientY,
	// Energy & advanced sensors
	"energy":              CondEnergy,
	"bot_carrying":        CondBotCarrying,
	"time_since_delivery": CondTimeSinceDelivery,
	"since_delivery":      CondTimeSinceDelivery,
	"recent_collision":    CondRecentCollision,
	"collision":           CondRecentCollision,
	"neighbor_min_dist":   CondNeighborMinDist,
	"nbr_min_dist":        CondNeighborMinDist,
	// A* pathfinding sensors
	"path_dist":  CondPathDist,
	"path_angle": CondPathAngle,
	"pdist":      CondPathDist,
	"pangle":     CondPathAngle,
	// Flocking (Boids) sensors
	"flock_align":      CondFlockAlign,
	"flock_cohesion":   CondFlockCohesion,
	"flock_separation": CondFlockSeparation,
	"flock_sep":        CondFlockSeparation,
	"f_align":          CondFlockAlign,
	"f_cohesion":       CondFlockCohesion,
	"f_sep":            CondFlockSeparation,
	// Dynamic Role sensors
	"role":        CondRole,
	"role_demand": CondRoleDemand,
	// Quorum Sensing sensors
	"vote":           CondVote,
	"quorum_count":   CondQuorumCount,
	"quorum_reached": CondQuorumReached,
	"quorum":         CondQuorumReached,
	// Rogue Detection sensors
	"reputation":     CondReputation,
	"suspect_nearby": CondSuspectNearby,
	"suspect":        CondSuspectNearby,
	"rep":            CondReputation,
	// Lévy-Flight sensors
	"levy_phase":        CondLevyPhase,
	"levy_step":         CondLevyStep,
	"levy":              CondLevyPhase,
	// Firefly Sync sensors
	"flash_phase":       CondFlashPhase,
	"flash_sync":        CondFlashSync,
	"flash":             CondFlashSync,
	// Collective Transport sensors
	"transport_nearby":  CondTransportNearby,
	"transport_count":   CondTransportCount,
	"transport":         CondTransportNearby,
	// Vortex Swarming sensors
	"vortex_strength":   CondVortexStrength,
	"vortex":            CondVortexStrength,
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
	// Wall-follow actions
	"WALL_FOLLOW_RIGHT":  {ActWallFollowRight, 0},
	"WALL_FOLLOW_LEFT":   {ActWallFollowLeft, 0},
	// Pheromone actions
	"FOLLOW_PHER":        {ActFollowPheromone, 0},
	"GOTO_PHER":          {ActFollowPheromone, 0},
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
	// Dash & emergency broadcast
	"DASH":                {ActDash, 0},
	"EMERGENCY_BROADCAST": {ActEmergencyBroadcast, 1},
	"EMERGENCY":           {ActEmergencyBroadcast, 1},
	// Movement extensions
	"REVERSE":             {ActReverse, 0},
	"BRAKE":               {ActBrake, 0},
	"SCATTER_RANDOM":      {ActScatterRandom, 0},
	"SCATTER":             {ActScatterRandom, 0},
	// A* pathfinding
	"FOLLOW_PATH":         {ActFollowPath, 0},
	"PATH":                {ActFollowPath, 0},
	// Flocking (Boids) actions
	"FLOCK":               {ActFlock, 0},
	"ALIGN":               {ActAlign, 0},
	"COHERE":              {ActCohere, 0},
	// Dynamic Role actions
	"BECOME_SCOUT":        {ActBecomeScout, 0},
	"BECOME_WORKER":       {ActBecomeWorker, 0},
	"BECOME_GUARD":        {ActBecomeGuard, 0},
	"SCOUT":               {ActBecomeScout, 0},
	"WORKER":              {ActBecomeWorker, 0},
	"GUARD":               {ActBecomeGuard, 0},
	// Quorum Sensing actions
	"VOTE":                {ActVote, 1},
	// Rogue Detection actions
	"FLAG_ROGUE":          {ActFlagRogue, 0},
	"FLAG":                {ActFlagRogue, 0},
	// Lévy-Flight actions
	"LEVY_WALK":           {ActLevyWalk, 0},
	"LEVY":                {ActLevyWalk, 0},
	// Firefly actions
	"FLASH":               {ActFlash, 0},
	// Collective Transport actions
	"ASSIST_TRANSPORT":    {ActAssistTransport, 0},
	"ASSIST":              {ActAssistTransport, 0},
	// Vortex actions
	"VORTEX":              {ActVortex, 0},
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
	// Wall sensors
	"wall_right": true, "wall_left": true, "wall_front": true,
	// Pheromone sensors
	"pheromone": true, "pher": true, "pher_ahead": true,
	// Team sensors
	"team": true, "team_score": true, "enemy_score": true,
	// Directional sensors
	"bot_ahead": true, "bot_behind": true, "bot_left": true, "bot_right": true,
	"ahead": true, "behind": true,
	"heading": true, "speed": true,
	// Memory sensors
	"visited_here": true, "visited_ahead": true, "explored": true, "visited": true,
	// Cooperative sensors
	"group_carry": true, "group_speed": true, "group_size": true,
	// Swarm awareness sensors
	"swarm_center_dist": true, "swarm_spread": true, "isolation_level": true,
	"resource_gradient_x": true, "resource_gradient_y": true,
	"center_dist": true, "isolation": true, "res_grad_x": true, "res_grad_y": true,
	// Energy & advanced sensors
	"energy": true, "bot_carrying": true, "time_since_delivery": true, "since_delivery": true,
	"recent_collision": true, "collision": true, "neighbor_min_dist": true, "nbr_min_dist": true,
	// A* pathfinding sensors
	"path_dist": true, "path_angle": true, "pdist": true, "pangle": true,
	// Flocking sensors
	"flock_align": true, "flock_cohesion": true, "flock_separation": true,
	"flock_sep": true, "f_align": true, "f_cohesion": true, "f_sep": true,
	// Role sensors
	"role": true, "role_demand": true,
	// Quorum sensors
	"vote": true, "quorum_count": true, "quorum_reached": true, "quorum": true,
	// Rogue sensors
	"reputation": true, "suspect_nearby": true, "suspect": true, "rep": true,
	// Lévy-Flight sensors
	"levy_phase": true, "levy_step": true, "levy": true,
	// Firefly sensors
	"flash_phase": true, "flash_sync": true, "flash": true,
	// Transport sensors
	"transport_nearby": true, "transport_count": true, "transport": true,
	// Vortex sensors
	"vortex_strength": true, "vortex": true,
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
	// Wall-follow
	"WALL_FOLLOW_RIGHT": true, "WALL_FOLLOW_LEFT": true,
	// Pheromone
	"FOLLOW_PHER": true, "GOTO_PHER": true,
	// Dash & emergency
	"DASH": true, "EMERGENCY_BROADCAST": true, "EMERGENCY": true,
	// Movement extensions
	"REVERSE": true, "BRAKE": true, "SCATTER_RANDOM": true, "SCATTER": true,
	// A* pathfinding
	"FOLLOW_PATH": true, "PATH": true,
	// Flocking
	"FLOCK": true, "ALIGN": true, "COHERE": true,
	// Roles
	"BECOME_SCOUT": true, "BECOME_WORKER": true, "BECOME_GUARD": true,
	"SCOUT": true, "WORKER": true, "GUARD": true,
	// Quorum
	"VOTE": true,
	// Rogue
	"FLAG_ROGUE": true, "FLAG": true,
	// Lévy-Flight
	"LEVY_WALK": true, "LEVY": true,
	// Firefly
	"FLASH": true,
	// Transport
	"ASSIST_TRANSPORT": true, "ASSIST": true,
	// Vortex
	"VORTEX": true,
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
	case CondWallRight:
		return "wall_right"
	case CondWallLeft:
		return "wall_left"
	case CondPherAhead:
		return "pher"
	case CondBotAhead:
		return "bot_ahead"
	case CondBotBehind:
		return "bot_behind"
	case CondBotLeft:
		return "bot_left"
	case CondBotRight:
		return "bot_right"
	case CondHeading:
		return "heading"
	case CondSpeed:
		return "speed"
	case CondVisitedHere:
		return "visited"
	case CondVisitedAhead:
		return "visited_ahead"
	case CondExplored:
		return "explored"
	case CondGroupCarry:
		return "group_carry"
	case CondGroupSpeed:
		return "group_speed"
	case CondGroupSize:
		return "group_size"
	case CondSwarmCenterDist:
		return "center_dist"
	case CondSwarmSpread:
		return "swarm_spread"
	case CondIsolationLevel:
		return "isolation"
	case CondResourceGradientX:
		return "res_grad_x"
	case CondResourceGradientY:
		return "res_grad_y"
	case CondEnergy:
		return "energy"
	case CondBotCarrying:
		return "bot_carrying"
	case CondTimeSinceDelivery:
		return "since_delivery"
	case CondRecentCollision:
		return "collision"
	case CondNeighborMinDist:
		return "nbr_min_dist"
	case CondPathDist:
		return "path_dist"
	case CondPathAngle:
		return "path_angle"
	case CondFlockAlign:
		return "flock_align"
	case CondFlockCohesion:
		return "flock_cohesion"
	case CondFlockSeparation:
		return "flock_sep"
	case CondRole:
		return "role"
	case CondRoleDemand:
		return "role_demand"
	case CondVote:
		return "vote"
	case CondQuorumCount:
		return "quorum_count"
	case CondQuorumReached:
		return "quorum"
	case CondReputation:
		return "rep"
	case CondSuspectNearby:
		return "suspect"
	case CondLevyPhase:
		return "levy_phase"
	case CondLevyStep:
		return "levy_step"
	case CondFlashPhase:
		return "flash_phase"
	case CondFlashSync:
		return "flash_sync"
	case CondTransportNearby:
		return "transport_nearby"
	case CondTransportCount:
		return "transport_count"
	case CondVortexStrength:
		return "vortex_strength"
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
	case ActWallFollowRight:
		return "WALL_FOLLOW_RIGHT"
	case ActWallFollowLeft:
		return "WALL_FOLLOW_LEFT"
	case ActFollowPheromone:
		return "FOLLOW_PHER"
	case ActDash:
		return "DASH"
	case ActEmergencyBroadcast:
		return "EMERGENCY"
	case ActReverse:
		return "REVERSE"
	case ActBrake:
		return "BRAKE"
	case ActScatterRandom:
		return "SCATTER"
	case ActFollowPath:
		return "FOLLOW_PATH"
	case ActFlock:
		return "FLOCK"
	case ActAlign:
		return "ALIGN"
	case ActCohere:
		return "COHERE"
	case ActBecomeScout:
		return "SCOUT"
	case ActBecomeWorker:
		return "WORKER"
	case ActBecomeGuard:
		return "GUARD"
	case ActVote:
		return "VOTE"
	case ActFlagRogue:
		return "FLAG"
	case ActLevyWalk:
		return "LEVY_WALK"
	case ActFlash:
		return "FLASH"
	case ActAssistTransport:
		return "ASSIST"
	case ActVortex:
		return "VORTEX"
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
	{"-- Navigation --", "edge", "obs_ahead", "obs_dist", "light", "wall_right", "wall_left", "wall_front", "pher", "path_dist", "path_angle", "flock_align", "flock_cohesion", "flock_sep"},
	{"-- Zufall --", "rnd", "true"},
	{"-- Delivery --", "carry", "match", "has_pkg", "p_dist", "d_dist", "pickup_color", "dropoff_color", "heard_beacon", "beacon_dist", "exploring"},
	{"-- Kommunikation --", "msg", "heard_pickup", "heard_dropoff", "led_dist"},
	{"-- LED --", "nearest_led_r", "nearest_led_g", "nearest_led_b"},
	{"-- Intern --", "state", "counter", "value1", "value2", "timer", "tick"},
	{"-- Truck --", "truck_here", "truck_pkg_count", "on_ramp", "nearest_truck_pkg"},
	{"-- Schwarm --", "center_dist", "swarm_spread", "isolation", "res_grad_x", "res_grad_y"},
	{"-- Erweitert --", "energy", "bot_carrying", "since_delivery", "collision", "nbr_min_dist"},
	{"-- Rollen & Quorum --", "role", "role_demand", "vote", "quorum_count", "quorum", "rep", "suspect"},
	{"-- Schwarm-KI --", "levy_phase", "levy_step", "flash_phase", "flash_sync", "transport_nearby", "transport_count", "vortex_strength"},
}

// ActionGrouped returns action names organized in groups for dropdown display.
var ActionGrouped = [][]string{
	{"-- Bewegung --", "FWD", "FWD_SLOW", "STOP", "TURN_LEFT", "TURN_RIGHT", "TURN_RANDOM"},
	{"-- Navigation --", "TURN_TO_NEAREST", "TURN_FROM_NEAREST", "TURN_TO_CENTER", "TURN_TO_LIGHT", "AVOID_OBSTACLE", "WALL_FOLLOW_RIGHT", "WALL_FOLLOW_LEFT", "FOLLOW_PHER", "FOLLOW_PATH", "FLOCK", "ALIGN", "COHERE"},
	{"-- Delivery --", "PICKUP", "DROP", "GOTO_PICKUP", "GOTO_DROPOFF", "GOTO_LED", "GOTO_BEACON", "SPIRAL"},
	{"-- Kommunikation --", "SEND_MESSAGE", "SEND_PICKUP", "SEND_DROPOFF", "GOTO_HEARD_PICKUP", "GOTO_HEARD_DROPOFF"},
	{"-- LED --", "SET_LED", "LED_PICKUP", "LED_DROPOFF", "COPY_LED"},
	{"-- Intern --", "SET_STATE", "SET_COUNTER", "INC_COUNTER", "DEC_COUNTER", "SET_TIMER", "SET_VALUE1", "SET_VALUE2"},
	{"-- Follow --", "FOLLOW_NEAREST", "UNFOLLOW"},
	{"-- Truck --", "GOTO_RAMP", "GOTO_TRUCK_PKG"},
	{"-- Spezial --", "DASH", "EMERGENCY", "REVERSE", "BRAKE", "SCATTER"},
	{"-- Rollen & Quorum --", "SCOUT", "WORKER", "GUARD", "VOTE", "FLAG"},
	{"-- Schwarm-KI --", "LEVY_WALK", "FLASH", "ASSIST", "VORTEX"},
}

// wordPos tracks a word and its column position in a line.
type wordPos struct {
	text string
	col  int
}
