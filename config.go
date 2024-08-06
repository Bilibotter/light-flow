package light_flow

import "time"

var (
	defaultConfig = createDefaultConfig()
)

type FlowConfig interface {
	ProcessConfig
	EnableRecover() FlowConfig
	DisableRecover() FlowConfig
}

type ProcessConfig interface {
	StepConfig
	ProcessTimeout(time.Duration) ProcessConfig
}

type StepConfig interface {
	StepTimeout(time.Duration) StepConfig
	StepRetry(int) StepConfig
}

type ternary int

type flowConfig struct {
	processConfig `flow:"skip"`
	recoverable   ternary
}

type processConfig struct {
	stepConfig  `flow:"skip"`
	procTimeout time.Duration
}

type stepConfig struct {
	extern      []*stepConfig `flow:"skip"`
	stepTimeout time.Duration
	stepRetry   int
	stepCfgInit bool
}

func createDefaultConfig() *flowConfig {
	return &flowConfig{
		recoverable: ternary(-1),
		processConfig: processConfig{
			procTimeout: 3 * time.Hour,
		},
	}
}

func (t ternary) HasInit() bool {
	return t != 0
}

func (t ternary) Valid() bool {
	return t == 1
}

func (f *flowConfig) EnableRecover() FlowConfig {
	if persister == nil {
		panic("Persist must be set to use the recovery feature, please invoke SetPersist(impl Persist) first")
	}
	f.recoverable = 1
	return f
}

func (f *flowConfig) DisableRecover() FlowConfig {
	f.recoverable = -1
	return f
}

func (f *flowConfig) isRecoverable() bool {
	if f.recoverable.HasInit() {
		return f.recoverable.Valid()
	}
	return defaultConfig.recoverable.Valid()
}

func (pc *processConfig) ProcessTimeout(duration time.Duration) ProcessConfig {
	pc.procTimeout = duration
	return pc
}

func (s *stepConfig) StepTimeout(duration time.Duration) StepConfig {
	s.stepCfgInit = true
	s.stepTimeout = duration
	return s
}

func (s *stepConfig) StepRetry(i int) StepConfig {
	s.stepCfgInit = true
	s.stepRetry = i
	return s
}
