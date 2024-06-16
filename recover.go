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
	maxSize                          = 2048
	enableEncrypt                    = true
	pwdEncryptor  SymmetricEncryptor = newBcryptEncryptor()
	enableRecover                    = false
	persister     Persist
)

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
	GetId() string
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
	GetRootId() string // root id is equal to workflow id
	GetRecoverId() string
	GetStatus() uint8
	GetName() string
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
	RegisterType[pointerValue]()
	RegisterType[time.Time]()
	RegisterType[outcome]()
	RegisterType[outcomeValue]()
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
	if err = markExecuted(flow, checkpoints); err != nil {
		return
	}
	persister.UpdateRecordStatus(&recoverRecord{RecoverId: record.GetRecoverId(), Status: RecoverRunning})
	flow.Done()
	if flow.Success() {
		persister.UpdateRecordStatus(&recoverRecord{RecoverId: record.GetRecoverId(), Status: RecoverSuccess})
	} else {
		persister.UpdateRecordStatus(&recoverRecord{RecoverId: record.GetRecoverId(), Status: RecoverFailed})
	}
	return nil
}

func markExecuted(workflow *runFlow, checkpoints []CheckPoint) error {
	for _, proc := range workflow.processes {
		for _, step := range proc.flowSteps {
			step.set(executed)
		}
	}
	id2Name := make(map[string]string)
	for _, checkpoint := range checkpoints {
		if checkpoint.GetScope() == ProcessScope {
			id2Name[checkpoint.GetId()] = checkpoint.GetName()
		}
	}
	for _, checkpoint := range checkpoints {
		if checkpoint.GetScope() == StepScope {
			proc, exist := workflow.processes[id2Name[checkpoint.GetParentId()]]
			if !exist {
				return fmt.Errorf("unable to recognize the process to which Step[%s] belongs", checkpoint.GetName())
			}
			_, find := proc.flowSteps[checkpoint.GetName()]
			if !find {
				return fmt.Errorf("step[%s] not belong to process[%s]", checkpoint.GetName(), proc.name)
			}
			proc.clearExecutedFromRoot(checkpoint.GetName())
		}
	}
	return nil
}

func loadCheckpoints(workflow *runFlow, checkpoints []CheckPoint) (err error) {
	id2Name := make(map[string]string)
	for _, checkpoint := range checkpoints {
		if checkpoint.GetScope() == ProcessScope {
			id2Name[checkpoint.GetId()] = checkpoint.GetName()
		}
	}
	defer func() {
		if r := recover(); r != nil {
			err = newPanicError("load checkpoint failed", r)
		}
	}()
	for _, checkpoint := range checkpoints {
		switch checkpoint.GetScope() {
		case FlowScope:
			err = workflow.loadCheckpoint(checkpoint)
		case ProcessScope:
			proc := workflow.processes[id2Name[checkpoint.GetId()]]
			err = proc.loadCheckpoint(checkpoint)
		case StepScope:
			belong := workflow.processes[id2Name[checkpoint.GetParentId()]]
			step := belong.flowSteps[checkpoint.GetName()]
			err = step.loadCheckpoint(checkpoint)
		default:
			err = fmt.Errorf("CheckPointp[%s] has unknown scope %d", checkpoint.GetName(), checkpoint.GetScope())
		}
		if err != nil {
			return err
		}
	}
	return
}

func wrapIfNeed(data any) {
	m, ok := data.(map[string]any)
	if !ok {
		return
	}
	for k := range m {
		kind := reflect.TypeOf(m[k]).Kind()
		if kind == reflect.Pointer {
			m[k] = pointerValue{Elem: m[k]}
		}
	}
}

