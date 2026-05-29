package service

import (
	"strings"
	"testing"
)

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
	if len(result.Preview) != 1 || result.Preview[0].Status != "invalid" || result.FailureCSV == "" {
		t.Fatalf("expected invalid preview and failure csv, got %+v", result)
	}
}

func TestParseCoursesImportRejectsInvalidTimes(t *testing.T) {
	data := []byte("课程名称,班级,星期,开始时间,结束时间\n非法时间,一班,三,10:15,09:30\n格式错误,一班,三,九点,10:15\n")
	courses, result := ParseCoursesImport("courses.csv", data)
	if len(courses) != 0 {
		t.Fatalf("expected no courses, got %+v", courses)
	}
	if result.Skipped != 2 || len(result.Preview) != 2 || result.FailureCSV == "" {
		t.Fatalf("expected invalid time previews and failure csv, got %+v", result)
	}
	for _, item := range result.Preview {
		if item.Status != "invalid" {
			t.Fatalf("expected invalid preview, got %+v", result.Preview)
		}
	}
}

func TestParseCoursesImportRejectsInvalidWeekdayAndLongFields(t *testing.T) {
	longTitle := strings.Repeat("课", 81)
	data := []byte("课程名称,班级,星期,开始时间,结束时间\n" +
		"非法星期,一班,周八,09:00,09:45\n" +
		longTitle + ",一班,周三,09:00,09:45\n")
	courses, result := ParseCoursesImport("courses.csv", data)
	if len(courses) != 0 {
		t.Fatalf("expected no courses, got %+v", courses)
	}
	if result.Skipped != 2 || len(result.Preview) != 2 || result.FailureCSV == "" {
		t.Fatalf("expected invalid import result, got %+v", result)
	}
	if result.Preview[0].Row != 2 || result.Preview[1].Row != 3 {
		t.Fatalf("expected original row numbers, got %+v", result.Preview)
	}
}

func TestParseParentsImportRejectsInvalidRiskAndLongFields(t *testing.T) {
	longContact := strings.Repeat("1", 81)
	data := []byte("学生姓名,班级,家长姓名,关系,联系方式,风险等级\n" +
		"林晓晓,一班,林妈妈,母亲,13800000000,紧急\n" +
		"陈小安,一班,陈爸爸,父亲," + longContact + ",low\n")
	parents, result := ParseParentsImport("parents.csv", data)
	if len(parents) != 0 {
		t.Fatalf("expected no parents, got %+v", parents)
	}
	if result.Skipped != 2 || len(result.Preview) != 2 || result.FailureCSV == "" {
		t.Fatalf("expected invalid import result, got %+v", result)
	}
	if !strings.Contains(result.Errors[0], "风险等级") || !strings.Contains(result.Errors[1], "联系方式") {
		t.Fatalf("expected risk and length errors, got %+v", result.Errors)
	}
}

func TestImportResultMarkDuplicateUpdatesPreview(t *testing.T) {
	result := ImportResult{Preview: []ImportPreview{{Row: 2, Status: "ready", Message: "可导入"}}}
	result.MarkDuplicate(2, "重复")
	if result.Skipped != 1 || len(result.Errors) != 1 || result.Preview[0].Status != "duplicate" {
		t.Fatalf("unexpected duplicate result: %+v", result)
	}
}

func TestNormalizeClockSupportsExcelSerialTime(t *testing.T) {
	if got := normalizeClock("0.3958333333"); got != "09:30" {
		t.Fatalf("expected 09:30, got %s", got)
	}
}

func TestImportTemplateCSVIncludesBOMAndHeaders(t *testing.T) {
	template := CourseImportTemplateCSV()
	if len(template) < 3 || template[0] != 0xef || !strings.Contains(string(template), "课程名称") {
		t.Fatalf("unexpected course template: %q", string(template))
	}
}
