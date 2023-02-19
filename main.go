package main

import (
	"context"
	"flag"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/go-fed/testsuite/server"
)

const (
	kCommonTemplate = "common.tmpl"
	kSiteTemplate   = "site.tmpl"
	kHomePage       = "home.html"
	kAboutPage      = "about.html"
	kNewTestPage    = "new_test.html"
	kTestStatusPage = "test_status.html"
)

type CommandLineFlags struct {
	Hostname     *string
	TemplatesDir *string
	StaticDir    *string
	TestTimeout  *time.Duration
	MaxTests     *int
	LogFile      *string
}

func NewCommandLineFlags() *CommandLineFlags {
	c := &CommandLineFlags{
		Hostname:     flag.String("host", "", "Host name of this instance (including TLD)"),
		TemplatesDir: flag.String("templates", "./templates", "Directory containing the Go template files"),
		StaticDir:    flag.String("static", "./static", "Directory containing statically-served files"),
		TestTimeout:  flag.Duration("test_timeout", time.Minute*15, "Maximum time tests will be kept"),
		MaxTests:     flag.Int("max_tests", 30, "Maximum number of concurrent tests"),
		LogFile:      flag.String("logfile", "log.txt", "Log file to be able to audit spam & abuse"),
	}
	flag.Parse()
	if err := c.validate(); err != nil {
		panic(err)
	}
	return c
}

func (c *CommandLineFlags) validate() error {
	return nil
}

func (c *CommandLineFlags) templateFilepaths(pageFile string) []string {
	return []string{
		filepath.Join(*c.TemplatesDir, kCommonTemplate),
		filepath.Join(*c.TemplatesDir, kSiteTemplate),
		filepath.Join(*c.TemplatesDir, pageFile),
	}
}

func (c *CommandLineFlags) homeTemplate() (*template.Template, error) {
	return template.ParseFiles(c.templateFilepaths(kHomePage)...)
}

func (c *CommandLineFlags) aboutTemplate() (*template.Template, error) {
	return template.ParseFiles(c.templateFilepaths(kAboutPage)...)
}

func (c *CommandLineFlags) newTestTemplate() (*template.Template, error) {
	return template.ParseFiles(c.templateFilepaths(kNewTestPage)...)
}

func (c *CommandLineFlags) testStatusTemplate() (*template.Template, error) {
	return template.ParseFiles(c.templateFilepaths(kTestStatusPage)...)
}

func main() {
	c := NewCommandLineFlags()
	rand.Seed(time.Now().Unix())

	httpsServer := &http.Server{
		Addr: ":8000",
	}

	homeTmpl, err := c.homeTemplate()
	if err != nil {
		panic(err)
	}
	aboutTmpl, err := c.aboutTemplate()
	if err != nil {
		panic(err)
	}
	newTestTmpl, err := c.newTestTemplate()
	if err != nil {
		panic(err)
	}
	testStatusTmpl, err := c.testStatusTemplate()
	if err != nil {
		panic(err)
	}
	_ = server.NewWebServer(homeTmpl, aboutTmpl, newTestTmpl, testStatusTmpl, httpsServer, *c.Hostname, *c.TestTimeout, *c.MaxTests, *c.StaticDir, *c.LogFile)

	redir := &http.Server{
		Addr:         ":http",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Connection", "close")
			http.Redirect(w, req, fmt.Sprintf("https://%s%s", req.Host, req.URL), http.StatusMovedPermanently)
		}),
	}
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		if err := redir.Shutdown(context.Background()); err != nil {
			log.Printf("HTTP redirect server Shutdown: %v", err)
		}
		if err := httpsServer.Shutdown(context.Background()); err != nil {
			log.Printf("HTTP server Shutdown: %v", err)
		}
	}()
	go func() {
		if err := redir.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("HTTP redirect server ListenAndServe: %v", err)
		}
	}()
	if err := httpsServer.ListenAndServe(); err != nil {
		panic(err)
	}
}
