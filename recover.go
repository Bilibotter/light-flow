package light_flow

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"reflect"
	"time"
)

const (
	StepScope uint8 = 16 * (iota + 1)
	ProcessScope
	FlowScope
)

const (
	RecoverIdle recoverCode = 1 << iota
	RecoverRunning
	RecoverSuccess
	RecoverFailed recoverCode = 32
)

var (
	maxSize                          = 4096
	enableEncrypt                    = true
	pwdEncryptor  SymmetricEncryptor = &aes256Encryptor{newRoutineUnsafeSet[string]("pwd", "password")}
	enableRecover                    = false
	persister     Persist
)

type proto interface {
	runtimeI
	isRecoverable() bool
	getInternal(key string) (value any, exist bool)
	setInternal(key string, value any)
}

type Persist interface {
	GetLatestRecord(rootId string) (RecoverRecord, error)
	ListCheckpoints(recoveryId string) ([]CheckPoint, error)
	UpdateRecordStatus(record RecoverRecord) error
	// save checkpoint and update workflow latest recordId
	SaveCheckpointAndRecord(checkpoint []CheckPoint, record RecoverRecord) error
}

type SymmetricEncryptor interface {
	encryptor
	NeedEncrypt(key string) bool
	GetSecret() []byte
}

type encryptor interface {
	Encrypt(plainText string, secret []byte) (string, error)
	Decrypt(cipherText string, secret []byte) (string, error)
}

type checkpointBuilder interface {
	CheckPoint
	setRecoverId(string)
	buildSnapshot() error
}

type CheckPoint interface {
	associative
	scene
}

type associative interface {
	GetPrimaryKey() string
	GetName() string
	GetParentId() string
	GetRootId() string
	GetScope() uint8
}

type scene interface {
	// Each recovery operation generates a new recovery ID that is consistent across new checkpoints
	GetRecoverId() string
	GetSnapshot() []byte
}

type RecoverRecord interface {
	GetName() string
	GetRootId() string // root id is equal to workflow id
	GetRecoverId() string
	GetStatus() uint8
}

type recoverCode uint8

type outcomeValue struct {
	O     outcome
	Value any
}

// Most serializers will convert the pointer to a struct and use pointerValue to identify the pointer.
type pointerValue struct {
	Elem interface{}
}

type recoverRecord struct {
	RecoverId string
	RootId    string
	Status    recoverCode
	Name      string
}

type flowCheckpoint struct {
	*runFlow
	recoverId string
	snapshot  []byte
}

type procCheckpoint struct {
	*runProcess
	recoverId string
	snapshot  []byte
}

type stepCheckpoint struct {
	*runStep
	recoverId string
	snapshot  []byte
}

type aes256Encryptor struct {
	*set[string]
}

func init() {
	RegisterType[time.Time]()

	RegisterType[outcome]()
	RegisterType[outcomeValue]()

	RegisterType[pointerValue]()
	RegisterType[breakPoint]()
}

func DisableEncrypt() {
	enableEncrypt = false
}

func SetEncryptor(encryptor SymmetricEncryptor) {
	pwdEncryptor = encryptor
}

func SetPersist(impl Persist) {
	persister = impl
}

func RegisterType[T any]() {
	var t T
	kind := reflect.TypeOf(t).Kind()
	if kind == reflect.Pointer {
		panic("can not register pointer type, use struct instead")
	}
	gob.Register(t)
}

func EnableRecover() {
	if persister == nil {
		panic("Persistence must be set to use the recovery function")
	}
	enableRecover = true
}

func DisableRecover() {
	enableRecover = false
}

// todo optimize performance
func RecoverFlow(flowId string) (err error) {
	record, err := persister.GetLatestRecord(flowId)
	if err != nil {
		return err
	}
	checkpoints, err := persister.ListCheckpoints(record.GetRecoverId())
	if err != nil {
		return err
	}
	factory, ok := allFlows.Load(record.GetName())
	if !ok {
		return fmt.Errorf("flow[%s] not found", record.GetName())
	}
	flow := factory.(*FlowMeta).buildRunFlow(nil)
	if err = loadCheckpoints(flow, checkpoints); err != nil {
		return
	}
	if err = loadStatus(flow, checkpoints); err != nil {
		return
	}
	persister.UpdateRecordStatus(&recoverRecord{RecoverId: record.GetRecoverId(), Status: RecoverRunning})
	flow.append(Recovering)
	flow.Done()
	if flow.Success() {
		persister.UpdateRecordStatus(&recoverRecord{RecoverId: record.GetRecoverId(), Status: RecoverSuccess})
	} else {
		persister.UpdateRecordStatus(&recoverRecord{RecoverId: record.GetRecoverId(), Status: RecoverFailed})
	}
	return nil
}

