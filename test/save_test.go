package test

import (
	flow "github.com/Bilibotter/light-flow"
	"sync/atomic"
	"testing"
	"time"
)

const (
	Activated int8 = iota
	Success
	Recovered
	Fail
)

type Steps struct {
	Id         string `gorm:"primaryKey"`
	Name       string
	Status     int8
	ProcId     string // Add ProcID to directly link Step to its parent Process
	FlowId     string // Add FlowID to directly link Step to its parent Flow
	CreatedAt  *time.Time
	UpdatedAt  *time.Time
	FinishedAt *time.Time
}

type Processes struct {
	Id         string `gorm:"primaryKey"`
	Name       string
	Status     int8
	FlowId     string // Add FlowID to directly link Process to its parent Flow
	CreatedAt  *time.Time
	UpdatedAt  *time.Time
	FinishedAt *time.Time
}

type Flows struct {
	Id         string `gorm:"primaryKey"`
	Name       string
	Status     int8
	CreatedAt  *time.Time
	UpdatedAt  *time.Time
	FinishedAt *time.Time
}

func onStepBegin(step flow.Step) error {
	db := <-pool
	defer func() {
		pool <- db
	}()
	entity := Steps{
		Id:        step.ID(),
		Name:      step.Name(),
		Status:    Activated,
		ProcId:    step.ProcessID(),
		FlowId:    step.FlowID(),
		CreatedAt: step.StartTime(),
		UpdatedAt: step.StartTime(),
	}
	return db.Create(&entity).Error
}

func onStepComplete(step flow.Step) error {
	db := <-pool
	defer func() {
		pool <- db
	}()
	entity := Steps{
		Id:         step.ID(),
		UpdatedAt:  step.EndTime(),
		FinishedAt: step.EndTime(),
	}
	if step.EndTime().IsZero() {
		println("Step end time is zero")
	}
	if entity.FinishedAt.IsZero() {
		println("Step finished at is zero")
	}
	if step.Success() {
		entity.Status = Success
	} else {
		entity.Status = Fail
	}
	return db.Model(&Steps{}).Where("id = ?", step.ID()).Updates(entity).Error
}

func onStepRecover(step flow.Step) error {
	db := <-pool
	defer func() {
		pool <- db
	}()
	entity := Steps{
		Id:        step.ID(),
		UpdatedAt: step.EndTime(),
	}
	if step.Success() {
		entity.Status = Success
	} else {
		entity.Status = Fail
	}
	return db.Model(&Steps{}).Where("id = ?", step.ID()).Updates(entity).Error
}

func onProcessBegin(process flow.Process) error {
	db := <-pool
	defer func() {
		pool <- db
	}()
	entity := Processes{
		Id:        process.ID(),
		Name:      process.Name(),
		Status:    Activated,
		FlowId:    process.FlowID(),
		CreatedAt: process.StartTime(),
		UpdatedAt: process.StartTime(),
	}
	return db.Create(&entity).Error
}

func onProcessComplete(process flow.Process) error {
	db := <-pool
	defer func() {
		pool <- db
	}()
	entity := Processes{
		Id:         process.ID(),
		UpdatedAt:  process.EndTime(),
		FinishedAt: process.EndTime(),
	}
	if process.Success() {
		entity.Status = Success
	} else {
		entity.Status = Fail
	}
	return db.Model(&Processes{}).Where("id = ?", process.ID()).Updates(entity).Error
}

func onProcessRecover(process flow.Process) error {
	db := <-pool
	defer func() {
		pool <- db
	}()
	entity := Processes{
		Id:        process.ID(),
		UpdatedAt: process.EndTime(),
	}
	if process.Success() {
		entity.Status = Success
	} else {
		entity.Status = Fail
	}
	return db.Model(&Processes{}).Where("id = ?", process.ID()).Updates(entity).Error
}

