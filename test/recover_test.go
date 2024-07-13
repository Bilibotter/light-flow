package test

import (
	"bufio"
	"fmt"
	flow "github.com/Bilibotter/light-flow"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var (
	pool = make(chan *gorm.DB, 3)
)

var (
	username string
	password string
	host     string
	dbname   string
)

type Checkpoints struct {
	Id        string    `gorm:"column:id;primary_key"`
	Uid       string    `gorm:"column:uid"`
	Name      string    `gorm:"column:name;NOT NULL"`
	RecoverId string    `gorm:"column:recover_id"`
	ParentUid string    `gorm:"column:parent_uid"`
	RootUid   string    `gorm:"column:root_uid"`
	Scope     uint8     `gorm:"column:scope;NOT NULL"`
	Snapshot  []byte    `gorm:"column:snapshot"`
	CreatedAt time.Time `gorm:"type:datetime;column:created_at;"`
	UpdatedAt time.Time `gorm:"type:datetime;column:updated_at;"`
}

func (c Checkpoints) GetId() string {
	return c.Id
}

func (c Checkpoints) GetUid() string {
	return c.Uid
}

func (c Checkpoints) GetName() string {
	return c.Name
}

func (c Checkpoints) GetParentUid() string {
	return c.ParentUid
}

func (c Checkpoints) GetRootUid() string {
	return c.RootUid
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
	RootUid   string    `gorm:"column:root_uid;NOT NULL"`
	RecoverId string    `gorm:"column:recover_id;primary_key"`
	Status    uint8     `gorm:"column:status;NOT NULL"`
	Name      string    `gorm:"column:name;NOT NULL"`
	CreatedAt time.Time `gorm:"type:datetime;column:created_at;"`
	UpdatedAt time.Time `gorm:"type:datetime;column:updated_at;"`
}

func (r RecoverRecords) GetRootUid() string {
	return r.RootUid
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

func (p persisitImpl) GetLatestRecord(rootUid string) (flow.RecoverRecord, error) {
	db := <-pool
	defer func() {
		pool <- db
	}()
	var record RecoverRecords
	result := db.Where("root_uid = ?", rootUid).Where("status = ?", flow.RecoverIdle).First(&record)
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
			Id:        cp.GetId(),
			Uid:       cp.GetUid(),
			Name:      cp.GetName(),
			RecoverId: cp.GetRecoverId(),
			ParentUid: cp.GetParentUid(),
			RootUid:   cp.GetRootUid(),
			Scope:     cp.GetScope(),
			Snapshot:  cp.GetSnapshot(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}
	if err := tx.Create(&checkpoints).Error; err != nil {
		tx.Rollback()
		panic(err)
	}
	record := RecoverRecords{
		RootUid:   records.GetRootUid(),
		RecoverId: records.GetRecoverId(),
		Status:    records.GetStatus(),
		Name:      records.GetName(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
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

func Recover0(t *testing.T, ff flow.FinishedWorkFlow) flow.FinishedWorkFlow {
	atomic.AddInt64(&times, 1)
	resetCurrent()
	println()
	t.Logf("start [%d]times recover >>>>>>>>>>>>>>", times)
	f, err := ff.Recover()
	if err != nil {
		panic(err)
	}
	println()
	t.Logf("end [%d]times recover <<<<<<<<<<<<<<<", times)
	return f
}

func Recover(name string) {
	println("\n\t----------Recovering----------\n")
	executeSuc = true
	flowId, err := getId(name)
	if err != nil {
		panic(err)
	}
	resetCurrent()
	if _, err = flow.RecoverFlow(flowId); err != nil {
		panic(err)
	}
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
	return record.GetRootUid(), nil
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
	flow.SetMaxSerializeSize(10240)
	flow.SetPersist(&persisitImpl{})
	flow.RegisterType[Person]()
	os.Exit(m.Run())
}

func TestCombineStepRecover(t *testing.T) {
	defer resetCurrent()
	defer flow.DefaultConfig().DisableRecover()
	flow.DefaultConfig().EnableRecover()
	executeSuc = false
	wf := flow.RegisterFlow("TestCombineStepRecover")
	proc := wf.Process("TestCombineStepRecover")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step1").
		Next(Fn(t).Do(SetCtx()).Step(), "step2").
		Next(Fn(t).Suc(CheckCtx("step2"), SetCtx()).ErrStep(), "step3").
		Next(Fn(t).Do(CheckCtx("step3")).Step(), "step4")
	wf.AfterFlow(false, CheckResult(t, 2, flow.Error)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 2, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	flow.DoneFlow("TestCombineStepRecover", nil)
	Recover("TestCombineStepRecover")
}

func TestSeparateStepRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestSeparateStepRecover")
	wf.EnableRecover()
	proc := wf.Process("TestSeparateStepRecover")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.NameStep(Fn(t).Do(SetCtx0("+")).Step(), "step2")
	proc.NameStep(Fn(t).
		Suc(CheckCtx("step1"), CheckCtx0("step2", "+"), SetCtx(), SetCtx0("+")).ErrStep(),
		"step3", "step1", "step2")
	proc.NameStep(Fn(t).Suc(CheckCtx("step3"), CheckCtx0("step3", "+")).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 2, flow.Error)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 2, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	flow.DoneFlow("TestSeparateStepRecover", nil)
	Recover("TestSeparateStepRecover")
}

func Test1ProcSuc1ProcRecover(t *testing.T) {
	defer resetCurrent()
	flow.DefaultConfig().DisableRecover()
	executeSuc = false
	wf := flow.RegisterFlow("Test1ProcSuc1ProcRecover")
	wf.EnableRecover()
	proc := wf.Process("Test1ProcSuc1ProcRecover0")
	proc.NameStep(GenerateStep(1), "step1").
		Next(GenerateStep(2), "step2").
		Next(GenerateStep(3), "step3")

	proc = wf.Process("Test1ProcSuc1ProcRecover")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step1").
		Next(Fn(t).Suc(CheckCtx("step1"), SetCtx()).ErrStep(), "step2").
		Next(Fn(t).Fail(SetCtx()).Suc(CheckCtx("step2")).Step(), "step3")
	wf.AfterFlow(false, CheckResult(t, 4, flow.Error)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 2, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	flow.DoneFlow("Test1ProcSuc1ProcRecover", nil)
	Recover("Test1ProcSuc1ProcRecover")
}

func TestSingleStepRecoverWith2Branch(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestSingleStepRecoverWith2Branch")
	wf.EnableRecover()
	proc := wf.Process("TestSingleStepRecoverWith2Branch")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step2-1").
		Next(Fn(t).Do(SetCtx()).Step(), "step2-2")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step1").
		Next(Fn(t).Suc(CheckCtx("step1"), SetCtx()).ErrStep(), "step2").
		Next(Fn(t).Fail(SetCtx()).Suc(CheckCtx("step2")).Step(), "step3")
	wf.AfterFlow(false, CheckResult(t, 3, flow.Error)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 2, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	flow.DoneFlow("TestSingleStepRecoverWith2Branch", nil)
	Recover("TestSingleStepRecoverWith2Branch")
}

func TestRecoverSuccessFlow(t *testing.T) {
	defer resetCurrent()
	id := ""
	wf := flow.RegisterFlow("TestRecoverSuccessFlow")
	wf.EnableRecover()
	proc := wf.Process("TestRecoverSuccessFlow")
	proc.NameStep(Ck(t).SetFn(), "Step1").
		Next(Ck(t).SetFn(), "Step2").
		Next(Ck(t).SetFn(), "Step3")
	wf.AfterFlow(false, func(workFlow flow.WorkFlow) (keepOn bool, err error) {
		id = workFlow.ID()
		return true, nil
	})
	wf.AfterFlow(false, CheckResult(t, 3, flow.Success))
	flow.DoneFlow("TestRecoverSuccessFlow", nil)
	defer func() {
		if r := recover(); r != nil {
			if r != gorm.ErrRecordNotFound {
				t.Errorf("recover error: %v", r)
			}
		} else {
			t.Errorf("recover success, but should fail")
		}
	}()
	_, err := flow.RecoverFlow(id)
	if err != nil {
		panic(err)
	}
}

func TestSingleStepRecoverWith1DependSuc1DependFail0(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestSingleStepRecover1DependSuc1DependFail0")
	wf.EnableRecover()
	proc := wf.Process("TestSingleStepRecover1DependSuc1DependFail0")
	proc.NameStep(Fn(t).Do(SetCtx0("+")).Step(), "step1-1")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step1-2")

	proc.NameStep(Fn(t).
		Do(CheckCtx0("step1-1", "+"), SetCtx0("+")).Step(),
		"step2-1", "step1-1")

	proc.NameStep(Fn(t).
		Suc(CheckCtx("step1-2"), SetCtx()).ErrStep(),
		"step2-2", "step1-2")

	proc.NameStep(Fn(t).Fail(SetCtx()).Suc(CheckCtx0("step2-1", "+"), CheckCtx("step2-2")).Step(),
		"step3-1", "step2-1", "step2-2")

	proc.NameStep(Fn(t).Suc(CheckCtx("step2-2")).ErrStep(),
		"step3-2", "step2-2")

	wf.AfterFlow(false, CheckResult(t, 3, flow.Error)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 3, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	flow.DoneFlow("TestSingleStepRecover1DependSuc1DependFail0", nil)
	Recover("TestSingleStepRecover1DependSuc1DependFail0")
}

func TestSingleStepRecoverWith1DependSuc1DependFail(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestSingleStepRecover1DependSuc1DependFail")
	wf.EnableRecover()
	proc := wf.Process("TestSingleStepRecover1DependSuc1DependFail")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step1-1")

	proc.NameStep(Fn(t).
		Do(CheckCtx("step1-1"), SetCtx0("+")).Step(),
		"step2-1", "step1-1")

	proc.NameStep(Fn(t).
		Suc(CheckCtx("step1-1"), SetCtx()).ErrStep(),
		"step2-2", "step1-1")

	proc.NameStep(Fn(t).Fail(SetCtx()).Suc(CheckCtx0("step2-1", "+"), CheckCtx("step2-2")).Step(),
		"step3-1", "step2-1", "step2-2")

	wf.AfterFlow(false, CheckResult(t, 2, flow.Error)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 2, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	flow.DoneFlow("TestSingleStepRecover1DependSuc1DependFail", nil)
	Recover("TestSingleStepRecover1DependSuc1DependFail")
}

func TestSingleStepRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestSingleStepRecover")
	wf.EnableRecover()
	proc := wf.Process("TestSingleStepRecover")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step1").
		Next(Fn(t).Suc(CheckCtx("step1"), SetCtx()).ErrStep(), "step2").
		Next(Fn(t).Fail(SetCtx()).Suc(CheckCtx("step2")).Step(), "step3")
	wf.AfterFlow(false, CheckResult(t, 1, flow.Error)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 2, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	ret := flow.DoneFlow("TestSingleStepRecover", nil)
	println("\n\t----------Recovering----------\n")
	executeSuc = true
	resetCurrent()
	if _, err := ret.Recover(); err != nil {
		panic(err)
	}
}

func TestPanicStepRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestPanicStepRecover")
	wf.EnableRecover()
	proc := wf.Process("TestPanicStepRecover")
	f := func(ctx Ctx) (any, error) { panic("panic") }
	proc.NameStep(Fn(t).Fail(SetCtx(), f).Suc(CheckCtx("step1", 0)).Step(), "step1").
		Next(Fn(t).Fail(SetCtx()).Suc(CheckCtx("step1"), SetCtx()).Step(), "step2").
		Next(Fn(t).Fail(SetCtx()).Suc(CheckCtx("step2")).Step(), "step3")
	wf.AfterFlow(false, CheckResult(t, 0, flow.Panic)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 3, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	flow.DoneFlow("TestPanicStepRecover", nil)
	Recover("TestPanicStepRecover")
}

func TestMultipleStepRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestMultipleStepRecover")
	wf.EnableRecover()
	proc := wf.Process("TestMultipleStepRecover")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step1-1").
		Next(Fn(t).Suc(CheckCtx("step1-1"), SetCtx()).ErrStep(), "step1-2").
		Next(Fn(t).Do(CheckCtx("step1-2")).Step(), "1-3")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step2-1").
		Next(Fn(t).Suc(CheckCtx("step2-1"), SetCtx()).ErrStep(), "step2-2").
		Next(Fn(t).Do(CheckCtx("step2-2")).Step(), "2-3")
	wf.AfterFlow(false, CheckResult(t, 2, flow.Error)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 4, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	flow.DoneFlow("TestMultipleStepRecover", nil)
	Recover("TestMultipleStepRecover")
}

func TestMultipleStepRecover0(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestMultipleStepRecover0")
	wf.EnableRecover()
	proc := wf.Process("TestMultipleStepRecover0")
	proc.NameStep(Fn(t).Fail(SetCtx()).Suc(CheckCtx("step1-1", 0)).ErrStep(), "step1-1").
		Next(Fn(t).Do(CheckCtx("step1-1")).Step(), "1-2")
	proc.NameStep(Fn(t).Suc(SetCtx()).ErrStep(), "step2-1").
		Next(Fn(t).Do(CheckCtx("step2-1")).Step(), "2-2")
	wf.AfterFlow(false, CheckResult(t, 1, flow.Error)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 4, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	flow.DoneFlow("TestMultipleStepRecover0", nil)
	Recover("TestMultipleStepRecover0")
}

func TestParallelStepRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestParallelStepRecover")
	wf.EnableRecover()
	proc1 := wf.Process("TestParallelStepRecover1")
	proc1.NameStep(Fn(t).Do(SetCtx()).Step(), "step1-1").
		Next(Fn(t).Suc(CheckCtx("step1-1"), SetCtx()).ErrStep(), "step1-2").
		Next(Fn(t).Do(CheckCtx("step1-2")).Step(), "step1-3")
	proc2 := wf.Process("TestParallelStepRecover2")
	proc2.NameStep(Fn(t).Do(SetCtx()).Step(), "step2-1").
		Next(Fn(t).Suc(CheckCtx("step2-1"), SetCtx()).ErrStep(), "step2-2").
		Next(Fn(t).Do(CheckCtx("step2-2")).Step(), "step2-3")
	proc3 := wf.Process("TestParallelStepRecover3")
	proc3.NameStep(Fn(t).Do(SetCtx()).Step(), "step3-1").
		Next(Fn(t).Suc(CheckCtx("step3-1"), SetCtx()).ErrStep(), "step3-2").
		Next(Fn(t).Do(CheckCtx("step3-2")).Step(), "step3-3")
	wf.AfterFlow(false, CheckResult(t, 3, flow.Error)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 6, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	flow.DoneFlow("TestParallelStepRecover", nil)
	Recover("TestParallelStepRecover")
}

func TestFlowInputRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestFlowInputRecover")
	wf.EnableRecover()
	proc := wf.Process("TestFlowInputRecover")
	proc.NameStep(Fn(t).Do(CheckCtx("TestFlowInputRecover", 0)).Suc(SetCtx()).ErrStep(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	wf.AfterFlow(false, CheckResult(t, 1, flow.Error)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 4, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	m := simpleContext{
		name:  "TestFlowInputRecover",
		table: make(map[string]any),
	}
	_, err := SetCtx()(&m)
	if err != nil {
		t.Errorf("set context error %v", err)
	}
	flow.DoneFlow("TestFlowInputRecover", m.table)
	Recover("TestFlowInputRecover")
}

func TestFlowCallbackSkipWhileRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestFlowCallbackSkipWhileRecover")
	wf.EnableRecover()
	proc := wf.Process("TestFlowCallbackSkipWhileRecover")
	proc.NameStep(Fn(t).Do(CheckCtx("TestFlowCallbackSkipWhileRecover", 0)).Suc(SetCtx()).ErrStep(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	wf.BeforeFlow(false, Fn0(t).Normal())
	wf.AfterFlow(false, CheckResult(t, 2, flow.Error)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 4, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	m := simpleContext{
		name:  "TestFlowCallbackSkipWhileRecover",
		table: make(map[string]any),
	}
	_, err := SetCtx()(&m)
	if err != nil {
		t.Errorf("set context error %v", err)
	}
	flow.DoneFlow("TestFlowCallbackSkipWhileRecover", m.table)
	Recover("TestFlowCallbackSkipWhileRecover")
}

func TestPreFlowCallbackFailRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestPreFlowCallbackFailRecover")
	wf.EnableRecover()
	proc := wf.Process("TestPreFlowCallbackFailRecover")
	proc.NameStep(Fn(t).Do(CheckCtx("TestFlowCallbackSkipWhileRecover", 0), SetCtx()).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 5, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	wf.BeforeFlow(true, func(workFlow flow.WorkFlow) (keepOn bool, err error) {
		t.Logf("start Flow[%s] pre-callback", workFlow.Name())
		atomic.AddInt64(&current, 1)
		if executeSuc {
			t.Logf("finish Flow[%s] pre-callback", workFlow.Name())
			return true, nil
		}
		t.Logf("excute Flow[%s] pre-callback failed", workFlow.Name())
		return false, fmt.Errorf("execute failed")
	})
	m := simpleContext{
		name:  "TestFlowCallbackSkipWhileRecover",
		table: make(map[string]any),
	}
	_, err := SetCtx()(&m)
	if err != nil {
		t.Errorf("set context error %v", err)
	}
	ctrl := flow.DoneFlow("TestPreFlowCallbackFailRecover", m.table)
	if !ctrl.HasAny(flow.CallbackFail) {
		t.Errorf("should have callback fail")
	}
	if atomic.LoadInt64(&current) != 1 {
		t.Errorf("current should be 1")
	}
	Recover("TestPreFlowCallbackFailRecover")
}

func TestPreFlowCallbackPanicRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestPreFlowCallbackPanicRecover")
	wf.EnableRecover()
	proc := wf.Process("TestPreFlowCallbackPanicRecover")
	proc.NameStep(Fn(t).Do(CheckCtx("TestFlowCallbackSkipWhileRecover", 0), SetCtx()).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 5, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	wf.BeforeFlow(true, func(workFlow flow.WorkFlow) (keepOn bool, err error) {
		t.Logf("start Flow[%s] pre-callback", workFlow.Name())
		atomic.AddInt64(&current, 1)
		if executeSuc {
			t.Logf("finish Flow[%s] pre-callback", workFlow.Name())
			return true, nil
		}
		t.Logf("excute Flow[%s] pre-callback panic", workFlow.Name())
		panic("panic")
	})
	m := simpleContext{
		name:  "TestFlowCallbackSkipWhileRecover",
		table: make(map[string]any),
	}
	_, err := SetCtx()(&m)
	if err != nil {
		t.Errorf("set context error %v", err)
	}
	ctrl := flow.DoneFlow("TestPreFlowCallbackPanicRecover", m.table)
	if !ctrl.HasAny(flow.CallbackFail) {
		t.Errorf("should have callback fail")
	}
	if atomic.LoadInt64(&current) != 1 {
		t.Errorf("current should be 1")
	}
	Recover("TestPreFlowCallbackPanicRecover")
}

func TestProcessRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestProcessRecover")
	wf.EnableRecover()
	proc := wf.Process("TestProcessRecover")
	proc.NameStep(Fn(t).Do(CheckCtx("TestProcessRecover", 0)).Suc(SetCtx()).ErrStep(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 2, flow.Error)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 5, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	wf.BeforeProcess(false, Fn(t).Do(SetCtx()).Proc())
	flow.DoneFlow("TestProcessRecover", nil)
	Recover("TestProcessRecover")
}

func TestPreProcessCallbackFailRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestPreProcessCallbackFailRecover")
	wf.EnableRecover()
	proc := wf.Process("TestPreProcessCallbackFailRecover")
	proc.NameStep(Fn(t).Do(CheckCtx("TestPreProcessCallbackFailRecover", 0), SetCtx()).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 1, flow.CallbackFail)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 5, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	wf.BeforeProcess(true, Fn(t).Fail(SetCtx()).Suc(CheckCtx("TestPreProcessCallbackFailRecover", 0)).ErrProc())
	flow.DoneFlow("TestPreProcessCallbackFailRecover", nil)
	Recover("TestPreProcessCallbackFailRecover")
}

func TestPreProcessCallbackFailRecover0(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestPreProcessCallbackFailRecover0")
	wf.EnableRecover()
	proc := wf.Process("TestPreProcessCallbackFailRecover0")
	proc.NameStep(Fn(t).Do(CheckCtx("TestPreProcessCallbackFailRecover0", 0), SetCtx()).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 2, flow.CallbackFail)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 5, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	wf.BeforeProcess(true, Fn(t).NormalProc())
	wf.BeforeProcess(true, Fn(t).Fail(SetCtx()).Suc(CheckCtx("TestPreProcessCallbackFailRecover0", 0)).ErrProc())
	flow.DoneFlow("TestPreProcessCallbackFailRecover0", nil)
	Recover("TestPreProcessCallbackFailRecover0")
}

func TestPreProcessCallbackFailRecover1(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestPreProcessCallbackFailRecover1")
	wf.EnableRecover()
	proc := wf.Process("TestPreProcessCallbackFailRecover1")
	proc.NameStep(Fn(t).Do(CheckCtx("TestPreProcessCallbackFailRecover1", 0), SetCtx()).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 1, flow.CallbackFail)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 6, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	wf.BeforeProcess(true, Fn(t).Fail(SetCtx()).Suc(CheckCtx("TestPreProcessCallbackFailRecover1", 0)).ErrProc())
	wf.BeforeProcess(true, Fn(t).NormalProc())
	flow.DoneFlow("TestPreProcessCallbackFailRecover1", nil)
	Recover("TestPreProcessCallbackFailRecover1")
}

func TestPreProcessCallbackFailRecover2(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestPreProcessCallbackFailRecover2")
	wf.EnableRecover()
	proc := wf.Process("TestPreProcessCallbackFailRecover2")
	proc.NameStep(Fn(t).Do(CheckCtx("TestPreProcessCallbackFailRecover2", 0), SetCtx()).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 2, flow.CallbackFail)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 6, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	wf.BeforeProcess(true, Fn(t).NormalProc())
	wf.BeforeProcess(true, Fn(t).Fail(SetCtx()).Suc(CheckCtx("TestPreProcessCallbackFailRecover2", 0)).ErrProc())
	wf.BeforeProcess(true, Fn(t).NormalProc())
	flow.DoneFlow("TestPreProcessCallbackFailRecover2", nil)
	Recover("TestPreProcessCallbackFailRecover2")
}

func TestPreProcessCallbackPanicRecover2(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestPreProcessCallbackPanicRecover2")
	wf.EnableRecover()
	proc := wf.Process("TestPreProcessCallbackPanicRecover2")
	proc.NameStep(Fn(t).Do(CheckCtx("TestPreProcessCallbackPanicRecover2", 0), SetCtx()).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 2, flow.CallbackFail)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 6, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	wf.BeforeProcess(true, Fn(t).NormalProc())
	wf.BeforeProcess(true, Fn(t).Fail(SetCtx()).Suc(CheckCtx("TestPreProcessCallbackPanicRecover2", 0)).PanicProc())
	wf.BeforeProcess(true, Fn(t).NormalProc())
	flow.DoneFlow("TestPreProcessCallbackPanicRecover2", nil)
	Recover("TestPreProcessCallbackPanicRecover2")
}

func TestDefaultPreProcessCallbackFailRecover(t *testing.T) {
	defer resetCurrent()
	defer flow.ResetDefaultCallback()
	executeSuc = false
	wf := flow.RegisterFlow("TestDefaultPreProcessCallbackFailRecover")
	wf.EnableRecover()
	proc := wf.Process("TestDefaultPreProcessCallbackFailRecover")
	proc.NameStep(Fn(t).Do(CheckCtx("TestDefaultPreProcessCallbackFailRecover", 0), SetCtx()).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	df := flow.DefaultCallback()
	df.AfterFlow(false, CheckResult(t, 1, flow.CallbackFail)).If(execFail)
	df.AfterFlow(false, CheckResult(t, 5, flow.Success)).If(execSuc)
	df.AfterStep(false, ErrorResultPrinter).If(execSuc)
	df.BeforeProcess(true, Fn(t).Fail(SetCtx()).Suc(CheckCtx("TestDefaultPreProcessCallbackFailRecover", 0)).ErrProc())
	flow.DoneFlow("TestDefaultPreProcessCallbackFailRecover", nil)
	Recover("TestDefaultPreProcessCallbackFailRecover")
}

func TestAfterProcessCallbackFailRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestAfterProcessCallbackFailRecover")
	wf.EnableRecover()
	proc := wf.Process("TestAfterProcessCallbackFailRecover")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 5, flow.CallbackFail)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 1, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	wf.AfterProcess(true, Fn(t).Fail(SetCtx()).Suc(CheckCtx("TestAfterProcessCallbackFailRecover", 0)).ErrProc())
	flow.DoneFlow("TestAfterProcessCallbackFailRecover", nil)
	Recover("TestAfterProcessCallbackFailRecover")
}

func TestAfterProcessCallbackFailReplay(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestAfterProcessCallbackFailReplay")
	wf.EnableRecover()
	proc := wf.Process("TestAfterProcessCallbackFailReplay")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 6, flow.CallbackFail)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 2, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	wf.AfterProcess(true, Fn(t).NormalProc())
	wf.AfterProcess(true, Fn(t).Fail(SetCtx()).Suc(CheckCtx("TestAfterProcessCallbackFailReplay", 0)).ErrProc())
	flow.DoneFlow("TestAfterProcessCallbackFailReplay", nil)
	Recover("TestAfterProcessCallbackFailReplay")
}

func TestAfterProcessCallbackFailReplay0(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestAfterProcessCallbackFailReplay0")
	wf.EnableRecover()
	proc := wf.Process("TestAfterProcessCallbackFailReplay0")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 5, flow.CallbackFail)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 2, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	wf.AfterProcess(true, Fn(t).Fail(SetCtx()).Suc(CheckCtx("TestAfterProcessCallbackFailReplay0", 0)).ErrProc())
	wf.AfterProcess(true, Fn(t).NormalProc())
	flow.DoneFlow("TestAfterProcessCallbackFailReplay0", nil)
	Recover("TestAfterProcessCallbackFailReplay0")
}

func TestAfterProcessCallbackFailReplay1(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestAfterProcessCallbackFailReplay1")
	wf.EnableRecover()
	proc := wf.Process("TestAfterProcessCallbackFailReplay1")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 6, flow.CallbackFail)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 3, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	wf.AfterProcess(true, Fn(t).NormalProc())
	wf.AfterProcess(true, Fn(t).Fail(SetCtx()).Suc(CheckCtx("TestAfterProcessCallbackFailReplay1", 0)).ErrProc())
	wf.AfterProcess(true, Fn(t).NormalProc())
	flow.DoneFlow("TestAfterProcessCallbackFailReplay1", nil)
	Recover("TestAfterProcessCallbackFailReplay1")
}

func TestAfterProcessCallbackPanicReplay1(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestAfterProcessCallbackPanicReplay1")
	wf.EnableRecover()
	proc := wf.Process("TestAfterProcessCallbackPanicReplay1")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 6, flow.CallbackFail)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 3, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	wf.AfterProcess(true, Fn(t).NormalProc())
	wf.AfterProcess(true, Fn(t).Fail(SetCtx()).Suc(CheckCtx("TestAfterProcessCallbackPanicReplay1", 0)).PanicProc())
	wf.AfterProcess(true, Fn(t).NormalProc())
	flow.DoneFlow("TestAfterProcessCallbackPanicReplay1", nil)
	Recover("TestAfterProcessCallbackPanicReplay1")
}

func TestAfterFlowCallbackFailReplay(t *testing.T) {
	defer resetCurrent()
	defer flow.ResetDefaultCallback()
	executeSuc = false
	wf := flow.RegisterFlow("TestAfterFlowCallbackFailReplay")
	wf.EnableRecover()
	proc := wf.Process("TestAfterFlowCallbackFailReplay")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	df := flow.DefaultCallback()
	df.AfterFlow(true, Fn0(t).Normal())
	df.AfterFlow(true, Fn0(t).Normal())
	df.AfterFlow(true, Fn0(t).Fail().Suc().ErrFlow())
	wf.AfterFlow(false, CheckResult(t, 7, flow.CallbackFail)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 3, flow.Success)).If(execSuc)
	flow.DoneFlow("TestAfterFlowCallbackFailReplay", nil)
	Recover("TestAfterFlowCallbackFailReplay")
}

func TestAfterFlowCallbackPanicReplay(t *testing.T) {
	defer resetCurrent()
	defer flow.ResetDefaultCallback()
	executeSuc = false
	wf := flow.RegisterFlow("TestAfterFlowCallbackPanicReplay")
	wf.EnableRecover()
	proc := wf.Process("TestAfterFlowCallbackPanicReplay")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	df := flow.DefaultCallback()
	df.AfterFlow(true, Fn0(t).Normal())
	df.AfterFlow(true, Fn0(t).Normal())
	df.AfterFlow(true, Fn0(t).Fail().Suc().PanicFlow())
	wf.AfterFlow(false, CheckResult(t, 7, flow.CallbackFail)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 3, flow.Success)).If(execSuc)
	flow.DoneFlow("TestAfterFlowCallbackPanicReplay", nil)
	Recover("TestAfterFlowCallbackPanicReplay")
}

func TestProcessRecover1Suc1Fail(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestProcessRecover1Suc1Fail")
	wf.EnableRecover()
	proc1 := wf.Process("TestProcessRecover1Suc1Fail1")
	proc1.NameStep(Fn(t).Do(CheckCtx("TestProcessRecover1Suc1Fail1", 0)).Suc(SetCtx()).ErrStep(), "step1")
	proc1.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc1.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc1.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	proc1.BeforeProcess(false, Fn(t).Do(SetCtx()).Proc())

	proc2 := wf.Process("TestProcessRecover1Suc1Fail2")
	proc2.NameStep(Fn(t).Normal(), "step2-1")
	proc2.BeforeProcess(false, Fn(t).Do(SetCtx()).Proc())
	wf.AfterFlow(false, CheckResult(t, 4, flow.Error)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 5, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	flow.DoneFlow("TestProcessRecover1Suc1Fail", nil)
	Recover("TestProcessRecover1Suc1Fail")
}

func TestStepInterruptRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestStepInterruptRecover")
	wf.EnableRecover()
	proc := wf.Process("TestStepInterruptRecover")
	proc.NameStep(Fn(t).Fail(SetCtx()).Suc(CheckCtx("step1-1", 0)).ErrStep(), "step1-1")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step2-1", "step1-1")
	proc.NameStep(Fn(t).Normal(), "step1-2")
	proc.NameStep(Fn(t).Normal(), "step2-2", "step1-2")
	wf.AfterFlow(false, CheckResult(t, 3, flow.Error)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 2, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	flow.DoneFlow("TestStepInterruptRecover", nil)
	Recover("TestStepInterruptRecover")
}

func TestStepBeforeCallbackFailRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestStepBeforeCallbackFailRecover")
	wf.EnableRecover()
	proc := wf.Process("TestStepBeforeCallbackFailRecover")
	proc.BeforeStep(true, Fn(t).Fail(SetCtx()).Suc(CheckCtx("step1", 0)).ErrStepCall()).
		OnlyFor("step1")
	proc.NameStep(Fn(t).Do(CheckCtx("step1", 0)).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 1, flow.CallbackFail)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 5, flow.Success)).If(execSuc)
	flow.DoneFlow("TestStepBeforeCallbackFailRecover", nil)
	Recover("TestStepBeforeCallbackFailRecover")
}

func TestStepBeforeCallbackPanicRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestStepBeforeCallbackPanicRecover")
	wf.EnableRecover()
	proc := wf.Process("TestStepBeforeCallbackPanicRecover")
	proc.BeforeStep(true, Fn(t).Fail(SetCtx()).Suc(CheckCtx("step1", 0)).PanicStepCall()).
		OnlyFor("step1")
	proc.NameStep(Fn(t).Do(CheckCtx("step1", 0)).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 1, flow.CallbackFail)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 5, flow.Success)).If(execSuc)
	flow.DoneFlow("TestStepBeforeCallbackPanicRecover", nil)
	Recover("TestStepBeforeCallbackPanicRecover")
}

func TestStepBeforeCallbackFailRecover0(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestStepBeforeCallbackFailRecover0")
	wf.EnableRecover()
	proc := wf.Process("TestStepBeforeCallbackFailRecover0")
	proc.BeforeStep(true, Fn(t).StepCall("proc")).OnlyFor("step1")
	proc.BeforeStep(true, Fn(t).Fail(SetCtx()).Suc(CheckCtx("step1", 0)).ErrStepCall()).
		OnlyFor("step1")
	proc.BeforeStep(true, Fn(t).StepCall("proc")).OnlyFor("step1")
	proc.NameStep(Fn(t).Do(CheckCtx("step1", 0)).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 2, flow.CallbackFail)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 6, flow.Success)).If(execSuc)
	flow.DoneFlow("TestStepBeforeCallbackFailRecover0", nil)
	Recover("TestStepBeforeCallbackFailRecover0")
}