func SetMaxSerializeSize(size int) {
	maxSize = size
}

func loadStatus(workflow *runFlow, checkpoints []CheckPoint) error {
	point, exist := workflow.getInternal(fmt.Sprintf(flowBreakPoint, workflow.name))
	// the must before callback failed to execute last time, so all steps and processes need to be executed at this time
	if exist && !point.(*breakPoint).SkipRun {
		return nil
	}
	for _, proc := range workflow.processes {
		proc.append(executed)
		for _, step := range proc.flowSteps {
			step.append(executed)
		}
	}
	id2Name := make(map[string]string)
	for _, checkpoint := range checkpoints {
		if checkpoint.GetScope() != ProcessScope {
			continue
		}
		name := checkpoint.GetName()
		workflow.processes[name].clear(executed)
		workflow.processes[name].append(Recovering)
		point, exist = workflow.processes[name].getInternal(fmt.Sprintf(procBreakPoint, name))
		if exist && !point.(*breakPoint).SkipRun {
			for _, step := range workflow.processes[name].flowSteps {
				step.clear(executed)
			}
		}
		id2Name[checkpoint.GetPrimaryKey()] = name
	}
	for _, checkpoint := range checkpoints {
		if checkpoint.GetScope() != StepScope {
			continue
		}
		name := checkpoint.GetName()
		proc, ok := workflow.processes[id2Name[checkpoint.GetParentId()]]
		if !ok {
			return fmt.Errorf("unable to recognize the process to which Step[%s] belongs", checkpoint.GetName())
		}
		current, find := proc.flowSteps[name]
		if !find {
			return fmt.Errorf("Step[%s] not belong to Process[%s]", name, proc.name)
		}
		current.append(Recovering)
		proc.clearExecutedFromRoot(name)
	}
	return nil
}

func loadCheckpoints(workflow *runFlow, checkpoints []CheckPoint) (err error) {
	id2Name := make(map[string]string)
	for _, checkpoint := range checkpoints {
		if checkpoint.GetScope() == ProcessScope {
			id2Name[checkpoint.GetPrimaryKey()] = checkpoint.GetName()
		}
	}
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("[Recovery] panic recovered:\n%s\n%s\n", r, stack())
		}
	}()
	for _, checkpoint := range checkpoints {
		switch checkpoint.GetScope() {
		case FlowScope:
			err = workflow.loadCheckpoint(checkpoint)
		case ProcessScope:
			proc := workflow.processes[id2Name[checkpoint.GetPrimaryKey()]]
			err = proc.loadCheckpoint(checkpoint)
		case StepScope:
			belong := workflow.processes[id2Name[checkpoint.GetParentId()]]
			step := belong.flowSteps[checkpoint.GetName()]
			err = step.loadCheckpoint(checkpoint)
		default:
			err = fmt.Errorf("CheckPoint[%s] has unknown scope %d", checkpoint.GetName(), checkpoint.GetScope())
		}
		if err != nil {
			return err
		}
	}
	return
}

func wrapIfNeed(data any) {
	switch m := data.(type) {
	case map[string][]node:
		wrapNodeMap(m)
	case []map[string]any:
		wrapInterfaceMap(m)
	default:
		return
	}
}

func wrapNodeMap(m map[string][]node) {
	for k := range m {
		for i := range m[k] {
			if m[k][i].Value == nil {
				continue
			}
			kind := reflect.TypeOf(m[k][i].Value).Kind()
			if kind == reflect.Pointer {
				m[k][i].Value = pointerValue{Elem: m[k][i].Value}
			}
		}
	}
}

func wrapInterfaceMap(listMap []map[string]any) {
	for i := range listMap {
		for k := range listMap[i] {
			kind := reflect.TypeOf(listMap[i][k]).Kind()
			if kind == reflect.Pointer {
				listMap[i][k] = pointerValue{Elem: listMap[i][k]}
			}
		}
	}

}

func unwrapIfNeed(data any) {
	switch m := data.(type) {
	case map[string][]node:
		unwrapNodeMap(m)
	case []map[string]any:
		unwrapInterfaceMap(m)
	default:
		return
	}
}

func unwrapNodeMap(m map[string][]node) {
	for k := range m {
		for i := range m[k] {
			if m[k][i].Value == nil {
				continue
			}
			if pv, match := m[k][i].Value.(pointerValue); match {
				pointer := reflect.New(reflect.TypeOf(pv.Elem))
				pointer.Elem().Set(reflect.ValueOf(pv.Elem))
				m[k][i].Value = pointer.Interface()
			}
		}
	}
}

