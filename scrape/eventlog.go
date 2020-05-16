package scrape

import (
	"fmt"
	"time"

	"github.com/PuerkitoBio/goquery"
	_ "github.com/influxdata/influxdb1-client" // this is important because of a bug in go mod
	client "github.com/influxdata/influxdb1-client/v2"
)

const (
	dateTimeLayout = "01/02/2006 15:04 MST"
)

// EventLog holds data pulled from the /cmeventlog.html page.
type EventLog struct {
	DateTime    string
	EventID     int
	EventLevel  int
	Description string
}

// ToInfluxPoints converts EventLog to "points"
func (e EventLog) ToInfluxPoints() ([]*client.Point, error) {
	var points []*client.Point

	// No tags for this specific struct.
	tags := map[string]string{}
	fields := map[string]interface{}{
		"date_time":   e.DateTime,
		"event_id":    e.EventID,
		"event_level": e.EventLevel,
		"description": e.Description,
	}
	point, err := client.NewPoint("event_log", tags, fields, time.Now())
	if err != nil {
		return nil, fmt.Errorf("error generating points data for EventLog: %s", err.Error())
	}

	points = append(points, point)

	return points, nil
}

const eventLogTableSelector = "#bg3 > div.container > div.content > form > center > table"

func scrapeEventLogs(doc *goquery.Document) []EventLog {
	eventLogTable := doc.Find(eventLogTableSelector)
	eventLogTableTbody := eventLogTable.Children()
	eventLogTableTbodyRows := eventLogTableTbody.Children()

	eventLogs := []EventLog{}
	eventLogTableTbodyRows.Each(func(index int, row *goquery.Selection) {
		// Skip the "title" row as well as the "header" row.
		// These are both regular old <tr> rows on this page.
		if index > 0 {
			eventLogs = append(eventLogs, makeEventLog(row))
		}
	})

	return eventLogs
}

func makeEventLog(selection *goquery.Selection) EventLog {
	rowData := selection.Children()
	eventLog := EventLog{
		DateTime:    rowData.Get(0).FirstChild.Data,
		EventID:     getIntRowData(rowData, 1),
		EventLevel:  getIntRowData(rowData, 2),
		Description: rowData.Get(3).FirstChild.Data,
	}

	eventLog.DateTime, _ = formatTime(eventLog.DateTime)

	return eventLog
}

func formatTime(datetime string) (string, error) {
	now := time.Now()
	zone, _ := now.Zone()
	t, err := time.Parse(dateTimeLayout, datetime+" "+zone)
	if err != nil {
		fmt.Println(err)
		return datetime, err
	}
	return t.Format(time.RFC3339), nil
}

func buildEventLogPoints(logs []EventLog) ([]*client.Point, error) {
	var points []*client.Point

	for _, log := range logs {
		influxPoints, err := log.ToInfluxPoints()
		if err != nil {
			return nil, err
		}
		points = append(points, influxPoints...)
	}

	return points, nil
}