func TestStepBeforeCallbackPanicRecover0(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestStepBeforeCallbackPanicRecover0")
	wf.EnableRecover()
	proc := wf.Process("TestStepBeforeCallbackPanicRecover0")
	proc.BeforeStep(true, Fn(t).StepCall("proc")).OnlyFor("step1")
	proc.BeforeStep(true, Fn(t).Fail(SetCtx()).Suc(CheckCtx("step1", 0)).PanicStepCall()).
		OnlyFor("step1")
	proc.BeforeStep(true, Fn(t).StepCall("proc")).OnlyFor("step1")
	proc.NameStep(Fn(t).Do(CheckCtx("step1", 0)).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 2, flow.CallbackFail)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 6, flow.Success)).If(execSuc)
	flow.DoneFlow("TestStepBeforeCallbackPanicRecover0", nil)
	Recover("TestStepBeforeCallbackPanicRecover0")
}

func TestStepBeforeCallbackFailRecoverAnd2CommonStepCallback(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestStepBeforeCallbackFailRecoverAnd2CommonStepCallback")
	wf.EnableRecover()
	proc := wf.Process("TestStepBeforeCallbackFailRecoverAnd2CommonStepCallback")
	proc.BeforeStep(true, Fn(t).StepCall("proc"))
	proc.BeforeStep(true, Fn(t).Fail(SetCtx()).Suc(CheckCtx("step1", 0)).ErrStepCall()).
		OnlyFor("step1")
	proc.BeforeStep(true, Fn(t).StepCall("proc"))
	proc.NameStep(Fn(t).Do(SetCtx(), CheckCtx("step1", 0)).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 2, flow.CallbackFail)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 12, flow.Success)).If(execSuc)
	flow.DoneFlow("TestStepBeforeCallbackFailRecoverAnd2CommonStepCallback", nil)
	Recover("TestStepBeforeCallbackFailRecoverAnd2CommonStepCallback")
}

func TestStepAfterCallbackFailRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestStepAfterCallbackFailRecover")
	wf.EnableRecover()
	proc := wf.Process("TestStepAfterCallbackFailRecover")
	proc.AfterStep(true, Fn(t).Fail(CheckCtx("step1")).Suc(CheckCtx("step1")).ErrStepCall()).
		OnlyFor("step1")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 2, flow.CallbackFail)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 4, flow.Success)).If(execSuc)
	flow.DoneFlow("TestStepAfterCallbackFailRecover", nil)
	Recover("TestStepAfterCallbackFailRecover")
}

func TestStepAfterCallbackPanicRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestStepAfterCallbackPanicRecover")
	wf.EnableRecover()
	proc := wf.Process("TestStepAfterCallbackPanicRecover")
	proc.AfterStep(true, Fn(t).Fail(CheckCtx("step1")).Suc(CheckCtx("step1")).PanicStepCall()).
		OnlyFor("step1")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 2, flow.CallbackFail)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 4, flow.Success)).If(execSuc)
	flow.DoneFlow("TestStepAfterCallbackPanicRecover", nil)
	Recover("TestStepAfterCallbackPanicRecover")
}

func TestStepAfterCallbackFailRecover0(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestStepAfterCallbackFailRecover0")
	wf.EnableRecover()
	proc := wf.Process("TestStepAfterCallbackFailRecover0")
	proc.AfterStep(true, Fn(t).Fail(CheckCtx("step1")).Suc(CheckCtx("step1")).ErrStepCall()).
		OnlyFor("step1")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 2, flow.CallbackFail)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 4, flow.Success)).If(execSuc)
	flow.DoneFlow("TestStepAfterCallbackFailRecover0", nil)
	Recover("TestStepAfterCallbackFailRecover0")
}

