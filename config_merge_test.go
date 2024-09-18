package light_flow

import (
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var (
	current int64
)

type persistMock struct{}

func (p persistMock) GetLatestRecord(rootId string) (RecoverRecord, error) {
	//TODO implement me
	panic("implement me")
}

func (p persistMock) ListCheckpoints(recoveryId string) ([]CheckPoint, error) {
	//TODO implement me
	panic("implement me")
}

func (p persistMock) UpdateRecordStatus(record RecoverRecord) error {
	//TODO implement me
	panic("implement me")
}

func (p persistMock) SaveCheckpointAndRecord(checkpoint []CheckPoint, record RecoverRecord) error {
	//TODO implement me
	panic("implement me")
}

func emptyStep(ctx Step) (any, error) {
	return nil, nil
}

func errorStep(ctx Step) (any, error) {
	fmt.Printf("[Step: %s ] execute error\n", ctx.Name())
	atomic.AddInt64(&current, 1)
	return nil, fmt.Errorf("error step")
}

func CheckResult(t *testing.T, check int64, statuses ...*StatusEnum) func(WorkFlow) (keepOn bool, err error) {
	return func(workFlow WorkFlow) (keepOn bool, err error) {
		ss := make([]string, len(statuses))
		for i, status := range statuses {
			ss[i] = status.Message()
		}
		t.Logf("\n")
		t.Logf(`// ======= Start Check ======= //`)
		t.Logf("expect [ current ] = %d, [ status ] include { %s }", check, strings.Join(ss, ","))
		if atomic.LoadInt64(&current) != check {
			t.Errorf("execute %d step, but current = %d", check, current)
		}
		for _, status := range statuses {
			if status == Success && !workFlow.Success() {
				t.Errorf("WorkFlow executed failed")
			}
			if !workFlow.HasAny(status) {
				t.Errorf("workFlow has not %s status", status.Message())
			}
		}
		t.Logf("[ status ] expalin = { %s }", strings.Join(workFlow.ExplainStatus(), ","))
		t.Logf(`// ======= Finish Check ====== //`)
		println()
		return true, nil
	}
}

func resetCurrent() {
	atomic.StoreInt64(&current, 0)
}

func resetDefaultConfig() {
	defaultConfig = createDefaultConfig()
	persister = nil
}

func GenerateStep(i int, args ...any) func(ctx Step) (any, error) {
	return func(ctx Step) (any, error) {
		if len(args) > 0 {
			time.Sleep(10 * time.Millisecond)
		}
		fmt.Printf("%d.step finish\n", i)
		atomic.AddInt64(&current, 1)
		return i, nil
	}
}

func fieldsContainedInSet(structure interface{}, strSet *set[string]) (string, bool) {
	value := reflect.ValueOf(structure)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		panic("input must be a struct or a pointer to a struct")
	}

	structType := value.Type()
	for i := 0; i < structType.NumField(); i++ {
		fieldName := structType.Field(i).Name
		if ok := strSet.Contains(fieldName); !ok {
			return fieldName, false
		}
		strSet.Remove(fieldName)
	}
	if strSet.Size() != 0 {
		for k := range strSet.Data {
			return k, false
		}
	}
	return "", true
}

func StepIncr(info Step) (bool, error) {
	println("step inc current")
	atomic.AddInt64(&current, 1)
	return true, nil
}

func ProcIncr(info Process) (bool, error) {
	println("proc inc current")
	atomic.AddInt64(&current, 1)
	return true, nil
}

func FlowIncr(prefix string) func(info WorkFlow) (bool, error) {
	return func(info WorkFlow) (bool, error) {
		fmt.Printf("%s flow inc current\n", prefix)
		atomic.AddInt64(&current, 1)
		return true, nil
	}
}

func StepCurrentChecker(i int64, flag int) func(info Step) (bool, error) {
	return func(info Step) (bool, error) {
		if flag == 0 {
			fmt.Printf("execute before step checker\n")
		} else {
			fmt.Printf("execute after step checker\n")
		}
		if atomic.LoadInt64(&current) != i {
			fmt.Printf("current is %d not %d\n", atomic.LoadInt64(&current), i)
			return false, fmt.Errorf("flow check failed, current is %d not %d", atomic.LoadInt64(&current), i)
		}
		atomic.AddInt64(&current, 1)
		println("current", atomic.LoadInt64(&current))
		return true, nil
	}
}

