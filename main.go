package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type HasuraScheduleResponse struct {
	EventID string `json:"event_id"`
	Message string `json:"message"`
}

func setSchedule(endpoint string, webhook string, scheduleTime time.Time) (*HasuraScheduleResponse, error) {
	p, err := json.Marshal(
		map[string]any{
			"type": "create_scheduled_event",
			"args": map[string]string{
				"webhook":     webhook,
				"schedule_at": scheduleTime.Format(time.RFC3339),
			},
		},
	)

	if err != nil {
		return nil, err
	}

	resp, err := http.Post(
		endpoint,
		"application/json",
		bytes.NewBuffer(p),
	)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var r HasuraScheduleResponse

	err = json.Unmarshal(
		respBody,
		&r,
	)

	if err != nil {
		return nil, err
	}

	return &r, nil
}

type ScheduleResponse struct {
	ID           string `json:"id"`
	StartEventID string `json:"start_event_id"`
	EndEventID   string `json:"end_event_id"`
}

type ScheduleObject struct {
	ID           string
	StartWebhook string
	EndWebhook   string
	StartTime    time.Time
	EndTime      time.Time
}

type HasuraScheduler struct {
	HasuraEndpoint string
}

func (r *HasuraScheduler) Schedule(s ScheduleObject) (*ScheduleResponse, error) {

	sr, err := setSchedule(
		r.HasuraEndpoint,
		s.StartWebhook,
		s.StartTime,
	)

	if err != nil {
		return nil, err
	}

	er, err := setSchedule(
		r.HasuraEndpoint,
		s.EndWebhook,
		s.EndTime,
	)

	return &ScheduleResponse{
		ID:           s.ID,
		StartEventID: sr.EventID,
		EndEventID:   er.EventID,
	}, nil
}

func main() {

	e := echo.New()
	e.Use(middleware.Logger())
	apiv1g := e.Group("/api/v1")

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	schedulerRepo := &HasuraScheduler{
		// TODO: change me
		HasuraEndpoint: "http://graphql-engine:8080/v1/metadata",
	}

	loc, err := time.LoadLocation(
		"Asia/Bangkok",
	)

	if err != nil {
		panic(err)
	}

	apiv1g.POST(
		"/start",
		func(c echo.Context) error {
			return c.JSON(
				http.StatusOK,
				map[string]string{
					"message":      "success",
					"current_time": time.Now().In(loc).Format(time.DateTime),
				},
			)
		},
	)

	apiv1g.POST(
		"/stop",
		func(c echo.Context) error {
			return c.JSON(
				http.StatusOK,
				map[string]string{
					"message":      "success",
					"current_time": time.Now().In(loc).Format(time.DateTime),
				},
			)
		},
	)

	apiv1g.POST(
		"/schedule",
		func(c echo.Context) error {

			var req struct {
				ID        string    `json:"id"`
				StartTime time.Time `json:"start_time"`
				EndTime   time.Time `json:"end_time"`
			}

			err := c.Bind(&req)
			if err != nil {
				return c.JSON(
					http.StatusInternalServerError,
					map[string]string{
						"message": err.Error(),
					},
				)
			}

			slog.Info(
				"request schedule",
				slog.String("id", req.ID),
				slog.String("start_time", req.StartTime.String()),
				slog.String("end_time", req.EndTime.String()),
			)

			r, err := schedulerRepo.Schedule(
				ScheduleObject{
					ID:           req.ID,
					StartWebhook: "http://scheduler:8000/api/v1/start", //TODO : change me
					EndWebhook:   "http://scheduler:8000/api/v1/stop",  //TODO: change me
					StartTime:    req.StartTime,
					EndTime:      req.EndTime,
				},
			)

			return c.JSON(
				http.StatusOK,
				map[string]string{
					"id":      r.ID,
					"message": "success",
				},
			)
		},
	)

	err = e.Start(":8000")
	if err != nil {
		panic(err)
	}

}
