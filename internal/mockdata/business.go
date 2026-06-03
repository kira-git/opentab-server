package mockdata

import "opentab-server/internal/models"

var ApprovalSummary = models.ApprovalSummary{
	PendingCount:  12,
	ApprovedToday: 8,
	Items: []models.ApprovalItem{
		{
			ID:        "apv-001",
			Title:     "测试设备采购申请",
			Applicant: "陈磊",
			Amount:    1299,
			Reason:    "用于 Android 端真机兼容性测试",
			Status:    "pending",
			CreatedAt: "2026-06-03T09:20:00+08:00",
		},
		{
			ID:        "apv-002",
			Title:     "客户走访物料报销",
			Applicant: "李静",
			Amount:    320,
			Reason:    "客户走访使用的资料打印和贴纸物料",
			Status:    "approved",
			CreatedAt: "2026-06-02T10:05:00+08:00",
		},
		{
			ID:        "apv-003",
			Title:     "周五下午请假",
			Applicant: "陈磊",
			Reason:    "周五下午处理个人事务，上午完成接口联调记录交接",
			Status:    "rejected",
			CreatedAt: "2026-06-01T11:20:00+08:00",
			Comment:   "请补充交接安排",
		},
	},
}

var CalendarSummary = models.CalendarSummary{
	TodayCount: 3,
	Events: []models.CalendarEvent{
		{
			ID:           "evt-001",
			Title:        "产品研发部晨会",
			Description:  "确认 Tab 注册、权限和 AI OnCall 联调进展",
			StartTime:    "2026-06-03T09:30:00+08:00",
			EndTime:      "2026-06-03T10:00:00+08:00",
			Location:     "线上会议",
			Participants: []string{"刘洋", "陈磊"},
		},
		{
			ID:           "evt-002",
			Title:        "Tab 容器联调复盘",
			Description:  "检查客户端 Tab 列表、审批和日程数据展示",
			StartTime:    "2026-06-03T14:00:00+08:00",
			EndTime:      "2026-06-03T15:00:00+08:00",
			Location:     "开发群语音",
			Participants: []string{"刘洋", "陈磊"},
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