func ProcessCurrentChecker(i int64, flag int) func(info Process) (bool, error) {
	return func(info Process) (bool, error) {
		if flag == 0 {
			fmt.Printf("execute before process checker\n")
		} else {
			fmt.Printf("execute after process checker\n")
		}
		if atomic.LoadInt64(&current) != i {
			fmt.Printf("current is %d not %d\n", atomic.LoadInt64(&current), i)
			return false, fmt.Errorf("flow check failed, current is %d not %d", atomic.LoadInt64(&current), i)
		}
		atomic.AddInt64(&current, 1)
		return true, nil
	}
}

func FlowCurrentChecker(i int64, flag int) func(info WorkFlow) (bool, error) {
	return func(info WorkFlow) (bool, error) {
		if flag == 0 {
			fmt.Printf("execute before flow checker\n")
		} else {
			fmt.Printf("execute after flow checker\n")
		}
		if atomic.LoadInt64(&current) != i {
			fmt.Printf("flow check failed, current is %d not %d\n", atomic.LoadInt64(&current), i)
			return false, fmt.Errorf("flow check failed, current is %d not %d", atomic.LoadInt64(&current), i)
		}
		atomic.AddInt64(&current, 1)
		return true, nil
	}
}

func FlowConfigChecker(t *testing.T, config flowConfig) func(info WorkFlow) (bool, error) {
	return func(info WorkFlow) (bool, error) {
		t.Logf("start check [Flow: %s] config", info.Name())
		rf := info.(*runFlow)
		if rf.recoverable != config.recoverable {
			return false, fmt.Errorf("[Flow: %s] recoverable is %d not %d", info.Name(), rf.recoverable, config.recoverable)
		}
		atomic.AddInt64(&current, 1)
		t.Logf("check [Flow: %s] config success", info.Name())
		println()
		return true, nil
	}
}

func ProcessConfigChecker(t *testing.T, config processConfig) func(info Process) (bool, error) {
	return func(info Process) (bool, error) {
		t.Logf("start check [Process: %s] config", info.Name())
		rp := info.(*runProcess)
		if rp.procTimeout != config.procTimeout {
			return false, fmt.Errorf("[Process: %s] timeout is %d not %d", info.Name(), rp.procTimeout, config.procTimeout)
		}
		if rp.stepTimeout != config.stepTimeout {
			return false, fmt.Errorf("[Process: %s] step timeout is %d not %d", info.Name(), rp.stepTimeout, config.stepTimeout)
		}
		if rp.stepRetry != config.stepRetry {
			return false, fmt.Errorf("[Process: %s] step retry is %d not %d", info.Name(), rp.stepRetry, config.stepRetry)
		}
		atomic.AddInt64(&current, 1)
		t.Logf("check [Process: %s] config success", info.Name())
		println()
		return true, nil
	}
}

func StepConfigChecker(t *testing.T, config stepConfig) func(info Step) (bool, error) {
	return func(info Step) (bool, error) {
		t.Logf("start check [Step: %s ] config", info.Name())
		rs := info.(*runStep)
		if rs.stepTimeout != config.stepTimeout {
			return false, fmt.Errorf("[Step: %s ] timeout is %d not %d", info.Name(), rs.stepTimeout, config.stepTimeout)
		}
		if rs.stepRetry != config.stepRetry {
			return false, fmt.Errorf("[Step: %s ] retry is %d not %d", info.Name(), rs.stepRetry, config.stepRetry)
		}
		atomic.AddInt64(&current, 1)
		t.Logf("check [Step: %s ] config success", info.Name())
		println()
		return true, nil
	}
}