func TestStepAfterCallbackPanicRecover0(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestStepAfterCallbackPanicRecover0")
	wf.EnableRecover()
	proc := wf.Process("TestStepAfterCallbackPanicRecover0")
	proc.AfterStep(true, Fn(t).Fail(CheckCtx("step1")).Suc(CheckCtx("step1")).PanicStepCall()).
		OnlyFor("step1")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 2, flow.CallbackFail)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 4, flow.Success)).If(execSuc)
	flow.DoneFlow("TestStepAfterCallbackPanicRecover0", nil)
	Recover("TestStepAfterCallbackPanicRecover0")
}

func TestStepCallbackWithAllScope0(t *testing.T) {
	defer resetCurrent()
	defer flow.ResetDefaultCallback()
	executeSuc = false
	df := flow.DefaultCallback()
	wf := flow.RegisterFlow("TestStepCallbackWithAllScope0")
	wf.EnableRecover()
	proc := wf.Process("TestStepCallbackWithAllScope0")
	df.BeforeStep(true, Fn(t).StepCall("default")).OnlyFor("step1")
	wf.BeforeStep(true, Fn(t).StepCall("flow")).OnlyFor("step1")
	proc.BeforeStep(true, Fn(t).StepCall("proc")).OnlyFor("step1")
	df.AfterStep(true, Fn(t).StepCall("default")).OnlyFor("step1")
	wf.AfterStep(true, Fn(t).StepCall("flow")).OnlyFor("step1")
	proc.AfterStep(true, Fn(t).Fail(CheckCtx("step1")).Suc(CheckCtx("step1")).ErrStepCall()).
		OnlyFor("step1")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 7, flow.CallbackFail)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 4, flow.Success)).If(execSuc)
	flow.DoneFlow("TestStepCallbackWithAllScope0", nil)
	Recover("TestStepCallbackWithAllScope0")
}

