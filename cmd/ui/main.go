package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"go.uber.org/config"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

func NewLogger(c Database) *log.Logger {
	//func NewLogger(c Configuration) *log.Logger {
	logger := log.New(os.Stdout, "" /* prefix */, 0 /* flags */)
	logger.Printf("Executing NewLogger. %v", c)
	return logger
}

type Configuration struct {
	Application struct {
		Name string
	}
	Database struct {
		Username string
		Password string
	}
	Logger struct {
		LoggingLevel string `[yaml: logging_level]`
	}
}

type Database struct {
	Username string
	Password string
}

func LoadDatabase() (Database, error) {
	var c Database

	cfg, err := config.NewYAML(config.File("./config.yaml"))
	if err != nil {
		return c, err
	}

	if err := cfg.Get("database").Populate(&c); err != nil {
		return c, err
	}

	return c, nil

}

func LoadConfig() (Configuration, error) {
	var c Configuration

	cfg, err := config.NewYAML(config.File("./config.yaml"))
	if err != nil {
		return c, err
	}

	if err := cfg.Get("").Populate(&c); err != nil {
		return c, err
	}

	return c, nil
}

func Register(mux *http.ServeMux, h http.Handler) {
	mux.Handle("/", h)
}

func NewMux(lc fx.Lifecycle, logger *log.Logger) *http.ServeMux {
	logger.Print("Executing NewMux.")
	// First, we construct the mux and server. We don't want to start the server
	// until all handlers are registered.
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    "0.0.0.0:8081",
		Handler: mux,
	}
	// If NewMux is called, we know that another function is using the mux. In
	// that case, we'll use the Lifecycle type to register a Hook that starts
	// and stops our HTTP server.
	//
	// Hooks are executed in dependency order. At startup, NewLogger's hooks run
	// before NewMux's. On shutdown, the order is reversed.
	//
	// Returning an error from OnStart hooks interrupts application startup. Fx
	// immediately runs the OnStop portions of any successfully-executed OnStart
	// hooks (so that types which started cleanly can also shut down cleanly),
	// then exits.
	//
	// Returning an error from OnStop hooks logs a warning, but Fx continues to
	// run the remaining hooks.
	lc.Append(fx.Hook{
		// To mitigate the impact of deadlocks in application startup and
		// shutdown, Fx imposes a time limit on OnStart and OnStop hooks. By
		// default, hooks have a total of 15 seconds to complete. Timeouts are
		// passed via Go's usual context.Context.
		OnStart: func(context.Context) error {
			logger.Print("Starting HTTP server.")
			// In production, we'd want to separate the Listen and Serve phases for
			// better error-handling.
			go server.ListenAndServe()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Print("Stopping HTTP server.")
			return server.Shutdown(ctx)
		},
	})

	return mux
}

func NewHandler(logger *log.Logger) (http.Handler, error) {
	logger.Print("Executing NewHandler.")
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		logger.Print("Got a request.")
		//		tm := time.Now()
		w.Write([]byte("The time is: "))
	}), nil
}

func NewConfigHandler(arg string) func(*log.Logger) (http.Handler, error) {
	return func(logger *log.Logger) (http.Handler, error) {
		logger.Printf("Executing NewHandler %s.", arg)
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			logger.Print("Got a request.")
			//		tm := time.Now()
			w.Write([]byte("The time is: "))
		}), nil
	}
}

func main() {
	app := fx.New(
		fx.Provide(
			LoadConfig,
			LoadDatabase,
			NewLogger,
			NewConfigHandler("configured"),
			NewMux,
		),
		fx.Invoke(Register),

		fx.WithLogger(
			func() fxevent.Logger {
				return &fxevent.ConsoleLogger{W: os.Stdout}
			},
		),
	)
	app.Run()
}
