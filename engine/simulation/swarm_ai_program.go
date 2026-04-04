package simulation

import (
	"fmt"
	"math"
	"math/rand"
	"swarmsim/domain/swarm"
	"swarmsim/engine/swarmscript"
	"swarmsim/logger"
)

// executeSwarmProgram runs all matching rules on a bot.
// Conditions evaluate against a snapshot; actions mutate the bot live.
func executeSwarmProgram(ss *swarm.SwarmState, i int) {
	bot := &ss.Bots[i]
	prog := ss.Program

	// Genetic Programming: use bot's own program if GP is ON
	if ss.GPEnabled && bot.OwnProgram != nil {
		prog = bot.OwnProgram
	}

	// Teams: use team-specific program if teams enabled
	if ss.TeamsEnabled {
		if bot.Team == 1 && ss.TeamAProgram != nil {
			prog = ss.TeamAProgram
		} else if bot.Team == 2 && ss.TeamBProgram != nil {
			prog = ss.TeamBProgram
		}
	}

	if prog == nil {
		return
	}

	// Snapshot mutable vars for condition evaluation
	snapState := bot.State
	snapCounter := bot.Counter
	snapTimer := bot.Timer

	// Reset per-tick outputs
	bot.Speed = 0
	bot.PendingMsg = 0

	// Track matched rules for GP visualization
	if ss.GPEnabled {
		bot.LastMatchedRules = bot.LastMatchedRules[:0]
	}

	// Decision trace: for the selected bot, record per-condition results
	tracing := ss.ShowDecisionTrace && i == ss.SelectedBot
	if tracing {
		ss.DecisionTrace = ss.DecisionTrace[:0]
	}

	for ri, rule := range prog.Rules {
		// Evaluate all conditions
		allMatch := true

		if tracing {
			step := swarm.DecisionStep{
				RuleIdx:    ri,
				ActionName: swarmscript.ActionTypeName(rule.Action.Type),
			}
			for _, cond := range rule.Conditions {
				actual := resolveCondActual(cond, bot, snapState, snapCounter, snapTimer, ss, i)
				cv := resolveCondValue(cond, bot, ss)
				passed := evaluateSwarmCondition(cond, bot, snapState, snapCounter, snapTimer, ss.Rng, ss, i)
				cr := swarm.ConditionResult{
					SensorName:  swarmscript.CondTypeName(cond.Type),
					Operator:    swarmscript.OpString(cond.Op),
					Threshold:   fmt.Sprintf("%d", cv),
					ActualValue: fmt.Sprintf("%d", actual),
					Passed:      passed,
				}
				if cond.Type == swarmscript.CondTrue {
					cr.SensorName = "true"
					cr.Operator = ""
					cr.Threshold = ""
					cr.ActualValue = ""
					cr.Passed = true
				}
				step.Conditions = append(step.Conditions, cr)
				if !passed {
					allMatch = false
				}
			}
			step.Matched = allMatch
			// Build rule text summary
			step.RuleText = buildRuleText(step)
			ss.DecisionTrace = append(ss.DecisionTrace, step)
		} else {
			for _, cond := range rule.Conditions {
				if !evaluateSwarmCondition(cond, bot, snapState, snapCounter, snapTimer, ss.Rng, ss, i) {
					allMatch = false
					break
				}
			}
		}

		if allMatch {
			executeSwarmAction(rule.Action, bot, ss, i)
			if ss.GPEnabled {
				bot.LastMatchedRules = append(bot.LastMatchedRules, ri)
			}
		}
	}
}

// buildRuleText creates a human-readable summary of a decision step.
func buildRuleText(step swarm.DecisionStep) string {
	s := "IF "
	for ci, c := range step.Conditions {
		if ci > 0 {
			s += " AND "
		}
		if c.SensorName == "true" {
			s += "true"
		} else {
			s += c.SensorName + " " + c.Operator + " " + c.Threshold
		}
	}
	s += " THEN " + step.ActionName
	return s
}

// resolveCondActual returns the actual sensor value for a condition (for decision trace).
func resolveCondActual(cond swarmscript.Condition, bot *swarm.SwarmBot, snapState, snapCounter, snapTimer int, ss *swarm.SwarmState, botIdx int) int {
	switch cond.Type {
	case swarmscript.CondTrue:
		return 1
	case swarmscript.CondNeighborsCount:
		return bot.NeighborCount
	case swarmscript.CondNearestDistance:
		return int(bot.NearestDist)
	case swarmscript.CondState, swarmscript.CondMyState:
		return snapState
	case swarmscript.CondCounter:
		return snapCounter
	case swarmscript.CondTimer:
		return snapTimer
	case swarmscript.CondOnEdge:
		if bot.OnEdge {
			return 1
		}
		return 0
	case swarmscript.CondReceivedMessage:
		return bot.ReceivedMsg
	case swarmscript.CondLightValue:
		return bot.LightValue
	case swarmscript.CondRandom:
		return -1 // random, cannot show
	case swarmscript.CondCarrying:
		if bot.CarryingPkg >= 0 {
			return 1
		}
		return 0
	case swarmscript.CondNearestPickupDist:
		return int(bot.NearestPickupDist)
	case swarmscript.CondNearestDropoffDist:
		return int(bot.NearestDropoffDist)
	case swarmscript.CondObstacleAhead:
		if bot.ObstacleAhead {
			return 1
		}
		return 0
	case swarmscript.CondObstacleDist:
		return int(bot.ObstacleDist)
	case swarmscript.CondValue1:
		return bot.Value1
	case swarmscript.CondValue2:
		return bot.Value2
	case swarmscript.CondTick:
		return ss.Tick
	case swarmscript.CondHeading:
		deg := int(bot.Angle * 180 / math.Pi)
		if deg < 0 {
			deg += 360
		}
		return deg
	case swarmscript.CondSpeed:
		return int(bot.Speed * 100)
	case swarmscript.CondEnergy:
		return int(bot.Energy)
	case swarmscript.CondTeam:
		return bot.Team
	default:
		return 0
	}
}

