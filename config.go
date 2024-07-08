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
	DisableDefaultConfig() ProcessConfig
}

type StepConfig interface {
	StepTimeout(time.Duration) StepConfig
	StepRetry(int) StepConfig
}

type ternary int

type flowConfig struct {
	processConfig
	recoverable ternary
}

type processConfig struct {
	stepConfig
	procCfg           []*processConfig
	procTimeout       time.Duration
	disableDefaultCfg bool
	isDefaultCfg      bool
}

type stepConfig struct {
	stepCfg     []*stepConfig
	stepTimeout time.Duration
	stepRetry   int
}

func createDefaultConfig() *flowConfig {
	return &flowConfig{
		recoverable: ternary(-1),
		processConfig: processConfig{
			isDefaultCfg: true,
			procTimeout:  3 * time.Hour,
			stepConfig: stepConfig{
				stepTimeout: 3 * time.Hour,
			},
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

func (pc *processConfig) getProcessTimeout() time.Duration {
	if pc.procTimeout != 0 {
		return pc.procTimeout
	}
	for _, prev := range pc.procCfg {
		if timeout := prev.getProcessTimeout(); timeout != 0 {
			return timeout
		}
	}
	return 0
}

func (pc *processConfig) DisableDefaultConfig() ProcessConfig {
	if pc.isDefaultCfg {
		panic("default config can't disable itself")
	}
	pc.disableDefaultCfg = true
	return pc
}

func (s *stepConfig) StepTimeout(duration time.Duration) StepConfig {
	s.stepTimeout = duration
	return s
}

func (s *stepConfig) getStepTimeout() time.Duration {
	if s.stepTimeout != 0 {
		return s.stepTimeout
	}
	for _, prev := range s.stepCfg {
		if timeout := prev.getStepTimeout(); timeout != 0 {
			return timeout
		}
	}
	return 0
}

func (s *stepConfig) StepRetry(i int) StepConfig {
	s.stepRetry = i
	return s
}

func (s *stepConfig) getStepRetry() int {
	if s.stepRetry != 0 {
		return s.stepRetry
	}
	for _, prev := range s.stepCfg {
		if retry := prev.getStepRetry(); retry != 0 {
			return retry
		}
	}
	return 0
}
