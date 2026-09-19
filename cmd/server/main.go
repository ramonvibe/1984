package main

import (
 "context"
 "errors"
 "flag"
 "fmt"
 "log/slog"
 "net/http"
 "os"
 "os/signal"
 "syscall"
 "time"
 "github.com/jackc/pgx/v5/pgxpool"
 "github.com/ramon/trackline/internal/config"
 "github.com/ramon/trackline/internal/database"
 "github.com/ramon/trackline/internal/server"
)

func main() { if err:=run(); err!=nil { slog.Error("server stopped","error",err); os.Exit(1) } }
func run() error {
 migrateOnly:=flag.Bool("migrate",false,"Apply pending database migrations and exit")
 healthcheck:=flag.Bool("healthcheck",false,"Check the local server and exit")
 flag.Parse()
 configuration,err:=config.Load(); if err!=nil { return err }
 if *healthcheck {
  client:=http.Client{Timeout:3*time.Second}
  response,err:=client.Get("http://127.0.0.1"+configuration.Address+"/healthz"); if err!=nil { return err }; defer response.Body.Close()
  if response.StatusCode!=200 { return fmt.Errorf("unhealthy: HTTP %d",response.StatusCode) }; return nil
 }
 ctx,stop:=signal.NotifyContext(context.Background(),os.Interrupt,syscall.SIGTERM); defer stop()
 poolConfig,err:=pgxpool.ParseConfig(configuration.DatabaseURL); if err!=nil { return errors.New("invalid DATABASE_URL") }
 poolConfig.MaxConns=10; poolConfig.MinConns=0; poolConfig.MaxConnIdleTime=5*time.Minute
 poolConfig.ConnConfig.RuntimeParams["timezone"]="UTC"
 pool,err:=pgxpool.NewWithConfig(ctx,poolConfig); if err!=nil { return err }; defer pool.Close()
 startup,cancel:=context.WithTimeout(ctx,30*time.Second)
 err=database.Migrate(startup,pool); cancel(); if err!=nil { return err }
 if *migrateOnly { slog.Info("database migrations applied"); return nil }
 app,err:=server.New(pool,configuration); if err!=nil { return err }
 httpServer:=&http.Server{Addr:configuration.Address,Handler:app.Handler(),ReadHeaderTimeout:5*time.Second,ReadTimeout:15*time.Second,WriteTimeout:75*time.Second,IdleTimeout:60*time.Second,MaxHeaderBytes:16<<10}
 failed:=make(chan error,1)
  go func(){ slog.Info("1984 is ready","address",configuration.Address); failed<-httpServer.ListenAndServe() }()
 ticker:=time.NewTicker(time.Hour); defer ticker.Stop()
 for {
  select {
  case err:=<-failed: if errors.Is(err,http.ErrServerClosed) { return nil }; return err
  case <-ticker.C:
   cleanup,cancel:=context.WithTimeout(ctx,10*time.Second)
   if err:=app.Queries.DeleteExpiredSessions(cleanup); err!=nil { slog.Warn("session cleanup failed","error",err) }
   if err:=app.Queries.DeleteOldGitHubDeliveries(cleanup); err!=nil { slog.Warn("webhook cleanup failed","error",err) }; cancel()
  case <-ctx.Done():
   shutdown,cancel:=context.WithTimeout(context.Background(),configuration.ShutdownTimeout); defer cancel()
   return httpServer.Shutdown(shutdown)
  }
 }
}
