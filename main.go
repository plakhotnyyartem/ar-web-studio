package main

import (
	"context"
	"errors"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
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
	// За балансировщиком хостинга (Render и т.п.) реальный IP посетителя приходит
	// в X-Forwarded-For. Доверяем этому заголовку только от адресов прокси из
	// TRUSTED_PROXIES, иначе все посетители выглядели бы как один IP.
	// На Render прокси подключается к приложению с локального адреса (::1),
	// поэтому вместе с заданными сетями доверяем и loopback: снаружи с него
	// подключиться невозможно.
	var proxies []string
	if v := os.Getenv("TRUSTED_PROXIES"); v != "" {
		proxies = append(strings.Split(v, ","), "127.0.0.1/32", "::1/128")
	}
	if err := r.SetTrustedProxies(proxies); err != nil {
		log.Fatalf("TRUSTED_PROXIES: %v", err)
	}
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
	r.HEAD("/", func(c *gin.Context) { c.Status(http.StatusOK) }) // проверка «жив ли сервис»
	r.POST("/api/lead", h.CreateLead)
	r.NoRoute(h.NotFound)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	logTelegramStatus()

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