func onFlowBegin(flow flow.WorkFlow) error {
	db := <-pool
	defer func() {
		pool <- db
	}()
	entity := Flows{
		Id:        flow.ID(),
		Name:      flow.Name(),
		Status:    Activated,
		CreatedAt: flow.StartTime(),
		UpdatedAt: flow.StartTime(),
	}
	return db.Create(&entity).Error
}

func onFlowComplete(flow flow.WorkFlow) error {
	db := <-pool
	defer func() {
		pool <- db
	}()
	entity := Flows{
		Id:        flow.ID(),
		UpdatedAt: flow.EndTime(),
	}
	if !flow.EndTime().IsZero() {
		entity.FinishedAt = flow.EndTime()
	}
	if flow.Success() {
		entity.Status = Success
	} else {
		entity.Status = Fail
	}
	return db.Model(&Flows{}).Where("id = ?", flow.ID()).Updates(entity).Error
}

func onFlowRecover(flow flow.WorkFlow) error {
	db := <-pool
	defer func() {
		pool <- db
	}()
	entity := Flows{
		Id:        flow.ID(),
		UpdatedAt: flow.EndTime(),
	}
	if flow.Success() {
		entity.Status = Success
	} else {
		entity.Status = Fail
	}
	return db.Model(&Flows{}).Where("id = ?", flow.ID()).Updates(entity).Error
}

func setPersist() {
	flow.StepPersist().OnInsert(onStepBegin).OnUpdate(onStepComplete)
	flow.ProcPersist().OnInsert(onProcessBegin).OnUpdate(onProcessComplete)
	flow.FlowPersist().OnInsert(onFlowBegin).OnUpdate(onFlowComplete)
}

func stringStatus(status int8) string {
	switch status {
	case Activated:
		return "Activated"
	case Success:
		return "Success"
	case Fail:
		return "Fail"
	case Recovered:
		return "Recovered"
	default:
		return "Unknown"
	}
}