func TestExecuteDefaultOrder(t *testing.T) {
	defer resetCurrent()
	defer ResetDefaultCallback()
	config := DefaultCallback()
	config.BeforeFlow(true, FlowCurrentChecker(0, 0))
	config.BeforeProcess(true, ProcessCurrentChecker(1, 0))
	config.BeforeStep(true, StepCurrentChecker(2, 0))
	config.AfterStep(true, StepCurrentChecker(4, 1))
	config.AfterProcess(true, ProcessCurrentChecker(5, 1))
	config.AfterFlow(true, FlowCurrentChecker(6, 1))
	workflow := RegisterFlow("TestExecuteDefaultOrder")
	process := workflow.Process("TestExecuteDefaultOrder")
	process.NameStep(GenerateStep(1), "1")
	workflow.AfterFlow(false, CheckResult(t, 7, Success))
	DoneFlow("TestExecuteDefaultOrder", nil)
}

func TestExecuteOwnOrder(t *testing.T) {
	defer resetCurrent()
	workflow := RegisterFlow("TestExecuteOwnOrder")
	workflow.BeforeFlow(true, FlowCurrentChecker(0, 0))
	workflow.BeforeProcess(true, ProcessCurrentChecker(1, 0))
	workflow.BeforeStep(true, StepCurrentChecker(2, 0))
	workflow.AfterStep(true, StepCurrentChecker(4, 1))
	workflow.AfterProcess(true, ProcessCurrentChecker(5, 1))
	workflow.AfterFlow(true, FlowCurrentChecker(6, 1))
	process := workflow.Process("TestExecuteOwnOrder")
	process.NameStep(GenerateStep(1), "1")
	workflow.AfterFlow(false, CheckResult(t, 7, Success))
	DoneFlow("TestExecuteOwnOrder", nil)
}

func TestExecuteMixOrder(t *testing.T) {
	defer resetCurrent()
	defer ResetDefaultCallback()
	config := DefaultCallback()
	config.BeforeFlow(true, FlowCurrentChecker(0, 0))
	config.BeforeProcess(true, ProcessCurrentChecker(2, 0))
	config.BeforeStep(true, StepCurrentChecker(4, 0))
	config.AfterStep(true, StepCurrentChecker(7, 1))
	config.AfterProcess(true, ProcessCurrentChecker(9, 1))
	config.AfterFlow(true, FlowCurrentChecker(11, 1))
	workflow := RegisterFlow("TestExecuteMixOrder")
	workflow.BeforeFlow(true, FlowCurrentChecker(1, 0))
	workflow.BeforeProcess(true, ProcessCurrentChecker(3, 0))
	workflow.BeforeStep(true, StepCurrentChecker(5, 0))
	workflow.AfterStep(true, StepCurrentChecker(8, 1))
	workflow.AfterProcess(true, ProcessCurrentChecker(10, 1))
	workflow.AfterFlow(true, FlowCurrentChecker(12, 1))
	process := workflow.Process("TestExecuteMixOrder")
	process.NameStep(GenerateStep(1), "1")
	workflow.AfterFlow(false, CheckResult(t, 13, Success))
	DoneFlow("TestExecuteMixOrder", nil)
}

func TestExecuteDeepMixOrder(t *testing.T) {
	defer resetCurrent()
	defer ResetDefaultCallback()
	config := DefaultCallback()
	config.BeforeFlow(true, FlowCurrentChecker(0, 0))
	config.BeforeProcess(true, ProcessCurrentChecker(2, 0))
	config.BeforeStep(true, StepCurrentChecker(5, 0))
	config.AfterStep(true, StepCurrentChecker(9, 1))
	config.AfterProcess(true, ProcessCurrentChecker(12, 1))
	config.AfterFlow(true, FlowCurrentChecker(15, 1))
	workflow := RegisterFlow("TestExecuteDeepMixOrder")
	workflow.BeforeFlow(true, FlowCurrentChecker(1, 0))
	workflow.BeforeProcess(true, ProcessCurrentChecker(3, 0))
	workflow.BeforeStep(true, StepCurrentChecker(6, 0))
	workflow.AfterStep(true, StepCurrentChecker(10, 1))
	workflow.AfterProcess(true, ProcessCurrentChecker(13, 1))
	workflow.AfterFlow(true, FlowCurrentChecker(16, 1))
	process := workflow.Process("TestExecuteDeepMixOrder")
	process.NameStep(GenerateStep(1), "1")
	process.BeforeProcess(true, ProcessCurrentChecker(4, 0))
	process.AfterProcess(true, ProcessCurrentChecker(14, 1))
	process.BeforeStep(true, StepCurrentChecker(7, 0))
	process.AfterStep(true, StepCurrentChecker(11, 1))
	workflow.AfterFlow(false, CheckResult(t, 17, Success))
	DoneFlow("TestExecuteDeepMixOrder", nil)
}

