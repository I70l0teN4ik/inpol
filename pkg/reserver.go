package pkg

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/sync/errgroup"
)

type Reserver interface {
	ReserveResidence() error
	AsyncReserve(limit int) error
	CheckDates() error
	WatchDates(sleep int) error
	GetReservationQueues() error
	Auth() error
	GetMFA() string
}

type reserver struct {
	client Client
	logger *log.Logger
}

func NewReserver(client Client) Reserver {
	return &reserver{
		client: client,
		logger: log.New(os.Stdout, "svc: ", log.Ltime),
	}
}

func (r reserver) GetMFA() string {
	return r.client.GetMFA(context.Background())
}

func (r reserver) Auth() error {
	_, err := r.client.Login("username", "password")
	if err != nil {
		r.logger.Println(err)
		return err
	}

	return nil
}

func (r reserver) ReserveResidence() (err error) {
	d := time.Now()
	day := d.Add(time.Hour * 24 * 6)

	var slots []Slot

	ctx := context.Background()
	for {
		slots, err = r.getSlots(ctx, day.Format("2006-01-02"))
		if err != nil {
			return err
		}
		if len(slots) > 0 {
			break
		}
	}

	r.reserve(ctx, slots)

	return
}

func (r reserver) AsyncReserve(limit int) error {
	d := time.Now()
	day := d.Add(time.Hour * 24 * 6)

	var err error
	var slots []Slot
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(limit)

	for {
		select {
		case <-egCtx.Done():
			break
		default:
			eg.Go(func() error {
				slots, err = r.getSlots(egCtx, day.Format("2006-01-02"))
				if err != nil {
					return err
				}
				if len(slots) > 0 {
					cancel() // Cancel other goroutines
				}

				return nil
			})
		}

		if len(slots) > 0 {
			r.reserve(context.Background(), slots)
			break
		}
		if egCtx.Err() != nil {
			break
		}
	}

	return eg.Wait()
}

func (r reserver) CheckDates() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	//ctx := context.Background()
	dates, err := r.client.Dates(ctx)
	if err != nil {
		r.logger.Println(err)
		return err
	}

	eg, egCtx := errgroup.WithContext(ctx)

	for _, date := range dates[:7] {
		eg.Go(func() error {
			slots, err := r.client.Slots(egCtx, date[:10])
			if err != nil {
				r.logger.Println(err)
				return err
			}
			if len(slots) > 0 {
				r.reserve(ctx, slots)
				cancel()
			}
			return nil
		})
	}

	return eg.Wait()
}

func (r reserver) GetReservationQueues() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := r.client.ReservationQueues(ctx)
	return err
}

func (r reserver) WatchDates(sleep int) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dates := make([]string, 0, 5)
	d := time.Now()

	for i := 0; i < 7; i++ {
		d = d.Add(time.Hour * 24)
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
			dates = append(dates, d.Format("2006-01-02"))
		}
	}
	r.logger.Println(dates)

	var done bool
	for !done {
		for _, date := range dates {
			slots, _ := r.client.Slots(ctx, date)
			if len(slots) > 0 {
				message := fmt.Sprintf("Found slots for date %s", date)
				r.client.NotifyTelegramUsers(message)

				done = r.reserve(ctx, slots)
				break
			}
		}

		r.logger.Println("---------------------------")
		time.Sleep(time.Duration(sleep) * time.Second)
	}

	return nil
}

func (r reserver) getSlots(ctx context.Context, day string) ([]Slot, error) {
	slots, err := r.client.Slots(ctx, day)
	if err != nil {
		r.logger.Println(err)
		time.Sleep(time.Millisecond * 100)
		return nil, err
	}

	if len(slots) == 0 {
		time.Sleep(time.Millisecond * 100)
	} else {
		r.logger.Printf("found %d slots\n", len(slots))
	}

	return slots, err
}

func (r reserver) reserve(ctx context.Context, slots []Slot) bool {
	// Find the slot with the largest count
	// If multiple slots have the same largest count, select the last one
	largestCountSlot := slots[0]
	for _, s := range slots {
		if s.Count >= largestCountSlot.Count {
			largestCountSlot = s
		}
	}

	slot := largestCountSlot

	r.logger.Println(slot)

	done, err := r.client.Reserve(ctx, slot)
	if err != nil {
		r.logger.Println(err)
	}

	if done {
		r.logger.Println("Reservation done:", slot)
	} else {
		r.logger.Println("Reservation failed", slot)

		for _, s := range slots {
			ok, err := r.client.Reserve(ctx, s)
			if err != nil {
				r.logger.Println(err)
			}
			if ok {
				done = true
				r.logger.Println("Reservation done:", slot.Id, slot.Date)
				break
			} else {
				r.logger.Println()
			}
		}
	}

	return done
}
