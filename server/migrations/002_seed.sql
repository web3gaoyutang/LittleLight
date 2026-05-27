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
