package main

import (
	"bufio"
	"errors"
	"os"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"

	_ "github.com/heroku/x/hmetrics/onload"
)

const bufferLen = 500

type logData struct {
	app    *string
	tags   *[]string
	prefix *string
	line   *string
}

type ServerCtx struct {
	Port              string
	AppTags           map[string][]string
	AppPrefix         map[string]string
	StatsdUrl         string
	BasicAuthUsername string
	BasicAuthPassword string
	Debug             bool
	in                chan *logData
	out               chan *logMetrics
}

//STATSD_URL=..  Required. Default: localhost:8125
//DATADOG_DRAIN_DEBUG=         Optional. If DEBUG is set, a lot of stuff w
func loadServerCtx() *ServerCtx {

	s := &ServerCtx{"8080",
		make(map[string][]string),
		make(map[string]string),
		"localhost:8125",
		"",
		"",
		false,
		nil,
		nil,
	}
	port := os.Getenv("PORT")
	if port != "" {
		s.Port = port
	}

	if os.Getenv("DATADOG_DRAIN_DEBUG") != "" {
		s.Debug = true
	}

	s.StatsdUrl = os.Getenv("STATSD_URL")
	if s.StatsdUrl == "" {
		log.Panic(errors.New("Missing STATSD_URL"))
	}

	s.BasicAuthUsername = os.Getenv("BASIC_AUTH_USERNAME")
	if s.BasicAuthUsername == "" {
		log.Panic(errors.New("Missing BASIC_AUTH_USERNAME"))
	}

	s.BasicAuthPassword = os.Getenv("BASIC_AUTH_PASSWORD")
	if s.BasicAuthPassword == "" {
		log.Panic(errors.New("Missing BASIC_AUTH_PASSWORD"))
	}

	log.WithFields(log.Fields{
		"port":      s.Port,
		"AppTags":   s.AppTags,
		"AppPrefix": s.AppPrefix,
		"StatsdUrl": s.StatsdUrl,
		"Debug":     s.Debug,
	}).Info("Configuration loaded")

	return s
}

func init() {
	// Output to stderr instead of stdout
	log.SetOutput(os.Stderr)

	// Only log the Info severity or above.
	log.SetLevel(log.InfoLevel)
}

func (s *ServerCtx) getTags(c *gin.Context, app string) []string {
	requestTags := c.DefaultQuery("tags", "")
	if requestTags == "" {
		return s.AppTags[app]
	} else {
		return strings.Split(requestTags, ",")
	}
}

func (s *ServerCtx) processLogs(c *gin.Context) {
	app := c.DefaultQuery("app", "")
	if app == "" {
		log.Error(errors.New("app query parameter not passed"))
		c.String(500, "Missing app query parameter")
		return
	}

	tags := s.getTags(c, app)
	tags = append(tags, "app:"+app)
	prefix := c.DefaultQuery("prefix", s.AppPrefix[app])

	scanner := bufio.NewScanner(c.Request.Body)
	for scanner.Scan() {
		line := scanner.Text()
		log.WithField("line", line).Debug("LINE")
		s.in <- &logData{&app, &tags, &prefix, &line}
	}
	if err := scanner.Err(); err != nil {
		log.Error(err)
	}

	c.String(200, "OK")
}

func main() {
	gin.SetMode(gin.ReleaseMode)

	s := loadServerCtx()
	if s.Debug {
		log.SetLevel(log.DebugLevel)
		gin.SetMode(gin.DebugMode)
	}

	c, err := statsdClient(s.StatsdUrl)
	if err != nil {
		log.WithField("statsdUrl", s.StatsdUrl).Fatal("Could not connect to statsd")
	}

	if v := os.Getenv("EXCLUDED_TAGS"); v != "" {
		for _, t := range strings.Split(v, ",") {
			c.ExcludedTags[t] = true
		}
	}

	r := gin.Default()
	r.GET("/status", func(c *gin.Context) {
		c.String(200, "OK")
	})

	accounts := map[string]string{}
	accounts[s.BasicAuthUsername] = s.BasicAuthPassword
	auth := r.Group("/", gin.BasicAuth(accounts))
	auth.POST("/", s.processLogs)

	s.in = make(chan *logData, bufferLen)
	defer close(s.in)
	s.out = make(chan *logMetrics, bufferLen)
	defer close(s.out)
	go logProcess(s.in, s.out)
	go c.sendToStatsd(s.out)
	log.Infoln("Server ready ...")
	r.Run(":" + s.Port)
}