func unwrapInterfaceMap(listMap []map[string]any) {
	for i := range listMap {
		for k := range listMap[i] {
			wrap, ok := listMap[i][k].(pointerValue)
			if !ok {
				continue
			}
			pointer := reflect.New(reflect.TypeOf(wrap.Elem))
			pointer.Elem().Set(reflect.ValueOf(wrap.Elem))
			listMap[i][k] = pointer.Interface()
		}
	}
}

func getSecret(checkpoint CheckPoint) []byte {
	secret := pwdEncryptor.GetSecret()
	if secret == nil {
		if _, ok := pwdEncryptor.(*aes256Encryptor); ok {
			secret = []byte(checkpoint.GetRecoverId())
		}
	}
	return secret
}

func serialize[T any](value T) ([]byte, error) {
	wrapIfNeed(value)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(value); err != nil {
		return nil, err
	}
	if buf.Len() > maxSize {
		return nil, errors.New("data size exceeds max limit")
	}
	return buf.Bytes(), nil
}

func deserialize[T any](data []byte) (result T, err error) {
	if len(data) > maxSize {
		err = errors.New("data size exceeds max limit")
		return
	}

	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	if err = dec.Decode(&result); err != nil {
		return
	}
	unwrapIfNeed(result)
	return result, nil
}

func encryptIfNeed(key string, value any, secret []byte) (any, error) {
	if !enableEncrypt {
		return value, nil
	}
	if !pwdEncryptor.NeedEncrypt(key) {
		return value, nil
	}
	if plainText, ok := value.(string); ok {
		return pwdEncryptor.Encrypt(plainText, secret)
	}
	return value, nil
}

func decryptIfNeed(key string, value any, secret []byte) (any, error) {
	if !enableEncrypt {
		return value, nil
	}
	if !pwdEncryptor.NeedEncrypt(key) {
		return value, nil
	}
	if cipherText, ok := value.(string); ok {
		return pwdEncryptor.Decrypt(cipherText, secret)
	}
	return value, nil
}

func (r *recoverRecord) GetRootId() string {
	return r.RootId
}

func (r *recoverRecord) GetRecoverId() string {
	return r.RecoverId
}

func (r *recoverRecord) GetStatus() uint8 {
	return uint8(r.Status)
}

func (r *recoverRecord) GetName() string {
	return r.Name
}

func (b *aes256Encryptor) NeedEncrypt(key string) bool {
	return b.Contains(key)
}

func (b *aes256Encryptor) GetSecret() []byte {
	return nil
}

func (b *aes256Encryptor) Encrypt(plainText string, secret []byte) (string, error) {
	key := sha256.Sum256(secret)
	text := []byte(plainText)
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}

	cipherText := make([]byte, aes.BlockSize+len(text))
	iv := cipherText[:aes.BlockSize]
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], text)

	return base64.URLEncoding.EncodeToString(cipherText), nil
}

func (b *aes256Encryptor) Decrypt(cipherText string, secret []byte) (string, error) {
	cipherTextBytes, err := base64.URLEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}
	key := sha256.Sum256(secret)
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}

	if len(cipherTextBytes) < aes.BlockSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	iv := cipherTextBytes[:aes.BlockSize]
	cipherTextBytes = cipherTextBytes[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(cipherTextBytes, cipherTextBytes)

	return string(cipherTextBytes), nil
}

func (point *flowCheckpoint) GetPrimaryKey() string {
	return point.id
}

func (point *flowCheckpoint) GetName() string {
	return point.name
}

func (point *flowCheckpoint) GetParentId() string {
	return ""
}

func (point *flowCheckpoint) GetRootId() string {
	return point.id
}

func (point *flowCheckpoint) GetScope() uint8 {
	return FlowScope
}

func (point *flowCheckpoint) GetRecoverId() string {
	return point.recoverId
}

func (point *flowCheckpoint) setRecoverId(id string) {
	point.recoverId = id
}

func (point *flowCheckpoint) buildSnapshot() (err error) {
	secret := getSecret(point)
	ctx := make(map[string]any)
	for k, v := range point.table {
		ctx[k], err = encryptIfNeed(k, v, secret)
		if err != nil {
			return
		}
	}
	point.snapshot, err = serialize([]map[string]any{ctx, point.internal})
	return
}

func (point *flowCheckpoint) GetSnapshot() []byte {
	return point.snapshot
}

func (point *procCheckpoint) GetPrimaryKey() string {
	return point.id
}

func (point *procCheckpoint) GetName() string {
	return point.name
}

func (point *procCheckpoint) GetParentId() string {
	return point.FlowID()
}

