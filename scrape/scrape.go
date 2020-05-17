package scrape

import (
	"fmt"
	"net/http"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/pdunnavant/modem-scraper/config"
	"go.uber.org/zap"
)

// Scrape scrapes data from the modem.
func Scrape(logger *zap.Logger, config config.Configuration) (*ModemInformation, error) {
	doc, err := getDocumentFromURL(logger, config.Modem.Url+"/cmconnectionstatus.html")
	if err != nil {
		return nil, err
	}
	connectionStatus := scrapeConnectionStatus(doc)

	doc, err = getDocumentFromURL(logger, config.Modem.Url+"/cmswinfo.html")
	if err != nil {
		return nil, err
	}
	softwareInformation := scrapeSoftwareInformation(doc)

	doc, err = getDocumentFromURL(logger, config.Modem.Url+"/cmeventlog.html")
	if err != nil {
		return nil, err
	}
	eventLog := scrapeEventLogs(logger, doc)

	modemInformation := ModemInformation{
		ConnectionStatus:    *connectionStatus,
		SoftwareInformation: *softwareInformation,
		EventLog:            eventLog,
	}

	return &modemInformation, nil
}

func getDocumentFromURL(logger *zap.Logger, url string) (*goquery.Document, error) {
	logger.Debug(fmt.Sprintf("grabbing %s", url),
		zap.String("op", "scrape.getDocumentFromURL"),
	)

	start := time.Now()

	// TODO: add a timeout here (10s?)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	elapsed := time.Since(start)
	logger.Debug(fmt.Sprintf("got %s, took %s", url, elapsed),
		zap.String("op", "scrape.getDocumentFromURL"),
	)

	return doc, nil
}