func TestBreakWhileFlowError(t *testing.T) {
	defer resetCurrent()
	defer ResetDefaultCallback()
	config := DefaultCallback()
	config.BeforeFlow(true, FlowIncr("default config before panic"))
	config.BeforeFlow(true, FlowCurrentChecker(111, 0))
	config.BeforeFlow(true, FlowIncr("default config after panic"))
	config.BeforeProcess(true, ProcIncr)
	config.BeforeStep(true, StepIncr)
	config.AfterStep(true, StepIncr)
	config.AfterProcess(true, ProcIncr)
	config.AfterFlow(true, FlowIncr("default config after flow"))

	workflow := RegisterFlow("TestBreakWhileFlowError")
	workflow.BeforeFlow(true, FlowIncr("own config before flow"))
	workflow.BeforeProcess(true, ProcIncr)
	workflow.BeforeStep(true, StepIncr)
	workflow.AfterStep(true, StepIncr)
	workflow.AfterProcess(true, ProcIncr)
	workflow.AfterFlow(true, FlowIncr("own config after flow"))
	process := workflow.Process("TestBreakWhileFlowError")
	process.NameStep(GenerateStep(1), "1")
	process.BeforeProcess(true, ProcIncr)
	process.AfterProcess(true, ProcIncr)
	process.BeforeStep(true, StepIncr)
	process.AfterStep(true, StepIncr)
	workflow.AfterFlow(false, CheckResult(t, 3, CallbackFail))
	DoneFlow("TestBreakWhileFlowError", nil)
}

func TestFieldSkip(t *testing.T) {
	config := DefaultConfig()
	config.ProcessTimeout(1 * time.Second)
	config.StepTimeout(1 * time.Second)
	config.StepRetry(3)
	move := flowConfig{}
	move.ProcessTimeout(2 * time.Second)
	move.StepTimeout(2 * time.Second)
	move.StepRetry(4)
	tmp := move
	copyPropertiesSkipNotEmpty(config, &move)
	if tmp.procTimeout != move.procTimeout {
		t.Errorf("process timeout not equal")
	}
	if tmp.stepRetry != move.stepRetry {
		t.Errorf("step retry not equal")
	}
	if tmp.stepTimeout != move.stepTimeout {
		t.Errorf("step timeout not equal")
	}
}

