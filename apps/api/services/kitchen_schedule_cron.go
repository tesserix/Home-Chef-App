package services

// kitchen_schedule_cron.go — schedule-driven auto open/close (#kitchen-schedule).
//
// A chef who opts in (ChefProfile.AutoScheduleEnabled) has their kitchen's
// AcceptingOrders flipped on/off automatically to match their operating hours
// (ChefSchedule) for the current IST day — so they don't have to remember to open
// and close every day. Chefs who haven't opted in are untouched: they open/close
// by hand. A manual timed pause (PausedUntil) still wins while it's active.

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

const kitchenScheduleInterval = 5 * time.Minute

// scheduleIST is the wall-clock zone the chef's OpenTime/CloseTime are expressed
// in. Fixed offset fallback if tzdata is unavailable in the image.
var scheduleIST = func() *time.Location {
	if loc, err := time.LoadLocation("Asia/Kolkata"); err == nil {
		return loc
	}
	return time.FixedZone("IST", 5*3600+30*60)
}()

// StartKitchenScheduleCron launches the auto open/close loop. Returns immediately.
func StartKitchenScheduleCron(ctx context.Context) {
	go func() {
		runKitchenScheduleScan(ctx)
		ticker := time.NewTicker(kitchenScheduleInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("kitchen-schedule: shutting down on ctx cancel")
				return
			case <-ticker.C:
				runKitchenScheduleScan(ctx)
			}
		}
	}()
	log.Println("kitchen-schedule: cron started (interval=5m)")
}

// scheduleMinutes turns "HH:MM" into minutes-since-midnight, or -1 if unparseable.
func scheduleMinutes(s string) int {
	s = strings.TrimSpace(s)
	var h, m int
	if _, err := fmt.Sscanf(s, "%d:%d", &h, &m); err != nil {
		return -1
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return -1
	}
	return h*60 + m
}

func runKitchenScheduleScan(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("kitchen-schedule: panic recovered: %v", r)
		}
	}()

	now := time.Now()
	ist := now.In(scheduleIST)
	weekday := int(ist.Weekday()) // 0=Sunday .. 6=Saturday (matches ChefSchedule.DayOfWeek)
	nowMin := ist.Hour()*60 + ist.Minute()

	// Only opted-in, approved kitchens are managed.
	var chefs []models.ChefProfile
	if err := database.DB.
		Where("auto_schedule_enabled = ? AND is_verified = ?", true, true).
		Find(&chefs).Error; err != nil {
		log.Printf("kitchen-schedule: chef query failed: %v", err)
		return
	}
	if len(chefs) == 0 {
		return
	}

	ids := make([]uuid.UUID, 0, len(chefs))
	for _, ch := range chefs {
		ids = append(ids, ch.ID)
	}

	// Today's schedule row per chef.
	var scheds []models.ChefSchedule
	database.DB.Where("chef_id IN ? AND day_of_week = ?", ids, weekday).Find(&scheds)
	byChef := make(map[uuid.UUID]models.ChefSchedule, len(scheds))
	for _, s := range scheds {
		byChef[s.ChefID] = s
	}

	var toOpen, toClose []uuid.UUID
	for _, ch := range chefs {
		desiredOpen := false
		if s, ok := byChef[ch.ID]; ok && !s.IsClosed {
			openMin := scheduleMinutes(s.OpenTime)
			closeMin := scheduleMinutes(s.CloseTime)
			if openMin >= 0 && closeMin > openMin && nowMin >= openMin && nowMin < closeMin {
				desiredOpen = true
			}
		}
		// A manual "back in N min" pause overrides the schedule while it's active.
		if ch.PausedUntil != nil && ch.PausedUntil.After(now) {
			desiredOpen = false
		}
		switch {
		case desiredOpen && !ch.AcceptingOrders:
			toOpen = append(toOpen, ch.ID)
		case !desiredOpen && ch.AcceptingOrders:
			toClose = append(toClose, ch.ID)
		}
	}

	if len(toOpen) > 0 {
		if err := database.DB.Model(&models.ChefProfile{}).
			Where("id IN ?", toOpen).Update("accepting_orders", true).Error; err != nil {
			log.Printf("kitchen-schedule: open failed: %v", err)
		} else {
			log.Printf("kitchen-schedule: opened %d kitchen(s) on schedule", len(toOpen))
		}
	}
	if len(toClose) > 0 {
		if err := database.DB.Model(&models.ChefProfile{}).
			Where("id IN ?", toClose).Update("accepting_orders", false).Error; err != nil {
			log.Printf("kitchen-schedule: close failed: %v", err)
		} else {
			log.Printf("kitchen-schedule: closed %d kitchen(s) on schedule", len(toClose))
		}
	}
}
