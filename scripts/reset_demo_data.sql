-- OpenTab demo data reset script.
-- Run this in Navicat against the opentab database when demo data was polluted by testing.
-- It removes non-seed users, custom tabs, test business data, and rewrites readable demo records.

BEGIN;

CREATE TEMP TABLE seed_user_ids(id text PRIMARY KEY) ON COMMIT DROP;
INSERT INTO seed_user_ids(id) VALUES
  ('user-demo'),
  ('user-admin'),
  ('user-guest'),
  ('user-product-manager'),
  ('user-product-employee'),
  ('user-operation-manager'),
  ('user-operation-employee');

CREATE TEMP TABLE seed_team_ids(id text PRIMARY KEY) ON COMMIT DROP;
INSERT INTO seed_team_ids(id) VALUES
  ('team-product'),
  ('team-operation');

CREATE TEMP TABLE seed_approval_ids(id text PRIMARY KEY) ON COMMIT DROP;
INSERT INTO seed_approval_ids(id) VALUES
  ('apv-product-001'),
  ('apv-product-002'),
  ('apv-operation-001');

CREATE TEMP TABLE seed_event_ids(id text PRIMARY KEY) ON COMMIT DROP;
INSERT INTO seed_event_ids(id) VALUES
  ('evt-product-001'),
  ('evt-product-002'),
  ('evt-operation-001'),
  ('evt-operation-002'),
  ('evt-company-001');

CREATE TEMP TABLE seed_announcement_ids(id text PRIMARY KEY) ON COMMIT DROP;
INSERT INTO seed_announcement_ids(id) VALUES
  ('ann-company-001'),
  ('ann-product-001'),
  ('ann-operation-001');

DELETE FROM oncall_messages
WHERE session_id IN (
  SELECT id FROM oncall_sessions WHERE user_id NOT IN (SELECT id FROM seed_user_ids)
);
DELETE FROM audit_logs;
DELETE FROM oncall_sessions WHERE user_id NOT IN (SELECT id FROM seed_user_ids);
DELETE FROM auth_sessions WHERE user_id NOT IN (SELECT id FROM seed_user_ids);
DELETE FROM user_permissions WHERE user_id NOT IN (SELECT id FROM seed_user_ids);
DELETE FROM user_tabs WHERE user_id NOT IN (SELECT id FROM seed_user_ids);
DELETE FROM approval_items WHERE id NOT IN (SELECT id FROM seed_approval_ids);
DELETE FROM calendar_events WHERE id NOT IN (SELECT id FROM seed_event_ids);
DELETE FROM announcements WHERE id NOT IN (SELECT id FROM seed_announcement_ids);
DELETE FROM team_members WHERE user_id NOT IN (SELECT id FROM seed_user_ids)
  OR team_id NOT IN (SELECT id FROM seed_team_ids);
DELETE FROM tab_visibility_targets
WHERE tab_id IN (
  SELECT id FROM tabs WHERE is_system = false OR owner_user_id IS NOT NULL
);
DELETE FROM tabs WHERE is_system = false OR owner_user_id IS NOT NULL;
DELETE FROM teams WHERE id NOT IN (SELECT id FROM seed_team_ids);
DELETE FROM users WHERE id NOT IN (SELECT id FROM seed_user_ids);

UPDATE users SET
  account = 'opentab-demo',
  display_name = '李娜',
  global_role = '',
  enabled = true,
  updated_at = NOW()
WHERE id = 'user-demo';

UPDATE users SET
  account = 'admin',
  display_name = '张伟',
  global_role = 'admin',
  enabled = true,
  updated_at = NOW()
WHERE id = 'user-admin';

UPDATE users SET
  account = 'opentab-guest',
  display_name = '王芳',
  global_role = '',
  enabled = true,
  updated_at = NOW()
WHERE id = 'user-guest';

UPDATE users SET
  account = 'product-manager',
  display_name = '刘洋',
  global_role = '',
  enabled = true,
  updated_at = NOW()
WHERE id = 'user-product-manager';

UPDATE users SET
  account = 'product-employee',
  display_name = '陈磊',
  global_role = '',
  enabled = true,
  updated_at = NOW()
WHERE id = 'user-product-employee';

UPDATE users SET
  account = 'operation-manager',
  display_name = '张敏',
  global_role = '',
  enabled = true,
  updated_at = NOW()