// restrict
func TestMultipleCopy(t *testing.T) {
	src := stepConfig{
		stepTimeout: time.Hour,
		stepRetry:   1,
		stepCfgInit: true,
		extern:      []*stepConfig{{nil, time.Minute, 2, true}, {nil, time.Minute, 2, true}},
	}
	dst := stepConfig{}
	copyProperties(&src, &dst, true)
	copyProperties(&src, &dst, true)
	copyProperties(&src, &dst, true)
	copyProperties(&src, &dst, true)
	if dst.stepTimeout != time.Hour {
		t.Error("stepTimeout should be time.Hour")
	}
	if dst.stepRetry != 1 {
		t.Error("stepRetry should be 1")
	}
	if dst.stepCfgInit != true {
		t.Error("configInit should be true")
	}
	if len(dst.extern) != 0 {
		t.Error("extern should be empty")
	}
	src.stepTimeout = time.Minute
	src.stepRetry = 3
	dst = stepConfig{}
	copyProperties(&src, &dst, true)
	copyProperties(&src, &dst, true)
	copyProperties(&src, &dst, true)
	copyProperties(&src, &dst, true)
	if dst.stepTimeout != time.Minute {
		t.Error("stepTimeout should be time.Minute")
	}
	if dst.stepRetry != 3 {
		t.Error("stepRetry should be 3")
	}
	if len(dst.extern) != 0 {
		t.Error("extern should be empty")
	}

	src1 := processConfig{
		procTimeout: time.Minute,
		stepConfig:  src,
	}
	dst1 := processConfig{}
	copyProperties(&src1, &dst1, true)
	copyProperties(&src1, &dst1, true)
	copyProperties(&src1, &dst1, true)
	copyProperties(&src1, &dst1, true)
	if dst1.procTimeout != time.Minute {
		t.Error("procTimeout should be time.Minute")
	}
	if dst1.stepConfig.stepTimeout != 0 {
		t.Error("stepConfig.stepTimeout should be time.Hour")
	}
	if dst1.stepConfig.stepRetry != 0 {
		t.Error("stepConfig.stepRetry should be 0")
	}
	if dst1.stepConfig.stepCfgInit != false {
		t.Error("stepConfig.configInit should be false")
	}
	if len(dst1.stepConfig.extern) != 0 {
		t.Error("stepConfig.extern should be empty")
	}

	src2 := flowConfig{
		recoverable:   1,
		processConfig: src1,
	}
	dst2 := flowConfig{}
	copyProperties(&src2, &dst2, true)
	copyProperties(&src2, &dst2, true)
	copyProperties(&src2, &dst2, true)
	copyProperties(&src2, &dst2, true)
	if dst2.recoverable != 1 {
		t.Error("recoverable should be 1")
	}
	if dst2.processConfig.procTimeout != 0 {
		t.Error("processConfig.procTimeout should be time.Minute")
	}
	if dst2.processConfig.stepConfig.stepTimeout != 0 {
		t.Error("processConfig.stepConfig.stepTimeout should be time.Hour")
	}
	if dst2.processConfig.stepConfig.stepRetry != 0 {
		t.Error("processConfig.stepConfig.stepRetry should be 1")
	}
	if dst2.processConfig.stepConfig.stepCfgInit != false {
		t.Error("processConfig.stepConfig.configInit should be false")
	}
	if len(dst2.processConfig.stepConfig.extern) != 0 {
		t.Error("processConfig.stepConfig.extern should be empty")
	}
}

// restrict
func TestCopyPropertiesSkipFlowTag(t *testing.T) {
	src := stepConfig{
		stepTimeout: time.Hour,
		stepRetry:   1,
		stepCfgInit: true,
		extern:      []*stepConfig{{nil, time.Minute, 2, true}, {nil, time.Minute, 2, true}},
	}
	dst := stepConfig{}
	copyProperties(&src, &dst, true)
	if dst.stepTimeout != time.Hour {
		t.Error("stepTimeout should be time.Hour")
	}
	if dst.stepRetry != 1 {
		t.Error("stepRetry should be 1")
	}
	if dst.stepCfgInit != true {
		t.Error("configInit should be true")
	}
	if len(dst.extern) != 0 {
		t.Error("extern should be empty")
	}

	src1 := processConfig{
		procTimeout: time.Minute,
		stepConfig:  src,
	}
	dst1 := processConfig{}
	copyProperties(&src1, &dst1, true)
	if dst1.procTimeout != time.Minute {
		t.Error("procTimeout should be time.Minute")
	}
	if dst1.stepConfig.stepTimeout != 0 {
		t.Error("stepConfig.stepTimeout should be time.Hour")
	}
	if dst1.stepConfig.stepRetry != 0 {
		t.Error("stepConfig.stepRetry should be 1")
	}
	if dst.stepCfgInit != true {
		t.Error("configInit should be true")
	}
	if len(dst1.stepConfig.extern) != 0 {
		t.Error("stepConfig.extern should be empty")
	}

	src2 := flowConfig{
		recoverable:   1,
		processConfig: src1,
	}
	dst2 := flowConfig{}
	copyProperties(&src2, &dst2, true)
	if dst2.recoverable != 1 {
		t.Error("recoverable should be 1")
	}
	if dst2.processConfig.procTimeout != 0 {
		t.Error("processConfig.procTimeout should be time.Minute")
	}
	if dst2.processConfig.stepConfig.stepTimeout != 0 {
		t.Error("processConfig.stepConfig.stepTimeout should be time.Hour")
	}
	if dst2.processConfig.stepConfig.stepRetry != 0 {
		t.Error("processConfig.stepConfig.stepRetry should be 1")
	}
	if dst2.stepCfgInit != false {
		t.Error("configInit should be true")
	}
	if len(dst2.processConfig.stepConfig.extern) != 0 {
		t.Error("processConfig.stepConfig.extern should be empty")
	}
}