func CheckFlowPersist(t *testing.T, ff flow.FinishedWorkFlow, expect int) {
	t.Logf("Checking Flow %s", ff.Name())
	db := <-pool
	defer func() {
		pool <- db
	}()
	var f Flows
	count := 0
	if err := db.Where("id = ?", ff.ID()).First(&f).Error; err != nil {
		t.Errorf("Error getting Flow %s: %s", ff.Name(), err.Error())
	} else {
		count += 1
	}
	if ff.Name() != f.Name {
		t.Errorf("Flow %s has wrong name: %s", ff.Name(), f.Name)
	}
	if ff.ID() != f.Id {
		t.Errorf("Flow %s has wrong ID: %s", ff.Name(), f.Id)
	}
	if ff.StartTime().Sub(*f.CreatedAt) > time.Second {
		t.Errorf("Flow %s has wrong start time: %s, expected %s", ff.Name(), f.CreatedAt.UTC(), ff.StartTime().UTC())
	}
	if ff.EndTime().Sub(*f.UpdatedAt) > time.Second {
		t.Errorf("Flow %s has wrong end time: %s, expected %s", ff.Name(), f.UpdatedAt.UTC(), ff.EndTime().UTC())
	}
	if ff.EndTime().Sub(*f.FinishedAt) > time.Second {
		t.Errorf("Flow %s has wrong end time: %s, expected %s", ff.Name(), f.FinishedAt.UTC(), ff.EndTime().UTC())
	}
	if ff.Success() && f.Status != Success {
		t.Errorf("Flow %s should be Success but is %s", ff.Name(), stringStatus(f.Status))
	} else if !ff.Success() && f.Status != Fail {
		t.Errorf("Flow %s should be Fail but is %s", ff.Name(), stringStatus(f.Status))
	}
	if ff.Success() && f.Status != Success {
		t.Errorf("Flow %s should be Success but is %s", ff.Name(), stringStatus(f.Status))
	} else if !ff.Success() && f.Status != Fail {
		t.Errorf("Flow %s should be Fail but is %s", ff.Name(), stringStatus(f.Status))
	}
	t.Logf("Check Flow[ %s ] complete", ff.Name())
	for _, proc := range ff.Processes() {
		if !proc.Has(flow.Pending) {
			continue
		}
		var p Processes
		if err := db.Where("id = ?", proc.ID()).First(&p).Error; err != nil {
			t.Errorf("Error getting Process %s: %s", proc.Name(), err.Error())
		} else {
			count += 1
		}
		if proc.Name() != p.Name {
			t.Errorf("Process %s has wrong name: %s", proc.Name(), p.Name)
		}
		if proc.ID() != p.Id {
			t.Errorf("Process %s has wrong ID: %s", proc.Name(), p.Id)
		}
		if proc.StartTime().Sub(*p.CreatedAt) > time.Second {
			t.Errorf("Process %s has wrong start time: %s, expected %s", proc.Name(), p.CreatedAt.UTC(), proc.StartTime().UTC())
		}
		if proc.EndTime().Sub(*p.UpdatedAt) > time.Second {
			t.Errorf("Process %s has wrong end time: %s, expected %s", proc.Name(), p.UpdatedAt.UTC(), proc.EndTime().UTC())
		}
		if proc.EndTime().Sub(*p.FinishedAt) > time.Second {
			t.Errorf("Process %s has wrong end time: %s, expected %s", proc.Name(), p.FinishedAt.UTC(), proc.EndTime().UTC())
		}
		if proc.Success() && p.Status != Success {
			t.Errorf("Process %s should be Success but is %s", proc.Name(), stringStatus(p.Status))
		} else if !proc.Success() && p.Status != Fail {
			t.Errorf("Process %s should be Fail but is %s", proc.Name(), stringStatus(p.Status))
		}
		t.Logf("Check Process[ %s ] complete", proc.Name())
		for _, step := range proc.Steps() {
			if !step.Has(flow.Pending) {
				continue
			}
			var s Steps
			if err := db.Where("id = ?", step.ID()).First(&s).Error; err != nil {
				t.Errorf("Error getting Step %s: %s", step.Name(), err.Error())
			} else {
				count += 1
			}
			if step.Name() != s.Name {
				t.Errorf("Step %s has wrong name: %s", step.Name(), s.Name)
			}
			if step.ID() != s.Id {
				t.Errorf("Step %s has wrong ID: %s", step.Name(), s.Id)
			}
			if step.StartTime().Sub(*s.CreatedAt) > time.Second {
				t.Errorf("Step %s has wrong start time: %s, expected %s", step.Name(), s.CreatedAt.UTC(), step.StartTime().UTC())
			}
			if step.EndTime().Sub(*s.UpdatedAt) > time.Second {
				t.Errorf("Step %s has wrong end time: %s, expected %s", step.Name(), s.UpdatedAt.UTC(), step.EndTime().UTC())
			}
			if step.EndTime() == nil {
				t.Errorf("Step %s has no end time", step.Name())
			}
			if s.FinishedAt == nil {
				if s.CreatedAt != nil {
					t.Errorf("Step %s has no finished at time, but has created at time", step.Name())
				} else {
					t.Errorf("Step %s has no finished at time", step.Name())
				}
			}
			if step.EndTime().Sub(*s.FinishedAt) > time.Second {
				t.Errorf("Step %s has wrong end time: %s, expected %s", step.Name(), s.FinishedAt.UTC(), step.EndTime().UTC())
			}
			if step.Success() && s.Status != Success {
				t.Errorf("Step %s should be Success but is %s", step.Name(), stringStatus(s.Status))
			} else if !step.Success() && s.Status != Fail {
				t.Errorf("Step %s should be Fail but is %s", step.Name(), stringStatus(s.Status))
			}
			t.Logf("Check Step[ %s ] complete", step.Name())
		}
	}
	if count != expect {
		t.Errorf("Expected %d entities, got %d", expect, count)
	}
}

