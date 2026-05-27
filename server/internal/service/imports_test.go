package service

import "testing"

func TestParseCoursesImportCSV(t *testing.T) {
	data := []byte("课程名称,班级,星期,开始时间,结束时间,地点,备注\n心理健康,高二(3)班,周三,09:30,10:15,402,情绪识别\n")
	courses, result := ParseCoursesImport("courses.csv", data)
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %+v", result.Errors)
	}
	if len(courses) != 1 {
		t.Fatalf("expected one course, got %d", len(courses))
	}
	if courses[0].Title != "心理健康" || courses[0].Weekday != 3 || courses[0].StartTime != "09:30" {
		t.Fatalf("unexpected course: %+v", courses[0])
	}
}

func TestParseParentsImportCSV(t *testing.T) {
	data := []byte("学生姓名,班级,家长姓名,关系,联系方式,家长风格,风险等级,重点备注,下一步\n林晓晓,高二(5)班,林晓晓妈妈,母亲,13800000000,比较敏感,中,近期睡眠不足,周五回访\n")
	parents, result := ParseParentsImport("parents.csv", data)
	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors: %+v", result.Errors)
	}
	if len(parents) != 1 {
		t.Fatalf("expected one parent, got %d", len(parents))
	}
	if parents[0].StudentName != "林晓晓" || parents[0].RiskLevel != "medium" || parents[0].Contact == "" {
		t.Fatalf("unexpected parent: %+v", parents[0])
	}
}

func TestParseParentsImportSkipsIncompleteRows(t *testing.T) {
	data := []byte("学生姓名,班级,家长姓名\n只写姓名,,家长\n")
	parents, result := ParseParentsImport("parents.csv", data)
	if len(parents) != 0 {
		t.Fatalf("expected no parents, got %d", len(parents))
	}
	if result.Skipped != 1 || len(result.Errors) == 0 {
		t.Fatalf("expected skipped row with error, got %+v", result)
	}
}

func TestNormalizeClockSupportsExcelSerialTime(t *testing.T) {
	if got := normalizeClock("0.3958333333"); got != "09:30" {
		t.Fatalf("expected 09:30, got %s", got)
	}
}