// restrict
func TestCopyPropertiesSkipNotEmpty(t *testing.T) {
	src := stepConfig{
		stepTimeout: time.Hour,
		stepRetry:   1,
		extern:      []*stepConfig{{nil, time.Minute, 2, true}},
	}
	dst := stepConfig{
		stepTimeout: time.Minute,
		stepRetry:   2,
		stepCfgInit: true,
		extern:      []*stepConfig{{nil, time.Second, 3, true}, {nil, time.Second, 3, true}},
	}
	copyProperties(&src, &dst, true)
	if dst.stepTimeout != time.Minute {
		t.Error("stepTimeout should be time.Minute")
	}
	if dst.stepRetry != 2 {
		t.Error("stepRetry should be 2")
	}
	if dst.stepCfgInit != true {
		t.Error("configInit should be true")
	}
	if len(dst.extern) != 2 {
		t.Error("extern should have 2 elements")
	} else if dst.extern[0].stepTimeout != time.Second {
		t.Error("extern[0].stepTimeout should be time.Second")
	} else if dst.extern[0].stepRetry != 3 {
		t.Error("extern[0].stepRetry should be 3")
	}

	src1 := processConfig{
		procTimeout: time.Minute,
		stepConfig:  src,
	}
	dst1 := processConfig{
		procTimeout: time.Second,
		stepConfig:  dst,
	}
	copyProperties(&src1, &dst1, true)
	if dst1.procTimeout != time.Second {
		t.Error("procTimeout should be time.Minute")
	}
	if dst1.stepConfig.stepTimeout != time.Minute {
		t.Error("stepConfig.stepTimeout should be time.Hour")
	}
	if dst1.stepConfig.stepRetry != 2 {
		t.Error("stepConfig.stepRetry should be 1")
	}
	if dst.stepCfgInit != true {
		t.Error("configInit should be true")
	}
	if len(dst1.stepConfig.extern) != 2 {
		t.Error("stepConfig.extern should have 2 elements")
	} else if dst1.stepConfig.extern[0].stepTimeout != time.Second {
		t.Error("stepConfig.extern[0].stepTimeout should be time.Second")
	} else if dst1.stepConfig.extern[0].stepRetry != 3 {
		t.Error("stepConfig.extern[0].stepRetry should be 3")
	}

	src2 := flowConfig{recoverable: 2, processConfig: src1}
	dst2 := flowConfig{recoverable: 3, processConfig: dst1}
	copyProperties(&src2, &dst2, true)
	if dst2.recoverable != 3 {
		t.Error("recoverable should be 3")
	}
	if dst2.processConfig.procTimeout != time.Second {
		t.Error("processConfig.procTimeout should be time.Minute")
	}
	if dst2.processConfig.stepConfig.stepRetry != 2 {
		t.Error("processConfig.stepConfig.stepRetry should be 1")
	}
	if dst.stepCfgInit != true {
		t.Error("configInit should be true")
	}
	if len(dst2.processConfig.stepConfig.extern) != 2 {
		t.Error("processConfig.stepConfig.extern should have 2 elements")
	} else if dst2.processConfig.stepConfig.extern[0].stepTimeout != time.Second {
		t.Error("processConfig.stepConfig.extern[0].stepTimeout should be time.Second")
	} else if dst2.processConfig.stepConfig.extern[0].stepRetry != 3 {
		t.Error("processConfig.stepConfig.extern[0].stepRetry should be 3")
	}
}

