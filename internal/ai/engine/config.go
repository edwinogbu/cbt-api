package engine

// DefaultConfig returns a production-ready engine configuration.
func DefaultConfig() EngineConfig {
	return EngineConfig{
		MaxRefinementCycles: 2,
		QualityThreshold:    70,
		EnableSafety:        true,
		EnableScoring:       true,
		TimeoutSeconds:      60,
	}
}

// WithRefinement configures refinement cycles.
func (c EngineConfig) WithRefinement(cycles int) EngineConfig {
	c.MaxRefinementCycles = cycles
	return c
}

// WithThreshold configures quality threshold.
func (c EngineConfig) WithThreshold(threshold int) EngineConfig {
	c.QualityThreshold = threshold
	return c
}

// WithTimeout configures timeout in seconds.
func (c EngineConfig) WithTimeout(seconds int) EngineConfig {
	c.TimeoutSeconds = seconds
	return c
}

// DisableSafety disables safety checks.
func (c EngineConfig) DisableSafety() EngineConfig {
	c.EnableSafety = false
	return c
}


// package engine

// // DefaultConfig returns a production-ready engine configuration.
// func DefaultConfig() EngineConfig {
// 	return EngineConfig{
// 		MaxRefinementCycles: 2,
// 		QualityThreshold:    70,
// 		EnableSafety:        true,
// 		EnableScoring:       true,
// 		TimeoutSeconds:      60,
// 	}
// }

// // WithRefinement configures refinement cycles.
// func (c EngineConfig) WithRefinement(cycles int) EngineConfig {
// 	c.MaxRefinementCycles = cycles
// 	return c
// }

// // WithThreshold configures quality threshold.
// func (c EngineConfig) WithThreshold(threshold int) EngineConfig {
// 	c.QualityThreshold = threshold
// 	return c
// }