func (point *procCheckpoint) GetRootId() string {
	return point.FlowID()
}

func (point *procCheckpoint) GetScope() uint8 {
	return ProcessScope
}

func (point *procCheckpoint) GetRecoverId() string {
	return point.recoverId
}

func (point *procCheckpoint) setRecoverId(id string) {
	point.recoverId = id
}

func (point *procCheckpoint) buildSnapshot() (err error) {
	secret := getSecret(point)
	snapshot := make(map[string][]node)
	for k := range point.nodes {
		for head := point.nodes[k]; head != nil; head = head.Next {
			head.Value, err = encryptIfNeed(k, head.Value, secret)
			if err != nil {
				return
			}
			snapshot[k] = append(snapshot[k], *head)
		}
	}
	point.snapshot, err = serialize(snapshot)
	return
}

func (point *procCheckpoint) GetSnapshot() []byte {
	return point.snapshot
}

func (point *stepCheckpoint) GetPrimaryKey() string {
	return point.id
}

func (point *stepCheckpoint) GetName() string {
	return point.name
}

func (point *stepCheckpoint) GetParentId() string {
	return point.ProcessID()
}

func (point *stepCheckpoint) GetRootId() string {
	return point.FlowID()
}

func (point *stepCheckpoint) GetScope() uint8 {
	return StepScope
}

func (point *stepCheckpoint) GetRecoverId() string {
	return point.recoverId
}

func (point *stepCheckpoint) setRecoverId(id string) {
	point.recoverId = id
}

func (point *stepCheckpoint) buildSnapshot() (err error) {
	return nil
}

func (point *stepCheckpoint) GetSnapshot() []byte {
	return point.snapshot
}

func (rf *runFlow) loadCheckpoint(checkpoint CheckPoint) error {
	combine, err := deserialize[[]map[string]any](checkpoint.GetSnapshot())
	if err != nil {
		return err
	}
	snapshot := combine[0]
	for k := range snapshot {
		v := snapshot[k]
		snapshot[k], err = decryptIfNeed(k, v, getSecret(checkpoint))
		if err != nil {
			return err
		}
	}
	rf.table = snapshot
	rf.internal = combine[1]
	rf.id = checkpoint.GetPrimaryKey()
	return nil
}

func (rf *runFlow) isRecoverable() bool {
	// multiple recoveries are not supported
	if rf.Has(Recovering) {
		return false
	}
	if rf.enableRecover < 0 || rf.Success() {
		return false
	}
	if rf.enableRecover == 0 && !enableRecover {
		return false
	}
	return true
}

func (rf *runFlow) saveCheckpoints() {
	defer func() {
		if r := recover(); r != nil {
			// todo send panic event
		}
	}()
	checkpoints := make([]CheckPoint, 0)
	recoverId := generateId()
	checkpoints = append(checkpoints, &flowCheckpoint{runFlow: rf})
	for _, proc := range rf.processes {
		if proc.Success() {
			continue
		}
		for _, step := range proc.flowSteps {
			if step.needRecover() {
				checkpoints = append(checkpoints, &stepCheckpoint{runStep: step})
			}
		}
		checkpoints = append(checkpoints, &procCheckpoint{runProcess: proc})
	}
	for _, checkpoint := range checkpoints {
		checkpoint.(checkpointBuilder).setRecoverId(recoverId)
		if err := checkpoint.(checkpointBuilder).buildSnapshot(); err != nil {
			break
		}
	}
	record := &recoverRecord{
		RecoverId: recoverId,
		RootId:    rf.id,
		Status:    RecoverIdle,
		Name:      rf.name,
	}
	persister.SaveCheckpointAndRecord(checkpoints, record)
}

func (process *runProcess) loadCheckpoint(checkpoint CheckPoint) error {
	process.id = checkpoint.GetPrimaryKey()
	snapshot, err := deserialize[map[string][]node](checkpoint.GetSnapshot())
	if err != nil {
		return err
	}
	for k := range snapshot {
		var head, current *node
		nodeList := snapshot[k]
		for i := len(nodeList) - 1; i >= 0; i-- {
			newNode := &node{
				Path: nodeList[i].Path,
				Next: head,
			}
			newNode.Value, err = decryptIfNeed(k, nodeList[i].Value, getSecret(checkpoint))
			if err != nil {
				return err
			}
			head = newNode
			if current == nil {
				current = newNode
			}
		}
		process.nodes[k] = head
	}
	return nil
}

func (step *runStep) loadCheckpoint(checkpoint CheckPoint) error {
	step.id = checkpoint.GetPrimaryKey()
	return nil
}
