package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	"proteggo_api/types"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	taskspb "cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"cloud.google.com/go/logging"
)

// TODO: Configure the queue using glocud or console or create a code to do it programatically

// createTask creates a new task in your App Engine queue.
func CreateTask(context context.Context, client *cloudtasks.Client, logger *logging.Logger, upload *types.UploadImageToStorageModel) (*taskspb.Task, error) {
	// Build the Task queue path.
	queuePath := fmt.Sprintf("projects/%s/locations/%s/queues/%s", types.FIREBASE_PROJECT_ID, types.FIREBASE_LOCATION_ID, types.CLOUD_IMAGES_QUEUE_ID)

	// Serialize the UploadImageToStorageModel instance to JSON
	payload, err := json.Marshal(upload)
	if err != nil {
		logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload:  "Error serializing UploadImageToStorageModel",
			Labels:   map[string]string{"error": err.Error()},
		})
		return nil, err
	}

	// Build the Task payload.
	// https://godoc.org/google.golang.org/genproto/googleapis/cloud/tasks/v2#CreateTaskRequest
	req := &taskspb.CreateTaskRequest{
		Parent: queuePath,
		Task: &taskspb.Task{
			MessageType: &taskspb.Task_HttpRequest{
				HttpRequest: &taskspb.HttpRequest{
					HttpMethod: taskspb.HttpMethod_POST,
					Url:        types.CLOUD_RUN_SERVICE_URL + types.CLOUD_TASKS_HANDLER_PATH,
				},
			},
		},
	}

	// Add a payload message if one is present.
	req.Task.GetHttpRequest().Body = payload

	createdTask, err := client.CreateTask(context, req)
	if err != nil {
		logger.Log(logging.Entry{
			Severity: logging.Error,
			Payload:  "Error creating image processing task",
			Labels:   map[string]string{"error": err.Error()},
		})
		return nil, err
	}

	return createdTask, nil
}