func TestStepCallbackPanicWithAllScope0(t *testing.T) {
	defer resetCurrent()
	defer flow.ResetDefaultCallback()
	executeSuc = false
	df := flow.DefaultCallback()
	wf := flow.RegisterFlow("TestStepCallbackPanicWithAllScope0")
	wf.EnableRecover()
	proc := wf.Process("TestStepCallbackPanicWithAllScope0")
	df.BeforeStep(true, Fn(t).StepCall("default")).OnlyFor("step1")
	wf.BeforeStep(true, Fn(t).StepCall("flow")).OnlyFor("step1")
	proc.BeforeStep(true, Fn(t).StepCall("proc")).OnlyFor("step1")
	df.AfterStep(true, Fn(t).StepCall("default")).OnlyFor("step1")
	wf.AfterStep(true, Fn(t).StepCall("flow")).OnlyFor("step1")
	proc.AfterStep(true, Fn(t).Fail(CheckCtx("step1")).Suc(CheckCtx("step1")).PanicStepCall()).
		OnlyFor("step1")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 7, flow.CallbackFail)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 4, flow.Success)).If(execSuc)
	flow.DoneFlow("TestStepCallbackPanicWithAllScope0", nil)
	Recover("TestStepCallbackPanicWithAllScope0")
}

func TestStepCallbackWithAllScope1(t *testing.T) {
	defer resetCurrent()
	defer flow.ResetDefaultCallback()
	executeSuc = false
	df := flow.DefaultCallback()
	wf := flow.RegisterFlow("TestStepCallbackWithAllScope1")
	wf.EnableRecover()
	proc := wf.Process("TestStepCallbackWithAllScope1")
	df.BeforeStep(true, Fn(t).StepCall("default")).OnlyFor("step1")
	wf.BeforeStep(true, Fn(t).StepCall("flow")).OnlyFor("step1")
	proc.BeforeStep(true, Fn(t).StepCall("proc")).OnlyFor("step1")
	df.AfterStep(true, Fn(t).StepCall("default")).OnlyFor("step1")
	wf.AfterStep(true, Fn(t).Fail(CheckCtx("step1")).Suc(CheckCtx("step1")).ErrStepCall()).
		OnlyFor("step1")
	proc.AfterStep(true, Fn(t).StepCall("flow")).OnlyFor("step1")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 6, flow.CallbackFail)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 5, flow.Success)).If(execSuc)
	flow.DoneFlow("TestStepCallbackWithAllScope1", nil)
	Recover("TestStepCallbackWithAllScope1")
}

