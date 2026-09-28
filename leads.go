package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Lead struct {
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	Business  string    `json:"business,omitempty"`
	Package   string    `json:"package,omitempty"`
	Message   string    `json:"message,omitempty"`
	IP        string    `json:"ip"`
	CreatedAt time.Time `json:"created_at"`
}

// LeadStore дописывает заявки в JSON Lines файл: одна строка — одна заявка.
type LeadStore struct {
	mu   sync.Mutex
	path string
}

func NewLeadStore(path string) (*LeadStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return &LeadStore{path: path}, nil
}

func (s *LeadStore) Save(l Lead) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	line, err := json.Marshal(l)
	if err != nil {
		return err
	}
	_, err = f.Write(append(line, '\n'))
	if err == nil {
		log.Printf("новая заявка: %s, %s", l.Name, l.Phone)
	}
	return err
}

var tgClient = &http.Client{Timeout: 10 * time.Second}

// notifyTelegram отправляет заявку в Telegram, если заданы
// TELEGRAM_BOT_TOKEN и TELEGRAM_CHAT_ID. Без них просто ничего не делает.
func notifyTelegram(l Lead) {
	token, chatID := os.Getenv("TELEGRAM_BOT_TOKEN"), os.Getenv("TELEGRAM_CHAT_ID")
	if token == "" || chatID == "" {
		return
	}

	text := fmt.Sprintf("🔥 Новая заявка с сайта\n\n👤 %s\n📞 %s", l.Name, l.Phone)
	if l.Business != "" {
		text += "\n🏪 " + l.Business
	}
	if l.Package != "" {
		text += "\n📦 " + l.Package
	}
	if l.Message != "" {
		text += "\n💬 " + l.Message
	}

	resp, err := tgClient.PostForm(
		"https://api.telegram.org/bot"+token+"/sendMessage",
		url.Values{"chat_id": {chatID}, "text": {text}},
	)
	if err != nil {
		log.Printf("telegram: %v", err)
		return
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("telegram: статус %d", resp.StatusCode)
	}
}
