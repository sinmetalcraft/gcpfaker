package cloudtasks_test

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	tasksfaker "github.com/sinmetalcraft/gcpfaker/cloudtasks"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	_ "github.com/sinmetalcraft/gcpfaker/hook"
	taskspb "google.golang.org/genproto/googleapis/cloud/tasks/v2"
	"google.golang.org/protobuf/proto"
)

func TestNewFakerWithoutTesting(t *testing.T) {
	cases := []struct {
		name      string
		callCount int
	}{
		{"one", 1},
		{"two", 2},
	}

	parent := fmt.Sprintf("projects/%s/locations/%s/queues/%s", "[PROJECT]", "[LOCATION]", "[QUEUE]")
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			faker := tasksfaker.NewFakerWithoutTesting()
			defer faker.Stop()

			var expectedResponses []*taskspb.Task
			for i := 0; i < tt.callCount; i++ {
				name := fmt.Sprintf("%s/tasks/name%d", parent, rand.Int())
				var dispatchCount int32 = 1217252086
				var responseCount int32 = 424727441
				var expectedResponse = &taskspb.Task{
					Name:          name,
					DispatchCount: dispatchCount,
					ResponseCount: responseCount,
				}
				faker.AddMockResponse(nil, expectedResponse)
				expectedResponses = append(expectedResponses, expectedResponse)
			}

			var task *taskspb.Task = &taskspb.Task{}
			var request = &taskspb.CreateTaskRequest{
				Parent: parent,
				Task:   task,
			}

			c, err := cloudtasks.NewClient(context.Background(), faker.ClientOpt)
			if err != nil {
				t.Fatal(err)
			}

			for i := 0; i < tt.callCount; i++ {
				resp, err := c.CreateTask(context.Background(), request)
				if err != nil {
					t.Fatal(err)
				}

				req, err := faker.GetCreateTaskRequest(i)
				if err != nil {
					t.Fatal(err)
				}
				if e, g := request, req; !proto.Equal(e, g) {
					t.Errorf("request want %q, but got %q", e, g)
				}

				if e, g := expectedResponses[i], resp; !proto.Equal(e, g) {
					t.Errorf("response want %q, but got %q)", e, g)
				}
			}

			if e, g := tt.callCount, faker.GetCreateTaskCallCount(); e != g {
				t.Errorf("createTaskCallCount want %v but got %v", e, g)
			}
		})
	}
}

func TestMockCloudTasksServer_CreateTask(t *testing.T) {
	cases := []struct {
		name      string
		callCount int
	}{
		{"one", 1},
		{"two", 2},
	}

	parent := fmt.Sprintf("projects/%s/locations/%s/queues/%s", "[PROJECT]", "[LOCATION]", "[QUEUE]")
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			faker := tasksfaker.NewFaker(t)
			defer faker.Stop()

			var expectedResponses []*taskspb.Task
			for i := 0; i < tt.callCount; i++ {
				name := fmt.Sprintf("%s/tasks/name%d", parent, rand.Int())
				var dispatchCount int32 = 1217252086
				var responseCount int32 = 424727441
				var expectedResponse = &taskspb.Task{
					Name:          name,
					DispatchCount: dispatchCount,
					ResponseCount: responseCount,
				}
				faker.AddMockResponse(nil, expectedResponse)
				expectedResponses = append(expectedResponses, expectedResponse)
			}

			var task *taskspb.Task = &taskspb.Task{}
			var request = &taskspb.CreateTaskRequest{
				Parent: parent,
				Task:   task,
			}

			c, err := cloudtasks.NewClient(context.Background(), faker.ClientOpt)
			if err != nil {
				t.Fatal(err)
			}

			for i := 0; i < tt.callCount; i++ {
				resp, err := c.CreateTask(context.Background(), request)
				if err != nil {
					t.Fatal(err)
				}

				req, err := faker.GetCreateTaskRequest(i)
				if err != nil {
					t.Fatal(err)
				}
				if e, g := request, req; !proto.Equal(e, g) {
					t.Errorf("request want %q, but got %q", e, g)
				}

				if e, g := expectedResponses[i], resp; !proto.Equal(e, g) {
					t.Errorf("response want %q, but got %q)", e, g)
				}
			}

			if e, g := tt.callCount, faker.GetCreateTaskCallCount(); e != g {
				t.Errorf("createTaskCallCount want %v but got %v", e, g)
			}
		})
	}
}

