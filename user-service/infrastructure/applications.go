package infrastructure

import (
	"database/sql"
	"fmt"
	"helper"
	"log"
	"net"
	"net/http"
	"os"
	"time"
	"user-service/config"
	"user-service/user-proto/users"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgconn"
	_ "github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"google.golang.org/grpc"
)

type application struct {
	Config   config.Config
	InfoLog  *log.Logger
	ErrorLog *log.Logger
}

func Application() application {
	var cfg config.Config
	return application{
		Config:   cfg,
		InfoLog:  log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime),
		ErrorLog: log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime),
	}
}

func (app *application) StartGrpcServer(grpcServer users.UserServiceServer) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", helper.GetEnvInt("GRPC_PORT")))
	if err != nil {
		log.Fatal(err)
	}

	s := grpc.NewServer()

	users.RegisterUserServiceServer(s, grpcServer)

	app.InfoLog.Printf("starting server on %d mode on port %d", helper.GetEnvInt("GRPC_PORT"), helper.GetEnvInt("GRPC_PORT"))

	if err := s.Serve(lis); err != nil {
		log.Fatal(err)
	}
}

func (app *application) Serve(mux *gin.Engine) (err error) {

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", app.Config.Port),
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       30 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	app.InfoLog.Printf("starting server on %s mode on port %d", app.Config.Env, app.Config.Port)

	err = srv.ListenAndServe()
	return
}

func openDb(dsn string) (db *sql.DB, err error) {
	db, err = sql.Open("pgx", dsn)
	if err != nil {
		return
	}
	err = db.Ping()
	if err != nil {
		return
	}
	return
}

func (app *application) ConnectToDb() (db *sql.DB) {
	ticker := time.NewTicker(time.Second * 2)
	var err error
	count := 0
	for db == nil {
		db, err = openDb(os.Getenv("DSN"))
		if err != nil {
			app.InfoLog.Println("postgres is not ready yet")
		}
		count++
		if count > 5 {
			log.Fatal(err)
		}
		<-ticker.C
	}
	return
}