func unwrapIfNeed(data any) {
	m, ok := data.(map[string]any)
	if !ok {
		return
	}
	for k := range m {
		if pv, suc := m[k].(pointerValue); suc {
			pointer := reflect.New(reflect.TypeOf(pv.Elem))
			pointer.Elem().Set(reflect.ValueOf(pv.Elem))
			m[k] = pointer.Interface()
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

func newBcryptEncryptor() *aes256Encryptor {
	s := newRoutineUnsafeSet[string]("pwd", "password")
	return &aes256Encryptor{s}
}

func encryptIfNeed(snapshot map[string]any, secret []byte) error {
	if !enableEncrypt {
		return nil
	}
	for k := range snapshot {
		if !pwdEncryptor.NeedEncrypt(k) {
			continue
		}
		if v, ok := snapshot[k].(string); ok {
			encrypted, err := pwdEncryptor.Encrypt(v, secret)
			if err != nil {
				return fmt.Errorf("encrypt Key[%s] failed, error: %s", k, err.Error())
			}
			snapshot[k] = encrypted
		}
	}
	return nil
}

func decryptIfNeed(snapshot map[string]any, secret []byte) error {
	if !enableEncrypt {
		return nil
	}
	for k := range snapshot {
		if !pwdEncryptor.NeedEncrypt(k) {
			continue
		}
		if v, ok := snapshot[k].(string); ok {
			decrypted, err := pwdEncryptor.Decrypt(v, secret)
			if err != nil {
				return fmt.Errorf("decrypt Key[%s] failed, error: %s", k, err.Error())
			}
			snapshot[k] = decrypted
		}
	}
	return nil
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

func (point *flowCheckpoint) GetParentId() string {
	return ""
}

func (point *flowCheckpoint) GetRootId() string {
	return point.Id
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
	snapshot := make(map[string]any)
	if err = encryptIfNeed(snapshot, getSecret(point)); err != nil {
		return
	}
	point.snapshot, err = serialize(snapshot)
	return
}

func (point *flowCheckpoint) GetSnapshot() []byte {
	return point.snapshot
}

func (point *procCheckpoint) GetId() string {
	return point.id
}

func (point *procCheckpoint) GetName() string {
	return point.name
}

func (point *procCheckpoint) GetParentId() string {
	return point.GetFlowId()
}

func (point *procCheckpoint) GetRootId() string {
	return point.GetFlowId()
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
	snapshot := make(map[string]any)
	for k := range point.nodes {
		if value, exist := point.matchByIndex(resultMark, k); exist {
			snapshot[k] = value
		}
	}
	if err = encryptIfNeed(snapshot, getSecret(point)); err != nil {
		return
	}
	for name := range point.steps {
		wrap, exist := point.getOutCome(name)
		point.lock.RUnlock()
		if !exist {
			continue
		}
		snapshot[name] = outcomeValue{O: *wrap, Value: snapshot[name]}
	}

	point.snapshot, err = serialize(snapshot)

	return
}

func (point *procCheckpoint) GetSnapshot() []byte {
	return point.snapshot
}

func (point *stepCheckpoint) GetId() string {
	return point.id
}

func (point *stepCheckpoint) GetName() string {
	return point.name
}

func (point *stepCheckpoint) GetParentId() string {
	return point.GetProcessId()
}

func (point *stepCheckpoint) GetRootId() string {
	return point.GetFlowId()
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
	snapshot := make(map[string]any)
	for key := range point.nodes {
		if value, exist := point.Get(key); exist {
			snapshot[key] = value
		}
	}
	if err = encryptIfNeed(snapshot, getSecret(point)); err != nil {
		return
	}
	point.snapshot, err = serialize(snapshot)
	return
}

func (point *stepCheckpoint) GetSnapshot() []byte {
	return point.snapshot
}

func (rf *runFlow) loadCheckpoint(checkpoint CheckPoint) error {
	snapshot, err := deserialize[map[string]any](checkpoint.GetSnapshot())
	if err != nil {
		return err
	}
	if err = decryptIfNeed(snapshot, getSecret(checkpoint)); err != nil {
		return err
	}
	rf.table = snapshot
	rf.Id = checkpoint.GetId()
	return nil
}

func (rf *runFlow) shallRecover() bool {
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
	checkpoints := make([]CheckPoint, 0, 3)
	recoverId := generateId()
	checkpoints = append(checkpoints, &flowCheckpoint{runFlow: rf})
	for _, proc := range rf.processes {
		if proc.Success() {
			continue
		}
		checkpoints = append(checkpoints, &procCheckpoint{runProcess: proc})
		for _, step := range proc.flowSteps {
			if step.needRecover() {
				checkpoints = append(checkpoints, &stepCheckpoint{runStep: step})
			}
		}
	}
	for _, checkpoint := range checkpoints {
		checkpoint.(checkpointBuilder).setRecoverId(recoverId)
		if err := checkpoint.(checkpointBuilder).buildSnapshot(); err != nil {
			break
		}
	}
	record := &recoverRecord{
		RecoverId: recoverId,
		RootId:    rf.Id,
		Status:    RecoverIdle,
		Name:      rf.Name,
	}
	persister.SaveCheckpointAndRecord(checkpoints, record)
}

func (rp *runProcess) loadCheckpoint(checkpoint CheckPoint) error {
	rp.id = checkpoint.GetId()
	snapshot, err := deserialize[map[string]any](checkpoint.GetSnapshot())
	if err != nil {
		return err
	}
	if err = decryptIfNeed(snapshot, getSecret(checkpoint)); err != nil {
		return err
	}
	for k, v := range snapshot {
		if unwrap, ok := v.(outcomeValue); ok {
			rp.restoreOutcome(k, &unwrap.O)
			if unwrap.Value != nil {
				rp.Set(k, unwrap.Value)
			}
			continue
		}
		rp.Set(k, v)
	}
	return nil
}

func (step *runStep) loadCheckpoint(checkpoint CheckPoint) error {
	step.id = checkpoint.GetId()
	snapshot, err := deserialize[map[string]any](checkpoint.GetSnapshot())
	if err != nil {
		return err
	}
	if err = decryptIfNeed(snapshot, getSecret(checkpoint)); err != nil {
		return err
	}
	for k, v := range snapshot {
		step.Set(k, v)
	}
	return nil
}
