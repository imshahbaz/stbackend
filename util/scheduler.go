package util

import (
	"time"

	"github.com/robfig/cron/v3"
)

var (
	FifteenMinSpec = "10 */15 9-15 * * 1-5"
	FiveMinSpec    = "10 */5 9-15 * * 1-5"
	OneMinSpec     = "10 * 9-15 * * 1-5"
)

func ScheduleTask(spec string, task func()) (*cron.Cron, error) {
	ist := time.FixedZone("IST", 5*60*60+30*60)

	c := cron.New(
		cron.WithLocation(ist),
		cron.WithSeconds(),
	)

	_, err := c.AddFunc(spec, task)
	if err != nil {
		return nil, err
	}

	c.Start()

	return c, nil
}
