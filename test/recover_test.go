package test

import (
	"bufio"
	"errors"
	"fmt"
	flow "github.com/Bilibotter/light-flow"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
	"os"
	"strings"
	"sync/atomic"
	"testing"
)

var (
	pool = make(chan *gorm.DB, 3)
)

var (
	executeSuc = false
	username   string
	password   string
	host       string
	dbname     string
)

type StepFuncBuilder struct {
	f func(ctx flow.Step) (any, error)
}

type simpleContext struct {
	table map[string]any
	name  string
}

func (s *simpleContext) Get(key string) (value any, exist bool) {
	value, exist = s.table[key]
	return
}

func (s *simpleContext) Set(key string, value any) {
	s.table[key] = value
}

func (s *simpleContext) Name() string {
	return s.name
}

func (s *StepFuncBuilder) Do(f func(ctx flow.Step) (any, error)) *StepFuncBuilder {
	old := s.f
	if old == nil {
		old = func(ctx flow.Step) (any, error) {
			return nil, nil
		}
	}
	curr := func(ctx flow.Step) (any, error) {
		res1, err1 := old(ctx)
		res2, err2 := f(ctx)
		if err1 != nil {
			return res1, err1
		}
		return res2, err2
	}
	s.f = curr
	return s
}

func (s *StepFuncBuilder) End() func(ctx flow.Step) (any, error) {
	return s.f
}

func Fn() *StepFuncBuilder {
	return &StepFuncBuilder{}
}

type Checkpoints struct {
	Id        string `gorm:"column:id;primary_key"`
	Name      string `gorm:"column:name;NOT NULL"`
	RecoverId string `gorm:"column:recover_id"`
	ParentId  string `gorm:"column:parent_id"`
	RootId    string `gorm:"column:root_id"`
	Scope     uint8  `gorm:"column:scope;NOT NULL"`
	Snapshot  []byte `gorm:"column:snapshot"`
}

type Person struct {
	Name string
	Age  int
}

type Named interface {
	name() string
}

func (p *Person) name() string {
	return p.Name
}

func (c Checkpoints) GetPrimaryKey() string {
	return c.Id
}

func (c Checkpoints) GetName() string {
	return c.Name
}

func (c Checkpoints) GetParentId() string {
	return c.ParentId
}

func (c Checkpoints) GetRootId() string {
	return c.RootId
}

func (c Checkpoints) GetScope() uint8 {
	return c.Scope
}

func (c Checkpoints) GetRecoverId() string {
	return c.RecoverId
}

func (c Checkpoints) GetSnapshot() []byte {
	return c.Snapshot
}

type RecoverRecords struct {
	RootId    string `gorm:"column:root_id;NOT NULL"`
	RecoverId string `gorm:"column:recover_id;primary_key"`
	Status    uint8  `gorm:"column:status;NOT NULL"`
	Name      string `gorm:"column:name;NOT NULL"`
}

func (r RecoverRecords) GetRootId() string {
	return r.RootId
}

func (r RecoverRecords) GetRecoverId() string {
	return r.RecoverId
}

func (r RecoverRecords) GetStatus() uint8 {
	return r.Status
}

func (r RecoverRecords) GetName() string {
	return r.Name
}

type persisitImpl struct {
}

func (p persisitImpl) GetLatestRecord(rootId string) (flow.RecoverRecord, error) {
	db := <-pool
	defer func() {
		pool <- db
	}()
	var record RecoverRecords
	result := db.Where("root_id = ?", rootId).Where("status = ?", uint8(flow.RecoverIdle)).First(&record)
	if result.Error != nil {
		return record, result.Error
	}
	return record, nil
}

func (p persisitImpl) ListCheckpoints(recoveryId string) ([]flow.CheckPoint, error) {
	db := <-pool
	defer func() {
		pool <- db
	}()
	var checkpoints []Checkpoints
	result := db.Where("recover_id = ?", recoveryId).Find(&checkpoints)
	if result.Error != nil {
		panic(result.Error)
	}
	cps := make([]flow.CheckPoint, len(checkpoints))
	for i, cp := range checkpoints {
		cps[i] = cp
	}
	return cps, nil
}