func TestStepCallbackWithAllScope2(t *testing.T) {
	defer resetCurrent()
	defer flow.ResetDefaultCallback()
	executeSuc = false
	df := flow.DefaultCallback()
	wf := flow.RegisterFlow("TestStepCallbackWithAllScope2")
	wf.EnableRecover()
	proc := wf.Process("TestStepCallbackWithAllScope2")
	df.BeforeStep(true, Fn(t).StepCall("default")).OnlyFor("step1")
	wf.BeforeStep(true, Fn(t).StepCall("flow")).OnlyFor("step1")
	proc.BeforeStep(true, Fn(t).StepCall("proc")).OnlyFor("step1")
	df.AfterStep(true, Fn(t).Fail(CheckCtx("step1")).Suc(CheckCtx("step1")).ErrStepCall()).
		OnlyFor("step1")
	wf.AfterStep(true, Fn(t).StepCall("default")).OnlyFor("step1")
	proc.AfterStep(true, Fn(t).StepCall("flow")).OnlyFor("step1")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 5, flow.CallbackFail)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 6, flow.Success)).If(execSuc)
	flow.DoneFlow("TestStepCallbackWithAllScope2", nil)
	Recover("TestStepCallbackWithAllScope2")
}

func TestStepCallbackWithAllScope3(t *testing.T) {
	defer resetCurrent()
	defer flow.ResetDefaultCallback()
	executeSuc = false
	df := flow.DefaultCallback()
	wf := flow.RegisterFlow("TestStepCallbackWithAllScope3")
	wf.EnableRecover()
	proc := wf.Process("TestStepCallbackWithAllScope3")
	df.BeforeStep(true, Fn(t).StepCall("default")).OnlyFor("step1")
	wf.BeforeStep(true, Fn(t).StepCall("flow")).OnlyFor("step1")
	proc.BeforeStep(true, Fn(t).Fail(Normal()).Suc(Normal()).ErrStepCall()).
		OnlyFor("step1")
	df.AfterStep(true, Fn(t).StepCall("proc")).OnlyFor("step1")
	wf.AfterStep(true, Fn(t).StepCall("default")).OnlyFor("step1")
	proc.AfterStep(true, Fn(t).StepCall("flow")).OnlyFor("step1")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 3, flow.CallbackFail)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 8, flow.Success)).If(execSuc)
	flow.DoneFlow("TestStepCallbackWithAllScope3", nil)
	Recover("TestStepCallbackWithAllScope3")
}

