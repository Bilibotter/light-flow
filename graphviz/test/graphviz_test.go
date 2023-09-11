package test

import (
	"fmt"
	"gitee.com/MetaphysicCoding/light-flow/core"
	"gitee.com/MetaphysicCoding/light-flow/graphviz"
	"os"
	"testing"
	"time"
)

func TaskA(ctx *core.Context) (any, error) {
	time.Sleep(100 * time.Millisecond)
	return nil, nil
}

func TaskB(ctx *core.Context) (any, error) {
	time.Sleep(100 * time.Millisecond)
	return nil, nil
}

func TaskC(ctx *core.Context) (any, error) {
	time.Sleep(100 * time.Millisecond)
	return nil, nil
}

func TaskD(ctx *core.Context) (any, error) {
	time.Sleep(100 * time.Millisecond)
	return nil, nil
}

func TestDraw(t *testing.T) {
	workflow := core.NewWorkflow(nil)
	conf := core.ProduceConfig{
		CompleteProcessor: graphviz.Decorate(func(info *core.ProcedureInfo) {}),
	}
	procedure := workflow.AddProcedure("procedureA", &conf)
	procedure.AddStep(TaskA)
	procedure.AddStep(TaskB, TaskA)
	procedure.AddStep(TaskC, TaskA)
	procedure.AddStep(TaskD, TaskB, TaskC)
	workflow.WaitToDone()
	filename := fmt.Sprintf("%s%s.png", graphviz.GraphPrefix, "procedureA")
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		t.Errorf("graph file %s not exist\n", filename)
	}
}
