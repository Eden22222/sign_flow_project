package util

import (
	"time"

	"sign_flow_project/internal/model"
)

// FormatWorkflowCreatedAt 将 workflow 创建时间格式化为 API 展示用字符串；nil 返回空串。
func FormatWorkflowCreatedAt(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

// FormatWorkflowDueDate 将 workflow 截止时间格式化为分钟精度字符串；nil 返回空串。
func FormatWorkflowDueDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02T15:04")
}

// BuildSignerStepStatus 根据流程状态与当前步，生成签署人步骤在详情中的展示状态。
func BuildSignerStepStatus(wf *model.WorkflowModel, stepIndex int) string {
	switch wf.Status {
	case model.WorkflowStatusCompleted:
		return "completed"
	case model.WorkflowStatusDraft:
		return "waiting"
	case model.WorkflowStatusCancelled:
		if stepIndex < wf.CurrentStep {
			return "completed"
		}
		return "waiting"
	case model.WorkflowStatusPending:
		if stepIndex < wf.CurrentStep {
			return "completed"
		}
		if stepIndex == wf.CurrentStep {
			return "current"
		}
		return "waiting"
	default:
		return "waiting"
	}
}
