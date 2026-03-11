package telegram

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"kiro-relay/internal/relay"
)

type Adapter struct {
	api      *tgbotapi.BotAPI
	cfg      *Config
	sessions *relay.SessionManager
}

func New(cfg *Config, sm *relay.SessionManager) (*Adapter, error) {
	api, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		return nil, err
	}
	slog.Info("telegram adapter started", "username", api.Self.UserName)
	return &Adapter{api: api, cfg: cfg, sessions: sm}, nil
}

func (a *Adapter) Run(stop <-chan struct{}) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := a.api.GetUpdatesChan(u)

	for {
		select {
		case <-stop:
			slog.Info("telegram adapter shutting down")
			return
		case update := <-updates:
			if update.Message == nil || update.Message.Text == "" {
				continue
			}
			a.handle(update.Message)
		}
	}
}

func (a *Adapter) sessionKey(uid int64) string {
	return "telegram:" + strconv.FormatInt(uid, 10)
}

func (a *Adapter) handle(msg *tgbotapi.Message) {
	uid := msg.From.ID

	if !a.cfg.IsAllowed(uid) {
		a.reply(msg.Chat.ID, "⛔ Not authorized.")
		return
	}

	if msg.IsCommand() {
		a.handleCommand(msg)
		return
	}

	key := a.sessionKey(uid)
	if _, active := a.sessions.Get(key); !active {
		a.reply(msg.Chat.ID, "No active session. Send /chat start first.")
		return
	}

	sent, _ := a.api.Send(tgbotapi.NewMessage(msg.Chat.ID, "⏳ Thinking..."))

	go func() {
		resp, err := a.sessions.Send(key, msg.Text)
		if err != nil {
			resp = "❌ " + err.Error()
		}
		a.sendChunked(msg.Chat.ID, sent.MessageID, resp)
	}()
}

func (a *Adapter) handleCommand(msg *tgbotapi.Message) {
	uid := msg.From.ID
	key := a.sessionKey(uid)

	switch msg.Command() {
	case "chat":
		args := strings.Fields(msg.CommandArguments())
		if len(args) == 0 {
			a.reply(msg.Chat.ID, "Usage: /chat start | /chat end")
			return
		}
		switch args[0] {
		case "start":
			a.sessions.Start(key)
			a.reply(msg.Chat.ID, fmt.Sprintf("🆕 New kiro session started.\nWork dir: %s\nSend any message to chat.\nUse /chat end to end the session.", a.sessions.GetWorkDir()))
		case "end":
			a.sessions.End(key)
			a.reply(msg.Chat.ID, "👋 Session ended.")
		default:
			a.reply(msg.Chat.ID, "Usage: /chat start | /chat end")
		}
	case "workdir":
		dir := strings.TrimSpace(msg.CommandArguments())
		if dir == "" {
			a.reply(msg.Chat.ID, fmt.Sprintf("📂 Current: %s\nUsage: /workdir /path/to/dir", a.sessions.GetWorkDir()))
			return
		}
		a.sessions.SetWorkDir(dir)
		a.reply(msg.Chat.ID, fmt.Sprintf("📂 Work dir changed to: %s", dir))
	case "agent":
		args := strings.Fields(msg.CommandArguments())
		if len(args) == 0 {
			current := a.sessions.GetAgent()
			a.reply(msg.Chat.ID, fmt.Sprintf("🤖 Current: %s\nUsage: /agent list | /agent set <name>", current))
			return
		}

		switch args[0] {
		case "list":
			agents, err := a.sessions.ListAgents()
			if err != nil {
				a.reply(msg.Chat.ID, "❌ "+err.Error())
				return
			}
			current := a.sessions.GetAgent()
			if current != "" {
				agents = fmt.Sprintf("🤖 Current: %s\n\n%s", current, agents)
			}
			a.reply(msg.Chat.ID, agents)
		case "set":
			if len(args) < 2 {
				a.reply(msg.Chat.ID, "Usage: /agent set <name>")
				return
			}
			agent := args[1]
			a.sessions.SetAgent(agent)
			a.reply(msg.Chat.ID, fmt.Sprintf("🤖 Agent set to: %s", agent))
		default:
			a.reply(msg.Chat.ID, "Usage: /agent list | /agent set <name>")
		}
	default:
		a.reply(msg.Chat.ID, "Unknown command. Use /chat, /workdir, or /agent.")
	}
}

func (a *Adapter) reply(chatID int64, text string) {
	a.api.Send(tgbotapi.NewMessage(chatID, text))
}

func (a *Adapter) sendChunked(chatID int64, editMsgID int, text string) {
	const maxChunk = 4000
	for i := 0; i < len(text); i += maxChunk {
		end := i + maxChunk
		if end > len(text) {
			end = len(text)
		}
		chunk := text[i:end]
		if i == 0 && editMsgID != 0 {
			a.api.Send(tgbotapi.NewEditMessageText(chatID, editMsgID, chunk))
		} else {
			a.api.Send(tgbotapi.NewMessage(chatID, chunk))
		}
	}
}
