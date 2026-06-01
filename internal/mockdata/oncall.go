package mockdata

import "opentab-server/internal/models"

var OnCallSessions = map[string][]models.OnCallSession{
	"user-demo": {
		{
			SessionID:    "sess-001",
			Title:        "Tab 接入咨询",
			CreatedAt:    "2026-05-31T12:00:00+08:00",
			UpdatedAt:    "2026-05-31T12:10:00+08:00",
			MessageCount: 2,
		},
	},
}

var OnCallMessages = map[string][]models.OnCallMessage{
	"sess-001": {
		{
			MessageID:   "msg-001",
			SessionID:   "sess-001",
			Role:        "user",
			Content:     "如何接入一个审批 Tab？",
			ContentType: "text",
			CreatedAt:   "2026-05-31T12:01:00+08:00",
		},
		{
			MessageID:   "msg-002",
			SessionID:   "sess-001",
			Role:        "assistant",
			Content:     "接入审批 Tab 时，客户端需要先支持 native route。",
			ContentType: "text",
			CreatedAt:   "2026-05-31T12:01:05+08:00",
		},
	},
}
