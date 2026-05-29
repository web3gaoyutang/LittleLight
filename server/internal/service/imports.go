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
	"time"

	"github.com/web3gaoyutang/littlelight/server/internal/domain"
)

const maxImportRows = 300

type ImportResult struct {
	Imported   int             `json:"imported"`
	Skipped    int             `json:"skipped"`
	Errors     []string        `json:"errors"`
	Preview    []ImportPreview `json:"preview,omitempty"`
	FailureCSV string          `json:"failureCsv,omitempty"`
	ReadyRows  []int           `json:"-"`

	failureHeader []string
	failureRows   [][]string
	rawRows       map[int][]string
}

type ImportPreview struct {
	Row     int               `json:"row"`
	Status  string            `json:"status"`
	Message string            `json:"message"`
	Values  map[string]string `json:"values"`
}

func (r *ImportResult) MarkDuplicate(row int, message string) {
	if message == "" {
		message = fmt.Sprintf("第 %d 行重复，已跳过", row)
	}
	r.Skipped++
	r.Errors = append(r.Errors, message)
	r.addFailureRow(row, message)
	for index := range r.Preview {
		if r.Preview[index].Row == row {
			r.Preview[index].Status = "duplicate"
			r.Preview[index].Message = message
			return
		}
	}
	r.Preview = append(r.Preview, ImportPreview{Row: row, Status: "duplicate", Message: message, Values: map[string]string{}})
}

func (r *ImportResult) ReadyRow(index int) int {
	if index >= 0 && index < len(r.ReadyRows) && r.ReadyRows[index] > 0 {
		return r.ReadyRows[index]
	}
	return index + 2
}

func (r *ImportResult) setFailureHeader(row []string) {
	r.failureHeader = append([]string(nil), row...)
	r.rebuildFailureCSV()
}

func (r *ImportResult) markReady(rowNumber int, row []string, values map[string]string) {
	r.trackRawRow(rowNumber, row)
	r.ReadyRows = append(r.ReadyRows, rowNumber)
	r.Preview = append(r.Preview, importPreview(rowNumber, "ready", "可导入", values))
}

func (r *ImportResult) markInvalid(rowNumber int, row []string, values map[string]string, message string) {
	r.Skipped++
	r.Errors = append(r.Errors, message)
	r.trackRawRow(rowNumber, row)
	r.Preview = append(r.Preview, importPreview(rowNumber, "invalid", message, values))
	r.addFailureRow(rowNumber, message)
}

func (r *ImportResult) trackRawRow(rowNumber int, row []string) {
	if r.rawRows == nil {
		r.rawRows = map[int][]string{}
	}
	r.rawRows[rowNumber] = append([]string(nil), row...)
}

func (r *ImportResult) addFailureRow(rowNumber int, message string) {
	row, ok := r.rawRows[rowNumber]
	if !ok {
		return
	}
	r.failureRows = append(r.failureRows, appendFailureReason(row, message))
	r.rebuildFailureCSV()
}

func (r *ImportResult) rebuildFailureCSV() {
	r.FailureCSV = failureCSV(r.failureHeader, r.failureRows)
}

func CourseImportTemplateCSV() []byte {
	return csvTemplate([][]string{
		{"课程名称", "班级", "地点", "星期", "开始时间", "结束时间", "备注"},
		{"心理健康", "高二(3)班", "教学楼 B 座 402 室", "三", "09:30", "10:15", "情绪识别与压力调节"},
	})
}