// evaluateSwarmCondition checks a single condition against bot sensors.
func evaluateSwarmCondition(cond swarmscript.Condition, bot *swarm.SwarmBot, snapState, snapCounter, snapTimer int, rng *rand.Rand, ss *swarm.SwarmState, botIdx int) bool {
	cv := resolveCondValue(cond, bot, ss)
	switch cond.Type {
	case swarmscript.CondTrue:
		return true

	case swarmscript.CondNeighborsCount:
		return compareInt(bot.NeighborCount, cond.Op, cv)

	case swarmscript.CondNearestDistance:
		return compareInt(int(bot.NearestDist), cond.Op, cv)

	case swarmscript.CondState, swarmscript.CondMyState:
		return compareInt(snapState, cond.Op, cv)

	case swarmscript.CondCounter:
		return compareInt(snapCounter, cond.Op, cv)

	case swarmscript.CondTimer:
		return compareInt(snapTimer, cond.Op, cv)

	case swarmscript.CondOnEdge:
		onEdgeVal := 0
		if bot.OnEdge {
			onEdgeVal = 1
		}
		return compareInt(onEdgeVal, cond.Op, cv)

	case swarmscript.CondReceivedMessage:
		return compareInt(bot.ReceivedMsg, cond.Op, cv)

	case swarmscript.CondLightValue:
		return compareInt(bot.LightValue, cond.Op, cv)

	case swarmscript.CondRandom:
		// random < N means N% chance
		return rng.Intn(100) < cv

	case swarmscript.CondHasLeader:
		hasLeader := 0
		if bot.FollowTargetIdx >= 0 {
			hasLeader = 1
		}
		return compareInt(hasLeader, cond.Op, cv)

	case swarmscript.CondHasFollower:
		hasFollower := 0
		if bot.FollowerIdx >= 0 {
			hasFollower = 1
		}
		return compareInt(hasFollower, cond.Op, cv)

	case swarmscript.CondChainLength:
		cl := computeChainLength(ss, botIdx)
		return compareInt(cl, cond.Op, cv)

	case swarmscript.CondNearestLEDR:
		return compareInt(int(bot.NearestLEDR), cond.Op, cv)

	case swarmscript.CondNearestLEDG:
		return compareInt(int(bot.NearestLEDG), cond.Op, cv)

	case swarmscript.CondNearestLEDB:
		return compareInt(int(bot.NearestLEDB), cond.Op, cv)

	case swarmscript.CondTick:
		return compareInt(ss.Tick, cond.Op, cv)

	case swarmscript.CondObstacleAhead:
		obsVal := 0
		if bot.ObstacleAhead {
			obsVal = 1
		}
		return compareInt(obsVal, cond.Op, cv)

	case swarmscript.CondObstacleDist:
		return compareInt(int(bot.ObstacleDist), cond.Op, cv)

	case swarmscript.CondValue1:
		return compareInt(bot.Value1, cond.Op, cv)

	case swarmscript.CondValue2:
		return compareInt(bot.Value2, cond.Op, cv)

	case swarmscript.CondCarrying:
		carryVal := 0
		if bot.CarryingPkg >= 0 {
			carryVal = 1
		}
		return compareInt(carryVal, cond.Op, cv)

	case swarmscript.CondCarryingColor:
		cc := 0
		if bot.CarryingPkg >= 0 && bot.CarryingPkg < len(ss.Packages) {
			cc = ss.Packages[bot.CarryingPkg].Color
		}
		return compareInt(cc, cond.Op, cv)

	case swarmscript.CondNearestPickupDist:
		return compareInt(int(bot.NearestPickupDist), cond.Op, cv)

	case swarmscript.CondNearestPickupColor:
		return compareInt(bot.NearestPickupColor, cond.Op, cv)

	case swarmscript.CondNearestPickupHasPkg:
		v := 0
		if bot.NearestPickupHasPkg {
			v = 1
		}
		return compareInt(v, cond.Op, cv)

	case swarmscript.CondNearestDropoffDist:
		return compareInt(int(bot.NearestDropoffDist), cond.Op, cv)

	case swarmscript.CondNearestDropoffColor:
		return compareInt(bot.NearestDropoffColor, cond.Op, cv)

	case swarmscript.CondDropoffMatch:
		v := 0
		if bot.DropoffMatch {
			v = 1
		}
		return compareInt(v, cond.Op, cv)

	case swarmscript.CondHeardPickupColor:
		return compareInt(bot.HeardPickupColor, cond.Op, cv)

	case swarmscript.CondHeardDropoffColor:
		return compareInt(bot.HeardDropoffColor, cond.Op, cv)

	case swarmscript.CondNearestMatchLEDDist:
		return compareInt(int(bot.NearestMatchLEDDist), cond.Op, cv)

	case swarmscript.CondTruckHere:
		v := 0
		if bot.TruckHere {
			v = 1
		}
		return compareInt(v, cond.Op, cv)

	case swarmscript.CondTruckPkgCount:
		return compareInt(bot.TruckPkgCount, cond.Op, cv)

	case swarmscript.CondOnRamp:
		v := 0
		if bot.OnRamp {
			v = 1
		}
		return compareInt(v, cond.Op, cv)

	case swarmscript.CondNearestTruckPkgDist:
		return compareInt(int(bot.NearestTruckPkgDist), cond.Op, cv)

	case swarmscript.CondHeardBeaconDropoff:
		v := 0
		if bot.HeardBeaconDropoffColor > 0 {
			v = 1
		}
		return compareInt(v, cond.Op, cv)

	case swarmscript.CondHeardBeaconDropoffDist:
		return compareInt(int(bot.HeardBeaconDropoffDist), cond.Op, cv)

	case swarmscript.CondExploring:
		v := 0
		if bot.ExplorationTimer > 60 {
			v = 1
		}
		return compareInt(v, cond.Op, cv)

	case swarmscript.CondWallRight:
		v := 0
		if bot.WallRight {
			v = 1
		}
		return compareInt(v, cond.Op, cv)

	case swarmscript.CondWallLeft:
		v := 0
		if bot.WallLeft {
			v = 1
		}
		return compareInt(v, cond.Op, cv)

	case swarmscript.CondPherAhead:
		return compareInt(int(bot.PherAhead*100), cond.Op, cv)

	case swarmscript.CondTeam:
		return compareInt(bot.Team, cond.Op, cv)

	case swarmscript.CondTeamScore:
		score := 0
		if bot.Team == 1 {
			score = ss.TeamAScore
		} else if bot.Team == 2 {
			score = ss.TeamBScore
		}
		return compareInt(score, cond.Op, cv)

	case swarmscript.CondEnemyScore:
		score := 0
		if bot.Team == 1 {
			score = ss.TeamBScore
		} else if bot.Team == 2 {
			score = ss.TeamAScore
		}
		return compareInt(score, cond.Op, cv)

	case swarmscript.CondBotAhead:
		return compareInt(bot.BotAhead, cond.Op, cv)
	case swarmscript.CondBotBehind:
		return compareInt(bot.BotBehind, cond.Op, cv)
	case swarmscript.CondBotLeft:
		return compareInt(bot.BotLeft, cond.Op, cv)
	case swarmscript.CondBotRight:
		return compareInt(bot.BotRight, cond.Op, cv)
	case swarmscript.CondHeading:
		deg := int(bot.Angle * 180 / math.Pi)
		if deg < 0 {
			deg += 360
		}
		return compareInt(deg, cond.Op, cv)
	case swarmscript.CondSpeed:
		return compareInt(int(bot.Speed*100), cond.Op, cv)

	case swarmscript.CondVisitedHere:
		return compareInt(swarm.BotVisitedHere(bot), cond.Op, cv)
	case swarmscript.CondVisitedAhead:
		return compareInt(swarm.BotVisitedAhead(bot), cond.Op, cv)
	case swarmscript.CondExplored:
		return compareInt(swarm.BotExploredPercent(bot), cond.Op, cv)
	case swarmscript.CondGroupCarry:
		return compareInt(bot.GroupCarry, cond.Op, cv)
	case swarmscript.CondGroupSpeed:
		return compareInt(bot.GroupSpeed, cond.Op, cv)
	case swarmscript.CondGroupSize:
		return compareInt(bot.GroupSize, cond.Op, cv)
	case swarmscript.CondSwarmCenterDist:
		return compareInt(bot.SwarmCenterDist, cond.Op, cv)
	case swarmscript.CondSwarmSpread:
		return compareInt(bot.SwarmSpreadSensor, cond.Op, cv)
	case swarmscript.CondIsolationLevel:
		return compareInt(bot.IsolationLevel, cond.Op, cv)
	case swarmscript.CondResourceGradientX:
		return compareInt(bot.ResourceGradientX, cond.Op, cv)
	case swarmscript.CondResourceGradientY:
		return compareInt(bot.ResourceGradientY, cond.Op, cv)

	case swarmscript.CondEnergy:
		return compareInt(int(bot.Energy), cond.Op, cv)
	case swarmscript.CondBotCarrying:
		return compareInt(bot.BotCarryingCount, cond.Op, cv)
	case swarmscript.CondTimeSinceDelivery:
		return compareInt(bot.TimeSinceDelivery, cond.Op, cv)
	case swarmscript.CondRecentCollision:
		return compareInt(bot.RecentCollision, cond.Op, cv)
	case swarmscript.CondNeighborMinDist:
		return compareInt(bot.NeighborMinDist, cond.Op, cv)
	case swarmscript.CondPathDist:
		return compareInt(bot.PathDist, cond.Op, cv)
	case swarmscript.CondPathAngle:
		return compareInt(bot.PathAngle, cond.Op, cv)

	// Flocking (Boids) sensors
	case swarmscript.CondFlockAlign:
		return compareInt(bot.FlockAlign, cond.Op, cv)
	case swarmscript.CondFlockCohesion:
		return compareInt(bot.FlockCohesion, cond.Op, cv)
	case swarmscript.CondFlockSeparation:
		return compareInt(bot.FlockSeparation, cond.Op, cv)

	// Dynamic Role sensors
	case swarmscript.CondRole:
		return compareInt(bot.Role, cond.Op, cv)
	case swarmscript.CondRoleDemand:
		return compareInt(bot.RoleDemand, cond.Op, cv)

	// Quorum Sensing sensors
	case swarmscript.CondVote:
		return compareInt(bot.Vote, cond.Op, cv)
	case swarmscript.CondQuorumCount:
		return compareInt(bot.QuorumCount, cond.Op, cv)
	case swarmscript.CondQuorumReached:
		return compareInt(bot.QuorumReached, cond.Op, cv)

	// Rogue Detection sensors
	case swarmscript.CondReputation:
		return compareInt(bot.Reputation, cond.Op, cv)
	case swarmscript.CondSuspectNearby:
		return compareInt(bot.SuspectNearby, cond.Op, cv)

	// Lévy-Flight sensors
	case swarmscript.CondLevyPhase:
		return compareInt(bot.LevyPhase, cond.Op, cv)
	case swarmscript.CondLevyStep:
		return compareInt(bot.LevyStep, cond.Op, cv)

	// Firefly Sync sensors
	case swarmscript.CondFlashPhase:
		return compareInt(bot.FlashPhase, cond.Op, cv)
	case swarmscript.CondFlashSync:
		return compareInt(bot.FlashSync, cond.Op, cv)

	// Collective Transport sensors
	case swarmscript.CondTransportNearby:
		return compareInt(bot.TransportNearby, cond.Op, cv)
	case swarmscript.CondTransportCount:
		return compareInt(bot.TransportCount, cond.Op, cv)

	// Vortex Swarming sensors
	case swarmscript.CondVortexStrength:
		return compareInt(bot.VortexStrength, cond.Op, cv)

	// Waggle Dance sensors
	case swarmscript.CondWaggleDancing:
		return compareInt(bot.WaggleDancing, cond.Op, cv)
	case swarmscript.CondWaggleTarget:
		return compareInt(bot.WaggleTarget, cond.Op, cv)

	// Morphogen sensors
	case swarmscript.CondMorphA:
		return compareInt(bot.MorphA, cond.Op, cv)
	case swarmscript.CondMorphH:
		return compareInt(bot.MorphH, cond.Op, cv)

	// Evasion Wave sensors
	case swarmscript.CondEvasionAlert:
		return compareInt(bot.EvasionAlert, cond.Op, cv)
	case swarmscript.CondEvasionWave:
		return compareInt(bot.EvasionWave, cond.Op, cv)

	// Slime Mold sensors
	case swarmscript.CondSlimeTrail:
		return compareInt(bot.SlimeTrail, cond.Op, cv)
	case swarmscript.CondSlimeGrad:
		return compareInt(bot.SlimeGrad, cond.Op, cv)

	// Ant Bridge sensors
	case swarmscript.CondBridgeActive:
		return compareInt(bot.BridgeActive, cond.Op, cv)
	case swarmscript.CondBridgeNearby:
		return compareInt(bot.BridgeNearby, cond.Op, cv)

	// Shape Formation sensors
	case swarmscript.CondShapeDist:
		return compareInt(bot.ShapeDist, cond.Op, cv)
	case swarmscript.CondShapeAngle:
		return compareInt(bot.ShapeAngle, cond.Op, cv)
	case swarmscript.CondShapeProgress:
		return compareInt(bot.ShapeProgress, cond.Op, cv)

	// Mexican Wave sensors
	case swarmscript.CondWaveFlash:
		return compareInt(bot.WaveFlash, cond.Op, cv)
	case swarmscript.CondWavePhase:
		return compareInt(bot.WavePhase, cond.Op, cv)

	// Shepherd-Flock sensors
	case swarmscript.CondShepherdRole:
		return compareInt(bot.ShepherdRole, cond.Op, cv)
	case swarmscript.CondShepherdDist:
		return compareInt(bot.ShepherdDist, cond.Op, cv)
	case swarmscript.CondFlockToTarget:
		return compareInt(bot.FlockToTarget, cond.Op, cv)

	// PSO sensors
	case swarmscript.CondPSOFitness:
		return compareInt(bot.PSOFitness, cond.Op, cv)
	case swarmscript.CondPSOBest:
		return compareInt(bot.PSOBest, cond.Op, cv)
	case swarmscript.CondPSOGlobalDist:
		return compareInt(bot.PSOGlobalDist, cond.Op, cv)

	// Predator-Prey sensors
	case swarmscript.CondPredRole:
		return compareInt(bot.PredRole, cond.Op, cv)
	case swarmscript.CondPreyDist:
		return compareInt(bot.PreyDist, cond.Op, cv)
	case swarmscript.CondPredCatches:
		return compareInt(bot.PredCatches, cond.Op, cv)

	// Magnetic Chain sensors
	case swarmscript.CondMagChainLen:
		return compareInt(bot.MagChainLen, cond.Op, cv)
	case swarmscript.CondMagLinked:
		return compareInt(bot.MagLinked, cond.Op, cv)
	case swarmscript.CondMagAlign:
		return compareInt(bot.MagAlign, cond.Op, cv)

	// Cell Division sensors
	case swarmscript.CondDivGroup:
		return compareInt(bot.DivGroup, cond.Op, cv)
	case swarmscript.CondDivPhase:
		return compareInt(bot.DivPhase, cond.Op, cv)
	case swarmscript.CondDivDist:
		return compareInt(bot.DivDist, cond.Op, cv)
	// V-Formation conditions
	case swarmscript.CondVFormPos:
		return compareInt(bot.VFormPos, cond.Op, cv)
	case swarmscript.CondVFormDraft:
		return compareInt(bot.VFormDraft, cond.Op, cv)
	case swarmscript.CondVFormLeader:
		return compareInt(bot.VFormLeader, cond.Op, cv)
	// Brood Sorting conditions
	case swarmscript.CondBroodCarrying:
		return compareInt(bot.BroodCarrying, cond.Op, cv)
	case swarmscript.CondBroodItemColor:
		return compareInt(bot.BroodItemColor, cond.Op, cv)
	case swarmscript.CondBroodDensity:
		return compareInt(bot.BroodDensity, cond.Op, cv)
	case swarmscript.CondBroodSameColor:
		return compareInt(bot.BroodSameColor, cond.Op, cv)
	// Jellyfish Pulse conditions
	case swarmscript.CondJellyPhase:
		return compareInt(bot.JellyPhase, cond.Op, cv)
	case swarmscript.CondJellyExpanding:
		return compareInt(bot.JellyExpanding, cond.Op, cv)
	case swarmscript.CondJellyRadius:
		return compareInt(bot.JellyRadius, cond.Op, cv)
	// Immune System conditions
	case swarmscript.CondImmuneRole:
		return compareInt(bot.ImmuneRole, cond.Op, cv)
	case swarmscript.CondImmuneAlert:
		return compareInt(bot.ImmuneAlert, cond.Op, cv)
	case swarmscript.CondImmunePathDist:
		return compareInt(bot.ImmunePathDist, cond.Op, cv)
	// Gravitational N-Body conditions
	case swarmscript.CondGravMass:
		return compareInt(bot.GravMass, cond.Op, cv)
	case swarmscript.CondGravForce:
		return compareInt(bot.GravForce, cond.Op, cv)
	case swarmscript.CondGravNearHeavy:
		return compareInt(bot.GravNearHeavy, cond.Op, cv)
	// Crystallization conditions
	case swarmscript.CondCrystalNeigh:
		return compareInt(bot.CrystalNeigh, cond.Op, cv)
	case swarmscript.CondCrystalDefect:
		return compareInt(bot.CrystalDefect, cond.Op, cv)
	case swarmscript.CondCrystalSettled:
		return compareInt(bot.CrystalSettled, cond.Op, cv)
	// Amoeba conditions
	case swarmscript.CondAmoebaDistCenter:
		return compareInt(bot.AmoebaDistCenter, cond.Op, cv)
	case swarmscript.CondAmoebaSkin:
		return compareInt(bot.AmoebaSkin, cond.Op, cv)
	case swarmscript.CondAmoebaPseudo:
		return compareInt(bot.AmoebaPseudo, cond.Op, cv)
	// ACO conditions
	case swarmscript.CondACOTrail:
		return compareInt(bot.ACOTrail, cond.Op, cv)
	case swarmscript.CondACOGrad:
		return compareInt(bot.ACOGrad, cond.Op, cv)
	// Bacterial Foraging conditions
	case swarmscript.CondBFOHealth:
		return compareInt(bot.BFOHealth, cond.Op, cv)
	case swarmscript.CondBFOSwimming:
		return compareInt(bot.BFOSwimming, cond.Op, cv)
	case swarmscript.CondBFONutrient:
		return compareInt(bot.BFONutrient, cond.Op, cv)
	// Grey Wolf Optimizer conditions
	case swarmscript.CondGWORank:
		return compareInt(bot.GWORank, cond.Op, cv)
	case swarmscript.CondGWOFitness:
		return compareInt(bot.GWOFitness, cond.Op, cv)
	case swarmscript.CondGWOAlphaDist:
		return compareInt(bot.GWOAlphaDist, cond.Op, cv)
	// Whale Optimization conditions
	case swarmscript.CondWOAPhase:
		return compareInt(bot.WOAPhase, cond.Op, cv)
	case swarmscript.CondWOAFitness:
		return compareInt(bot.WOAFitness, cond.Op, cv)
	case swarmscript.CondWOABestDist:
		return compareInt(bot.WOABestDist, cond.Op, cv)
	// Moth-Flame Optimization conditions
	case swarmscript.CondMFOFlame:
		return compareInt(bot.MFOFlame, cond.Op, cv)
	case swarmscript.CondMFOFitness:
		return compareInt(bot.MFOFitness, cond.Op, cv)
	case swarmscript.CondMFOFlameDist:
		return compareInt(bot.MFOFlameDist, cond.Op, cv)
	// Cuckoo Search conditions
	case swarmscript.CondCuckooFitness:
		return compareInt(bot.CuckooFitness, cond.Op, cv)
	case swarmscript.CondCuckooNestAge:
		return compareInt(bot.CuckooNestAge, cond.Op, cv)
	case swarmscript.CondCuckooBest:
		return compareInt(bot.CuckooBest, cond.Op, cv)
	// Differential Evolution conditions
	case swarmscript.CondDEFitness:
		return compareInt(bot.DEFitness, cond.Op, cv)
	case swarmscript.CondDEBestDist:
		return compareInt(bot.DEBestDist, cond.Op, cv)
	case swarmscript.CondDEPhase:
		return compareInt(bot.DEPhase, cond.Op, cv)
	// Artificial Bee Colony conditions
	case swarmscript.CondABCFitness:
		return compareInt(bot.ABCFitness, cond.Op, cv)
	case swarmscript.CondABCRole:
		return compareInt(bot.ABCRole, cond.Op, cv)
	case swarmscript.CondABCBestDist:
		return compareInt(bot.ABCBestDist, cond.Op, cv)
	// Harmony Search Optimization conditions
	case swarmscript.CondHSOFitness:
		return compareInt(bot.HSOFitness, cond.Op, cv)
	case swarmscript.CondHSOPhase:
		return compareInt(bot.HSOPhase, cond.Op, cv)
	case swarmscript.CondHSOBestDist:
		return compareInt(bot.HSOBestDist, cond.Op, cv)
	// Bat Algorithm conditions
	case swarmscript.CondBatLoud:
		return compareInt(bot.BatLoud, cond.Op, cv)
	case swarmscript.CondBatPulse:
		return compareInt(bot.BatPulse, cond.Op, cv)
	case swarmscript.CondBatFitness:
		return compareInt(bot.BatFitness, cond.Op, cv)
	case swarmscript.CondBatBestDist:
		return compareInt(bot.BatBestDist, cond.Op, cv)
	// Salp Swarm Algorithm conditions
	case swarmscript.CondSSARole:
		return compareInt(bot.SSARole, cond.Op, cv)
	case swarmscript.CondSSAFitness:
		return compareInt(bot.SSAFitness, cond.Op, cv)
	case swarmscript.CondSSAFoodDist:
		return compareInt(bot.SSAFoodDist, cond.Op, cv)
	// Gravitational Search Algorithm conditions
	case swarmscript.CondGSAMass:
		return compareInt(bot.GSAMass, cond.Op, cv)
	case swarmscript.CondGSAForce:
		return compareInt(bot.GSAForce, cond.Op, cv)
	case swarmscript.CondGSABestDist:
		return compareInt(bot.GSABestDist, cond.Op, cv)
	// Flower Pollination Algorithm conditions
	case swarmscript.CondFPAFitness:
		return compareInt(bot.FPAFitness, cond.Op, cv)
	case swarmscript.CondFPAType:
		return compareInt(bot.FPAType, cond.Op, cv)
	case swarmscript.CondFPABestDist:
		return compareInt(bot.FPABestDist, cond.Op, cv)
	// Harris Hawks Optimization conditions
	case swarmscript.CondHHOPhase:
		return compareInt(bot.HHOPhase, cond.Op, cv)
	case swarmscript.CondHHOFitness:
		return compareInt(bot.HHOFitness, cond.Op, cv)
	case swarmscript.CondHHOBestDist:
		return compareInt(bot.HHOBestDist, cond.Op, cv)
	// Simulated Annealing conditions
	case swarmscript.CondSAFitness:
		return compareInt(bot.SAFitness, cond.Op, cv)
	case swarmscript.CondSATemp:
		return compareInt(bot.SATemp, cond.Op, cv)
	case swarmscript.CondSABestDist:
		return compareInt(bot.SABestDist, cond.Op, cv)
	// Aquila Optimizer conditions
	case swarmscript.CondAOPhase:
		return compareInt(bot.AOPhase, cond.Op, cv)
	case swarmscript.CondAOFitness:
		return compareInt(bot.AOFitness, cond.Op, cv)
	case swarmscript.CondAOBestDist:
		return compareInt(bot.AOBestDist, cond.Op, cv)
	// Sine Cosine Algorithm conditions
	case swarmscript.CondSCAFitness:
		return compareInt(bot.SCAFitness, cond.Op, cv)
	case swarmscript.CondSCAPhase:
		return compareInt(bot.SCAPhase, cond.Op, cv)
	case swarmscript.CondSCABestDist:
		return compareInt(bot.SCABestDist, cond.Op, cv)
	// Dragonfly Algorithm conditions
	case swarmscript.CondDAFitness:
		return compareInt(bot.DAFitness, cond.Op, cv)
	case swarmscript.CondDARole:
		return compareInt(bot.DARole, cond.Op, cv)
	case swarmscript.CondDAFoodDist:
		return compareInt(bot.DAFoodDist, cond.Op, cv)
	// Teaching-Learning-Based Optimization conditions
	case swarmscript.CondTLBOFitness:
		return compareInt(bot.TLBOFitness, cond.Op, cv)
	case swarmscript.CondTLBOPhase:
		return compareInt(bot.TLBOPhase, cond.Op, cv)
	case swarmscript.CondTLBOTeacherDist:
		return compareInt(bot.TLBOTeacherDist, cond.Op, cv)
	// Equilibrium Optimizer conditions
	case swarmscript.CondEOFitness:
		return compareInt(bot.EOFitness, cond.Op, cv)
	case swarmscript.CondEOPhase:
		return compareInt(bot.EOPhase, cond.Op, cv)
	case swarmscript.CondEOEquilDist:
		return compareInt(bot.EOEquilDist, cond.Op, cv)

	// Jaya Algorithm conditions
	case swarmscript.CondJayaFitness:
		return compareInt(bot.JayaFitness, cond.Op, cv)
	case swarmscript.CondJayaBestDist:
		return compareInt(bot.JayaBestDist, cond.Op, cv)
	case swarmscript.CondJayaWorstDist:
		return compareInt(bot.JayaWorstDist, cond.Op, cv)
	}

	return false
}


