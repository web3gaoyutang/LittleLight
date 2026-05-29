package repository

import (
	"time"

	"github.com/web3gaoyutang/littlelight/server/internal/domain"
)

func dueFollowUpParentIDs(records []domain.CommunicationRecord) ([]domain.ID, int) {
	seen := map[domain.ID]bool{}
	ids := make([]domain.ID, 0)
	for _, record := range records {
		if record.ParentID == nil || *record.ParentID == "" || seen[*record.ParentID] {
			continue
		}
		seen[*record.ParentID] = true
		ids = append(ids, *record.ParentID)
	}
	return ids, len(records)
}

func pendingReminderCount(reminders []domain.Reminder) int {
	count := 0
	for _, reminder := range reminders {
		if reminder.Status == "" || reminder.Status == "pending" || reminder.Status == "snoozed" {
			count++
		}
	}
	return count
}

func endOfDay(day time.Time) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), day.Location())
}
