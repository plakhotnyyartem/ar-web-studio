package main

import (
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	leads   *LeadStore
	limiter *rateLimiter
}

func (h *Handlers) Index(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"S":        Studio,
		"Services": Services,
		"Projects": Projects,
		"Plans":    Plans,
		"Steps":    Steps,
		"Reviews":  Reviews,
		"FAQ":      FAQ,
		"Stack":    Stack,
	})
}

func (h *Handlers) NotFound(c *gin.Context) {
	c.HTML(http.StatusNotFound, "404.html", gin.H{"S": Studio})
}

type leadRequest struct {
	Name     string `json:"name" binding:"required,max=80"`
	Phone    string `json:"phone" binding:"required,max=30"`
	Business string `json:"business" binding:"max=100"`
	Package  string `json:"package" binding:"max=60"`
	Message  string `json:"message" binding:"max=1000"`
	Website  string `json:"website"` // honeypot: люди это поле не видят, боты заполняют
}

func (h *Handlers) CreateLead(c *gin.Context) {
	var req leadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "Проверьте, что имя и телефон заполнены."})
		return
	}

	// Бот заполнил скрытое поле — делаем вид, что всё хорошо, но не сохраняем.
	if req.Website != "" {
		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "Укажите, как к вам обращаться."})
		return
	}
	if countDigits(req.Phone) < 10 {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "Похоже, в номере телефона не хватает цифр."})
		return
	}
	if !h.limiter.Allow(c.ClientIP()) {
		c.JSON(http.StatusTooManyRequests, gin.H{"ok": false, "error": "Заявка уже отправлена. Подождите немного перед повторной отправкой."})
		return
	}

	lead := Lead{
		Name:      req.Name,
		Phone:     strings.TrimSpace(req.Phone),
		Business:  strings.TrimSpace(req.Business),
		Package:   strings.TrimSpace(req.Package),
		Message:   strings.TrimSpace(req.Message),
		IP:        c.ClientIP(),
		CreatedAt: time.Now(),
	}
	if err := h.leads.Save(lead); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "Не получилось сохранить заявку. Напишите нам в WhatsApp."})
		return
	}
	go notifyTelegram(lead)

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func countDigits(s string) int {
	n := 0
	for _, r := range s {
		if unicode.IsDigit(r) {
			n++
		}
	}
	return n
}

// rateLimiter разрешает одну заявку с IP за указанный интервал.
type rateLimiter struct {
	mu     sync.Mutex
	window time.Duration
	seen   map[string]time.Time
}

func newRateLimiter(window time.Duration) *rateLimiter {
	return &rateLimiter{window: window, seen: make(map[string]time.Time)}
}

func (l *rateLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if last, ok := l.seen[key]; ok && now.Sub(last) < l.window {
		return false
	}
	l.seen[key] = now
	// Не даём карте разрастаться бесконечно.
	if len(l.seen) > 10000 {
		for k, t := range l.seen {
			if now.Sub(t) > l.window {
				delete(l.seen, k)
			}
		}
	}
	return true
}