func TestNormalPersist(t *testing.T) {
	defer resetPersist()
	defer resetCurrent()
	setPersist()
	wf := flow.RegisterFlow("TestNormalPersist")
	proc := wf.Process("TestNormalPersist")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3", "1")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4", "1", "3")
	wf.AfterFlow(false, CheckResult(t, 4, flow.Success))
	ff := flow.DoneFlow("TestNormalPersist", nil)
	CheckFlowPersist(t, ff, 6)
}

func TestMultipleErrorStepPersist(t *testing.T) {
	defer resetPersist()
	defer resetCurrent()
	setPersist()
	workflow := flow.RegisterFlow("TestMultipleErrorStepPersist0")
	process := workflow.Process("TestMultipleErrorStepPersist0")
	process.NameStep(Fn(t).Errors(), "1")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Error))
	ff := flow.DoneFlow("TestMultipleErrorStepPersist0", nil)
	CheckFlowPersist(t, ff, 3)

	resetCurrent()
	workflow = flow.RegisterFlow("TestMultipleErrorStepPersist1")
	process = workflow.Process("TestMultipleErrorStepPersist1")
	process.NameStep(Fn(t).Panic(), "2")
	workflow.AfterFlow(false, CheckResult(t, 1, flow.Panic))
	ff = flow.DoneFlow("TestMultipleErrorStepPersist1", nil)
	CheckFlowPersist(t, ff, 3)

	resetCurrent()
	atomic.StoreInt64(&letGo, 0)
	workflow = flow.RegisterFlow("TestMultipleErrorStepPersist2")
	process = workflow.Process("TestMultipleErrorStepPersist2")
	step := process.NameStep(Fn(t).WaitLetGO(1), "3")
	step.StepTimeout(time.Nanosecond)
	ff = flow.DoneFlow("TestMultipleErrorStepPersist2", nil)
	atomic.StoreInt64(&letGo, 1)
	CheckFlowPersist(t, ff, 3)
	waitCurrent(2)
	CheckResult(t, 2, flow.Timeout)(any(ff).(flow.WorkFlow))
}

