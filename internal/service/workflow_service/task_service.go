package service

import (
	"errors"
	"fmt"

	"sign_flow_project/internal/dao"
	"sign_flow_project/internal/model"

	"gorm.io/gorm"
)

type WorkflowTaskItem struct {
	TaskID     uint             `json:"taskId"`
	WorkflowID uint             `json:"workflowId"`
	SignerID   string           `json:"signerId"`
	StepIndex  int              `json:"stepIndex"`
	Status     model.TaskStatus `json:"status"`
}

type WorkflowTaskListResult struct {
	WorkflowID uint               `json:"workflowId"`
	Tasks      []WorkflowTaskItem `json:"tasks"`
}

func (s *workflowQueryServiceImpl) GetTasks(workflowID uint) (*WorkflowTaskListResult, error) {
	if workflowID == 0 {
		return nil, fmt.Errorf("workflowId is required")
	}

	_, err := dao.WorkflowDao.SelectByID(workflowID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("workflow not found")
		}
		return nil, err
	}

	tasks, err := dao.TaskDao.SelectByWorkflowID(workflowID)
	if err != nil {
		return nil, err
	}

	items := make([]WorkflowTaskItem, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, WorkflowTaskItem{
			TaskID:     task.ID,
			WorkflowID: task.WorkflowID,
			SignerID:   task.SignerID,
			StepIndex:  task.StepIndex,
			Status:     task.Status,
		})
	}

	return &WorkflowTaskListResult{
		WorkflowID: workflowID,
		Tasks:      items,
	}, nil
}