func TestStepCallbackWithAllScope4(t *testing.T) {
	defer resetCurrent()
	defer flow.ResetDefaultCallback()
	executeSuc = false
	df := flow.DefaultCallback()
	wf := flow.RegisterFlow("TestStepCallbackWithAllScope4")
	wf.EnableRecover()
	proc := wf.Process("TestStepCallbackWithAllScope4")
	df.BeforeStep(true, Fn(t).StepCall("default")).OnlyFor("step1")
	wf.BeforeStep(true, Fn(t).Fail(Normal()).Suc(Normal()).ErrStepCall()).
		OnlyFor("step1")
	proc.BeforeStep(true, Fn(t).StepCall("flow")).OnlyFor("step1")
	df.AfterStep(true, Fn(t).StepCall("proc")).OnlyFor("step1")
	wf.AfterStep(true, Fn(t).StepCall("default")).OnlyFor("step1")
	proc.AfterStep(true, Fn(t).StepCall("flow")).OnlyFor("step1")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 2, flow.CallbackFail)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 9, flow.Success)).If(execSuc)
	flow.DoneFlow("TestStepCallbackWithAllScope4", nil)
	Recover("TestStepCallbackWithAllScope4")
}

func TestStepCallbackWithAllScope5(t *testing.T) {
	defer resetCurrent()
	defer flow.ResetDefaultCallback()
	executeSuc = false
	df := flow.DefaultCallback()
	wf := flow.RegisterFlow("TestStepCallbackWithAllScope5")
	wf.EnableRecover()
	proc := wf.Process("TestStepCallbackWithAllScope5")
	df.BeforeStep(true, Fn(t).Fail(Normal()).Suc(Normal()).ErrStepCall()).
		OnlyFor("step1")
	wf.BeforeStep(true, Fn(t).StepCall("default")).OnlyFor("step1")
	proc.BeforeStep(true, Fn(t).StepCall("flow")).OnlyFor("step1")
	df.AfterStep(true, Fn(t).StepCall("proc")).OnlyFor("step1")
	wf.AfterStep(true, Fn(t).StepCall("default")).OnlyFor("step1")
	proc.AfterStep(true, Fn(t).StepCall("flow")).OnlyFor("step1")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 1, flow.CallbackFail)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 10, flow.Success)).If(execSuc)
	flow.DoneFlow("TestStepCallbackWithAllScope5", nil)
	Recover("TestStepCallbackWithAllScope5")
}