// TestCreateTask_defaultResponse is AddMockResponse を呼ばずに default の Response を受け取ることで問題が起きないかをテスト
func TestCreateTask_defaultResponse(t *testing.T) {
	cases := []struct {
		name      string
		callCount int
	}{
		{"one", 1},
		{"two", 2},
		{"51", 51},
	}

	parent := fmt.Sprintf("projects/%s/locations/%s/queues/%s", "[PROJECT]", "[LOCATION]", "[QUEUE]")
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			faker := tasksfaker.NewFaker(t)
			defer faker.Stop()

			c, err := cloudtasks.NewClient(context.Background(), faker.ClientOpt)
			if err != nil {
				t.Fatal(err)
			}

			var expectedResponses []*taskspb.Task
			for i := 0; i < tt.callCount; i++ {
				var expectedResponse = &taskspb.Task{
					Name: fmt.Sprintf("%s/tasks/name%d", parent, rand.Int()),
					MessageType: &taskspb.Task_AppEngineHttpRequest{
						AppEngineHttpRequest: &taskspb.AppEngineHttpRequest{
							HttpMethod:  taskspb.HttpMethod_GET,
							RelativeUri: "/tq/hoge",
						},
					},
					DispatchCount: 0,
					ResponseCount: 0,
				}
				expectedResponses = append(expectedResponses, expectedResponse)
			}

			var formattedParent string = fmt.Sprintf("projects/%s/locations/%s/queues/%s", "[PROJECT]", "[LOCATION]", "[QUEUE]")
			for i := 0; i < tt.callCount; i++ {
				var task *taskspb.Task = &taskspb.Task{
					Name: expectedResponses[i].GetName(),
					MessageType: &taskspb.Task_AppEngineHttpRequest{
						AppEngineHttpRequest: &taskspb.AppEngineHttpRequest{
							HttpMethod:  taskspb.HttpMethod_GET,
							RelativeUri: "/tq/hoge",
						},
					},
				}
				var request = &taskspb.CreateTaskRequest{
					Parent: formattedParent,
					Task:   task,
				}

				resp, err := c.CreateTask(context.Background(), request)
				if err != nil {
					t.Fatal(err)
				}

				req, err := faker.GetCreateTaskRequest(i)
				if err != nil {
					t.Fatal(err)
				}
				if e, g := request, req; !proto.Equal(e, g) {
					t.Errorf("request want %q, but got %q", e, g)
				}

				if e, g := expectedResponses[i], resp; !proto.Equal(e, g) {
					t.Errorf("response want %q, but got %q)", e, g)
				}
			}

			if e, g := tt.callCount, faker.GetCreateTaskCallCount(); e != g {
				t.Errorf("createTaskCallCount want %v but got %v", e, g)
			}
			_, err = faker.GetCreateTaskRequest(tt.callCount - 1)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestFaker_AddMockResponseWithTaskName(t *testing.T) {
	cases := []struct {
		name      string
		callCount int
	}{
		{"51", 51},
		{"99", 99},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			faker := tasksfaker.NewFaker(t)
			defer faker.Stop()

			c, err := cloudtasks.NewClient(context.Background(), faker.ClientOpt)
			if err != nil {
				t.Fatal(err)
			}

			var parent string = fmt.Sprintf("projects/%s/locations/%s/queues/%s", "[PROJECT]", "[LOCATION]", "[QUEUE]")
			const num = 10
			mockTaskName := fmt.Sprintf("%s/tasks/tn%d", parent, num)
			var dispatchCount int32 = 99
			var responseCount int32 = 99
			var expectedResponse = &taskspb.Task{
				Name:          mockTaskName,
				DispatchCount: dispatchCount,
				ResponseCount: responseCount,
			}
			faker.AddMockResponseWithTaskName(mockTaskName, nil, expectedResponse)

			for i := 0; i < tt.callCount; i++ {
				task := &taskspb.Task{
					Name: fmt.Sprintf("%s/tasks/tn%d", parent, i),
					MessageType: &taskspb.Task_AppEngineHttpRequest{
						AppEngineHttpRequest: &taskspb.AppEngineHttpRequest{
							HttpMethod:  taskspb.HttpMethod_GET,
							RelativeUri: "/tq/hoge",
						},
					},
				}
				var request = &taskspb.CreateTaskRequest{
					Parent: parent,
					Task:   task,
				}

				resp, err := c.CreateTask(context.Background(), request)
				if err != nil {
					t.Fatal(err)
				}
				if mockTaskName == task.GetName() {
					if e, g := dispatchCount, resp.GetDispatchCount(); e != g {
						t.Errorf("want MockResponse.dispatchCount %d but got %d", e, g)
					}
				}
			}

			if e, g := tt.callCount, faker.GetCreateTaskCallCount(); e != g {
				t.Errorf("want createTaskRequests.len %d but got %d", e, g)
			}
		})
	}
}