// restrict
func TestFieldRestrict(t *testing.T) {
	restrict := newRoutineUnsafeSet[string]("extern", "stepTimeout", "stepRetry", "stepCfgInit")
	exclude, ok := fieldsContainedInSet(stepConfig{}, restrict)
	if !ok {
		t.Errorf("update step config merge test case while new fields are added to stepConfig!!")
		t.Errorf("field %s matched failed", exclude)
	}

	restrict = newRoutineUnsafeSet[string]("procTimeout", "stepConfig")
	exclude, ok = fieldsContainedInSet(processConfig{}, restrict)
	if !ok {
		t.Errorf("update process config merge test case while new fields are added to processConfig!!")
		t.Errorf("field %s matched failed", exclude)
	}

	restrict = newRoutineUnsafeSet[string]("recoverable", "processConfig")
	exclude, ok = fieldsContainedInSet(flowConfig{}, restrict)
	if !ok {
		t.Errorf("update flow config merge test case while new fields are added to flowConfig!!")
		t.Errorf("field %s matched failed", exclude)
	}
}

// restrict
func TestDefaultFlowConfigValid(t *testing.T) {
	defer resetDefaultConfig()
	defer resetCurrent()
	SuspendPersist(persistMock{})
	DefaultConfig().EnableRecover()
	wf1 := RegisterFlow("TestDefaultFlowConfigValid1")
	wf1.DisableRecover()
	wf1.BeforeFlow(true, FlowConfigChecker(t, flowConfig{recoverable: -1}))
	wf1.AfterFlow(true, FlowConfigChecker(t, flowConfig{recoverable: -1}))
	wf1.AfterFlow(false, CheckResult(t, 2, Success))

	wf2 := RegisterFlow("TestDefaultFlowConfigValid2")
	wf2.BeforeFlow(true, FlowConfigChecker(t, flowConfig{recoverable: 1}))
	wf2.AfterFlow(true, FlowConfigChecker(t, flowConfig{recoverable: 1}))
	wf2.AfterFlow(false, CheckResult(t, 2, Success))
	DoneFlow("TestDefaultFlowConfigValid2", nil)
}

// restrict
func TestDefaultProcessConfigValid(t *testing.T) {
	defer resetDefaultConfig()
	defer resetCurrent()
	SuspendPersist(persistMock{})
	DefaultConfig().EnableRecover()
	DefaultConfig().StepRetry(1)
	DefaultConfig().StepTimeout(time.Hour)
	DefaultConfig().ProcessTimeout(time.Minute)
	wf := RegisterFlow("TestDefaultProcessConfigValid")
	proc1 := wf.Process("ProcConfigFill1")
	proc1.StepRetry(3)
	proc1.StepTimeout(3 * time.Hour)
	proc1.ProcessTimeout(3 * time.Minute)
	copy1 := processConfig{stepConfig: stepConfig{stepTimeout: 3 * time.Hour, stepRetry: 3, stepCfgInit: true}, procTimeout: 3 * time.Minute}
	proc1.BeforeProcess(true, ProcessConfigChecker(t, copy1))
	proc1.AfterProcess(true, ProcessConfigChecker(t, copy1))
	proc1.BeforeStep(true, StepConfigChecker(t, copy1.stepConfig))
	proc1.AfterStep(true, StepConfigChecker(t, copy1.stepConfig))
	proc1.NameStep(emptyStep, "Step1")

	proc2 := wf.Process("ProcConfigEmpty1")
	copy2 := processConfig{procTimeout: 1 * time.Minute}
	proc2.BeforeProcess(true, ProcessConfigChecker(t, copy2))
	proc2.AfterProcess(true, ProcessConfigChecker(t, copy2))
	proc2.BeforeStep(true, StepConfigChecker(t, stepConfig{}))
	proc2.AfterStep(true, StepConfigChecker(t, stepConfig{}))
	proc2.NameStep(emptyStep, "Step2")
	wf.AfterFlow(true, CheckResult(t, 8, Success))
	DoneFlow("TestDefaultProcessConfigValid", nil)
}

