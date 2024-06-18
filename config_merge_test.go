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

func CheckResult(t *testing.T, check int64, statuses ...*StatusEnum) func(*WorkFlow) (keepOn bool, err error) {
	return func(workFlow *WorkFlow) (keepOn bool, err error) {
		ss := make([]string, len(statuses))
		for i, status := range statuses {
			ss[i] = status.Message()
		}
		t.Logf("start check, expected current=%d, status include %s", check, strings.Join(ss, ","))
		if current != check {
			t.Errorf("execute %d step, but current = %d\n", check, current)
		}
		for _, status := range statuses {
			if status == Success && !workFlow.Success() {
				t.Errorf("workFlow not success\n")
			}
			//if status == Timeout {
			//	time.Sleep(50 * time.Millisecond)
			//}
			if !workFlow.Has(status) {
				t.Errorf("workFlow has not %s status\n", status.Message())
			}
		}
		t.Logf("status expalin=%s", strings.Join(workFlow.ExplainStatus(), ","))
		t.Logf("finish check")
		println()
		return true, nil
	}
}

func resetCurrent() {
	atomic.StoreInt64(&current, 0)
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

func CheckField(check any, fields ...string) {
	fieldSet := createSetBySliceFunc(fields, func(s string) string {
		return s
	})
	// 获取结构体的反射类型
	t := reflect.TypeOf(check)
	v := reflect.ValueOf(check)
	// 遍历结构体的所有字段
	for i := 0; i < t.NumField(); i++ {
		// 获取当前字段的信息
		field := t.Field(i)
		value := v.Field(i)
		if tag := t.Field(i).Tag.Get("flow"); len(tag) != 0 {
			splits := strings.Split(tag, ";")
			skip := false
			for j := 0; j < len(splits) && !skip; j++ {
				skip = splits[j] == "skip"
			}
			if skip {
				continue
			}
		}
		// 检查字段名是否在集合中
		if !fieldSet.Contains(field.Name) {
			panic(fmt.Sprintf("field %s not in fields", field.Name))
		}
		if value.IsZero() {
			panic(fmt.Sprintf("field %s is zero", field.Name))
		}
	}
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

func FlowIncr(prefix string) func(info *WorkFlow) (bool, error) {
	return func(info *WorkFlow) (bool, error) {
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

func FlowCurrentChecker(i int64, flag int) func(info *WorkFlow) (bool, error) {
	return func(info *WorkFlow) (bool, error) {
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

func TestExecuteDefaultOrder(t *testing.T) {
	defer resetCurrent()
	defer func() {
		CreateDefaultConfig()
	}()
	config := CreateDefaultConfig()
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
	defer func() {
		CreateDefaultConfig()
	}()
	CreateDefaultConfig()
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
	defer func() {
		CreateDefaultConfig()
	}()
	config := CreateDefaultConfig()
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
	defer func() {
		CreateDefaultConfig()
	}()
	config := CreateDefaultConfig()
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
	defer CreateDefaultConfig()
	config := CreateDefaultConfig()
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
	workflow.AfterFlow(false, CheckResult(t, 4, CallbackFail))
	DoneFlow("TestBreakWhileFlowError", nil)
}

func TestFieldSkip(t *testing.T) {
	config := CreateDefaultConfig()
	config.ProcessTimeout(1 * time.Second)
	config.StepsTimeout(1 * time.Second)
	config.StepsRetry(3)
	move := FlowConfig{}
	move.ProcessTimeout(2 * time.Second)
	move.NotUseDefault()
	move.StepsTimeout(2 * time.Second)
	move.StepsRetry(4)
	tmp := move
	copyPropertiesSkipNotEmpty(config, &move)
	if tmp.ProcNotUseDefault != move.ProcNotUseDefault {
		t.Errorf("not use default not equal")
	}
	if tmp.ProcTimeout != move.ProcTimeout {
		t.Errorf("process timeout not equal")
	}
	if tmp.StepRetry != move.StepRetry {
		t.Errorf("step retry not equal")
	}
	if tmp.StepTimeout != move.StepTimeout {
		t.Errorf("step timeout not equal")
	}
}

func TestFieldCorrect(t *testing.T) {
	defer CreateDefaultConfig()
	config := CreateDefaultConfig()
	config.ProcessTimeout(1 * time.Second)
	config.NotUseDefault()
	config.StepsTimeout(1 * time.Second)
	config.StepsRetry(3)
	move := FlowConfig{}
	copyPropertiesSkipNotEmpty(config, &move)
	copyPropertiesSkipNotEmpty(&config.ProcessConfig, &move.ProcessConfig)
	CheckField(move.ProcessConfig, "ProcTimeout", "ProcNotUseDefault", "StepConfig")
	CheckField(move.ProcessConfig.StepConfig, "StepTimeout", "StepRetry")
	return
}