func (p persisitImpl) UpdateRecordStatus(record flow.RecoverRecord) error {
	db := <-pool
	defer func() {
		pool <- db
	}()
	result := db.Model(&RecoverRecords{}).Where("recover_id = ?", record.GetRecoverId()).Update("status", record.GetStatus())
	if result.Error != nil {
		panic(result.Error)
	}
	return nil
}

func (p persisitImpl) SaveCheckpointAndRecord(cps []flow.CheckPoint, records flow.RecoverRecord) error {
	db := <-pool
	defer func() {
		pool <- db
	}()
	tx := db.Begin()
	checkpoints := make([]Checkpoints, len(cps))
	for i, cp := range cps {
		checkpoints[i] = Checkpoints{
			Id:        cp.GetPrimaryKey(),
			Name:      cp.GetName(),
			RecoverId: cp.GetRecoverId(),
			ParentId:  cp.GetParentId(),
			RootId:    cp.GetRootId(),
			Scope:     cp.GetScope(),
			Snapshot:  cp.GetSnapshot(),
		}
	}
	if err := tx.Create(&checkpoints).Error; err != nil {
		tx.Rollback()
		panic(err)
	}
	record := RecoverRecords{
		RootId:    records.GetRootId(),
		RecoverId: records.GetRecoverId(),
		Status:    records.GetStatus(),
		Name:      records.GetName(),
	}
	if err := tx.Create(&record).Error; err != nil {
		tx.Rollback()
		panic(err)
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		panic(err)
	}
	return nil
}

type Ctx interface {
	Get(key string) (value any, exist bool)
	Set(key string, value any)
	Name() string
}

func StepSet(ctx Ctx) {
	ctx.Set("int", 1)
	ctx.Set("int8", int8(2))
	ctx.Set("int16", int16(3))
	ctx.Set("int32", int32(4))
	ctx.Set("int64", int64(5))
	ctx.Set("uint", uint(6))
	ctx.Set("uint8", uint8(7))
	ctx.Set("uint16", uint16(8))
	ctx.Set("uint32", uint32(9))
	ctx.Set("uint64", uint64(10))
	ctx.Set("float32", float32(11.01))
	ctx.Set("float64", float64(12.01))
	ctx.Set("bool", true)
	ctx.Set("string", ctx.Name())
	ctx.Set("password", ctx.Name())
	ctx.Set("pwd", ctx.Name())
	ctx.Set("Person", Person{ctx.Name(), 11})
	ctx.Set("*Person", &Person{ctx.Name(), 11})
	ctx.Set("Named", &Person{ctx.Name(), 11})
}

func StepSet0(ctx Ctx) {
	ctx.Set("int+", 1)
	ctx.Set("int8+", int8(2))
	ctx.Set("int16+", int16(3))
	ctx.Set("int32+", int32(4))
	ctx.Set("int64+", int64(5))
	ctx.Set("uint+", uint(6))
	ctx.Set("uint8+", uint8(7))
	ctx.Set("uint16+", uint16(8))
	ctx.Set("uint32+", uint32(9))
	ctx.Set("uint64+", uint64(10))
	ctx.Set("float32+", float32(11.01))
	ctx.Set("float64+", float64(12.01))
	ctx.Set("bool+", true)
	ctx.Set("string+", ctx.Name())
	ctx.Set("password+", ctx.Name())
	ctx.Set("pwd+", ctx.Name())
	ctx.Set("Person+", Person{ctx.Name(), 11})
	ctx.Set("*Person+", &Person{ctx.Name(), 11})
	ctx.Set("Named+", &Person{ctx.Name(), 11})
}

