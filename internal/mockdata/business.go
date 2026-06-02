package mockdata

import "opentab-server/internal/models"

var ApprovalSummary = models.ApprovalSummary{
	PendingCount:  12,
	ApprovedToday: 8,
	Items: []models.ApprovalItem{
		{
			ID:        "apv-001",
			Title:     "采购申请",
			Applicant: "张三",
			Amount:    1200,
			Reason:    "项目采购设备",
			Status:    "pending",
			CreatedAt: "2026-05-31T12:00:00+08:00",
		},
		{
			ID:        "apv-002",
			Title:     "差旅报销",
			Applicant: "李四",
			Amount:    860,
			Reason:    "客户现场沟通差旅费用",
			Status:    "approved",
			CreatedAt: "2026-05-30T09:15:00+08:00",
		},
		{
			ID:        "apv-003",
			Title:     "请假申请",
			Applicant: "王五",
			Reason:    "个人事务",
			Status:    "rejected",
			CreatedAt: "2026-05-29T11:20:00+08:00",
			Comment:   "请补充交接安排",
		},
	},
}

var CalendarSummary = models.CalendarSummary{
	TodayCount: 3,
	Events: []models.CalendarEvent{
		{
			ID:           "evt-001",
			Title:        "项目周会",
			Description:  "同步本周服务端和客户端联调进展",
			StartTime:    "2026-05-31T14:00:00+08:00",
			EndTime:      "2026-05-31T15:00:00+08:00",
			Location:     "线上会议",
			Participants: []string{"张三", "李四"},
		},
		{
			ID:           "evt-002",
			Title:        "接口联调",
			Description:  "联调 TabManifest 和 AI OnCall",
			StartTime:    "2026-05-31T16:00:00+08:00",
			EndTime:      "2026-05-31T17:00:00+08:00",
			Location:     "开发群语音",
			Participants: []string{"王铮", "客户端同学"},
		},
	},
}

var Permissions = []map[string]string{
	{"code": "team.manage", "description": "管理团队、成员和团队角色"},
	{"code": "team.all.read", "description": "查看所有团队数据"},
	{"code": "team.member.read", "description": "查看自己团队成员"},
	{"code": "tab.company.read", "description": "查看公司介绍"},
	{"code": "tab.announcement.read", "description": "查看公告"},
	{"code": "tab.announcement.write", "description": "发布和编辑公告"},
	{"code": "tab.fun.read", "description": "查看放松一刻"},
	{"code": "tab.approval.read", "description": "查看审批中心"},
	{"code": "tab.approval.create", "description": "发起审批"},
	{"code": "tab.approval.approve", "description": "处理审批"},
	{"code": "tab.approval.all", "description": "查看全部审批"},
	{"code": "tab.calendar.read", "description": "查看团队日程"},
	{"code": "tab.calendar.create", "description": "创建日程"},
	{"code": "tab.calendar.manage", "description": "管理团队日程"},
	{"code": "tab.calendar.all", "description": "查看全部日程"},
	{"code": "tab.admin.manage", "description": "管理 Tab 和系统配置"},
	{"code": "tab.debug.read", "description": "查看调试信息"},
	{"code": "tab.finance.read", "description": "查看财务看板"},
	{"code": "ai.oncall", "description": "使用 AI OnCall"},
}
