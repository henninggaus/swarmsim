package swarm

import (
	"math"
	"math/rand"
)

// swarm_steering.go provides shared steering, angle, and random-walk utilities
// used by multiple swarm algorithms (GWO, WOA, BFO, MFO, Cuckoo Search, etc.).
//
// The steerToward function is the canonical way to smoothly rotate a bot
// toward a desired heading with a maximum turn rate. It normalises the
// angular difference to [-π, π] and clamps the step to maxRate radians.
// Every algorithm that needs rate-limited steering should call steerToward
// rather than duplicating the angle-wrapping / clamping logic.
//
// WrapAngle normalises any angle into the range [-π, π]. Use it when you
// only need to normalise an angle difference without applying a rate limit.
//
// MantegnaLevy generates a Lévy-flight step using the Mantegna algorithm.
// It is used by HHO, FPA, Aquila, Dragonfly, and Cuckoo Search to avoid
// duplicating the same Mantegna implementation in each algorithm file.

// steerToward smoothly steers a bot toward a desired angle with a maximum
// turn rate per tick. The turn is clamped to ±maxRate radians so the bot
// cannot snap instantly to the target heading. This produces natural,
// smooth turning behaviour suitable for all bio-inspired algorithms.
func steerToward(bot *SwarmBot, desired, maxRate float64) {
	diff := desired - bot.Angle
	diff = WrapAngle(diff)
	if diff > maxRate {
		diff = maxRate
	} else if diff < -maxRate {
		diff = -maxRate
	}
	bot.Angle += diff
}

// WrapAngle normalises an angle (in radians) into the range [-π, π].
// This is useful for computing the shortest angular difference between
// two headings without applying any rate limiting.
func WrapAngle(a float64) float64 {
	for a > math.Pi {
		a -= 2 * math.Pi
	}
	for a < -math.Pi {
		a += 2 * math.Pi
	}
	return a
}

// mantegnaSigma15 is the precomputed sigma_u for beta = 1.5 using the formula:
//
//	sigma_u = { Γ(1+β) · sin(π·β/2) / [Γ((1+β)/2) · β · 2^((β-1)/2)] }^(1/β)
//
// For β = 1.5 this evaluates to ≈ 0.6966.
var mantegnaSigma15 = math.Pow(
	math.Gamma(1+1.5)*math.Sin(math.Pi*1.5/2)/
		(math.Gamma((1+1.5)/2)*1.5*math.Pow(2, (1.5-1)/2)),
	1.0/1.5,
)

// MantegnaLevy generates a Lévy-flight step size using Mantegna's algorithm
// for Lévy-stable distributions. The beta parameter controls the heavy-tail
// exponent (typically 1.5 for most swarm algorithms). The returned value may
// be positive or negative; callers that need a magnitude should take Abs.
//
// The Mantegna algorithm approximates a Lévy-stable random variable as:
//
//	step = (sigma_u · u) / |v|^(1/β)
//
// where u and v are independent standard normal variates and sigma_u is
// precomputed from β via Gamma functions.
func MantegnaLevy(rng *rand.Rand, beta float64) float64 {
	var sigmaU float64
	if beta == 1.5 {
		sigmaU = mantegnaSigma15
	} else {
		sigmaU = math.Pow(
			math.Gamma(1+beta)*math.Sin(math.Pi*beta/2)/
				(math.Gamma((1+beta)/2)*beta*math.Pow(2, (beta-1)/2)),
			1.0/beta,
		)
	}

	u := rng.NormFloat64() * sigmaU
	v := math.Abs(rng.NormFloat64())
	if v < 1e-10 {
		v = 1e-10
	}
	return u / math.Pow(v, 1.0/beta)
}