func StepCheck(preview string, ctx Ctx) (string, bool) {
	if value, exist := ctx.Get("int"); exist {
		if value.(int) != 1 {
			return "step-int", false
		}
	} else {
		return "step-int", false
	}
	if value, exist := ctx.Get("int8"); exist {
		if value.(int8) != 2 {
			return "step-int8", false
		}
	} else {
		return "step-int8", false
	}
	if value, exist := ctx.Get("int16"); exist {
		if value.(int16) != 3 {
			return "step-int16", false
		}
	} else {
		return "step-int16", false
	}
	if value, exist := ctx.Get("int32"); exist {
		if value.(int32) != 4 {
			return "step-int32", false
		}
	} else {
		return "step-int32", false
	}
	if value, exist := ctx.Get("int64"); exist {
		if value.(int64) != 5 {
			return "step-int64", false
		}
	} else {
		return "step-int64", false
	}
	if value, exist := ctx.Get("uint"); exist {
		if value.(uint) != 6 {
			return "step-uint", false
		}
	} else {
		return "step-uint", false
	}
	if value, exist := ctx.Get("uint8"); exist {
		if value.(uint8) != 7 {
			return "step-uint8", false
		}
	} else {
		return "step-uint8", false
	}
	if value, exist := ctx.Get("uint16"); exist {
		if value.(uint16) != 8 {
			return "step-uint16", false
		}
	} else {
		return "step-uint16", false
	}
	if value, exist := ctx.Get("uint32"); exist {
		if value.(uint32) != 9 {
			return "step-uint32", false
		}
	} else {
		return "step-uint32", false
	}
	if value, exist := ctx.Get("uint64"); exist {
		if value.(uint64) != 10 {
			return "step-uint64", false
		}
	} else {
		return "step-uint64", false
	}
	if value, exist := ctx.Get("float32"); exist {
		if value.(float32) != 11.01 {
			return "step-float32", false
		}
	} else {
		return "step-float32", false
	}
	if value, exist := ctx.Get("float64"); exist {
		if value.(float64) != 12.01 {
			return "step-float64", false
		}
	} else {
		return "step-float64", false
	}
	if value, exist := ctx.Get("bool"); exist {
		if value.(bool) != true {
			return "step-bool", false
		}
	} else {
		return "step-bool", false
	}
	if value, exist := ctx.Get("Person"); exist {
		if value.(Person).Name != preview || value.(Person).Age != 11 {
			return fmt.Sprintf("step-person[%#v]", value.(Person)), false
		}
	} else {
		return "step-person", false
	}
	if value, exist := ctx.Get("*Person"); exist {
		if value.(*Person).Name != preview || value.(*Person).Age != 11 {
			return "*step-person", false
		}
	} else {
		return "*step-person", false
	}
	if value, exist := ctx.Get("Named"); exist {
		if value.(*Person).Name != preview || value.(*Person).Age != 11 {
			return "Named", false
		}
	} else {
		return "Named", false
	}
	if value, exist := ctx.Get("string"); exist {
		if value.(string) != preview {
			return "step-string", false
		}
	} else {
		return "step-string", false
	}
	if value, exist := ctx.Get("password"); exist {
		if value.(string) != preview {
			return "step-password", false
		}
	} else {
		return "step-password", false
	}
	if value, exist := ctx.Get("pwd"); exist {
		if value.(string) != preview {
			return "step-pwd", false
		}
	} else {
		return "step-pwd", false
	}
	return "", true
}