func ParentImportTemplateCSV() []byte {
	return csvTemplate([][]string{
		{"学生姓名", "班级", "家长姓名", "关系", "联系方式", "沟通风格", "风险等级", "重点备注", "下一步"},
		{"林晓晓", "高二(5)班", "林晓晓妈妈", "母亲", "13800000000", "比较敏感", "medium", "睡眠与到校状态需要持续观察", "先确认睡眠，再同步课堂参与中的积极信号"},
	})
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
	headerRow := rows[0]
	result.setFailureHeader(headerRow)
	courses := make([]domain.Course, 0, len(rows)-1)
	for index, row := range rows[1:] {
		rowNumber := index + 2
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
		weekday, ok := parseWeekday(firstValue(values, "weekday", "day", "星期", "周几"))
		course := domain.Course{
			Title:     firstValue(values, "title", "course", "课程", "课程名称", "科目"),
			ClassName: firstValue(values, "class", "className", "班级", "班级名称"),
			Location:  firstValue(values, "location", "地点", "教室"),
			Weekday:   weekday,
			StartTime: normalizeClock(firstValue(values, "start", "startTime", "开始", "开始时间")),
			EndTime:   normalizeClock(firstValue(values, "end", "endTime", "结束", "结束时间")),
			Note:      firstValue(values, "note", "备注", "说明"),
		}
		if course.Title == "" || course.ClassName == "" {
			message := fmt.Sprintf("第 %d 行缺少课程名称或班级", rowNumber)
			result.markInvalid(rowNumber, row, values, message)
			continue
		}
		if !ok {
			message := fmt.Sprintf("第 %d 行星期必须是 0-6、周一到周日或星期一到星期日", rowNumber)
			result.markInvalid(rowNumber, row, values, message)
			continue
		}
		if course.StartTime == "" {
			course.StartTime = "08:00"
		}
		if course.EndTime == "" {
			course.EndTime = "08:45"
		}
		if !validImportClock(course.StartTime) || !validImportClock(course.EndTime) {
			message := fmt.Sprintf("第 %d 行开始或结束时间格式不正确", rowNumber)
			result.markInvalid(rowNumber, row, values, message)
			continue
		}
		if !importClockBefore(course.StartTime, course.EndTime) {
			message := fmt.Sprintf("第 %d 行结束时间必须晚于开始时间", rowNumber)
			result.markInvalid(rowNumber, row, values, message)
			continue
		}
		if message, ok := validateImportCourse(rowNumber, course); !ok {
			result.markInvalid(rowNumber, row, values, message)
			continue
		}
		courses = append(courses, course)
		result.markReady(rowNumber, row, values)
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
	headerRow := rows[0]
	result.setFailureHeader(headerRow)
	parents := make([]domain.ParentProfile, 0, len(rows)-1)
	for index, row := range rows[1:] {
		rowNumber := index + 2
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
		riskLevel, ok := normalizeRisk(firstValue(values, "risk", "riskLevel", "风险", "风险等级"))
		parent := domain.ParentProfile{
			StudentName:        firstValue(values, "student", "studentName", "学生", "学生姓名", "姓名"),
			ClassName:          firstValue(values, "class", "className", "班级", "班级名称"),
			ParentName:         firstValue(values, "parent", "parentName", "家长", "家长姓名"),
			Relationship:       firstValue(values, "relationship", "关系", "亲属关系"),
			Contact:            firstValue(values, "contact", "phone", "mobile", "联系方式", "手机号", "电话"),
			CommunicationStyle: firstValue(values, "style", "communicationStyle", "沟通风格", "家长风格"),
			RiskLevel:          riskLevel,
			ImportantNotes:     firstValue(values, "notes", "importantNotes", "重点备注", "备注"),
			NextAction:         firstValue(values, "nextAction", "下一步", "跟进动作"),
		}
		if parent.StudentName == "" || parent.ClassName == "" {
			message := fmt.Sprintf("第 %d 行缺少学生姓名或班级", rowNumber)
			result.markInvalid(rowNumber, row, values, message)
			continue
		}
		if !ok {
			message := fmt.Sprintf("第 %d 行风险等级必须是 low、medium、high 或低/中/高", rowNumber)
			result.markInvalid(rowNumber, row, values, message)
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
		if message, ok := validateImportParent(rowNumber, parent); !ok {
			result.markInvalid(rowNumber, row, values, message)
			continue
		}
		parents = append(parents, parent)
		result.markReady(rowNumber, row, values)
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

type importFieldLimit struct {
	Label string
	Value string
	Max   int
}

func validateImportCourse(rowNumber int, course domain.Course) (string, bool) {
	return validateImportFieldLengths(rowNumber, []importFieldLimit{
		{Label: "课程名称", Value: course.Title, Max: 80},
		{Label: "班级", Value: course.ClassName, Max: 80},
		{Label: "地点", Value: course.Location, Max: 120},
		{Label: "备注", Value: course.Note, Max: 1000},
	})
}

func validateImportParent(rowNumber int, parent domain.ParentProfile) (string, bool) {
	return validateImportFieldLengths(rowNumber, []importFieldLimit{
		{Label: "学生姓名", Value: parent.StudentName, Max: 60},
		{Label: "班级", Value: parent.ClassName, Max: 80},
		{Label: "家长姓名", Value: parent.ParentName, Max: 80},
		{Label: "关系", Value: parent.Relationship, Max: 40},
		{Label: "联系方式", Value: parent.Contact, Max: 80},
		{Label: "沟通风格", Value: parent.CommunicationStyle, Max: 80},
		{Label: "重点备注", Value: parent.ImportantNotes, Max: 2000},
		{Label: "下一步", Value: parent.NextAction, Max: 1000},
	})
}

func validateImportFieldLengths(rowNumber int, fields []importFieldLimit) (string, bool) {
	for _, field := range fields {
		if len([]rune(field.Value)) > field.Max {
			return fmt.Sprintf("第 %d 行%s不能超过 %d 个字符", rowNumber, field.Label, field.Max), false
		}
	}
	return "", true
}

func parseWeekday(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 1, true
	}
	value = strings.TrimPrefix(value, "星期")
	value = strings.TrimPrefix(value, "周")
	value = strings.TrimPrefix(value, "礼拜")
	labels := map[string]int{
		"日": 0, "天": 0, "0": 0,
		"一": 1, "1": 1,
		"二": 2, "2": 2,
		"三": 3, "3": 3,
		"四": 4, "4": 4,
		"五": 5, "5": 5,
		"六": 6, "6": 6,
	}
	if weekday, ok := labels[value]; ok {
		return weekday, true
	}
	return 0, false
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

func validImportClock(value string) bool {
	_, err := time.Parse("15:04", value)
	return err == nil
}

func importClockBefore(start string, end string) bool {
	startTime, err := time.Parse("15:04", start)
	if err != nil {
		return false
	}
	endTime, err := time.Parse("15:04", end)
	if err != nil {
		return false
	}
	return startTime.Before(endTime)
}

func normalizeRisk(value string) (string, bool) {
	value = strings.TrimSpace(value)
	switch {
	case strings.Contains(value, "高") || strings.EqualFold(value, "high"):
		return "high", true
	case strings.Contains(value, "中") || strings.EqualFold(value, "medium"):
		return "medium", true
	case strings.Contains(value, "低") || strings.EqualFold(value, "low"):
		return "low", true
	case value == "":
		return "", true
	default:
		return "", false
	}
}

func csvTemplate(rows [][]string) []byte {
	var buffer bytes.Buffer
	buffer.Write([]byte{0xef, 0xbb, 0xbf})
	writer := csv.NewWriter(&buffer)
	_ = writer.WriteAll(rows)
	writer.Flush()
	return buffer.Bytes()
}

func importPreview(row int, status string, message string, values map[string]string) ImportPreview {
	copyValues := make(map[string]string, len(values))
	for key, value := range values {
		copyValues[key] = value
	}
	return ImportPreview{Row: row, Status: status, Message: message, Values: copyValues}
}

func appendFailureReason(row []string, reason string) []string {
	result := append([]string(nil), row...)
	result = append(result, reason)
	return result
}

func failureCSV(header []string, failedRows [][]string) string {
	if len(failedRows) == 0 {
		return ""
	}
	rows := make([][]string, 0, len(failedRows)+1)
	rows = append(rows, append(append([]string(nil), header...), "失败原因"))
	rows = append(rows, failedRows...)
	return string(csvTemplate(rows))
}
