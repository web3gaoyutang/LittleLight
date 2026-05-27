INSERT INTO users (id, name, school, stage, subject, is_head_teacher, pro_status)
VALUES ('00000000-0000-0000-0000-000000000001', '林小微', '微光实验小学', '小学', '语文', true, 'trial')
ON CONFLICT (id) DO NOTHING;

INSERT INTO parent_profiles (id, user_id, student_name, class_name, parent_name, relationship, communication_style, risk_level, important_notes, next_action)
VALUES
('10000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '林晓晓', '高二(5)班', '林晓晓妈妈', '母亲', '比较敏感', 'medium', '近期睡眠不足，到校后情绪波动。', '先确认睡眠和到校状态，再同步课堂参与中的积极信号。'),
('10000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', '陈子默', '高二(3)班', '陈子默爸爸', '父亲', '关注成绩', 'low', '关注测试反馈和订正计划。', '周五前同步订正安排。')
ON CONFLICT (id) DO NOTHING;

INSERT INTO courses (id, user_id, title, class_name, location, weekday, start_time, end_time, repeat_rule, note)
VALUES
('20000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', '心理健康', '高二(3)班', '教学楼 B 座 402 室', 3, '09:30', '10:15', 'weekly', '情绪识别与压力调节'),
('20000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', '个别谈话', '林晓晓', '咨询室', 3, '13:40', '14:00', 'weekly', '睡眠与到校状态')
ON CONFLICT (id) DO NOTHING;

INSERT INTO favorites (id, user_id, type, title, content)
VALUES
('30000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'communication_template', '先肯定再建议', '先同步孩子已经做到的部分，再给出一个可执行的小建议。'),
('30000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'ai_praise', '忙碌后恢复', '你今天处理了很多细碎但重要的事，先允许自己慢下来。')
ON CONFLICT (id) DO NOTHING;

INSERT INTO healing_entries (id, user_id, type, mood, content, ai_reply)
VALUES
('40000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'praise', 'warm', '今天处理了课程和家长反馈。', '你已经稳稳接住了很多复杂信息，先给自己一点恢复空间。'),
('40000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'breath', 'calm', '完成 1 分钟呼吸练习', '已完成一次短恢复。')
ON CONFLICT (id) DO NOTHING;

INSERT INTO ai_generations (id, user_id, scenario, input, output, safety_label, token_usage)
VALUES
(
  '50000000-0000-0000-0000-000000000001',
  '00000000-0000-0000-0000-000000000001',
  'praise',
  '{"persona":"温柔前辈","content":"今天处理了课程和家长反馈。"}'::jsonb,
  '{"draft":{"content":"你已经稳稳接住了很多复杂信息，先给自己一点恢复空间。"}}'::jsonb,
  'self_care',
  0
)
ON CONFLICT (id) DO NOTHING;