func TestStepCallbackPanicWithAllScope5(t *testing.T) {
	defer resetCurrent()
	defer flow.ResetDefaultCallback()
	executeSuc = false
	df := flow.DefaultCallback()
	wf := flow.RegisterFlow("TestStepCallbackPanicWithAllScope5")
	wf.EnableRecover()
	proc := wf.Process("TestStepCallbackPanicWithAllScope5")
	df.BeforeStep(true, Fn(t).Fail(Normal()).Suc(Normal()).PanicStepCall()).
		OnlyFor("step1")
	wf.BeforeStep(true, Fn(t).StepCall("default")).OnlyFor("step1")
	proc.BeforeStep(true, Fn(t).StepCall("flow")).OnlyFor("step1")
	df.AfterStep(true, Fn(t).StepCall("proc")).OnlyFor("step1")
	wf.AfterStep(true, Fn(t).StepCall("default")).OnlyFor("step1")
	proc.AfterStep(true, Fn(t).StepCall("flow")).OnlyFor("step1")
	proc.NameStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.NameStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.NameStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 1, flow.CallbackFail)).If(execFail)
	wf.AfterFlow(false, CheckResult(t, 10, flow.Success)).If(execSuc)
	flow.DoneFlow("TestStepCallbackPanicWithAllScope5", nil)
	Recover("TestStepCallbackPanicWithAllScope5")
}

func TestMultiTimesRecover(t *testing.T) {
	defer resetCurrent()
	defer resetTimes()
	wf := flow.RegisterFlow("TestMultiTimesRecover")
	fx0 := Fx[flow.WorkFlow](t)
	fx1 := Fx[flow.Process](t)
	wf.BeforeFlow(true, fx0.Normal().Callback())
	wf.BeforeProcess(true, fx1.Normal().Callback())
	wf.EnableRecover()
	fx2 := Fx[flow.Step](t)
	fx3 := Fx[flow.Step](t)
	fx4 := Fx[flow.Step](t)
	fx5 := Fx[flow.Step](t)
	proc := wf.Process("TestMultiTimesRecover")
	proc.BeforeStep(true, fx2.Normal().Callback()).OnlyFor("step1")
	proc.AfterStep(true, fx4.Normal().Callback()).OnlyFor("step1")
	proc.NameStep(fx3.Normal().Step(), "step1")
	proc.NameStep(fx5.Normal().Step(), "step2", "step1")
	fx6 := Fx[flow.Process](t)
	fx7 := Fx[flow.WorkFlow](t)
	wf.AfterProcess(true, fx6.Normal().Callback())
	wf.AfterFlow(true, fx7.Normal().Callback())
	wf.AfterFlow(false, CheckResult(t, 1, flow.CallbackFail)).If(Times(0))
	wf.AfterFlow(false, CheckResult(t, 3, flow.CallbackFail)).If(Times(1))
	wf.AfterFlow(false, CheckResult(t, 3, flow.CallbackFail)).If(Times(2))
	wf.AfterFlow(false, CheckResult(t, 4, flow.Error)).If(Times(3))
	wf.AfterFlow(false, CheckResult(t, 3, flow.CallbackFail)).If(Times(4))
	wf.AfterFlow(false, CheckResult(t, 3, flow.Error)).If(Times(5))
	wf.AfterFlow(false, CheckResult(t, 2, flow.CallbackFail)).If(Times(6))
	// times 7 will be skipped, so can't use CheckResult to check it.
	wf.AfterFlow(false, CheckResult(t, 1, flow.Success)).If(Times(8))
	fx0.Condition(0, errorRet)
	ff := flow.DoneFlow("TestMultiTimesRecover", nil)

	fx0.Condition(0, normalRet)
	fx1.Condition(0, errorRet)
	Recover0(t, ff)

	fx1.Condition(0, normalRet)
	fx2.Condition(0, errorRet)
	Recover0(t, ff)

	fx2.Condition(0, normalRet)
	fx3.Condition(0, errorRet)
	Recover0(t, ff)

	fx3.Condition(0, normalRet)
	fx4.Condition(0, errorRet)
	Recover0(t, ff)

	fx4.Condition(0, normalRet)
	fx5.Condition(0, errorRet)
	Recover0(t, ff)

	fx5.Condition(0, normalRet)
	fx6.Condition(0, errorRet)
	Recover0(t, ff)

	fx6.Condition(0, normalRet)
	fx7.Condition(0, errorRet)
	ff = Recover0(t, ff)
	if !ff.Has(flow.CallbackFail) {
		t.Error("TestMultiTimesRecover failed")
	}
	if atomic.LoadInt64(&current) != 1 {
		t.Error("TestMultiTimesRecover failed")
	}

	fx7.Condition(0, normalRet)
	ff = Recover0(t, ff)
}

func TestPool(t *testing.T) {
	db := <-pool
	defer func() { pool <- db }()
	t.Logf("success")
}