// restrict
func TestFlowProcessConfigValid(t *testing.T) {
	defer resetDefaultConfig()
	defer resetCurrent()
	SuspendPersist(persistMock{})
	DefaultConfig().EnableRecover()
	DefaultConfig().StepRetry(1)
	DefaultConfig().StepTimeout(time.Hour)
	DefaultConfig().ProcessTimeout(time.Minute)
	wf := RegisterFlow("TestFlowProcessConfigValid")
	wf.ProcessTimeout(2 * time.Minute)
	wf.StepTimeout(2 * time.Hour)
	wf.StepRetry(2)

	proc1 := wf.Process("ProcConfigFill2")
	proc1.StepRetry(3)
	proc1.StepTimeout(3 * time.Hour)
	proc1.ProcessTimeout(3 * time.Minute)
	copy1 := processConfig{stepConfig: stepConfig{stepTimeout: 3 * time.Hour, stepRetry: 3}, procTimeout: 3 * time.Minute}
	proc1.BeforeProcess(true, ProcessConfigChecker(t, copy1))
	proc1.AfterProcess(true, ProcessConfigChecker(t, copy1))
	proc1.BeforeStep(true, StepConfigChecker(t, copy1.stepConfig))
	proc1.AfterStep(true, StepConfigChecker(t, copy1.stepConfig))
	proc1.NameStep(emptyStep, "Step1")

	proc2 := wf.Process("ProcConfigEmpty2")
	copy2 := processConfig{procTimeout: 2 * time.Minute, stepConfig: stepConfig{stepTimeout: 2 * time.Hour, stepRetry: 2}}
	proc2.BeforeProcess(true, ProcessConfigChecker(t, copy2))
	proc2.AfterProcess(true, ProcessConfigChecker(t, copy2))
	fc := stepConfig{stepTimeout: 2 * time.Hour, stepRetry: 2, stepCfgInit: true}
	proc2.BeforeStep(true, StepConfigChecker(t, fc))
	proc2.AfterStep(true, StepConfigChecker(t, fc))
	proc2.NameStep(emptyStep, "Step2")
	wf.AfterFlow(true, CheckResult(t, 8, Success))
	DoneFlow("TestFlowProcessConfigValid", nil)
}

func TestConfigMergeValid(t *testing.T) {
	defer resetDefaultConfig()
	defer resetCurrent()
	wf1 := RegisterFlow("TestConfigMergeValid1")
	proc1 := wf1.Process("TestConfigMergeValid1")
	proc1.NameStep(errorStep, "Step1").StepRetry(2)
	proc1.NameStep(errorStep, "Step2").StepRetry(5)
	wf1.AfterFlow(false, CheckResult(t, 9, Error))
	wf2 := RegisterFlow("TestConfigMergeValid2")
	proc2 := wf2.Process("TestConfigMergeValid2")
	proc2.NameStep(errorStep, "Step1")
	proc2.Merge("TestConfigMergeValid1")
	wf2.AfterFlow(false, CheckResult(t, 9, Error))
	DoneFlow("TestConfigMergeValid2", nil)
	resetCurrent()
	DoneFlow("TestConfigMergeValid1", nil)
}

func TestStepStepConfigValid(t *testing.T) {
	defer resetDefaultConfig()
	defer resetCurrent()
	SuspendPersist(persistMock{})
	DefaultConfig().EnableRecover()
	DefaultConfig().StepRetry(1)
	DefaultConfig().StepTimeout(time.Hour)
	DefaultConfig().ProcessTimeout(time.Minute)
	wf := RegisterFlow("TestStepStepConfigValid")
	wf.ProcessTimeout(2 * time.Minute)
	wf.StepTimeout(2 * time.Hour)
	wf.StepRetry(2)

	proc1 := wf.Process("TestStepStepConfigValid")
	proc1.StepRetry(3)
	proc1.StepTimeout(3 * time.Hour)
	proc1.ProcessTimeout(3 * time.Minute)
	copy1 := stepConfig{stepTimeout: 4 * time.Hour, stepRetry: 4, stepCfgInit: true}
	proc1.NameStep(emptyStep, "Step1").StepRetry(4).StepTimeout(4 * time.Hour)
	proc1.BeforeStep(true, StepConfigChecker(t, copy1))
	proc1.AfterStep(true, StepConfigChecker(t, copy1))
	wf.AfterFlow(true, CheckResult(t, 2, Success))
	DoneFlow("TestStepStepConfigValid", nil)
}
