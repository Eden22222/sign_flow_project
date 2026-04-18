package service

import (
	"fmt"
	"sort"
	"time"

	"sign_flow_project/internal/dao"
	"sign_flow_project/internal/model"
	"sign_flow_project/pkg/util"

	log "github.com/sirupsen/logrus"
)

type MyTasksSummary struct {
	PendingCount   int `json:"pendingCount"`
	CompletedCount int `json:"completedCount"`
	OverdueCount   int `json:"overdueCount"`
}

type MyTaskItem struct {
	TaskID         uint   `json:"taskId"`
	WorkflowID     uint   `json:"workflowId"`
	DocumentID     uint   `json:"documentId"`
	Title          string `json:"title"`
	FileName       string `json:"fileName"`
	Priority       string `json:"priority"`
	DueDate        string `json:"dueDate"`
	WorkflowStatus string `json:"workflowStatus"`
	DocumentStatus string `json:"documentStatus"`
	StepIndex      int    `json:"stepIndex"`
	TotalSteps     int    `json:"totalSteps"`
	IsOverdue      bool   `json:"isOverdue"`
}

type MyTasksResult struct {
	Summary   MyTasksSummary `json:"summary"`
	Pending   []MyTaskItem   `json:"pending"`
	Completed []MyTaskItem   `json:"completed"`
}

func (s *workflowQueryServiceImpl) GetMyTasks(userID uint) (*MyTasksResult, error) {
	if userID == 0 {
		return nil, fmt.Errorf("userId is required")
	}

	tasks, err := dao.TaskDao.SelectBySignerID(userID)
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return &MyTasksResult{
			Summary:   MyTasksSummary{},
			Pending:   []MyTaskItem{},
			Completed: []MyTaskItem{},
		}, nil
	}

	workflowIDs := make([]uint, 0, len(tasks))
	seenWorkflow := make(map[uint]struct{}, len(tasks))
	for _, task := range tasks {
		if task.WorkflowID == 0 {
			continue
		}
		if _, ok := seenWorkflow[task.WorkflowID]; ok {
			continue
		}
		seenWorkflow[task.WorkflowID] = struct{}{}
		workflowIDs = append(workflowIDs, task.WorkflowID)
	}

	workflows, err := dao.WorkflowDao.SelectByIDs(workflowIDs)
	if err != nil {
		return nil, err
	}
	workflowMap := make(map[uint]model.WorkflowModel, len(workflows))
	documentIDs := make([]uint, 0, len(workflows))
	seenDocument := make(map[uint]struct{}, len(workflows))
	for _, wf := range workflows {
		workflowMap[wf.ID] = wf
		if wf.DocumentID == 0 {
			continue
		}
		if _, ok := seenDocument[wf.DocumentID]; ok {
			continue
		}
		seenDocument[wf.DocumentID] = struct{}{}
		documentIDs = append(documentIDs, wf.DocumentID)
	}

	documents, err := dao.DocumentDao.SelectByIDs(documentIDs)
	if err != nil {
		return nil, err
	}
	documentMap := make(map[uint]model.DocumentModel, len(documents))
	for _, document := range documents {
		documentMap[document.ID] = document
	}

	signerCountMap, err := dao.WorkflowSignerDao.CountSignersByWorkflowIDs(workflowIDs)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	result := &MyTasksResult{
		Summary:   MyTasksSummary{},
		Pending:   make([]MyTaskItem, 0),
		Completed: make([]MyTaskItem, 0),
	}

	for _, task := range tasks {
		workflow, ok := workflowMap[task.WorkflowID]
		if !ok {
			log.Warnf("my-tasks skipped task because workflow not found: taskId=%d workflowId=%d", task.ID, task.WorkflowID)
			continue
		}
		document, ok := documentMap[workflow.DocumentID]
		if !ok {
			log.Warnf("my-tasks skipped task because document not found: taskId=%d workflowId=%d documentId=%d", task.ID, workflow.ID, workflow.DocumentID)
			continue
		}

		isOverdue := task.Status == model.TaskStatusPending &&
			workflow.DueAt != nil &&
			workflow.DueAt.Before(now)

		item := MyTaskItem{
			TaskID:         task.ID,
			WorkflowID:     workflow.ID,
			DocumentID:     document.ID,
			Title:          document.Title,
			FileName:       document.FileName,
			Priority:       string(workflow.Priority),
			DueDate:        util.FormatWorkflowDueDate(workflow.DueAt),
			WorkflowStatus: string(workflow.Status),
			DocumentStatus: document.Status,
			StepIndex:      task.StepIndex,
			TotalSteps:     signerCountMap[workflow.ID],
			IsOverdue:      isOverdue,
		}

		if task.Status == model.TaskStatusPending {
			result.Pending = append(result.Pending, item)
			result.Summary.PendingCount++
			if isOverdue {
				result.Summary.OverdueCount++
			}
			continue
		}

		if task.Status == model.TaskStatusSigned {
			result.Completed = append(result.Completed, item)
			result.Summary.CompletedCount++
		}
	}

	sort.SliceStable(result.Pending, func(i, j int) bool {
		a := result.Pending[i]
		b := result.Pending[j]
		if a.IsOverdue != b.IsOverdue {
			return a.IsOverdue
		}
		wa := workflowMap[a.WorkflowID]
		wb := workflowMap[b.WorkflowID]
		if c := cmpMyTaskDueAt(a.IsOverdue, wa.DueAt, wb.DueAt); c != 0 {
			return c < 0
		}
		pa := myTaskPriorityRank(a.Priority)
		pb := myTaskPriorityRank(b.Priority)
		if pa != pb {
			return pa < pb
		}
		return a.TaskID < b.TaskID
	})

	return result, nil
}

// cmpMyTaskDueAt 比较两条 pending 的截止时间先后（配合 IsOverdue 分组）。
// 已逾期：DueAt 更晚（离当前时刻更近的那次截止）排前；无截止时间排最后。
// 未逾期：DueAt 更早（即将到期）排前；无截止时间排最后。
func cmpMyTaskDueAt(overdue bool, da, db *time.Time) int {
	if overdue {
		switch {
		case da == nil && db == nil:
			return 0
		case da == nil:
			return 1
		case db == nil:
			return -1
		default:
			if da.After(*db) {
				return -1
			}
			if da.Before(*db) {
				return 1
			}
			return 0
		}
	}
	switch {
	case da == nil && db == nil:
		return 0
	case da == nil:
		return 1
	case db == nil:
		return -1
	default:
		if da.Before(*db) {
			return -1
		}
		if da.After(*db) {
			return 1
		}
		return 0
	}
}

func myTaskPriorityRank(p string) int {
	switch model.WorkflowPriority(p) {
	case model.WorkflowPriorityUrgent:
		return 0
	case model.WorkflowPriorityHigh:
		return 1
	case model.WorkflowPriorityNormal:
		return 2
	default:
		return 3
	}
}