// resolveCondValue returns the effective comparison value for a condition,
// using per-bot evolved parameters when IsParamRef and EvolutionOn.
func resolveCondValue(cond swarmscript.Condition, bot *swarm.SwarmBot, ss *swarm.SwarmState) int {
	if cond.IsParamRef && ss.EvolutionOn {
		return int(bot.ParamValues[cond.ParamIdx])
	}
	return cond.Value
}

// compareInt compares two ints with the given operator.
func compareInt(a int, op swarmscript.ConditionOp, b int) bool {
	switch op {
	case swarmscript.OpGT:
		return a > b
	case swarmscript.OpLT:
		return a < b
	case swarmscript.OpEQ:
		return a == b
	}
	return false
}

// executeSwarmAction performs an action on a bot.
func executeSwarmAction(act swarmscript.Action, bot *swarm.SwarmBot, ss *swarm.SwarmState, botIdx int) {
	switch act.Type {
	case swarmscript.ActMoveForward:
		bot.Speed = swarm.SwarmBotSpeed

	case swarmscript.ActTurnLeft:
		bot.Angle -= float64(act.Param1) * math.Pi / 180.0

	case swarmscript.ActTurnRight:
		bot.Angle += float64(act.Param1) * math.Pi / 180.0

	case swarmscript.ActTurnToNearest:
		if bot.NeighborCount > 0 {
			bot.Angle = bot.NearestAngle
		}

	case swarmscript.ActTurnFromNearest:
		if bot.NeighborCount > 0 {
			bot.Angle = bot.NearestAngle + math.Pi
		}

	case swarmscript.ActTurnToCenter:
		if bot.NeighborCount > 0 {
			bot.Angle = math.Atan2(bot.AvgNeighborY, bot.AvgNeighborX)
		}

	case swarmscript.ActTurnToLight:
		if ss.Light.Active {
			dx, dy := swarm.NeighborDelta(bot.X, bot.Y, ss.Light.X, ss.Light.Y, ss)
			bot.Angle = math.Atan2(dy, dx)
		}

	case swarmscript.ActTurnRandom:
		bot.Angle = ss.Rng.Float64() * 2 * math.Pi

	case swarmscript.ActStop:
		bot.Speed = 0

	case swarmscript.ActSetState:
		bot.State = act.Param1

	case swarmscript.ActSetCounter:
		bot.Counter = act.Param1

	case swarmscript.ActIncCounter:
		bot.Counter++

	case swarmscript.ActDecCounter:
		bot.Counter--

	case swarmscript.ActSetLED:
		r := act.Param1
		g := act.Param2
		b := act.Param3
		if r < 0 {
			r = 0
		}
		if r > 255 {
			r = 255
		}
		if g < 0 {
			g = 0
		}
		if g > 255 {
			g = 255
		}
		if b < 0 {
			b = 0
		}
		if b > 255 {
			b = 255
		}
		bot.LEDColor = [3]uint8{uint8(r), uint8(g), uint8(b)}

	case swarmscript.ActSendMessage:
		bot.PendingMsg = act.Param1

	case swarmscript.ActSetTimer:
		bot.Timer = act.Param1

	case swarmscript.ActFollowNearest:
		// Find nearest bot within sensor range that has no follower (= chain tail or lone bot)
		if bot.FollowTargetIdx >= 0 {
			return // already following someone
		}
		if bot.StuckCooldown > 0 {
			return // in anti-stuck cooldown, forced solo movement
		}
		// Scan ALL neighbors for best available target (not just single nearest)
		candidateIDs := ss.Hash.Query(bot.X, bot.Y, swarm.SwarmSensorRange)
		bestIdx := -1
		bestDist := 1e9
		for _, cid := range candidateIDs {
			if cid == botIdx || cid < 0 || cid >= len(ss.Bots) {
				continue
			}
			other := &ss.Bots[cid]
			// Must not already have a follower
			if other.FollowerIdx >= 0 {
				continue
			}
			dx, dy := swarm.NeighborDelta(bot.X, bot.Y, other.X, other.Y, ss)
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist > swarm.SwarmSensorRange {
				continue
			}
			if dist < bestDist {
				bestDist = dist
				bestIdx = cid
			}
		}
		if bestIdx >= 0 {
			// Cycle detection: walk from target through its leader chain
			wouldCycle := false
			cur := bestIdx
			for steps := 0; steps < len(ss.Bots); steps++ {
				if cur == botIdx {
					wouldCycle = true
					break
				}
				next := ss.Bots[cur].FollowTargetIdx
				if next < 0 || next >= len(ss.Bots) {
					break
				}
				cur = next
			}
			if !wouldCycle {
				bot.FollowTargetIdx = bestIdx
				ss.Bots[bestIdx].FollowerIdx = botIdx
				logger.InfoBot(botIdx, "FOLLOW", "Bot #%d following Bot #%d", botIdx, bestIdx)
			}
		}

	case swarmscript.ActUnfollow:
		if bot.FollowTargetIdx >= 0 && bot.FollowTargetIdx < len(ss.Bots) {
			logger.InfoBot(botIdx, "FOLLOW", "Bot #%d unfollowed Bot #%d", botIdx, bot.FollowTargetIdx)
			// Clear leader's follower reference
			ss.Bots[bot.FollowTargetIdx].FollowerIdx = -1
		}
		bot.FollowTargetIdx = -1

	case swarmscript.ActTurnAwayObstacle:
		if bot.ObstacleAhead {
			// Turn 90° + random 0-45° in a random direction to avoid corner loops
			turn := math.Pi/2 + ss.Rng.Float64()*math.Pi/4
			if ss.Rng.Intn(2) == 0 {
				turn = -turn
			}
			bot.Angle += turn
		}

	case swarmscript.ActMoveForwardSlow:
		bot.Speed = swarm.SwarmBotSpeed * 0.5

	case swarmscript.ActSetValue1:
		bot.Value1 = act.Param1

	case swarmscript.ActSetValue2:
		bot.Value2 = act.Param1

	case swarmscript.ActCopyNearestLED:
		if ss.DeliveryOn {
			// In delivery mode, use extended range and only copy delivery colors (not white)
			copyRange := swarm.SwarmDeliverySensorRange
			candidates := ss.Hash.Query(bot.X, bot.Y, copyRange)
			bestIdx := -1
			bestDist := 1e9
			for _, cid := range candidates {
				if cid == botIdx || cid < 0 || cid >= len(ss.Bots) {
					continue
				}
				other := &ss.Bots[cid]
				// Only copy from bots that have a delivery color (not white/default)
				if ledToDeliveryColor(other.LEDColor) == 0 {
					continue
				}
				dx, dy := swarm.NeighborDelta(bot.X, bot.Y, other.X, other.Y, ss)
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist <= copyRange && dist < bestDist {
					bestDist = dist
					bestIdx = cid
				}
			}
			if bestIdx >= 0 {
				bot.LEDColor = ss.Bots[bestIdx].LEDColor
			}
		} else {
			// Standard behavior: copy from nearest neighbor (60px)
			if bot.NearestIdx >= 0 && bot.NearestIdx < len(ss.Bots) {
				other := &ss.Bots[bot.NearestIdx]
				bot.LEDColor = other.LEDColor
			}
		}

	case swarmscript.ActPickup:
		if !ss.DeliveryOn || bot.CarryingPkg >= 0 {
			return
		}
		// Truck packages: crane-based pickup at ramp edge
		// Bot waits at the ramp edge (OnRamp=true), a crane transfers the package
		if ss.TruckToggle && ss.TruckState != nil && bot.OnRamp &&
			ss.TruckState.CurrentTruck != nil && ss.TruckState.CurrentTruck.Phase == swarm.TruckParked {
			t := ss.TruckState.CurrentTruck
			// Find first unpicked package (crane handles transfer — no distance check)
			for tpi := range t.Packages {
				tpkg := &t.Packages[tpi]
				if tpkg.PickedUp {
					continue
				}
				tpkg.PickedUp = true
				ss.TruckState.TotalPkgs++
				// Convert to DeliveryPackage
				dpkg := swarm.DeliveryPackage{
					Color:      tpkg.Color,
					CarriedBy:  botIdx,
					X:          bot.X,
					Y:          bot.Y,
					Active:     true,
					PickupTick: ss.Tick,
				}
				ss.Packages = append(ss.Packages, dpkg)
				bot.CarryingPkg = len(ss.Packages) - 1
				bot.Stats.TotalPickups++
				// Emit pickup event for particles (at ramp edge where bot is)
				ss.DeliveryEvents = append(ss.DeliveryEvents, swarm.SwarmDeliveryEvent{
					X: bot.X, Y: bot.Y,
					Color: tpkg.Color, IsPickup: true,
				})
				// Stats tracker pickup event
				if ss.StatsTracker != nil {
					ss.StatsTracker.AddPickupEvent(botIdx, swarm.DeliveryColorName(tpkg.Color))
					ss.StatsTracker.RecordActionAt(bot.X, bot.Y, ss.ArenaW, ss.ArenaH)
				}
				// Turn away from ramp (toward arena interior) so bot leaves immediately
				bot.Angle = (ss.Rng.Float64() - 0.5) * math.Pi / 2 // roughly rightward ±45°
				bot.Speed = swarm.SwarmBotSpeed
				logger.InfoBot(botIdx, "TRUCK", "Bot #%d crane-pickup %s from truck", botIdx, swarm.DeliveryColorName(tpkg.Color))
				return
			}
		}
		// Check ground packages first (closer interaction)
		for pi := range ss.Packages {
			pkg := &ss.Packages[pi]
			if !pkg.Active || pkg.CarriedBy >= 0 {
				continue
			}
			pdx, pdy := swarm.NeighborDelta(bot.X, bot.Y, pkg.X, pkg.Y, ss)
			if math.Sqrt(pdx*pdx+pdy*pdy) < 20 {
				pkg.CarriedBy = botIdx
				pkg.OnGround = false
				pkg.PickupTick = ss.Tick
				bot.CarryingPkg = pi
				bot.Stats.TotalPickups++
				// Stats tracker pickup event
				if ss.StatsTracker != nil {
					ss.StatsTracker.AddPickupEvent(botIdx, swarm.DeliveryColorName(pkg.Color))
					ss.StatsTracker.RecordActionAt(bot.X, bot.Y, ss.ArenaW, ss.ArenaH)
				}
				logger.InfoBot(botIdx, "DELIVERY", "Bot #%d picked up %s package", botIdx, swarm.DeliveryColorName(pkg.Color))
				// Emit pickup event for particles
				ss.DeliveryEvents = append(ss.DeliveryEvents, swarm.SwarmDeliveryEvent{
					X: pkg.X, Y: pkg.Y,
					Color: pkg.Color, IsPickup: true,
				})
				// Mark station as empty if this was a station package
				for si := range ss.Stations {
					st := &ss.Stations[si]
					if st.IsPickup && st.HasPackage && st.Color == pkg.Color {
						sdx, sdy := swarm.NeighborDelta(pkg.X, pkg.Y, st.X, st.Y, ss)
						if math.Sqrt(sdx*sdx+sdy*sdy) < 30 {
							st.HasPackage = false
							break
						}
					}
				}
				return
			}
		}

	case swarmscript.ActDrop:
		if !ss.DeliveryOn || bot.CarryingPkg < 0 || bot.CarryingPkg >= len(ss.Packages) {
			return
		}
		pkg := &ss.Packages[bot.CarryingPkg]
		// DROP safety: only allow within 30px of a dropoff station
		nearStation := false
		delivered := false
		for si := range ss.Stations {
			st := &ss.Stations[si]
			if st.IsPickup {
				continue
			}
			sdx, sdy := swarm.NeighborDelta(bot.X, bot.Y, st.X, st.Y, ss)
			stDist := math.Sqrt(sdx*sdx + sdy*sdy)
			if stDist < 30 {
				nearStation = true
				correct := pkg.Color == st.Color
				if !correct {
					logger.WarnBot(botIdx, "DELIVERY", "Bot #%d dropped %s at wrong %s station",
						botIdx, swarm.DeliveryColorName(pkg.Color), swarm.DeliveryColorName(st.Color))
				}
				// Delivery!
				ss.DeliveryStats.TotalDelivered++
				if correct {
					ss.DeliveryStats.CorrectDelivered++
					st.FlashOK = true
				} else {
					ss.DeliveryStats.WrongDelivered++
					st.FlashOK = false
				}
				st.FlashTimer = 30
				st.DeliverCount++
				ss.DeliveryStats.ColorDelivered[pkg.Color]++
				// Emit score popup
				scoreText := "+5"
				scoreColor := [3]uint8{255, 80, 80}
				if correct {
					scoreText = "+10"
					scoreColor = [3]uint8{80, 255, 80}
				}
				ss.ScorePopups = append(ss.ScorePopups, swarm.ScorePopup{
					X: st.X + 35, Y: st.Y,
					Text: scoreText, Timer: 60, Color: scoreColor,
				})
				// Emit delivery event for particles
				ss.DeliveryEvents = append(ss.DeliveryEvents, swarm.SwarmDeliveryEvent{
					X: st.X, Y: st.Y,
					Color: pkg.Color, Correct: correct,
				})
				deliveryTime := ss.Tick - pkg.PickupTick
				if deliveryTime > 0 {
					ss.DeliveryStats.DeliveryTimes = append(ss.DeliveryStats.DeliveryTimes, deliveryTime)
				}
				bot.Stats.TotalDeliveries++
				bot.TimeSinceDelivery = 0 // reset time-since-delivery sensor
				if correct {
					bot.Stats.CorrectDeliveries++
				} else {
					bot.Stats.WrongDeliveries++
				}
				if deliveryTime > 0 {
					bot.Stats.DeliveryTimes = append(bot.Stats.DeliveryTimes, deliveryTime)
				}
				if correct {
					logger.InfoBot(botIdx, "DELIVERY", "Bot #%d delivered %s CORRECT (%d ticks)", botIdx, swarm.DeliveryColorName(pkg.Color), deliveryTime)
				} else {
					logger.WarnBot(botIdx, "DELIVERY", "Bot #%d delivered %s WRONG (%d ticks)", botIdx, swarm.DeliveryColorName(pkg.Color), deliveryTime)
				}
				// Stats tracker
				if ss.StatsTracker != nil {
					ss.StatsTracker.RecordDelivery(correct, bot.Team)
					ss.StatsTracker.AddDeliveryEvent(botIdx, swarm.DeliveryColorName(pkg.Color), correct, deliveryTime)
					ss.StatsTracker.RecordActionAt(bot.X, bot.Y, ss.ArenaW, ss.ArenaH)
				}
				// Truck scoring
				if ss.TruckToggle && ss.TruckState != nil {
					ts := ss.TruckState
					ts.DeliveredPkgs++
					if correct {
						ts.Score += 10
						ts.CorrectPkgs++
					} else {
						ts.Score += 3
						ts.WrongPkgs++
					}
				}
				// Evolution fitness
				if ss.EvolutionOn {
					if correct {
						bot.Fitness += 10
					} else {
						bot.Fitness += 3
					}
				}
				// Team scoring
				if ss.TeamsEnabled {
					if bot.Team == 1 {
						ss.TeamAScore++
					} else if bot.Team == 2 {
						ss.TeamBScore++
					}
				}
				// Deactivate package, schedule respawn at its pickup station
				pkg.Active = false
				pkg.CarriedBy = -1
				bot.CarryingPkg = -1
				// Find matching pickup and schedule respawn
				for psi := range ss.Stations {
					pst := &ss.Stations[psi]
					if pst.IsPickup && pst.Color == pkg.Color {
						pst.RespawnIn = 100
						break
					}
				}
				delivered = true
				break
			}
		}
		if !delivered && !nearStation {
			// Not near any station — ignore DROP, log warning
			logger.WarnBot(botIdx, "DELIVERY", "Bot #%d tried DROP without station nearby", botIdx)
		}

	case swarmscript.ActTurnToPickup:
		if !ss.DeliveryOn {
			return
		}
		if bot.NearestPickupIdx >= 0 && bot.NearestPickupIdx < len(ss.Stations) {
			st := &ss.Stations[bot.NearestPickupIdx]
			dx, dy := swarm.NeighborDelta(bot.X, bot.Y, st.X, st.Y, ss)
			bot.Angle = math.Atan2(dy, dx)
		}

	case swarmscript.ActTurnToDropoff:
		if !ss.DeliveryOn {
			return
		}
		if bot.NearestDropoffIdx >= 0 && bot.NearestDropoffIdx < len(ss.Stations) {
			st := &ss.Stations[bot.NearestDropoffIdx]
			dx, dy := swarm.NeighborDelta(bot.X, bot.Y, st.X, st.Y, ss)
			bot.Angle = math.Atan2(dy, dx)
		}

	case swarmscript.ActTurnToMatchingDropoff:
		if !ss.DeliveryOn || bot.CarryingPkg < 0 {
			return
		}
		pkg := &ss.Packages[bot.CarryingPkg]
		// Find nearest visible dropoff matching package color (beacon range for carrying bots)
		scanRange := swarm.SwarmBeaconRange
		bestDist := 1e9
		bestAngle := bot.Angle
		for si := range ss.Stations {
			st := &ss.Stations[si]
			if st.IsPickup || st.Color != pkg.Color {
				continue
			}
			dx, dy := swarm.NeighborDelta(bot.X, bot.Y, st.X, st.Y, ss)
			d := math.Sqrt(dx*dx + dy*dy)
			if d <= scanRange && d < bestDist {
				bestDist = d
				bestAngle = math.Atan2(dy, dx)
			}
		}
		if bestDist < 1e9 {
			bot.Angle = bestAngle
		}

	case swarmscript.ActSendPickup:
		if !ss.DeliveryOn {
			return
		}
		// Encode: 10 + color (11-14)
		color := act.Param1
		if bot.NearestPickupColor > 0 {
			color = bot.NearestPickupColor
		}
		if color >= 1 && color <= 4 {
			bot.PendingMsg = 10 + color
		}

	case swarmscript.ActSendDropoff:
		if !ss.DeliveryOn {
			return
		}
		// Encode: 20 + color (21-24)
		color := act.Param1
		if bot.NearestDropoffColor > 0 {
			color = bot.NearestDropoffColor
		}
		if color >= 1 && color <= 4 {
			bot.PendingMsg = 20 + color
		}

	case swarmscript.ActTurnToHeardPickup:
		if !ss.DeliveryOn || bot.HeardPickupColor == 0 {
			return
		}
		bot.Angle = bot.HeardPickupAngle

	case swarmscript.ActTurnToHeardDropoff:
		if !ss.DeliveryOn || bot.HeardDropoffColor == 0 {
			return
		}
		bot.Angle = bot.HeardDropoffAngle

	case swarmscript.ActTurnToMatchingLED:
		// Turn toward the nearest bot whose LED matches carrying color
		if !ss.DeliveryOn || bot.NearestMatchLEDDist >= 999 {
			return
		}
		bot.Angle = bot.NearestMatchLEDAngle

	case swarmscript.ActSetLEDPickupColor:
		// Set LED to the delivery color of nearest visible pickup station
		if !ss.DeliveryOn || bot.NearestPickupIdx < 0 || bot.NearestPickupIdx >= len(ss.Stations) {
			return
		}
		r, g, b := deliveryColorRGB(ss.Stations[bot.NearestPickupIdx].Color)
		bot.LEDColor = [3]uint8{r, g, b}

	case swarmscript.ActSetLEDDropoffColor:
		// Set LED to the delivery color of nearest visible dropoff station
		if !ss.DeliveryOn || bot.NearestDropoffIdx < 0 || bot.NearestDropoffIdx >= len(ss.Stations) {
			return
		}
		r, g, b := deliveryColorRGB(ss.Stations[bot.NearestDropoffIdx].Color)
		bot.LEDColor = [3]uint8{r, g, b}

	case swarmscript.ActTurnToRamp:
		if !ss.TruckToggle || ss.TruckState == nil {
			return
		}
		// Ramp semaphore: limit concurrent non-carrying bots on ramp
		if !bot.OnRamp {
			if bot.RampCooldown > 0 {
				// Cooldown active — turn random instead
				bot.Angle += (ss.Rng.Float64() - 0.5) * math.Pi
				bot.Speed = swarm.SwarmBotSpeed
				return
			}
			maxBots := ss.RampMaxBots
			if maxBots <= 0 {
				maxBots = 3
			}
			if ss.RampBotCount >= maxBots {
				// Ramp full — turn random, cooldown 60 ticks
				bot.Angle += (ss.Rng.Float64() - 0.5) * math.Pi
				bot.Speed = swarm.SwarmBotSpeed
				bot.RampCooldown = 60
				return
			}
		}
		ts := ss.TruckState
		// Target the right edge of the ramp, spread bots along Y axis
		cx := ts.RampX + ts.RampW + 20 // just outside ramp right edge
		// Distribute bots evenly along ramp height (use bot index for offset)
		slots := 20
		slot := botIdx % slots
		yFrac := (float64(slot) + 0.5) / float64(slots) // 0.025 .. 0.975
		cy := ts.RampY + ts.RampH*0.1 + ts.RampH*0.8*yFrac
		dx := cx - bot.X
		dy := cy - bot.Y
		bot.Angle = math.Atan2(dy, dx)

	case swarmscript.ActTurnToTruckPkg:
		if !ss.TruckToggle || ss.TruckState == nil || ss.TruckState.CurrentTruck == nil {
			return
		}
		t := ss.TruckState.CurrentTruck
		if t.Phase != swarm.TruckParked || bot.NearestTruckPkgIdx < 0 {
			return
		}
		pkg := &t.Packages[bot.NearestTruckPkgIdx]
		wpx := t.X + pkg.RelX + 18 + 4
		wpy := t.Y + pkg.RelY + 4
		dx := wpx - bot.X
		dy := wpy - bot.Y
		bot.Angle = math.Atan2(dy, dx)

	case swarmscript.ActTurnToBeaconDropoff:
		// Turn toward nearest beacon-heard matching dropoff station
		if !ss.DeliveryOn || bot.HeardBeaconDropoffColor == 0 {
			return
		}
		bot.Angle = bot.HeardBeaconDropoffAngle

	case swarmscript.ActSpiralFwd:
		// Spiral outward to explore arena when lost
		bot.ExplorationAngle += 0.03
		bot.Angle += bot.ExplorationAngle * 0.1
		bot.Speed = swarm.SwarmBotSpeed
		// If near edge, reverse spiral direction to bounce back into arena
		margin := 40.0
		if bot.X < margin || bot.X > ss.ArenaW-margin ||
			bot.Y < margin || bot.Y > ss.ArenaH-margin {
			bot.ExplorationAngle = -bot.ExplorationAngle
			bot.Angle += math.Pi * 0.3 // partial turn inward
		}

	case swarmscript.ActWallFollowRight:
		// Right-hand rule: keep wall on right side
		if bot.ObstacleAhead {
			bot.Angle -= math.Pi / 2 // wall in front → turn left 90°
		} else if !bot.WallRight {
			bot.Angle += math.Pi / 2 // lost wall on right → turn right 90° to refind
		}
		bot.Speed = swarm.SwarmBotSpeed

	case swarmscript.ActWallFollowLeft:
		// Left-hand rule: keep wall on left side
		if bot.ObstacleAhead {
			bot.Angle += math.Pi / 2 // wall in front → turn right 90°
		} else if !bot.WallLeft {
			bot.Angle -= math.Pi / 2 // lost wall on left → turn left 90° to refind
		}
		bot.Speed = swarm.SwarmBotSpeed

	case swarmscript.ActFollowPheromone:
		// Follow pheromone gradient uphill
		if ss.PherGrid != nil {
			gx, gy := ss.PherGrid.Gradient(bot.X, bot.Y)
			if gx != 0 || gy != 0 {
				bot.Angle = math.Atan2(gy, gx)
			}
		}
		bot.Speed = swarm.SwarmBotSpeed

	case swarmscript.ActFollowPath:
		// Follow computed A* path toward next waypoint
		if ss.AStarOn && ss.AStar != nil {
			swarm.FollowPath(bot, ss, botIdx)
		} else {
			bot.Speed = swarm.SwarmBotSpeed
		}

	case swarmscript.ActFlock:
		// Apply all three Reynolds rules (separation + alignment + cohesion)
		swarm.ApplyFlock(bot, ss, botIdx)

	case swarmscript.ActAlign:
		// Align heading with neighbors
		swarm.ApplyAlign(bot, ss, botIdx)

	case swarmscript.ActCohere:
		// Steer toward neighbor center of mass
		swarm.ApplyCohere(bot, ss, botIdx)

	case swarmscript.ActBecomeScout:
		swarm.SetRole(bot, swarm.BotRoleScout)

	case swarmscript.ActBecomeWorker:
		swarm.SetRole(bot, swarm.BotRoleWorker)

	case swarmscript.ActBecomeGuard:
		swarm.SetRole(bot, swarm.BotRoleGuard)

	case swarmscript.ActVote:
		bot.Vote = act.Param1

	case swarmscript.ActFlagRogue:
		swarm.FlagRogue(bot, ss, botIdx)

	case swarmscript.ActLevyWalk:
		swarm.ApplyLevyWalk(bot, ss, botIdx)

	case swarmscript.ActFlash:
		swarm.ApplyFlash(bot, ss, botIdx)

	case swarmscript.ActAssistTransport:
		swarm.ApplyAssistTransport(bot, ss, botIdx)

	case swarmscript.ActVortex:
		swarm.ApplyVortex(bot, ss, botIdx)

	case swarmscript.ActWaggleDance:
		swarm.ApplyWaggleDance(bot, ss, botIdx)

	case swarmscript.ActFollowDance:
		swarm.ApplyFollowDance(bot, ss, botIdx)

	case swarmscript.ActMorphColor:
		swarm.ApplyMorphColor(bot, ss, botIdx)

	case swarmscript.ActEvade:
		swarm.ApplyEvade(bot, ss, botIdx)

	case swarmscript.ActFollowSlime:
		swarm.ApplyFollowSlime(bot, ss, botIdx)

	case swarmscript.ActFormBridge:
		swarm.ApplyFormBridge(bot, ss, botIdx)

	case swarmscript.ActCrossBridge:
		swarm.ApplyCrossBridge(bot, ss, botIdx)

	case swarmscript.ActFormShape:
		if ss.ShapeFormationOn && ss.ShapeFormation != nil {
			// Steer toward assigned shape target
			if botIdx < len(ss.ShapeFormation.Assigned) && ss.ShapeFormation.Assigned[botIdx] >= 0 {
				target := ss.ShapeFormation.TargetPositions[ss.ShapeFormation.Assigned[botIdx]]
				dx := target[0] - bot.X
				dy := target[1] - bot.Y
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist > 8 {
					targetAngle := math.Atan2(dy, dx)
					diff := targetAngle - bot.Angle
					for diff > math.Pi {
						diff -= 2 * math.Pi
					}
					for diff < -math.Pi {
						diff += 2 * math.Pi
					}
					if diff > 0.15 {
						diff = 0.15
					} else if diff < -0.15 {
						diff = -0.15
					}
					bot.Angle += diff
					if dist < 30 {
						bot.Speed = swarm.SwarmBotSpeed * 0.4
					} else {
						bot.Speed = swarm.SwarmBotSpeed
					}
				} else {
					bot.Speed = 0
					bot.LEDColor = [3]uint8{0, 255, 100}
				}
			}
		}

	case swarmscript.ActWaveFlash:
		swarm.ApplyWaveFlash(bot, ss, botIdx)

	case swarmscript.ActShepherd:
		swarm.ApplyShepherd(bot, ss, botIdx)

	case swarmscript.ActPSOMove:
		swarm.ApplyPSOMove(bot, ss, botIdx)

	case swarmscript.ActPredator:
		swarm.ApplyPredator(bot, ss, botIdx)

	case swarmscript.ActMagnetic:
		swarm.ApplyMagnetic(bot, ss, botIdx)

	case swarmscript.ActDivide:
		swarm.ApplyDivision(bot, ss, botIdx)

	case swarmscript.ActVFormation:
		swarm.ApplyVFormation(bot, ss, botIdx)

	case swarmscript.ActBroodSort:
		swarm.ApplyBroodSort(bot, ss, botIdx)

	case swarmscript.ActJellyfishPulse:
		swarm.ApplyJellyfishPulse(bot, ss, botIdx)

	case swarmscript.ActImmune:
		swarm.ApplyImmuneSwarm(bot, ss, botIdx)

	case swarmscript.ActGravity:
		swarm.ApplyGravity(bot, ss, botIdx)

	case swarmscript.ActCrystal:
		swarm.ApplyCrystal(bot, ss, botIdx)

	case swarmscript.ActAmoeba:
		swarm.ApplyAmoeba(bot, ss, botIdx)

	case swarmscript.ActACO:
		swarm.ApplyACO(bot, ss, botIdx)

	case swarmscript.ActBFO:
		swarm.ApplyBFO(bot, ss, botIdx)

	case swarmscript.ActGWO:
		swarm.ApplyGWO(bot, ss, botIdx)

	case swarmscript.ActWOA:
		swarm.ApplyWOA(bot, ss, botIdx)

	case swarmscript.ActMFO:
		swarm.ApplyMFO(bot, ss, botIdx)

	case swarmscript.ActCuckoo:
		swarm.ApplyCuckoo(bot, ss, botIdx)

	case swarmscript.ActDE:
		swarm.ApplyDE(bot, ss, botIdx)

	case swarmscript.ActABC:
		swarm.ApplyABC(bot, ss, botIdx)

	case swarmscript.ActDash:
		// Double-speed burst for 10 ticks (costs 15 energy, 60 tick cooldown)
		if bot.DashCooldown <= 0 && bot.Energy >= 15 {
			bot.DashTimer = 10
			bot.DashCooldown = 60
			bot.Energy -= 15
		}
		bot.Speed = swarm.SwarmBotSpeed

	case swarmscript.ActEmergencyBroadcast:
		// Broadcast with 3x communication range (message value from param)
		msgVal := act.Param1
		if msgVal == 0 {
			msgVal = 99 // default emergency value
		}
		// Send 3 copies at wider offsets to simulate 3x range
		for _, offset := range [][2]float64{{0, 0}, {swarm.SwarmCommRange, 0}, {-swarm.SwarmCommRange, 0}, {0, swarm.SwarmCommRange}, {0, -swarm.SwarmCommRange}} {
			ss.NextMessages = append(ss.NextMessages, swarm.SwarmMessage{
				Value: msgVal,
				X:     bot.X + offset[0],
				Y:     bot.Y + offset[1],
			})
		}
		// Visual: brighter wave ring
		if ss.ShowMsgWaves {
			ss.MsgWaves = append(ss.MsgWaves, swarm.MsgWave{
				X: bot.X, Y: bot.Y, Radius: 5, Timer: 45, Value: msgVal,
			})
		}

	case swarmscript.ActReverse:
		// Turn 180° and move forward
		bot.Angle += math.Pi
		bot.Speed = swarm.SwarmBotSpeed

	case swarmscript.ActBrake:
		// Initiate braking: speed ramps down over 3 ticks
		bot.BrakeTimer = 3

	case swarmscript.ActScatterRandom:
		// Scatter away from neighbors with random perturbation
		if bot.NeighborCount > 0 {
			// Turn away from center of neighbors + random offset
			awayAngle := math.Atan2(-bot.AvgNeighborY, -bot.AvgNeighborX)
			awayAngle += (ss.Rng.Float64() - 0.5) * math.Pi / 2 // ±45° random
			bot.Angle = awayAngle
		} else {
			bot.Angle = ss.Rng.Float64() * 2 * math.Pi
		}
		bot.Speed = swarm.SwarmBotSpeed
	}
}
