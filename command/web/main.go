package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/O-Nicolinho/LT-TeaCompany-WebApp/internal/driver"
	"github.com/O-Nicolinho/LT-TeaCompany-WebApp/internal/mail"
	"github.com/O-Nicolinho/LT-TeaCompany-WebApp/internal/models"
	"github.com/alexedwards/scs/v2"
)

const version = "1.0.0"

const cssVersion = "1"

var session *scs.SessionManager

type config struct {
	port int
	env  string
	api  string
	db   struct {
		dsn string
	}
	stripe struct {
		secret string
		key    string
	}
	smtp struct {
		host string
		port int
		user string
		pass string
		from string
	}
}

type application struct {
	mailer        mail.Sender
	config        config
	infoLog       *log.Logger
	errorLog      *log.Logger
	templateCache map[string]*template.Template
	version       string
	DB            models.DBModel
	Session       *scs.SessionManager
}

func (app *application) serve() error {
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", app.config.port),
		Handler:           app.routes(),
		IdleTimeout:       30 * time.Second,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Second,
	}

	app.infoLog.Println(fmt.Sprintf("Starting HTTP server in %s mode on port %d", app.config.env, app.config.port))

	return srv.ListenAndServe()
}

func main() {

	gob.Register(TransactionData{})

	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "Server port to listen on")

	flag.StringVar(&cfg.env, "env", "development", "Application environment {development | production}")

	flag.StringVar(&cfg.api, "api", "http://localhost:4001", "URL to api")

	flag.StringVar(&cfg.db.dsn, "dsn", "nicolinho:StrongPwd123!@unix(/tmp/mysql.sock)/TEA?parseTime=true&tls=false",
		"DSN (e.g. user:pass@unix(/path/to/mysql.sock)/dbname?parseTime=true)")
	flag.Parse()

	cfg.stripe.key = os.Getenv("STRIPE_KEY")
	cfg.stripe.secret = os.Getenv("STRIPE_SECRET")

	log.Println("Loaded Stripe secret:", cfg.stripe.secret != "")

	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	conn, err := driver.OpenDB(cfg.db.dsn)

	if err != nil {
		errorLog.Fatal(err)
	}
	defer conn.Close()

	// set up session

	session = scs.New()
	session.Lifetime = 24 * time.Hour

	tc := make(map[string]*template.Template)

	app := &application{
		config: cfg,
		mailer: mail.Sender{
			Host: cfg.smtp.host, Port: cfg.smtp.port,
			User: cfg.smtp.user, Pass: cfg.smtp.pass, From: cfg.smtp.from,
		},
		infoLog:       infoLog,
		errorLog:      errorLog,
		templateCache: tc,
		version:       version,
		DB:            models.DBModel{DB: conn},
		Session:       session,
	}

	cfg.smtp.host = os.Getenv("SMTP_HOST")
	cfg.smtp.port, _ = strconv.Atoi(os.Getenv("SMTP_PORT"))
	cfg.smtp.user = os.Getenv("SMTP_USER")
	cfg.smtp.pass = os.Getenv("SMTP_PASS")
	cfg.smtp.from = os.Getenv("SMTP_FROM")

	err = app.serve()

	if err != nil {
		app.errorLog.Println(err)
		log.Fatal(err)
	}

}