func TestStepRecoverPersist(t *testing.T) {
	defer resetCurrent()
	defer resetPersist()
	defer flow.DefaultConfig().DisableRecover()
	setPersist()
	flow.DefaultConfig().EnableRecover()
	executeSuc = false
	wf := flow.RegisterFlow("TestStepRecoverPersist")
	proc := wf.Process("TestStepRecoverPersist")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step1").
		Next(Fn(t).Do(SetCtx()).Step(), "step2").
		Next(Fn(t).Suc(CheckCtx("step2"), SetCtx()).ErrStep(), "step3").
		Next(Fn(t).Do(CheckCtx("step3")).Step(), "step4")
	wf.AfterFlow(false, CheckResult(t, 2, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	res := flow.DoneFlow("TestStepRecoverPersist", nil)
	CheckResult(t, 2, flow.Error)(any(res).(flow.WorkFlow))
	CheckFlowPersist(t, res, 5)
	res = Recover("TestStepRecoverPersist")
	CheckFlowPersist(t, res, 4)
}

func TestCallbackFailedPersist(t *testing.T) {
	defer resetCurrent()
	defer resetPersist()
	setPersist()
	wf := flow.RegisterFlow("TestCallbackFailedPersist0")
	proc := wf.Process("TestCallbackFailedPersist0")
	wf.BeforeFlow(true, Fx[flow.WorkFlow](t).Error().Callback())
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "0")
	ff := flow.DoneFlow("TestCallbackFailedPersist0", nil)
	CheckResult(t, 1, flow.CallbackFail)(any(ff).(flow.WorkFlow))
	CheckFlowPersist(t, ff, 1)

	resetCurrent()
	wf = flow.RegisterFlow("TestCallbackFailedPersist1")
	proc = wf.Process("TestCallbackFailedPersist1")
	wf.BeforeProcess(true, Fx[flow.Process](t).Error().Callback())
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "1")
	ff = flow.DoneFlow("TestCallbackFailedPersist1", nil)
	CheckResult(t, 1, flow.CallbackFail)(any(ff).(flow.WorkFlow))
	CheckFlowPersist(t, ff, 2)

	resetCurrent()
	wf = flow.RegisterFlow("TestCallbackFailedPersist2")
	proc = wf.Process("TestCallbackFailedPersist2")
	wf.BeforeStep(true, Fx[flow.Step](t).Error().Callback())
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "2")
	ff = flow.DoneFlow("TestCallbackFailedPersist2", nil)
	CheckResult(t, 1, flow.CallbackFail)(any(ff).(flow.WorkFlow))
	CheckFlowPersist(t, ff, 3)

	resetCurrent()
	wf = flow.RegisterFlow("TestCallbackFailedPersist3")
	proc = wf.Process("TestCallbackFailedPersist3")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "3")
	ff = flow.DoneFlow("TestCallbackFailedPersist3", nil)
	CheckResult(t, 1, flow.Success)(any(ff).(flow.WorkFlow))
	CheckFlowPersist(t, ff, 3)

	resetCurrent()
	wf = flow.RegisterFlow("TestCallbackFailedPersist4")
	wf.AfterStep(true, Fx[flow.Step](t).Error().Callback())
	proc = wf.Process("TestCallbackFailedPersist4")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "4")
	ff = flow.DoneFlow("TestCallbackFailedPersist4", nil)
	CheckResult(t, 2, flow.CallbackFail)(any(ff).(flow.WorkFlow))
	CheckFlowPersist(t, ff, 3)

	resetCurrent()
	wf = flow.RegisterFlow("TestCallbackFailedPersist5")
	wf.AfterProcess(true, Fx[flow.Process](t).Error().Callback())
	proc = wf.Process("TestCallbackFailedPersist5")
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "5")
	ff = flow.DoneFlow("TestCallbackFailedPersist5", nil)
	CheckResult(t, 2, flow.CallbackFail)(any(ff).(flow.WorkFlow))
	CheckFlowPersist(t, ff, 3)

	resetCurrent()
	wf = flow.RegisterFlow("TestCallbackFailedPersist6")
	proc = wf.Process("TestCallbackFailedPersist6")
	wf.AfterFlow(true, Fx[flow.WorkFlow](t).Error().Callback())
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "6")
	ff = flow.DoneFlow("TestCallbackFailedPersist6", nil)
	CheckResult(t, 2, flow.CallbackFail)(any(ff).(flow.WorkFlow))
	CheckFlowPersist(t, ff, 3)
}

func TestCallbackSucceedPersist(t *testing.T) {
	defer resetCurrent()
	defer resetPersist()
	setPersist()
	wf := flow.RegisterFlow("TestCallbackSucceedPersist0")
	proc := wf.Process("TestCallbackSucceedPersist0")
	wf.BeforeFlow(true, Fx[flow.WorkFlow](t).Inc().Callback())
	wf.BeforeProcess(true, Fx[flow.Process](t).Inc().Callback())
	wf.BeforeStep(true, Fx[flow.Step](t).Inc().Callback())
	wf.AfterStep(true, Fx[flow.Step](t).Inc().Callback())
	wf.AfterProcess(true, Fx[flow.Process](t).Inc().Callback())
	wf.AfterFlow(true, Fx[flow.WorkFlow](t).Inc().Callback())
	proc.NameStep(Fx[flow.Step](t).Inc().Step(), "0")
	ff := flow.DoneFlow("TestCallbackSucceedPersist0", nil)
	CheckResult(t, 7, flow.Success)(any(ff).(flow.WorkFlow))
	CheckFlowPersist(t, ff, 3)
}
