package cloudtasks

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"google.golang.org/api/option"
	taskspb "google.golang.org/genproto/googleapis/cloud/tasks/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

type Faker struct {
	serv      *grpc.Server
	mock      *mockCloudTasksServer
	ClientOpt option.ClientOption

	mockForIndexResponseIndex int
}

func NewFaker(t *testing.T) *Faker {
	t.Helper()

	mockCloudTasks := mockCloudTasksServer{
		mutex:                   &sync.RWMutex{},
		mockResponseForIndex:    make(map[int]*mockTaskResponse),
		mockResponseForTaskName: make(map[string]*mockTaskResponse),
	}

	serv := grpc.NewServer()
	taskspb.RegisterCloudTasksServer(serv, &mockCloudTasks)

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		log.Fatal(err)
	}
	go serv.Serve(lis)

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}

	return &Faker{
		serv:      serv,
		mock:      &mockCloudTasks,
		ClientOpt: option.WithGRPCConn(conn),
	}
}

func (f *Faker) Stop() {
	f.serv.Stop()
}

// AddMockResponse is Call された回数
func (f *Faker) AddMockResponse(err error, resp ...proto.Message) {
	f.mock.mutex.Lock()
	defer f.mock.mutex.Unlock()

	f.mockForIndexResponseIndex++

	f.mock.mockResponseForIndex[f.mockForIndexResponseIndex] = &mockTaskResponse{
		err:  err,
		resp: resp,
	}
}

// AddMockResponseWithIndex is CreateTask が Call した回数が一致した時に返す Mock Response を追加する
func (f *Faker) AddMockResponseWithIndex(callCount int, err error, resp ...proto.Message) {
	f.mock.mutex.Lock()
	defer f.mock.mutex.Unlock()

	f.mock.mockResponseForIndex[callCount] = &mockTaskResponse{
		err:  err,
		resp: resp,
	}
}

// AddMockResponseWithTaskName is TaskName が一致した時に返す Mock Response を追加する
func (f *Faker) AddMockResponseWithTaskName(taskName string, err error, resp ...proto.Message) {
	f.mock.mutex.Lock()
	defer f.mock.mutex.Unlock()

	f.mock.mockResponseForTaskName[taskName] = &mockTaskResponse{
		err:  err,
		resp: resp,
	}
}

// GetCreateTaskCallCount is Returns the number of times CreateTask was called
func (f *Faker) GetCreateTaskCallCount() int {
	f.mock.mutex.RLock()
	defer f.mock.mutex.RUnlock()

	return len(f.mock.callCreateTaskReqs)
}

// GetCreateTaskRequest is Returns the request passed to CreateTask
func (f *Faker) GetCreateTaskRequest(i int) (*taskspb.CreateTaskRequest, error) {
	f.mock.mutex.RLock()
	defer f.mock.mutex.RUnlock()

	if i > len(f.mock.callCreateTaskReqs)-1 {
		return nil, fmt.Errorf("GetCreateTaskRequest out of range. arg=%d,len=%d", i, len(f.mock.callCreateTaskReqs))
	}
	return f.mock.callCreateTaskReqs[i], nil
}

type mockTaskResponse struct {
	err  error
	resp []proto.Message
}

type mockCloudTasksServer struct {
	// Embed for forward compatibility.
	// Tests will keep working if more methods are added
	// in the future.
	taskspb.CloudTasksServer

	mutex *sync.RWMutex

	// CreateTask が呼ばれた時の Request が順番に入っている
	callCreateTaskReqs []*taskspb.CreateTaskRequest

	// 呼んだ回数を指定してMockResponseを返す
	mockResponseForIndex map[int]*mockTaskResponse

	// TaskNameを指定してMockResponseを返す
	mockResponseForTaskName map[string]*mockTaskResponse
}

func (s *mockCloudTasksServer) CreateTask(ctx context.Context, req *taskspb.CreateTaskRequest) (*taskspb.Task, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}

	s.callCreateTaskReqs = append(s.callCreateTaskReqs, req)

	v, ok := s.mockResponseForTaskName[req.Task.GetName()]
	if ok {
		return v.resp[0].(*taskspb.Task), nil
	}
	v, ok = s.mockResponseForIndex[len(s.callCreateTaskReqs)]
	if ok {
		return v.resp[0].(*taskspb.Task), nil
	}

	return s.createDefaultResponse(ctx, req)
}

func (s *mockCloudTasksServer) createDefaultResponse(ctx context.Context, req *taskspb.CreateTaskRequest) (*taskspb.Task, error) {
	if req.GetTask() == nil {
		return nil, fmt.Errorf("task is required")
	}
	t := req.GetTask()
	mockTask := &taskspb.Task{
		Name:             t.GetName(),
		MessageType:      t.GetMessageType(),
		ScheduleTime:     t.GetScheduleTime(),
		CreateTime:       t.GetCreateTime(),
		DispatchDeadline: t.GetDispatchDeadline(),
		DispatchCount:    t.GetDispatchCount(),
		ResponseCount:    t.GetResponseCount(),
		FirstAttempt:     t.GetFirstAttempt(),
		LastAttempt:      t.GetLastAttempt(),
		View:             t.GetView(),
	}
	if len(mockTask.GetName()) < 1 {
		mockTask.Name = fmt.Sprintf("%s/tasks/%s", req.GetParent(), uuid.New().String())
	}

	return mockTask, nil
}