func TestCreateTask_defaultResponse_HeavyLoop(t *testing.T) {
	cases := []struct {
		name      string
		callCount int
	}{
		{"one", 1},
		{"two", 2},
		{"1000", 1000},
		{"1100", 1100},
		{"1200", 1200},
		{"1300", 1300},
		{"1400", 1400},
		{"1500", 1500},
		{"1600", 1600},
		{"1700", 1700},
		{"1800", 1800},
		{"1900", 1900},
	}

	parent := fmt.Sprintf("projects/%s/locations/%s/queues/%s", "[PROJECT]", "[LOCATION]", "[QUEUE]")
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			faker := tasksfaker.NewFaker(t)
			defer faker.Stop()

			c, err := cloudtasks.NewClient(context.Background(), faker.ClientOpt)
			if err != nil {
				t.Fatal(err)
			}

			var expectedResponses []*taskspb.Task
			for i := 0; i < tt.callCount; i++ {
				var expectedResponse = &taskspb.Task{
					Name: fmt.Sprintf("%s/tasks/name%d", parent, rand.Int()),
					MessageType: &taskspb.Task_AppEngineHttpRequest{
						AppEngineHttpRequest: &taskspb.AppEngineHttpRequest{
							HttpMethod:  taskspb.HttpMethod_GET,
							RelativeUri: "/tq/hoge",
						},
					},
					DispatchCount: 0,
					ResponseCount: 0,
				}
				expectedResponses = append(expectedResponses, expectedResponse)
			}

			var formattedParent string = fmt.Sprintf("projects/%s/locations/%s/queues/%s", "[PROJECT]", "[LOCATION]", "[QUEUE]")
			for i := 0; i < tt.callCount; i++ {
				var task *taskspb.Task = &taskspb.Task{
					Name: expectedResponses[i].GetName(),
					MessageType: &taskspb.Task_AppEngineHttpRequest{
						AppEngineHttpRequest: &taskspb.AppEngineHttpRequest{
							HttpMethod:  taskspb.HttpMethod_GET,
							RelativeUri: "/tq/hoge",
						},
					},
				}
				var request = &taskspb.CreateTaskRequest{
					Parent: formattedParent,
					Task:   task,
				}

				resp, err := c.CreateTask(context.Background(), request)
				if err != nil {
					t.Fatal(err)
				}

				req, err := faker.GetCreateTaskRequest(i)
				if err != nil {
					t.Fatal(err)
				}
				if e, g := request, req; !proto.Equal(e, g) {
					t.Errorf("request want %q, but got %q", e, g)
				}

				if e, g := expectedResponses[i], resp; !proto.Equal(e, g) {
					t.Errorf("response want %q, but got %q)", e, g)
				}
			}

			if e, g := tt.callCount, faker.GetCreateTaskCallCount(); e != g {
				t.Errorf("createTaskCallCount want %v but got %v", e, g)
			}
			_, err = faker.GetCreateTaskRequest(tt.callCount - 1)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
