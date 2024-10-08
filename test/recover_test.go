package test

import (
	"bufio"
	"fmt"
	"github.com/Bilibotter/light-flow/flow"
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

func (p persisitImpl) ListCheckpoints(recoverId string) ([]flow.CheckPoint, error) {
	db := <-pool
	defer func() {
		pool <- db
	}()
	var checkpoints []Checkpoints
	result := db.Where("recover_id = ?", recoverId).Find(&checkpoints)
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
	t.Logf("start [%d] times recover >>>>>>>>>>>>>>", times)
	f, err := ff.Recover()
	if err != nil {
		if !strings.Contains(fmt.Sprintf("%v", err), "Flow[Name") {
			t.Errorf("error message should contain 'Flow[Name', but got %v", err)
		}
	}
	println()
	t.Logf("end [%d]times recover <<<<<<<<<<<<<<<", times)
	return f
}

func Recover(name string) flow.FinishedWorkFlow {
	println("\n\t=========Recovering=========\n")
	executeSuc = true
	flowId, err := getId(name)
	if err != nil {
		panic(err)
	}
	resetCurrent()
	if ff, err := flow.RecoverFlow(flowId); err != nil {
		panic(err)
	} else {
		return ff
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
	flow.SetEncryptor(flow.NewAES256Encryptor([]byte("light-flow"), "pwd", "password"))
	flow.SetMaxSerializeSize(10240)
	flow.SuspendPersist(&persisitImpl{})
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
	proc.CustomStep(Fn(t).Do(SetCtx()).Step(), "step1").
		Next(Fn(t).Do(SetCtx()).Step(), "step2").
		Next(Fn(t).Suc(CheckCtx("step2"), SetCtx()).ErrStep(), "step3").
		Next(Fn(t).Do(CheckCtx("step3")).Step(), "step4")
	wf.AfterFlow(false, CheckResult(t, 2, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	res := flow.DoneFlow("TestCombineStepRecover", nil)
	CheckResult(t, 2, flow.Error)(any(res).(flow.WorkFlow))
	Recover("TestCombineStepRecover")
}

func TestSeparateStepRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestSeparateStepRecover")
	wf.EnableRecover()
	proc := wf.Process("TestSeparateStepRecover")
	proc.CustomStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.CustomStep(Fn(t).Do(SetCtx0("+")).Step(), "step2")
	proc.CustomStep(Fn(t).
		Suc(CheckCtx("step1"), CheckCtx0("step2", "+"), SetCtx(), SetCtx0("+")).ErrStep(),
		"step3", "step1", "step2")
	proc.CustomStep(Fn(t).Suc(CheckCtx("step3"), CheckCtx0("step3", "+")).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 2, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	res := flow.DoneFlow("TestSeparateStepRecover", nil)
	CheckResult(t, 2, flow.Error)(any(res).(flow.WorkFlow))
	Recover("TestSeparateStepRecover")
}

func Test1ProcSuc1ProcRecover(t *testing.T) {
	defer resetCurrent()
	flow.DefaultConfig().DisableRecover()
	executeSuc = false
	wf := flow.RegisterFlow("Test1ProcSuc1ProcRecover")
	wf.EnableRecover()
	proc := wf.Process("Test1ProcSuc1ProcRecover0")
	proc.CustomStep(GenerateStep(1), "step1").
		Next(GenerateStep(2), "step2").
		Next(GenerateStep(3), "step3")

	proc = wf.Process("Test1ProcSuc1ProcRecover")
	proc.CustomStep(Fn(t).Do(SetCtx()).Step(), "step1").
		Next(Fn(t).Suc(CheckCtx("step1"), SetCtx()).ErrStep(), "step2").
		Next(Fn(t).Fail(SetCtx()).Suc(CheckCtx("step2")).Step(), "step3")
	wf.AfterFlow(false, CheckResult(t, 2, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	res := flow.DoneFlow("Test1ProcSuc1ProcRecover", nil)
	CheckResult(t, 4, flow.Error)(any(res).(flow.WorkFlow))
	Recover("Test1ProcSuc1ProcRecover")
}

func TestSingleStepRecoverWith2Branch(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestSingleStepRecoverWith2Branch")
	wf.EnableRecover()
	proc := wf.Process("TestSingleStepRecoverWith2Branch")
	proc.CustomStep(Fn(t).Do(SetCtx()).Step(), "step2w1").
		Next(Fn(t).Do(SetCtx()).Step(), "step2w2")
	proc.CustomStep(Fn(t).Do(SetCtx()).Step(), "step1").
		Next(Fn(t).Suc(CheckCtx("step1"), SetCtx()).ErrStep(), "step2").
		Next(Fn(t).Fail(SetCtx()).Suc(CheckCtx("step2")).Step(), "step3")
	wf.AfterFlow(false, CheckResult(t, 2, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	res := flow.DoneFlow("TestSingleStepRecoverWith2Branch", nil)
	CheckResult(t, 3, flow.Error)(any(res).(flow.WorkFlow))
	Recover("TestSingleStepRecoverWith2Branch")
}

func TestRecoverSuccessFlow(t *testing.T) {
	defer resetCurrent()
	id := ""
	wf := flow.RegisterFlow("TestRecoverSuccessFlow")
	wf.EnableRecover()
	proc := wf.Process("TestRecoverSuccessFlow")
	proc.CustomStep(Ck(t).SetFn(), "Step1").
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
			if strings.Contains(fmt.Sprintf("%v", r), "no latest record found for") {
				t.Logf("recover error: %v", r)
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

func TestRecoverSuccessFlow0(t *testing.T) {
	defer resetCurrent()
	wf := flow.RegisterFlow("TestRecoverSuccessFlow0")
	wf.EnableRecover()
	proc := wf.Process("TestRecoverSuccessFlow0")
	proc.CustomStep(Ck(t).SetFn(), "Step1").
		Next(Ck(t).SetFn(), "Step2").
		Next(Ck(t).SetFn(), "Step3")
	wf.AfterFlow(false, func(workFlow flow.WorkFlow) (keepOn bool, err error) {
		return true, nil
	})
	wf.AfterFlow(false, CheckResult(t, 3, flow.Success))
	res := flow.DoneFlow("TestRecoverSuccessFlow0", nil)
	_, err := res.Recover()
	if err == nil {
		t.Errorf("recover success, but should fail")
	} else if !strings.Contains(err.Error(), "isn't failed, can't recover") {
		t.Errorf("error msg not match expect, err: %v", err)
	}
}

func TestSingleStepRecoverWith1DependSuc1DependFail0(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestSingleStepRecover1DependSuc1DependFail0")
	wf.EnableRecover()
	proc := wf.Process("TestSingleStepRecover1DependSuc1DependFail0")
	proc.CustomStep(Fn(t).Do(SetCtx0("+")).Step(), "step1w1")
	proc.CustomStep(Fn(t).Do(SetCtx()).Step(), "step1w2")

	proc.CustomStep(Fn(t).
		Do(CheckCtx0("step1w1", "+"), SetCtx0("+")).Step(),
		"step2w1", "step1w1")

	proc.CustomStep(Fn(t).
		Suc(CheckCtx("step1w2"), SetCtx()).ErrStep(),
		"step2w2", "step1w2")

	proc.CustomStep(Fn(t).Fail(SetCtx()).Suc(CheckCtx0("step2w1", "+"), CheckCtx("step2w2")).Step(),
		"step3w1", "step2w1", "step2w2")

	proc.CustomStep(Fn(t).Suc(CheckCtx("step2w2")).ErrStep(),
		"step3w2", "step2w2")

	wf.AfterFlow(false, CheckResult(t, 3, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	res := flow.DoneFlow("TestSingleStepRecover1DependSuc1DependFail0", nil)
	CheckResult(t, 3, flow.Error)(any(res).(flow.WorkFlow))
	Recover("TestSingleStepRecover1DependSuc1DependFail0")
}

func TestSingleStepRecoverWith1DependSuc1DependFail(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestSingleStepRecover1DependSuc1DependFail")
	wf.EnableRecover()
	proc := wf.Process("TestSingleStepRecover1DependSuc1DependFail")
	proc.CustomStep(Fn(t).Do(SetCtx()).Step(), "step1w1")

	proc.CustomStep(Fn(t).
		Do(CheckCtx("step1w1"), SetCtx0("+")).Step(),
		"step2w1", "step1w1")

	proc.CustomStep(Fn(t).
		Suc(CheckCtx("step1w1"), SetCtx()).ErrStep(),
		"step2w2", "step1w1")

	proc.CustomStep(Fn(t).Fail(SetCtx()).Suc(CheckCtx0("step2w1", "+"), CheckCtx("step2w2")).Step(),
		"step3w1", "step2w1", "step2w2")

	wf.AfterFlow(false, CheckResult(t, 2, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	res := flow.DoneFlow("TestSingleStepRecover1DependSuc1DependFail", nil)
	CheckResult(t, 2, flow.Error)(any(res).(flow.WorkFlow))
	Recover("TestSingleStepRecover1DependSuc1DependFail")
}

func TestSingleStepRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestSingleStepRecover")
	wf.EnableRecover()
	proc := wf.Process("TestSingleStepRecover")
	proc.CustomStep(Fn(t).Do(SetCtx()).Step(), "step1").
		Next(Fn(t).Suc(CheckCtx("step1"), SetCtx()).ErrStep(), "step2").
		Next(Fn(t).Fail(SetCtx()).Suc(CheckCtx("step2")).Step(), "step3")
	wf.AfterFlow(false, CheckResult(t, 2, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	ret := flow.DoneFlow("TestSingleStepRecover", nil)
	CheckResult(t, 1, flow.Error)(any(ret).(flow.WorkFlow))
	println("\n\t=========Recovering=========\n")
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
	proc.CustomStep(Fn(t).Fail(SetCtx(), f).Suc(CheckCtx("step1", 0)).Step(), "step1").
		Next(Fn(t).Fail(SetCtx()).Suc(CheckCtx("step1"), SetCtx()).Step(), "step2").
		Next(Fn(t).Fail(SetCtx()).Suc(CheckCtx("step2")).Step(), "step3")
	wf.AfterFlow(false, CheckResult(t, 3, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	res := flow.DoneFlow("TestPanicStepRecover", nil)
	CheckResult(t, 0, flow.Panic)(any(res).(flow.WorkFlow))
	Recover("TestPanicStepRecover")
}

func TestMultipleStepRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestMultipleStepRecover")
	wf.EnableRecover()
	proc := wf.Process("TestMultipleStepRecover")
	proc.CustomStep(Fn(t).Do(SetCtx()).Step(), "step1w1").
		Next(Fn(t).Suc(CheckCtx("step1w1"), SetCtx()).ErrStep(), "step1w2").
		Next(Fn(t).Do(CheckCtx("step1w2")).Step(), "1w3")
	proc.CustomStep(Fn(t).Do(SetCtx()).Step(), "step2w1").
		Next(Fn(t).Suc(CheckCtx("step2w1"), SetCtx()).ErrStep(), "step2w2").
		Next(Fn(t).Do(CheckCtx("step2w2")).Step(), "2w3")
	wf.AfterFlow(false, CheckResult(t, 4, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	res := flow.DoneFlow("TestMultipleStepRecover", nil)
	CheckResult(t, 2, flow.Error)(any(res).(flow.WorkFlow))
	Recover("TestMultipleStepRecover")
}

func TestMultipleStepRecover0(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestMultipleStepRecover0")
	wf.EnableRecover()
	proc := wf.Process("TestMultipleStepRecover0")
	proc.CustomStep(Fn(t).Fail(SetCtx()).Suc(CheckCtx("step1w1", 0)).ErrStep(), "step1w1").
		Next(Fn(t).Do(CheckCtx("step1w1")).Step(), "1w2")
	proc.CustomStep(Fn(t).Suc(SetCtx()).ErrStep(), "step2w1").
		Next(Fn(t).Do(CheckCtx("step2w1")).Step(), "2w2")
	wf.AfterFlow(false, CheckResult(t, 4, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	res := flow.DoneFlow("TestMultipleStepRecover0", nil)
	CheckResult(t, 1, flow.Error)(any(res).(flow.WorkFlow))
	Recover("TestMultipleStepRecover0")
}

func TestParallelStepRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestParallelStepRecover")
	wf.EnableRecover()
	proc1 := wf.Process("TestParallelStepRecover1")
	proc1.CustomStep(Fn(t).Do(SetCtx()).Step(), "step1w1").
		Next(Fn(t).Suc(CheckCtx("step1w1"), SetCtx()).ErrStep(), "step1w2").
		Next(Fn(t).Do(CheckCtx("step1w2")).Step(), "step1w3")
	proc2 := wf.Process("TestParallelStepRecover2")
	proc2.CustomStep(Fn(t).Do(SetCtx()).Step(), "step2w1").
		Next(Fn(t).Suc(CheckCtx("step2w1"), SetCtx()).ErrStep(), "step2w2").
		Next(Fn(t).Do(CheckCtx("step2w2")).Step(), "step2w3")
	proc3 := wf.Process("TestParallelStepRecover3")
	proc3.CustomStep(Fn(t).Do(SetCtx()).Step(), "step3w1").
		Next(Fn(t).Suc(CheckCtx("step3w1"), SetCtx()).ErrStep(), "step3w2").
		Next(Fn(t).Do(CheckCtx("step3w2")).Step(), "step3w3")
	wf.AfterFlow(false, CheckResult(t, 6, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	res := flow.DoneFlow("TestParallelStepRecover", nil)
	CheckResult(t, 3, flow.Error)(any(res).(flow.WorkFlow))
	Recover("TestParallelStepRecover")
}

func TestFlowInputRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestFlowInputRecover")
	wf.EnableRecover()
	proc := wf.Process("TestFlowInputRecover")
	proc.CustomStep(Fn(t).Do(CheckCtx("TestFlowInputRecover", 0)).Suc(SetCtx()).ErrStep(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
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
	res := flow.DoneFlow("TestFlowInputRecover", m.table)
	CheckResult(t, 1, flow.Error)(any(res).(flow.WorkFlow))
	Recover("TestFlowInputRecover")
}

func TestFlowCallbackSkipWhileRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestFlowCallbackSkipWhileRecover")
	wf.EnableRecover()
	proc := wf.Process("TestFlowCallbackSkipWhileRecover")
	proc.CustomStep(Fn(t).Do(CheckCtx("TestFlowCallbackSkipWhileRecover", 0)).Suc(SetCtx()).ErrStep(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	wf.BeforeFlow(false, Fn0(t).Normal())
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
	res := flow.DoneFlow("TestFlowCallbackSkipWhileRecover", m.table)
	CheckResult(t, 2, flow.Error)(any(res).(flow.WorkFlow))
	Recover("TestFlowCallbackSkipWhileRecover")
}

func TestPreFlowCallbackFailRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestPreFlowCallbackFailRecover")
	wf.EnableRecover()
	proc := wf.Process("TestPreFlowCallbackFailRecover")
	proc.CustomStep(Fn(t).Do(CheckCtx("TestFlowCallbackSkipWhileRecover", 0), SetCtx()).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 5, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	wf.BeforeFlow(true, func(workFlow flow.WorkFlow) (keepOn bool, err error) {
		t.Logf("start [Flow: %s] pre-callback", workFlow.Name())
		atomic.AddInt64(&current, 1)
		if executeSuc {
			t.Logf("finish [Flow: %s] pre-callback", workFlow.Name())
			return true, nil
		}
		t.Logf("excute [Flow: %s] pre-callback failed", workFlow.Name())
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
	proc.CustomStep(Fn(t).Do(CheckCtx("TestFlowCallbackSkipWhileRecover", 0), SetCtx()).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 5, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	wf.BeforeFlow(true, func(workFlow flow.WorkFlow) (keepOn bool, err error) {
		t.Logf("start [Flow: %s] pre-callback", workFlow.Name())
		atomic.AddInt64(&current, 1)
		if executeSuc {
			t.Logf("finish [Flow: %s] pre-callback", workFlow.Name())
			return true, nil
		}
		t.Logf("excute [Flow: %s] pre-callback panic", workFlow.Name())
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
	proc.CustomStep(Fn(t).Do(CheckCtx("TestProcessRecover", 0)).Suc(SetCtx()).ErrStep(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 5, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	wf.BeforeProcess(false, Fn(t).Do(SetCtx()).Proc())
	res := flow.DoneFlow("TestProcessRecover", nil)
	CheckResult(t, 2, flow.Error)(any(res).(flow.WorkFlow))
	Recover("TestProcessRecover")
}

func TestPreProcessCallbackFailRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestPreProcessCallbackFailRecover")
	wf.EnableRecover()
	proc := wf.Process("TestPreProcessCallbackFailRecover")
	proc.CustomStep(Fn(t).Do(CheckCtx("TestPreProcessCallbackFailRecover", 0), SetCtx()).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 5, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	wf.BeforeProcess(true, Fn(t).Fail(SetCtx()).Suc(CheckCtx("TestPreProcessCallbackFailRecover", 0)).ErrProc())
	res := flow.DoneFlow("TestPreProcessCallbackFailRecover", nil)
	CheckResult(t, 1, flow.CallbackFail)(any(res).(flow.WorkFlow))
	Recover("TestPreProcessCallbackFailRecover")
}

func TestPreProcessCallbackFailRecover0(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestPreProcessCallbackFailRecover0")
	wf.EnableRecover()
	proc := wf.Process("TestPreProcessCallbackFailRecover0")
	proc.CustomStep(Fn(t).Do(CheckCtx("TestPreProcessCallbackFailRecover0", 0), SetCtx()).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 5, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	wf.BeforeProcess(true, Fn(t).NormalProc())
	wf.BeforeProcess(true, Fn(t).Fail(SetCtx()).Suc(CheckCtx("TestPreProcessCallbackFailRecover0", 0)).ErrProc())
	res := flow.DoneFlow("TestPreProcessCallbackFailRecover0", nil)
	CheckResult(t, 2, flow.CallbackFail)(any(res).(flow.WorkFlow))
	Recover("TestPreProcessCallbackFailRecover0")
}

func TestPreProcessCallbackFailRecover1(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestPreProcessCallbackFailRecover1")
	wf.EnableRecover()
	proc := wf.Process("TestPreProcessCallbackFailRecover1")
	proc.CustomStep(Fn(t).Do(CheckCtx("TestPreProcessCallbackFailRecover1", 0), SetCtx()).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 6, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	wf.BeforeProcess(true, Fn(t).Fail(SetCtx()).Suc(CheckCtx("TestPreProcessCallbackFailRecover1", 0)).ErrProc())
	wf.BeforeProcess(true, Fn(t).NormalProc())
	res := flow.DoneFlow("TestPreProcessCallbackFailRecover1", nil)
	CheckResult(t, 1, flow.CallbackFail)(any(res).(flow.WorkFlow))
	Recover("TestPreProcessCallbackFailRecover1")
}

func TestPreProcessCallbackFailRecover2(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestPreProcessCallbackFailRecover2")
	wf.EnableRecover()
	proc := wf.Process("TestPreProcessCallbackFailRecover2")
	proc.CustomStep(Fn(t).Do(CheckCtx("TestPreProcessCallbackFailRecover2", 0), SetCtx()).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 6, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	wf.BeforeProcess(true, Fn(t).NormalProc())
	wf.BeforeProcess(true, Fn(t).Fail(SetCtx()).Suc(CheckCtx("TestPreProcessCallbackFailRecover2", 0)).ErrProc())
	wf.BeforeProcess(true, Fn(t).NormalProc())
	res := flow.DoneFlow("TestPreProcessCallbackFailRecover2", nil)
	CheckResult(t, 2, flow.CallbackFail)(any(res).(flow.WorkFlow))
	Recover("TestPreProcessCallbackFailRecover2")
}

func TestPreProcessCallbackPanicRecover2(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestPreProcessCallbackPanicRecover2")
	wf.EnableRecover()
	proc := wf.Process("TestPreProcessCallbackPanicRecover2")
	proc.CustomStep(Fn(t).Do(CheckCtx("TestPreProcessCallbackPanicRecover2", 0), SetCtx()).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 6, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	wf.BeforeProcess(true, Fn(t).NormalProc())
	wf.BeforeProcess(true, Fn(t).Fail(SetCtx()).Suc(CheckCtx("TestPreProcessCallbackPanicRecover2", 0)).PanicProc())
	wf.BeforeProcess(true, Fn(t).NormalProc())
	res := flow.DoneFlow("TestPreProcessCallbackPanicRecover2", nil)
	CheckResult(t, 2, flow.CallbackFail)(any(res).(flow.WorkFlow))
	Recover("TestPreProcessCallbackPanicRecover2")
}

func TestDefaultPreProcessCallbackFailRecover(t *testing.T) {
	defer resetCurrent()
	defer flow.ResetDefaultCallback()
	executeSuc = false
	wf := flow.RegisterFlow("TestDefaultPreProcessCallbackFailRecover")
	wf.EnableRecover()
	proc := wf.Process("TestDefaultPreProcessCallbackFailRecover")
	proc.CustomStep(Fn(t).Do(CheckCtx("TestDefaultPreProcessCallbackFailRecover", 0), SetCtx()).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	df := flow.DefaultCallback()
	df.AfterFlow(false, CheckResult(t, 5, flow.Success)).If(execSuc)
	df.AfterStep(false, ErrorResultPrinter).If(execSuc)
	df.BeforeProcess(true, Fn(t).Fail(SetCtx()).Suc(CheckCtx("TestDefaultPreProcessCallbackFailRecover", 0)).ErrProc())
	res := flow.DoneFlow("TestDefaultPreProcessCallbackFailRecover", nil)
	CheckResult(t, 1, flow.CallbackFail)(any(res).(flow.WorkFlow))
	Recover("TestDefaultPreProcessCallbackFailRecover")
}

func TestAfterProcessCallbackFailRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestAfterProcessCallbackFailRecover")
	wf.EnableRecover()
	proc := wf.Process("TestAfterProcessCallbackFailRecover")
	proc.CustomStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 1, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	wf.AfterProcess(true, Fn(t).Fail(SetCtx()).Suc(CheckCtx("TestAfterProcessCallbackFailRecover", 0)).ErrProc())
	res := flow.DoneFlow("TestAfterProcessCallbackFailRecover", nil)
	CheckResult(t, 5, flow.CallbackFail)(any(res).(flow.WorkFlow))
	Recover("TestAfterProcessCallbackFailRecover")
}

func TestAfterProcessCallbackFailReplay(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestAfterProcessCallbackFailReplay")
	wf.EnableRecover()
	proc := wf.Process("TestAfterProcessCallbackFailReplay")
	proc.CustomStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 2, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	wf.AfterProcess(true, Fn(t).NormalProc())
	wf.AfterProcess(true, Fn(t).Fail(SetCtx()).Suc(CheckCtx("TestAfterProcessCallbackFailReplay", 0)).ErrProc())
	res := flow.DoneFlow("TestAfterProcessCallbackFailReplay", nil)
	CheckResult(t, 6, flow.CallbackFail)(any(res).(flow.WorkFlow))
	Recover("TestAfterProcessCallbackFailReplay")
}

func TestAfterProcessCallbackFailReplay0(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestAfterProcessCallbackFailReplay0")
	wf.EnableRecover()
	proc := wf.Process("TestAfterProcessCallbackFailReplay0")
	proc.CustomStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 2, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	wf.AfterProcess(true, Fn(t).Fail(SetCtx()).Suc(CheckCtx("TestAfterProcessCallbackFailReplay0", 0)).ErrProc())
	wf.AfterProcess(true, Fn(t).NormalProc())
	res := flow.DoneFlow("TestAfterProcessCallbackFailReplay0", nil)
	CheckResult(t, 5, flow.CallbackFail)(any(res).(flow.WorkFlow))
	Recover("TestAfterProcessCallbackFailReplay0")
}

func TestAfterProcessCallbackFailReplay1(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestAfterProcessCallbackFailReplay1")
	wf.EnableRecover()
	proc := wf.Process("TestAfterProcessCallbackFailReplay1")
	proc.CustomStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 3, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	wf.AfterProcess(true, Fn(t).NormalProc())
	wf.AfterProcess(true, Fn(t).Fail(SetCtx()).Suc(CheckCtx("TestAfterProcessCallbackFailReplay1", 0)).ErrProc())
	wf.AfterProcess(true, Fn(t).NormalProc())
	res := flow.DoneFlow("TestAfterProcessCallbackFailReplay1", nil)
	CheckResult(t, 6, flow.CallbackFail)(any(res).(flow.WorkFlow))
	Recover("TestAfterProcessCallbackFailReplay1")
}

func TestAfterProcessCallbackPanicReplay1(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestAfterProcessCallbackPanicReplay1")
	wf.EnableRecover()
	proc := wf.Process("TestAfterProcessCallbackPanicReplay1")
	proc.CustomStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 3, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	wf.AfterProcess(true, Fn(t).NormalProc())
	wf.AfterProcess(true, Fn(t).Fail(SetCtx()).Suc(CheckCtx("TestAfterProcessCallbackPanicReplay1", 0)).PanicProc())
	wf.AfterProcess(true, Fn(t).NormalProc())
	res := flow.DoneFlow("TestAfterProcessCallbackPanicReplay1", nil)
	CheckResult(t, 6, flow.CallbackFail)(any(res).(flow.WorkFlow))
	Recover("TestAfterProcessCallbackPanicReplay1")
}

func TestAfterFlowCallbackFailReplay(t *testing.T) {
	defer resetCurrent()
	defer flow.ResetDefaultCallback()
	executeSuc = false
	wf := flow.RegisterFlow("TestAfterFlowCallbackFailReplay")
	wf.EnableRecover()
	proc := wf.Process("TestAfterFlowCallbackFailReplay")
	proc.CustomStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	df := flow.DefaultCallback()
	df.AfterFlow(true, Fn0(t).Normal())
	df.AfterFlow(true, Fn0(t).Normal())
	df.AfterFlow(true, Fn0(t).Fail().Suc().ErrFlow())
	wf.AfterFlow(false, CheckResult(t, 4, flow.Success)).If(execSuc)
	res := flow.DoneFlow("TestAfterFlowCallbackFailReplay", nil)
	CheckResult(t, 8, flow.CallbackFail)(any(res).(flow.WorkFlow))
	Recover("TestAfterFlowCallbackFailReplay")
}

func TestAfterFlowCallbackPanicReplay(t *testing.T) {
	defer resetCurrent()
	defer flow.ResetDefaultCallback()
	executeSuc = false
	wf := flow.RegisterFlow("TestAfterFlowCallbackPanicReplay")
	wf.EnableRecover()
	proc := wf.Process("TestAfterFlowCallbackPanicReplay")
	proc.CustomStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	df := flow.DefaultCallback()
	df.AfterFlow(true, Fn0(t).Normal())
	df.AfterFlow(true, Fn0(t).Normal())
	df.AfterFlow(true, Fn0(t).Fail().Suc().PanicFlow())
	wf.AfterFlow(false, CheckResult(t, 4, flow.Success)).If(execSuc)
	res := flow.DoneFlow("TestAfterFlowCallbackPanicReplay", nil)
	CheckResult(t, 8, flow.CallbackFail)(any(res).(flow.WorkFlow))
	Recover("TestAfterFlowCallbackPanicReplay")
}

func TestProcessRecover1Suc1Fail(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestProcessRecover1Suc1Fail")
	wf.EnableRecover()
	proc1 := wf.Process("TestProcessRecover1Suc1Fail1")
	proc1.CustomStep(Fn(t).Do(CheckCtx("TestProcessRecover1Suc1Fail1", 0)).Suc(SetCtx()).ErrStep(), "step1")
	proc1.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc1.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc1.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	proc1.BeforeProcess(false, Fn(t).Do(SetCtx()).Proc())

	proc2 := wf.Process("TestProcessRecover1Suc1Fail2")
	proc2.CustomStep(Fn(t).Normal(), "step2w1")
	proc2.BeforeProcess(false, Fn(t).Do(SetCtx()).Proc())
	wf.AfterFlow(false, CheckResult(t, 5, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	res := flow.DoneFlow("TestProcessRecover1Suc1Fail", nil)
	CheckResult(t, 4, flow.Error)(any(res).(flow.WorkFlow))
	Recover("TestProcessRecover1Suc1Fail")
}

func TestStepInterruptRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestStepInterruptRecover")
	wf.EnableRecover()
	proc := wf.Process("TestStepInterruptRecover")
	proc.CustomStep(Fn(t).Fail(SetCtx()).Suc(CheckCtx("step1w1", 0)).ErrStep(), "step1w1")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step2w1", "step1w1")
	proc.CustomStep(Fn(t).Normal(), "step1w2")
	proc.CustomStep(Fn(t).Normal(), "step2w2", "step1w2")
	wf.AfterFlow(false, CheckResult(t, 2, flow.Success)).If(execSuc)
	wf.AfterStep(false, ErrorResultPrinter).If(execSuc)
	res := flow.DoneFlow("TestStepInterruptRecover", nil)
	CheckResult(t, 3, flow.Error)(any(res).(flow.WorkFlow))
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
	proc.CustomStep(Fn(t).Do(CheckCtx("step1", 0)).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 5, flow.Success)).If(execSuc)
	res := flow.DoneFlow("TestStepBeforeCallbackFailRecover", nil)
	CheckResult(t, 1, flow.CallbackFail)(any(res).(flow.WorkFlow))
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
	proc.CustomStep(Fn(t).Do(CheckCtx("step1", 0)).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 5, flow.Success)).If(execSuc)
	res := flow.DoneFlow("TestStepBeforeCallbackPanicRecover", nil)
	CheckResult(t, 1, flow.CallbackFail)(any(res).(flow.WorkFlow))
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
	proc.CustomStep(Fn(t).Do(CheckCtx("step1", 0)).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 6, flow.Success)).If(execSuc)
	res := flow.DoneFlow("TestStepBeforeCallbackFailRecover0", nil)
	CheckResult(t, 2, flow.CallbackFail)(any(res).(flow.WorkFlow))
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
	proc.CustomStep(Fn(t).Do(CheckCtx("step1", 0)).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 6, flow.Success)).If(execSuc)
	res := flow.DoneFlow("TestStepBeforeCallbackPanicRecover0", nil)
	CheckResult(t, 2, flow.CallbackFail)(any(res).(flow.WorkFlow))
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
	proc.CustomStep(Fn(t).Do(SetCtx(), CheckCtx("step1", 0)).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 12, flow.Success)).If(execSuc)
	res := flow.DoneFlow("TestStepBeforeCallbackFailRecoverAnd2CommonStepCallback", nil)
	CheckResult(t, 2, flow.CallbackFail)(any(res).(flow.WorkFlow))
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
	proc.CustomStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 4, flow.Success)).If(execSuc)
	res := flow.DoneFlow("TestStepAfterCallbackFailRecover", nil)
	CheckResult(t, 2, flow.CallbackFail)(any(res).(flow.WorkFlow))
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
	proc.CustomStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 4, flow.Success)).If(execSuc)
	res := flow.DoneFlow("TestStepAfterCallbackPanicRecover", nil)
	CheckResult(t, 2, flow.CallbackFail)(any(res).(flow.WorkFlow))
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
	proc.CustomStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 4, flow.Success)).If(execSuc)
	res := flow.DoneFlow("TestStepAfterCallbackFailRecover0", nil)
	CheckResult(t, 2, flow.CallbackFail)(any(res).(flow.WorkFlow))
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
	proc.CustomStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 4, flow.Success)).If(execSuc)
	res := flow.DoneFlow("TestStepAfterCallbackPanicRecover0", nil)
	CheckResult(t, 2, flow.CallbackFail)(any(res).(flow.WorkFlow))
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
	proc.CustomStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 4, flow.Success)).If(execSuc)
	res := flow.DoneFlow("TestStepCallbackWithAllScope0", nil)
	CheckResult(t, 7, flow.CallbackFail)(any(res).(flow.WorkFlow))
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
	proc.CustomStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 4, flow.Success)).If(execSuc)
	res := flow.DoneFlow("TestStepCallbackPanicWithAllScope0", nil)
	CheckResult(t, 7, flow.CallbackFail)(any(res).(flow.WorkFlow))
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
	proc.CustomStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 5, flow.Success)).If(execSuc)
	res := flow.DoneFlow("TestStepCallbackWithAllScope1", nil)
	CheckResult(t, 6, flow.CallbackFail)(any(res).(flow.WorkFlow))
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
	proc.CustomStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 6, flow.Success)).If(execSuc)
	res := flow.DoneFlow("TestStepCallbackWithAllScope2", nil)
	CheckResult(t, 5, flow.CallbackFail)(any(res).(flow.WorkFlow))
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
	proc.CustomStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 8, flow.Success)).If(execSuc)
	res := flow.DoneFlow("TestStepCallbackWithAllScope3", nil)
	CheckResult(t, 3, flow.CallbackFail)(any(res).(flow.WorkFlow))
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
	proc.CustomStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 9, flow.Success)).If(execSuc)
	res := flow.DoneFlow("TestStepCallbackWithAllScope4", nil)
	CheckResult(t, 2, flow.CallbackFail)(any(res).(flow.WorkFlow))
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
	proc.CustomStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 10, flow.Success)).If(execSuc)
	res := flow.DoneFlow("TestStepCallbackWithAllScope5", nil)
	CheckResult(t, 1, flow.CallbackFail)(any(res).(flow.WorkFlow))
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
	proc.CustomStep(Fn(t).Do(SetCtx()).Step(), "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step2", "step1")
	proc.CustomStep(Fn(t).Do(CheckCtx1(), SetCtx()).Step(), "step3", "step2")
	proc.CustomStep(Fn(t).Do(CheckCtx1()).Step(), "step4", "step3")
	wf.AfterFlow(false, CheckResult(t, 10, flow.Success)).If(execSuc)
	res := flow.DoneFlow("TestStepCallbackPanicWithAllScope5", nil)
	CheckResult(t, 1, flow.CallbackFail)(any(res).(flow.WorkFlow))
	Recover("TestStepCallbackPanicWithAllScope5")
}

// disable log before running this test case will make it clearer
func TestMultiTimesRecover0(t *testing.T) {
	defer resetCurrent()
	defer resetTimes()
	defer flow.ResetDefaultCallback()
	atomic.StoreInt64(&recoverFlag, 1)
	df := flow.DefaultCallback()
	wf := flow.RegisterFlow("TestMultiTimesRecover0")
	wf.EnableRecover()
	proc := wf.Process("TestMultiTimesRecover0")
	df.BeforeFlow(true, Fx[flow.WorkFlow](t).MultiRecover(1).Callback("Default->Before"))
	wf.BeforeFlow(true, Fx[flow.WorkFlow](t).MultiRecover(2).Callback("Flow->Before"))

	df.BeforeProcess(true, Fx[flow.Process](t).MultiRecover(3).Callback("Default->Before"))
	wf.BeforeProcess(true, Fx[flow.Process](t).MultiRecover(4).Callback("Flow->Before"))
	proc.BeforeProcess(true, Fx[flow.Process](t).MultiRecover(5).Callback("Process->Before"))

	df.BeforeStep(true, Fx[flow.Step](t).MultiRecover(6).Callback("Default->Before"))
	wf.BeforeStep(true, Fx[flow.Step](t).MultiRecover(7).Callback("Flow->Before"))
	proc.BeforeStep(true, Fx[flow.Step](t).MultiRecover(8).Callback("Process->Before"))

	proc.CustomStep(Fx[flow.Step](t).MultiRecover(9).Step(), "step1")

	df.AfterStep(true, Fx[flow.Step](t).MultiRecover(10).Callback("Default->After"))
	wf.AfterStep(true, Fx[flow.Step](t).MultiRecover(11).Callback("Flow->After"))
	proc.AfterStep(true, Fx[flow.Step](t).MultiRecover(12).Callback("Process->After"))

	df.AfterProcess(true, Fx[flow.Process](t).MultiRecover(13).Callback("Default->After"))
	wf.AfterProcess(true, Fx[flow.Process](t).MultiRecover(14).Callback("Flow->After"))
	proc.AfterProcess(true, Fx[flow.Process](t).MultiRecover(15).Callback("Process->After"))

	df.AfterFlow(true, Fx[flow.WorkFlow](t).MultiRecover(16).Callback("Default->After"))
	wf.AfterFlow(true, Fx[flow.WorkFlow](t).MultiRecover(17).Callback("Flow->After"))

	// default before flow failed
	ff := flow.DoneFlow("TestMultiTimesRecover0", nil)
	CheckResult(t, 1, flow.CallbackFail)(any(ff).(flow.WorkFlow))

	atomic.StoreInt64(&recoverFlag, 2)
	// default before flow success, but flow before flow failed
	ff = Recover0(t, ff)
	CheckResult(t, 102, flow.CallbackFail)(any(ff).(flow.WorkFlow))

	atomic.StoreInt64(&recoverFlag, 2)
	// flow before flow success, but default before process failed
	ff = Recover0(t, ff)
	CheckResult(t, 303, flow.CallbackFail)(any(ff).(flow.WorkFlow))

	atomic.StoreInt64(&recoverFlag, 2)
	// default before process success, but flow before process failed
	ff = Recover0(t, ff)
	CheckResult(t, 304, flow.CallbackFail)(any(ff).(flow.WorkFlow))

	atomic.StoreInt64(&recoverFlag, 2)
	// flow before process success, but process before process failed
	ff = Recover0(t, ff)
	CheckResult(t, 305, flow.CallbackFail)(any(ff).(flow.WorkFlow))

	atomic.StoreInt64(&recoverFlag, 2)
	// process before process success, but default before step failed
	ff = Recover0(t, ff)
	CheckResult(t, 606, flow.CallbackFail)(any(ff).(flow.WorkFlow))

	atomic.StoreInt64(&recoverFlag, 2)
	// default before step success, but flow before step failed
	ff = Recover0(t, ff)
	CheckResult(t, 607, flow.CallbackFail)(any(ff).(flow.WorkFlow))

	atomic.StoreInt64(&recoverFlag, 2)
	// flow before step success, but process before step failed
	ff = Recover0(t, ff)
	CheckResult(t, 608, flow.CallbackFail)(any(ff).(flow.WorkFlow))

	atomic.StoreInt64(&recoverFlag, 2)
	// process before step success, but step execute failed
	ff = Recover0(t, ff)
	CheckResult(t, 909, flow.Error)(any(ff).(flow.WorkFlow))

	atomic.StoreInt64(&recoverFlag, 2)
	// step execute success, but default after step failed
	ff = Recover0(t, ff)
	CheckResult(t, 610, flow.CallbackFail)(any(ff).(flow.WorkFlow))

	atomic.StoreInt64(&recoverFlag, 2)
	// default after step success, but flow after step failed
	ff = Recover0(t, ff)
	CheckResult(t, 611, flow.CallbackFail)(any(ff).(flow.WorkFlow))

	atomic.StoreInt64(&recoverFlag, 2)
	// flow after step success, but process before step failed
	ff = Recover0(t, ff)
	CheckResult(t, 612, flow.CallbackFail)(any(ff).(flow.WorkFlow))

	atomic.StoreInt64(&recoverFlag, 2)
	// process before step success, but default after process failed
	ff = Recover0(t, ff)
	CheckResult(t, 313, flow.CallbackFail)(any(ff).(flow.WorkFlow))

	atomic.StoreInt64(&recoverFlag, 2)
	// default after process success, but flow after process failed
	ff = Recover0(t, ff)
	CheckResult(t, 314, flow.CallbackFail)(any(ff).(flow.WorkFlow))

	atomic.StoreInt64(&recoverFlag, 2)
	// replay
	ff = Recover0(t, ff)
	CheckResult(t, 314, flow.CallbackFail)(any(ff).(flow.WorkFlow))

	atomic.StoreInt64(&recoverFlag, 3)
	// flow after process success, but process after process failed
	ff = Recover0(t, ff)
	CheckResult(t, 415, flow.CallbackFail)(any(ff).(flow.WorkFlow))

	atomic.StoreInt64(&recoverFlag, 4)
	// process after process success, but default after flow failed
	ff = Recover0(t, ff)
	CheckResult(t, 316, flow.CallbackFail)(any(ff).(flow.WorkFlow))

	atomic.StoreInt64(&recoverFlag, 2)
	// default after flow success, but flow after flow failed
	ff = Recover0(t, ff)
	CheckResult(t, 117, flow.CallbackFail)(any(ff).(flow.WorkFlow))

	atomic.StoreInt64(&recoverFlag, 3)
	// flow after flow success, final success
	ff = Recover0(t, ff)
	CheckResult(t, 200, flow.Success)(any(ff).(flow.WorkFlow))
}

func TestMultiTimesRecover(t *testing.T) {
	defer resetCurrent()
	defer resetTimes()
	wf := flow.RegisterFlow("TestMultiTimesRecover")
	fx0 := Fx[flow.WorkFlow](t)
	fx1 := Fx[flow.Process](t)
	wf.BeforeFlow(true, fx0.Inc().Callback())
	wf.BeforeProcess(true, fx1.Inc().Callback())
	wf.EnableRecover()
	fx2 := Fx[flow.Step](t)
	fx3 := Fx[flow.Step](t)
	fx4 := Fx[flow.Step](t)
	fx5 := Fx[flow.Step](t)
	proc := wf.Process("TestMultiTimesRecover")
	proc.BeforeStep(true, fx2.Inc().Callback()).OnlyFor("step1")
	proc.AfterStep(true, fx4.Inc().Callback()).OnlyFor("step1")
	proc.CustomStep(fx3.Inc().Step(), "step1")
	proc.CustomStep(fx5.Inc().Step(), "step2", "step1")
	fx6 := Fx[flow.Process](t)
	fx7 := Fx[flow.WorkFlow](t)
	wf.AfterProcess(true, fx6.Inc().Callback())
	wf.AfterFlow(true, fx7.Inc().Callback())
	wf.AfterFlow(false, CheckResult(t, 1, flow.CallbackFail)).If(Times(0))
	wf.AfterFlow(false, CheckResult(t, 2, flow.CallbackFail)).If(Times(1))
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
	if !ff.Success() {
		t.Error("TestMultiTimesRecover failed final")
	}
}

func TestTimeoutRecover(t *testing.T) {
	defer resetCurrent()
	atomic.StoreInt64(&letGo, 0)
	executeSuc = false
	wf := flow.RegisterFlow("TestTimeoutRecover")
	wf.EnableRecover()
	proc := wf.Process("TestTimeoutRecover")
	proc.CustomStep(Fn(t).WaitLetGO(1), "step1")
	proc.StepTimeout(50 * time.Millisecond)
	wf.AfterFlow(false, CheckResult(t, 2, flow.Success)).If(execSuc)
	res := flow.DoneFlow("TestTimeoutRecover", nil)
	CheckResult(t, 1, flow.Timeout)(any(res).(flow.WorkFlow))
	atomic.StoreInt64(&letGo, 1)
	executeSuc = true
	waitCurrent(2)
	Recover("TestTimeoutRecover")
}

func TestStepAndCallbackFailedSameTimeRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestStepAndCallbackFailedSameTimeRecover0")
	wf.EnableRecover()
	proc := wf.Process("TestStepAndCallbackFailedSameTimeRecover0")
	proc.CustomStep(Fn(t).Suc(CheckCtx0("1", "", 1)).Fail(SetCtx()).ErrStep(), "1")
	proc.CustomStep(Fn(t).Suc(CheckCtx("1")).Fail(SetCtx()).Step(), "2", "1")
	proc.AfterStep(true, Fn(t).Suc(CheckCtx0("1", "", 1)).Fail(CheckCtx0("1", "", 1)).ErrStepCall()).
		OnlyFor("1")
	wf.AfterFlow(false, CheckResult(t, 3, flow.Success)).If(execSuc)
	res := flow.DoneFlow("TestStepAndCallbackFailedSameTimeRecover0", nil)
	CheckResult(t, 2, flow.Error, flow.CallbackFail)(any(res).(flow.WorkFlow))
	Recover("TestStepAndCallbackFailedSameTimeRecover0")
}

func TestStepAndCallbackPanicSameTimeRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestStepAndCallbackPanicSameTimeRecover")
	wf.EnableRecover()
	proc := wf.Process("TestStepAndCallbackPanicSameTimeRecover")
	proc.CustomStep(Fn(t).Suc(CheckCtx0("1", "", 1)).Fail(SetCtx()).PanicStep(), "1")
	proc.CustomStep(Fn(t).Suc(CheckCtx("1")).Fail(SetCtx()).Step(), "2", "1")
	proc.AfterStep(true, Fn(t).Suc(CheckCtx0("1", "", 1)).Fail(CheckCtx0("1", "", 1)).PanicStepCall()).
		OnlyFor("1")
	wf.AfterFlow(false, CheckResult(t, 3, flow.Success)).If(execSuc)
	res := flow.DoneFlow("TestStepAndCallbackPanicSameTimeRecover", nil)
	CheckResult(t, 2, flow.Panic, flow.CallbackFail)(any(res).(flow.WorkFlow))
	Recover("TestStepAndCallbackPanicSameTimeRecover")
}

func TestDuplicateBreakPointRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	wf := flow.RegisterFlow("TestDuplicateBreakPointRecover")
	wf.EnableRecover()
	proc := wf.Process("TestDuplicateBreakPointRecover")
	proc.CustomStep(Fn(t).Suc(CheckCtx0("1", "", 1)).Fail(SetCtx()).ErrStep(), "1")
	proc.CustomStep(Fn(t).Suc(CheckCtx("1")).Step(), "2", "1")
	proc.AfterStep(true, Fn(t).Suc(CheckCtx0("1", "", 1)).Fail(CheckCtx0("1", "", 1)).PanicStepCall()).
		OnlyFor("1")
	wf.AfterFlow(false, CheckResult(t, 3, flow.Success)).If(execSuc)
	res := flow.DoneFlow("TestDuplicateBreakPointRecover", nil)
	CheckResult(t, 2, flow.Error, flow.CallbackFail)(any(res).(flow.WorkFlow))
	resetCurrent()
	res, _ = res.Recover()
	CheckResult(t, 2, flow.Error, flow.CallbackFail)(any(res).(flow.WorkFlow))
	Recover("TestDuplicateBreakPointRecover")
}

func TestNilPwdEncryptRecover(t *testing.T) {
	defer resetCurrent()
	executeSuc = false
	flow.SetEncryptor(nil)
	flow.DisableEncrypt()
	wf := flow.RegisterFlow("TestNilPwdEncryptRecover")
	wf.EnableRecover()
	proc := wf.Process("TestNilPwdEncryptRecover")
	proc.CustomStep(Fn(t).Suc(CheckCtx0("1", "", 1)).Fail(SetCtx()).ErrStep(), "1")
	ff := flow.DoneFlow("TestNilPwdEncryptRecover", nil)
	CheckResult(t, 1, flow.Error)(any(ff).(flow.WorkFlow))
	executeSuc = true
	ff = Recover0(t, ff)
	CheckResult(t, 1, flow.Success)(any(ff).(flow.WorkFlow))
}

func TestPool(t *testing.T) {
	db := <-pool
	defer func() { pool <- db }()
	t.Logf("success")
}
