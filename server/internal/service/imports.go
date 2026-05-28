package service

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/web3gaoyutang/littlelight/server/internal/domain"
)

const maxImportRows = 300

type ImportResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors"`
}

type xlsxCell struct {
	Reference string `xml:"r,attr"`
	Type      string `xml:"t,attr"`
	Value     string `xml:"v"`
	Inline    struct {
		Text string `xml:"t"`
	} `xml:"is"`
}

func ParseCoursesImport(filename string, data []byte) ([]domain.Course, ImportResult) {
	rows, result := parseTabularFile(filename, data)
	if len(rows) == 0 {
		result.Errors = append(result.Errors, "未读取到可导入的课表数据")
		return nil, result
	}
	header := normalizeHeader(rows[0])
	courses := make([]domain.Course, 0, len(rows)-1)
	for index, row := range rows[1:] {
		if index >= maxImportRows {
			result.Skipped += len(rows) - index - 1
			result.Errors = append(result.Errors, fmt.Sprintf("超过 %d 行的内容已跳过", maxImportRows))
			break
		}
		values := rowValues(header, row)
		if isBlankRow(values) {
			result.Skipped++
			continue
		}
		course := domain.Course{
			Title:     firstValue(values, "title", "course", "课程", "课程名称", "科目"),
			ClassName: firstValue(values, "class", "className", "班级", "班级名称"),
			Location:  firstValue(values, "location", "地点", "教室"),
			Weekday:   parseWeekday(firstValue(values, "weekday", "day", "星期", "周几")),
			StartTime: normalizeClock(firstValue(values, "start", "startTime", "开始", "开始时间")),
			EndTime:   normalizeClock(firstValue(values, "end", "endTime", "结束", "结束时间")),
			Note:      firstValue(values, "note", "备注", "说明"),
		}
		if course.Title == "" || course.ClassName == "" {
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("第 %d 行缺少课程名称或班级", index+2))
			continue
		}
		if course.StartTime == "" {
			course.StartTime = "08:00"
		}
		if course.EndTime == "" {
			course.EndTime = "08:45"
		}
		courses = append(courses, course)
	}
	return courses, result
}

func ParseParentsImport(filename string, data []byte) ([]domain.ParentProfile, ImportResult) {
	rows, result := parseTabularFile(filename, data)
	if len(rows) == 0 {
		result.Errors = append(result.Errors, "未读取到可导入的班级名单")
		return nil, result
	}
	header := normalizeHeader(rows[0])
	parents := make([]domain.ParentProfile, 0, len(rows)-1)
	for index, row := range rows[1:] {
		if index >= maxImportRows {
			result.Skipped += len(rows) - index - 1
			result.Errors = append(result.Errors, fmt.Sprintf("超过 %d 行的内容已跳过", maxImportRows))
			break
		}
		values := rowValues(header, row)
		if isBlankRow(values) {
			result.Skipped++
			continue
		}
		parent := domain.ParentProfile{
			StudentName:        firstValue(values, "student", "studentName", "学生", "学生姓名", "姓名"),
			ClassName:          firstValue(values, "class", "className", "班级", "班级名称"),
			ParentName:         firstValue(values, "parent", "parentName", "家长", "家长姓名"),
			Relationship:       firstValue(values, "relationship", "关系", "亲属关系"),
			Contact:            firstValue(values, "contact", "phone", "mobile", "联系方式", "手机号", "电话"),
			CommunicationStyle: firstValue(values, "style", "communicationStyle", "沟通风格", "家长风格"),
			RiskLevel:          normalizeRisk(firstValue(values, "risk", "riskLevel", "风险", "风险等级")),
			ImportantNotes:     firstValue(values, "notes", "importantNotes", "重点备注", "备注"),
			NextAction:         firstValue(values, "nextAction", "下一步", "跟进动作"),
		}
		if parent.StudentName == "" || parent.ClassName == "" {
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("第 %d 行缺少学生姓名或班级", index+2))
			continue
		}
		if parent.ParentName == "" {
			parent.ParentName = parent.StudentName + "家长"
		}
		if parent.Relationship == "" {
			parent.Relationship = "家长"
		}
		if parent.RiskLevel == "" {
			parent.RiskLevel = "low"
		}
		parents = append(parents, parent)
	}
	return parents, result
}

func parseTabularFile(filename string, data []byte) ([][]string, ImportResult) {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".csv":
		return parseCSV(data)
	case ".xlsx":
		return parseXLSX(data)
	default:
		return nil, ImportResult{Errors: []string{"仅支持 .xlsx 或 .csv 文件"}}
	}
}

func parseCSV(data []byte) ([][]string, ImportResult) {
	reader := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})))
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, ImportResult{Errors: []string{fmt.Sprintf("CSV 解析失败：%v", err)}}
	}
	return trimRows(rows), ImportResult{}
}

func parseXLSX(data []byte) ([][]string, ImportResult) {
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, ImportResult{Errors: []string{fmt.Sprintf("Excel 解析失败：%v", err)}}
	}
	files := map[string]*zip.File{}
	for _, file := range archive.File {
		files[file.Name] = file
	}
	shared, err := readSharedStrings(files["xl/sharedStrings.xml"])
	if err != nil {
		return nil, ImportResult{Errors: []string{err.Error()}}
	}
	sheet := firstExisting(files, "xl/worksheets/sheet1.xml")
	if sheet == nil {
		return nil, ImportResult{Errors: []string{"Excel 中未找到第一个工作表"}}
	}
	rows, err := readSheetRows(sheet, shared)
	if err != nil {
		return nil, ImportResult{Errors: []string{err.Error()}}
	}
	return trimRows(rows), ImportResult{}
}