func StepCheck0(preview string, ctx Ctx) (string, bool) {
	if value, exist := ctx.Get("int+"); exist {
		if value.(int) != 1 {
			return "step-int", false
		}
	} else {
		return "step-int", false
	}
	if value, exist := ctx.Get("int8+"); exist {
		if value.(int8) != 2 {
			return "step-int8", false
		}
	} else {
		return "step-int8", false
	}
	if value, exist := ctx.Get("int16+"); exist {
		if value.(int16) != 3 {
			return "step-int16", false
		}
	} else {
		return "step-int16", false
	}
	if value, exist := ctx.Get("int32+"); exist {
		if value.(int32) != 4 {
			return "step-int32", false
		}
	} else {
		return "step-int32", false
	}
	if value, exist := ctx.Get("int64+"); exist {
		if value.(int64) != 5 {
			return "step-int64", false
		}
	} else {
		return "step-int64", false
	}
	if value, exist := ctx.Get("uint+"); exist {
		if value.(uint) != 6 {
			return "step-uint", false
		}
	} else {
		return "step-uint", false
	}
	if value, exist := ctx.Get("uint8+"); exist {
		if value.(uint8) != 7 {
			return "step-uint8", false
		}
	} else {
		return "step-uint8", false
	}
	if value, exist := ctx.Get("uint16+"); exist {
		if value.(uint16) != 8 {
			return "step-uint16", false
		}
	} else {
		return "step-uint16", false
	}
	if value, exist := ctx.Get("uint32+"); exist {
		if value.(uint32) != 9 {
			return "step-uint32", false
		}
	} else {
		return "step-uint32", false
	}
	if value, exist := ctx.Get("uint64+"); exist {
		if value.(uint64) != 10 {
			return "step-uint64", false
		}
	} else {
		return "step-uint64", false
	}
	if value, exist := ctx.Get("float32+"); exist {
		if value.(float32) != 11.01 {
			return "step-float32", false
		}
	} else {
		return "step-float32", false
	}
	if value, exist := ctx.Get("float64+"); exist {
		if value.(float64) != 12.01 {
			return "step-float64", false
		}
	} else {
		return "step-float64", false
	}
	if value, exist := ctx.Get("bool+"); exist {
		if value.(bool) != true {
			return "step-bool", false
		}
	} else {
		return "step-bool", false
	}
	if value, exist := ctx.Get("Person+"); exist {
		if value.(Person).Name != preview || value.(Person).Age != 11 {
			return fmt.Sprintf("step-person[%#v]", value.(Person)), false
		}
	} else {
		return "step-person", false
	}
	if value, exist := ctx.Get("*Person+"); exist {
		if value.(*Person).Name != preview || value.(*Person).Age != 11 {
			return "*step-person", false
		}
	} else {
		return "*step-person", false
	}
	if value, exist := ctx.Get("Named+"); exist {
		if value.(*Person).Name != preview || value.(*Person).Age != 11 {
			return "Named", false
		}
	} else {
		return "Named", false
	}
	if value, exist := ctx.Get("string+"); exist {
		if value.(string) != preview {
			return "step-string", false
		}
	} else {
		return "step-string", false
	}
	return "", true
}

func empty(ctx Ctx) {}

func GenerateAfter(t *testing.T, preview string, setValue func(c Ctx), checks ...func(p string, ctx Ctx) (string, bool)) func(ctx flow.Step) (any, error) {
	return func(ctx flow.Step) (any, error) {
		t.Logf("step[%s] start\n", ctx.Name())
		if !executeSuc {
			t.Logf("step[%s] execute error\n", ctx.Name())
			return nil, fmt.Errorf("execute failed")
		}
		if result, exist := ctx.Result(preview); !exist {
			if !exist {
				panic(fmt.Sprintf("step[%s] get result failed, step[%s] reuslt is missing.", ctx.Name(), preview))
			}
			if result.(string) != preview {
				panic(fmt.Sprintf("step[%s] get result failed, step[%s] reuslt is not equal to %s", ctx.Name(), preview, preview))
			}
		}
		for _, check := range checks {
			info, ok := check(preview, ctx)
			if !ok {
				panic(fmt.Sprintf("step[%s] check %s failed", ctx.Name(), info))
			}
		}
		setValue(ctx)
		atomic.AddInt64(&current, 1)
		return ctx.Name(), nil
	}
}

func PreviewProcessRecover(process flow.Process) (keepOn bool, err error) {
	if !executeSuc {
		StepSet(process)
		return true, nil
	}
	StepCheck(process.Name(), process)
	atomic.AddInt64(&current, 1)
	return true, nil
}