WHERE id = 'user-operation-manager';

UPDATE users SET
  account = 'operation-employee',
  display_name = '李静',
  global_role = '',
  enabled = true,
  updated_at = NOW()
WHERE id = 'user-operation-employee';

INSERT INTO teams(id, name, description, enabled, created_at, updated_at) VALUES
  ('team-product', '产品研发部', '负责产品、客户端和服务端联调', true, NOW(), NOW()),
  ('team-operation', '运营支持部', '负责运营支持和客户协同', true, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name,
  description = EXCLUDED.description,
  enabled = true,
  updated_at = NOW();

INSERT INTO team_members(team_id, user_id, team_role, enabled, joined_at, created_at, updated_at) VALUES
  ('team-product', 'user-product-manager', 'manager', true, '2026-06-01T09:00:00+08:00', NOW(), NOW()),
  ('team-product', 'user-product-employee', 'employee', true, '2026-06-01T09:00:00+08:00', NOW(), NOW()),
  ('team-operation', 'user-operation-manager', 'manager', true, '2026-06-01T09:00:00+08:00', NOW(), NOW()),
  ('team-operation', 'user-operation-employee', 'employee', true, '2026-06-01T09:00:00+08:00', NOW(), NOW())
ON CONFLICT (team_id, user_id) DO UPDATE SET
  team_role = EXCLUDED.team_role,
  enabled = true,
  joined_at = EXCLUDED.joined_at,
  updated_at = NOW();

INSERT INTO approval_items(
  id, user_id, team_id, type, title, applicant_id, applicant, approver_id, approver,
  amount, reason, summary, form_json, status, comment, created_at, updated_at
) VALUES
  (
    'apv-product-001', 'user-product-employee', 'team-product', 'leave', '周五下午请假',
    'user-product-employee', '陈磊', 'user-product-manager', '刘洋', 0,
    '周五下午处理个人事务，上午完成接口联调记录交接',
    '请假 0.5 天，已补充交接安排',
    '{"leaveType":"事假","days":0.5,"handover":"Tab 接入联调记录已同步到项目群"}'::jsonb,
    'pending', '', '2026-06-03T09:20:00+08:00', '2026-06-03T09:20:00+08:00'
  ),
  (
    'apv-operation-001', 'user-operation-employee', 'team-operation', 'expense', '客户走访物料报销',
    'user-operation-employee', '李静', 'user-operation-manager', '张敏', 320,
    '客户走访使用的资料打印和贴纸物料',
    '报销 320 元，客户走访物料',
    '{"amount":320,"category":"客户走访","invoice":"已上传电子发票"}'::jsonb,
    'pending', '', '2026-06-03T10:05:00+08:00', '2026-06-03T10:05:00+08:00'
  ),
  (
    'apv-product-002', 'user-product-employee', 'team-product', 'purchase', '测试设备采购申请',
    'user-product-employee', '陈磊', 'user-product-manager', '刘洋', 1299,
    '用于 Android 端真机兼容性测试',
    '采购一台测试机，预算 1299 元',
    '{"amount":1299,"category":"测试设备","assetRequired":true}'::jsonb,
    'approved', '同意采购，注意登记资产编号', '2026-06-02T15:40:00+08:00', '2026-06-02T16:10:00+08:00'
  )
ON CONFLICT (id) DO UPDATE SET
  user_id = EXCLUDED.user_id,
  team_id = EXCLUDED.team_id,
  type = EXCLUDED.type,
  title = EXCLUDED.title,
  applicant_id = EXCLUDED.applicant_id,
  applicant = EXCLUDED.applicant,
  approver_id = EXCLUDED.approver_id,
  approver = EXCLUDED.approver,
  amount = EXCLUDED.amount,
  reason = EXCLUDED.reason,
  summary = EXCLUDED.summary,
  form_json = EXCLUDED.form_json,
  status = EXCLUDED.status,
  comment = EXCLUDED.comment,
  created_at = EXCLUDED.created_at,
  updated_at = EXCLUDED.updated_at;

INSERT INTO calendar_events(
  id, user_id, team_id, visibility, creator_id, creator_name, title, description,
  start_time, end_time, location, participants_json, participant_ids_json, created_at, updated_at
) VALUES
  (
    'evt-product-001', 'user-product-manager', 'team-product', 'team', 'user-product-manager', '刘洋',
    '产品研发部晨会', '确认 Tab 注册、权限和 AI OnCall 联调进展',
    '2026-06-03T09:30:00+08:00', '2026-06-03T10:00:00+08:00', '线上会议',
    '["刘洋","陈磊"]'::jsonb, '["user-product-manager","user-product-employee"]'::jsonb, NOW(), NOW()
  ),
  (
    'evt-operation-001', 'user-operation-manager', 'team-operation', 'team', 'user-operation-manager', '张敏',
    '客户反馈整理', '汇总近期客户对工作台 Tab 的反馈',
    '2026-06-03T10:30:00+08:00', '2026-06-03T11:00:00+08:00', '会议室 A',
    '["张敏","李静"]'::jsonb, '["user-operation-manager","user-operation-employee"]'::jsonb, NOW(), NOW()
  ),
  (
    'evt-product-002', 'user-product-manager', 'team-product', 'team', 'user-product-manager', '刘洋',
    'Tab 容器联调复盘', '检查客户端 Tab 列表、审批和日程数据展示',
    '2026-06-03T14:00:00+08:00', '2026-06-03T15:00:00+08:00', '开发群语音',
    '["刘洋","陈磊"]'::jsonb, '["user-product-manager","user-product-employee"]'::jsonb, NOW(), NOW()
  ),
  (
    'evt-operation-002', 'user-operation-manager', 'team-operation', 'team', 'user-operation-manager', '张敏',
    '公告发布确认', '确认阶段演示公告内容和发布范围',
    '2026-06-03T16:00:00+08:00', '2026-06-03T16:40:00+08:00', '会议室 B',
    '["张敏","李静"]'::jsonb, '["user-operation-manager","user-operation-employee"]'::jsonb, NOW(), NOW()
  ),
  (
    'evt-company-001', 'user-admin', '', 'company', 'user-admin', '张伟',
    '阶段演示彩排', '开放式 Tab 容器与 AI OnCall 助理阶段演示',
    '2026-06-04T15:30:00+08:00', '2026-06-04T16:30:00+08:00', '线上会议',
    '["全员"]'::jsonb, '[]'::jsonb, NOW(), NOW()
  )
ON CONFLICT (id) DO UPDATE SET
  user_id = EXCLUDED.user_id,
  team_id = EXCLUDED.team_id,
  visibility = EXCLUDED.visibility,
  creator_id = EXCLUDED.creator_id,
  creator_name = EXCLUDED.creator_name,
  title = EXCLUDED.title,
  description = EXCLUDED.description,
  start_time = EXCLUDED.start_time,
  end_time = EXCLUDED.end_time,
  location = EXCLUDED.location,
  participants_json = EXCLUDED.participants_json,
  participant_ids_json = EXCLUDED.participant_ids_json,
  updated_at = NOW();

INSERT INTO announcements(
  id, team_id, scope, title, content, publisher_id, publisher_name, pinned, deleted_at, created_at, updated_at
) VALUES
  (
    'ann-company-001', '', 'company', '阶段演示安排',
    '本周四 15:30 进行开放式 Tab 容器与 AI OnCall 助理阶段演示，请相关成员提前完成数据检查。',
    'user-admin', '张伟', true, NULL, NOW(), NOW()
  ),
  (
    'ann-product-001', 'team-product', 'team', '产品研发部联调提醒',
    '请在今天 14:00 前确认 Tab 列表、审批中心和日程接口在客户端展示正常。',
    'user-product-manager', '刘洋', false, NULL, NOW(), NOW()
  ),
  (
    'ann-operation-001', 'team-operation', 'team', '客户反馈整理',
    '请在周三下班前整理客户反馈和常见问题，重点标注和工作台 Tab 相关的需求。',
    'user-operation-manager', '张敏', false, NULL, NOW(), NOW()
  )
ON CONFLICT (id) DO UPDATE SET
  team_id = EXCLUDED.team_id,
  scope = EXCLUDED.scope,
  title = EXCLUDED.title,
  content = EXCLUDED.content,
  publisher_id = EXCLUDED.publisher_id,
  publisher_name = EXCLUDED.publisher_name,
  pinned = EXCLUDED.pinned,
  deleted_at = NULL,
  updated_at = NOW();

COMMIT;