func readSharedStrings(file *zip.File) ([]string, error) {
	if file == nil {
		return nil, nil
	}
	body, err := openZipFile(file)
	if err != nil {
		return nil, fmt.Errorf("读取共享字符串失败：%w", err)
	}
	type textNode struct {
		Text string `xml:",chardata"`
	}
	type richText struct {
		Text textNode `xml:"t"`
	}
	type sharedItem struct {
		Text     string     `xml:"t"`
		RichText []richText `xml:"r"`
	}
	var doc struct {
		Items []sharedItem `xml:"si"`
	}
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("共享字符串解析失败：%w", err)
	}
	items := make([]string, 0, len(doc.Items))
	for _, item := range doc.Items {
		if item.Text != "" {
			items = append(items, item.Text)
			continue
		}
		var builder strings.Builder
		for _, part := range item.RichText {
			builder.WriteString(part.Text.Text)
		}
		items = append(items, builder.String())
	}
	return items, nil
}

func readSheetRows(file *zip.File, shared []string) ([][]string, error) {
	body, err := openZipFile(file)
	if err != nil {
		return nil, fmt.Errorf("读取工作表失败：%w", err)
	}
	type row struct {
		Cells []xlsxCell `xml:"c"`
	}
	var doc struct {
		Rows []row `xml:"sheetData>row"`
	}
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("工作表解析失败：%w", err)
	}
	rows := make([][]string, 0, len(doc.Rows))
	for _, sheetRow := range doc.Rows {
		values := []string{}
		for _, item := range sheetRow.Cells {
			index := columnIndex(item.Reference)
			for len(values) <= index {
				values = append(values, "")
			}
			values[index] = cellValue(item, shared)
		}
		rows = append(rows, values)
	}
	return rows, nil
}

func cellValue(item xlsxCell, shared []string) string {
	switch item.Type {
	case "s":
		index, _ := strconv.Atoi(item.Value)
		if index >= 0 && index < len(shared) {
			return strings.TrimSpace(shared[index])
		}
	case "inlineStr":
		return strings.TrimSpace(item.Inline.Text)
	}
	return strings.TrimSpace(item.Value)
}

func columnIndex(reference string) int {
	index := 0
	seenLetter := false
	for _, char := range reference {
		if char >= 'A' && char <= 'Z' {
			seenLetter = true
			index = index*26 + int(char-'A'+1)
			continue
		}
		if char >= 'a' && char <= 'z' {
			seenLetter = true
			index = index*26 + int(char-'a'+1)
			continue
		}
		break
	}
	if !seenLetter {
		return 0
	}
	return index - 1
}

func openZipFile(file *zip.File) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func firstExisting(files map[string]*zip.File, names ...string) *zip.File {
	for _, name := range names {
		if file := files[name]; file != nil {
			return file
		}
	}
	return nil
}

func normalizeHeader(row []string) map[int]string {
	header := map[int]string{}
	for index, value := range row {
		header[index] = normalizeKey(value)
	}
	return header
}

func rowValues(header map[int]string, row []string) map[string]string {
	values := map[string]string{}
	for index, value := range row {
		key := header[index]
		if key == "" {
			continue
		}
		values[key] = strings.TrimSpace(value)
	}
	return values
}

func firstValue(values map[string]string, aliases ...string) string {
	for _, alias := range aliases {
		if value := values[normalizeKey(alias)]; value != "" {
			return value
		}
	}
	return ""
}

func isBlankRow(values map[string]string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func normalizeKey(value string) string {
	return strings.ToLower(strings.NewReplacer(" ", "", "_", "", "-", "", "：", "", ":", "").Replace(strings.TrimSpace(value)))
}

func trimRows(rows [][]string) [][]string {
	result := make([][]string, 0, len(rows))
	for _, row := range rows {
		empty := true
		for index := range row {
			row[index] = strings.TrimSpace(row[index])
			if row[index] != "" {
				empty = false
			}
		}
		if !empty {
			result = append(result, row)
		}
	}
	return result
}

func parseWeekday(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 1
	}
	labels := map[string]int{"日": 0, "天": 0, "一": 1, "1": 1, "二": 2, "2": 2, "三": 3, "3": 3, "四": 4, "4": 4, "五": 5, "5": 5, "六": 6, "6": 6}
	for label, weekday := range labels {
		if strings.Contains(value, label) {
			return weekday
		}
	}
	return 1
}

func normalizeClock(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if numeric, err := strconv.ParseFloat(value, 64); err == nil && numeric > 0 && numeric < 1 {
		totalMinutes := int(numeric*24*60 + 0.5)
		return fmt.Sprintf("%02d:%02d", totalMinutes/60, totalMinutes%60)
	}
	if strings.Contains(value, ":") && len(value) >= 4 {
		parts := strings.Split(value, ":")
		hour, _ := strconv.Atoi(parts[0])
		minute := 0
		if len(parts) > 1 {
			minute, _ = strconv.Atoi(parts[1])
		}
		return fmt.Sprintf("%02d:%02d", hour, minute)
	}
	return value
}

func normalizeRisk(value string) string {
	switch {
	case strings.Contains(value, "高") || strings.EqualFold(value, "high"):
		return "high"
	case strings.Contains(value, "中") || strings.EqualFold(value, "medium"):
		return "medium"
	case value == "":
		return ""
	default:
		return "low"
	}
}
