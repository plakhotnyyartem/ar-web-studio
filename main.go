package main

import (
	"context"
	"errors"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	port := envOr("PORT", "8080")

	store, err := NewLeadStore("data/leads.jsonl")
	if err != nil {
		log.Fatalf("не удалось открыть хранилище заявок: %v", err)
	}

	r := gin.Default()
	r.SetTrustedProxies(nil)
	r.SetFuncMap(template.FuncMap{
		"icon": icon,
		"year": func() int { return time.Now().Year() },
		"inc":  func(i int) int { return i + 1 },
	})
	r.LoadHTMLGlob("templates/*.html")
	r.Static("/static", "./static")
	r.StaticFile("/favicon.svg", "./static/favicon.svg")

	h := &Handlers{leads: store, limiter: newRateLimiter(30 * time.Second)}
	r.GET("/", h.Index)
	r.POST("/api/lead", h.CreateLead)
	r.NoRoute(h.NotFound)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("AR Web Studio запущен: http://localhost:%s", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("ошибка сервера: %v", err)
		}
	}()

	// Корректное завершение: дожидаемся текущих запросов перед выходом.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("ошибка при остановке: %v", err)
	}
	log.Println("сервер остановлен")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