func GeneratePreview(t *testing.T, preview string, setValue func(c Ctx), checks ...func(p string, ctx Ctx) (string, bool)) func(ctx flow.Step) (any, error) {
	return func(ctx flow.Step) (any, error) {
		t.Logf("step[%s] start\n", ctx.Name())
		if executeSuc {
			if result, exist := ctx.Result(preview); !exist {
				panic(fmt.Sprintf("step[%s] get result failed, step[%s] reuslt is missing.", ctx.Name(), preview))
			} else if result.(string) != preview {
				panic(fmt.Sprintf("step[%s] get result failed, step[%s] reuslt is not equal to %s", ctx.Name(), preview, preview))
			}
			for _, check := range checks {
				info, ok := check(preview, ctx)
				if !ok {
					panic(fmt.Sprintf("step[%s] check %s failed", ctx.Name(), info))
				}
			}
			atomic.AddInt64(&current, 1)
			return ctx.Name(), nil
		}
		setValue(ctx)
		atomic.AddInt64(&current, 1)
		return ctx.Name(), nil
	}
}

func execSuc() bool {
	if executeSuc {
		return true
	}
	return false
}

func execFail() bool {
	if executeSuc {
		return false
	}
	return true
}

func readDBConfig(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	config := make(map[string]string)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid config line: %s", line)
		}
		config[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	username = config["username"]
	password = config["password"]
	host = config["host"]
	dbname = config["dbname"]
	return nil
}

func getId(name string) (string, error) {
	db := <-pool
	defer func() {
		pool <- db
	}()
	var record RecoverRecords
	result := db.Where("name = ?", name).Where("status = ?", uint8(flow.RecoverIdle)).First(&record)
	if result.Error != nil {
		return "", result.Error
	}
	return record.GetRootId(), nil
}

func TestMain(m *testing.M) {
	if err := readDBConfig("mysql.txt"); err != nil {
		log.Fatalf("Failed to read DB config: %v", err)
	}
	dataSourceName := "%s:%s@tcp(%s:3306)/%s?charset=utf8mb4&parseTime=True&loc=Local"
	dataSourceName = fmt.Sprintf(dataSourceName, username, password, host, dbname)
	for i := 0; i < 3; i++ {
		db, err := gorm.Open(mysql.Open(dataSourceName), &gorm.Config{})
		if err != nil {
			log.Fatalf("Failed to open database: %v", err)
		}
		pool <- db
	}
	flow.SetPersist(&persisitImpl{})
	flow.RegisterType[Person]()
	os.Exit(m.Run())
}

func TestCombineStepRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestCombineStepRecover")
	wf.EnableRecover()
	proc := wf.Process("TestCombineStepRecover")
	proc.NameStep(func(ctx flow.Step) (any, error) {
		t.Logf("step[%s] start\n", ctx.Name())
		StepSet(ctx)
		atomic.AddInt64(&current, 1)
		return ctx.Name(), nil
	}, "step1")
	step2 := proc.NameStep(func(ctx flow.Step) (any, error) {
		t.Logf("step[%s] start\n", ctx.Name())
		if !executeSuc {
			info, ok := StepCheck("step1", ctx)
			if !ok {
				panic(fmt.Sprintf("step[%s] check %s failed", ctx.Name(), info))
			}
			StepSet(ctx)
		}
		atomic.AddInt64(&current, 1)
		return ctx.Name(), nil
	}, "step2", "step1")
	step2.Next(GenerateAfter(t, "step2", StepSet, StepCheck), "step3").
		Next(GenerateAfter(t, "step3", StepSet, StepCheck), "step4")
	wf.AfterFlow(false, CheckResult(t, 2, flow.Error)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 2, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	flow.DoneFlow("TestCombineStepRecover", nil)
	executeSuc = true
	flowId, err := getId("TestCombineStepRecover")
	if err != nil {
		panic(err)
	}
	resetCurrent()
	if err = flow.RecoverFlow(flowId); err != nil {
		panic(err)
	}
}

func TestSeparateStepRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestSeparateStepRecover")
	wf.EnableRecover()
	proc := wf.Process("TestSeparateStepRecover")
	proc.NameStep(func(ctx flow.Step) (any, error) {
		t.Logf("step[%s] start\n", ctx.Name())
		StepSet(ctx)
		atomic.AddInt64(&current, 1)
		return ctx.Name(), nil
	}, "step1")
	proc.NameStep(func(ctx flow.Step) (any, error) {
		t.Logf("step[%s] start\n", ctx.Name())
		StepSet0(ctx)
		atomic.AddInt64(&current, 1)
		return ctx.Name(), nil
	}, "step2", "step1")
	f := Fn().
		Do(GenerateAfter(t, "step1", StepSet, StepCheck)).
		Do(GenerateAfter(t, "step2", StepSet0, StepCheck0))
	step := proc.NameStep(f.End(), "step3", "step1", "step2")
	step.Next(GenerateAfter(t, "step3", StepSet, StepCheck), "step4")
	step.Next(GenerateAfter(t, "step3", StepSet0, StepCheck0), "step5")
	wf.AfterFlow(false, CheckResult(t, 2, flow.Error)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 4, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	flow.DoneFlow("TestSeparateStepRecover", nil)
	executeSuc = true
	flowId, err := getId("TestSeparateStepRecover")
	if err != nil {
		panic(err)
	}
	resetCurrent()
	if err = flow.RecoverFlow(flowId); err != nil {
		panic(err)
	}
}

func TestSingleStepRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestSingleStepRecover")
	wf.EnableRecover()
	proc := wf.Process("TestSingleStepRecover")
	proc.NameStep(GeneratePreview(t, "step1", StepSet, StepCheck), "step1").
		Next(GenerateAfter(t, "step1", StepSet, StepCheck), "step2").
		Next(GenerateAfter(t, "step2", StepSet, StepCheck), "step3")
	wf.AfterFlow(false, CheckResult(t, 1, flow.Error)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 2, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	flow.DoneFlow("TestSingleStepRecover", nil)
	executeSuc = true
	flowId, err := getId("TestSingleStepRecover")
	if err != nil {
		panic(err)
	}
	resetCurrent()
	if err = flow.RecoverFlow(flowId); err != nil {
		panic(err)
	}
}

func TestPanicStepRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestPanicStepRecover")
	wf.EnableRecover()
	proc := wf.Process("TestPanicStepRecover")
	proc.NameStep(func(ctx flow.Step) (any, error) {
		if !executeSuc {
			StepSet(ctx)
			panic("panic")
		}
		t.Logf("step[%s] start\n", ctx.Name())
		StepCheck(ctx.Name(), ctx)
		atomic.AddInt64(&current, 1)
		return ctx.Name(), nil
	}, "step1").
		Next(GenerateAfter(t, "step1", StepSet, StepCheck), "step2").
		Next(GenerateAfter(t, "step2", StepSet, StepCheck), "step3")
	wf.AfterFlow(false, CheckResult(t, 0, flow.Panic)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 3, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	flow.DoneFlow("TestPanicStepRecover", nil)
	executeSuc = true
	flowId, err := getId("TestPanicStepRecover")
	if err != nil {
		panic(err)
	}
	resetCurrent()
	if err = flow.RecoverFlow(flowId); err != nil {
		panic(err)
	}
}

func TestMultipleStepRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestMultipleStepRecover")
	wf.EnableRecover()
	proc1 := wf.Process("TestMultipleStepRecover1")
	step := proc1.NameStep(GeneratePreview(t, "step1-1", StepSet, StepCheck), "step1-1")
	step.
		Next(GenerateAfter(t, "step1-1", StepSet, StepCheck), "step2-2").
		Next(GenerateAfter(t, "step2-2", StepSet, StepCheck), "step2-3")
	step.
		Next(GenerateAfter(t, "step1-1", StepSet, StepCheck), "step3-2").
		Next(GenerateAfter(t, "step3-2", StepSet, StepCheck), "step3-3")
	wf.AfterFlow(false, CheckResult(t, 1, flow.Error)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 4, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	flow.DoneFlow("TestMultipleStepRecover", nil)
	executeSuc = true
	flowId, err := getId("TestMultipleStepRecover")
	if err != nil {
		panic(err)
	}
	resetCurrent()
	if err = flow.RecoverFlow(flowId); err != nil {
		panic(err)
	}
}

func TestMultipleStepRecover0(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestMultipleStepRecover+")
	wf.EnableRecover()
	proc1 := wf.Process("TestMultipleStepRecover+")
	proc1.NameStep(GeneratePreview(t, "step1-1", StepSet, StepCheck), "step1-1")
	proc1.NameStep(GeneratePreview(t, "step1-2", StepSet0, StepCheck0), "step1-2")
	proc1.NameStep(GenerateAfter(t, "step1-1", StepSet, StepCheck), "step2-1", "step1-1")
	proc1.NameStep(GenerateAfter(t, "step1-2", StepSet0, StepCheck0), "step2-2", "step1-2")
	wf.AfterFlow(false, CheckResult(t, 2, flow.Error)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 2, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	flow.DoneFlow("TestMultipleStepRecover+", nil)
	executeSuc = true
	flowId, err := getId("TestMultipleStepRecover+")
	if err != nil {
		panic(err)
	}
	resetCurrent()
	if err = flow.RecoverFlow(flowId); err != nil {
		panic(err)
	}
}

func TestParallelStepRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestParallelStepRecover")
	wf.EnableRecover()
	proc1 := wf.Process("TestParallelStepRecover1")
	proc1.NameStep(GeneratePreview(t, "step1-1", StepSet, StepCheck), "step1-1").
		Next(GenerateAfter(t, "step1-1", StepSet, StepCheck), "step1-2").
		Next(GenerateAfter(t, "step1-2", StepSet, StepCheck), "step1-3")
	proc2 := wf.Process("TestParallelStepRecover2")
	proc2.NameStep(GeneratePreview(t, "step2-1", StepSet, StepCheck), "step2-1").
		Next(GenerateAfter(t, "step2-1", StepSet, StepCheck), "step2-2").
		Next(GenerateAfter(t, "step2-2", StepSet, StepCheck), "step2-3")
	proc3 := wf.Process("TestParallelStepRecover3")
	proc3.NameStep(GeneratePreview(t, "step3-1", StepSet, StepCheck), "step3-1").
		Next(GenerateAfter(t, "step3-1", StepSet, StepCheck), "step3-2").
		Next(GenerateAfter(t, "step3-2", StepSet, StepCheck), "step3-3")
	wf.AfterFlow(false, CheckResult(t, 3, flow.Error)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 6, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	flow.DoneFlow("TestParallelStepRecover", nil)
	executeSuc = true
	flowId, err := getId("TestParallelStepRecover")
	if err != nil {
		panic(err)
	}
	resetCurrent()
	if err = flow.RecoverFlow(flowId); err != nil {
		panic(err)
	}
}

func TestFlowRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestFlowRecover")
	wf.EnableRecover()
	proc := wf.Process("TestFlowRecover")
	proc.NameStep(func(ctx flow.Step) (any, error) {
		StepCheck("TestFlowRecover", ctx)
		if !executeSuc {
			StepSet(ctx)
			return nil, errors.New("error")
		}
		atomic.AddInt64(&current, 1)
		return ctx.Name(), nil
	}, "step1")
	proc.NameStep(GenerateAfter(t, "step1", StepSet, StepCheck), "step2", "step1")
	proc.NameStep(GenerateAfter(t, "step2", StepSet, StepCheck), "step3", "step2")
	wf.AfterFlow(false, CheckResult(t, 0, flow.Error)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 3, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	m := simpleContext{
		name:  "TestFlowRecover",
		table: make(map[string]any),
	}
	StepSet(&m)
	flow.DoneFlow("TestFlowRecover", m.table)
	executeSuc = true
	flowId, err := getId("TestFlowRecover")
	if err != nil {
		panic(err)
	}
	resetCurrent()
	if err = flow.RecoverFlow(flowId); err != nil {
		panic(err)
	}
}

func TestProcessRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestProcessRecover")
	wf.EnableRecover()
	proc := wf.Process("TestProcessRecover")
	proc.NameStep(func(ctx flow.Step) (any, error) {
		fmt.Printf("step[%s] start\n", ctx.Name())
		atomic.AddInt64(&current, 1)
		return ctx.Name(), nil
	}, "step1")
	proc.NameStep(func(ctx flow.Step) (any, error) {
		fmt.Printf("step[%s] start\n", ctx.Name())
		atomic.AddInt64(&current, 1)
		return ctx.Name(), nil
	}, "step2", "step1")
	proc.NameStep(func(ctx flow.Step) (any, error) {
		fmt.Printf("step[%s] start\n", ctx.Name())
		if executeSuc {
			if res, exist := ctx.Result("step1"); exist {
				if res.(string) != "step1" {
					panic(fmt.Sprintf("step[%s] get result failed, step[%s] reuslt is not equal to %s", ctx.Name(), "step1", "step1"))
				}
			} else {
				panic(fmt.Sprintf("step[%s] get result failed, step[%s] reuslt is missing.", ctx.Name(), "step1"))
			}
			if res, exist := ctx.Result("step2"); exist {
				if res.(string) != "step2" {
					panic(fmt.Sprintf("step[%s] get result failed, step[%s] reuslt is not equal to %s", ctx.Name(), "step2", "step2"))
				}
			} else {
				panic(fmt.Sprintf("step[%s] get result failed, step[%s] reuslt is missing.", ctx.Name(), "step2"))
			}
			StepCheck("TestProcessRecover", ctx)
			atomic.AddInt64(&current, 1)
			return ctx.Name(), nil
		}
		fmt.Printf("step[%s] failed\n", ctx.Name())
		return nil, fmt.Errorf("error")
	}, "step3", "step1", "step2")
	proc.NameStep(func(ctx flow.Step) (any, error) {
		fmt.Printf("step[%s] start\n", ctx.Name())
		atomic.AddInt64(&current, 1)
		return ctx.Name(), nil
	}, "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 2, flow.Error)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 3, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	wf.BeforeProcess(false, PreviewProcessRecover)
	flow.DoneFlow("TestProcessRecover", nil)
	executeSuc = true
	flowId, err := getId("TestProcessRecover")
	if err != nil {
		panic(err)
	}
	resetCurrent()
	if err = flow.RecoverFlow(flowId); err != nil {
		panic(err)
	}
}

func TestStepInterruptRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestStepInterruptRecover")
	wf.EnableRecover()
	proc := wf.Process("TestStepInterruptRecover")
	proc.NameStep(func(ctx flow.Step) (any, error) {
		fmt.Printf("step[%s] start\n", ctx.Name())
		StepSet0(ctx)
		atomic.AddInt64(&current, 1)
		return ctx.Name(), nil
	}, "step1")
	proc.NameStep(func(ctx flow.Step) (any, error) {
		fmt.Printf("step[%s] start\n", ctx.Name())
		if !executeSuc {
			StepSet(ctx)
			return ctx.Name(), fmt.Errorf("error")
		}
		StepCheck0("step1", ctx)
		StepCheck("step2", ctx)
		atomic.AddInt64(&current, 1)
		return ctx.Name(), nil
	}, "step2", "step1")
	proc.NameStep(GenerateStep(3), "step3", "step2")
	wf.AfterFlow(false, CheckResult(t, 1, flow.Error)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 2, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	flow.DoneFlow("TestStepInterruptRecover", nil)
	executeSuc = true
	flowId, err := getId("TestStepInterruptRecover")
	if err != nil {
		panic(err)
	}
	resetCurrent()
	if err = flow.RecoverFlow(flowId); err != nil {
		panic(err)
	}
}

func TestPool(t *testing.T) {
	db := <-pool
	defer func() { pool <- db }()
	t.Logf("success")
}
