package telegram

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/rgeraskin/joytime/internal/domain"
	"github.com/rgeraskin/joytime/internal/models"
	tele "gopkg.in/telebot.v4"
	"gorm.io/gorm"
)

// Input states for multi-step conversation flows (text input only)
const (
	stateJoinFamily       = "join_family"
	stateAddTaskName      = "add_task_name"
	stateAddTaskBulk      = "add_task_bulk"
	stateAddTaskTokens    = "add_task_tokens"
	stateEditTaskTokens   = "edit_task_tokens"
	stateReviewConfirm    = "review_confirm"
	stateAddRewardBulk    = "add_reward_bulk"
	stateAddRewardName    = "add_reward_name"
	stateAddRewardTokens  = "add_reward_tokens"
	stateEditRewardTokens = "edit_reward_tokens"
	stateAddPenaltyName   = "add_penalty_name"
	stateAddPenaltyTokens = "add_penalty_tokens"
	stateAddPenaltyBulk   = "add_penalty_bulk"
	stateEditPenaltyTokens = "edit_penalty_tokens"
	stateApplyPenaltyChild  = "apply_penalty_child"
	stateManualAdjustReason = "manual_adjust_reason"
	stateManualAdjustTokens = "manual_adjust_tokens"
	stateRenameMemberName   = "rename_member_name"
)

// Number grid settings
const gridMaxCols = 7

// Bot wraps the Telegram bot with domain services
type Bot struct {
	bot      *tele.Bot
	services *domain.Services
	logger   *log.Logger
}

// New creates and configures a new Telegram bot
func New(token string, services *domain.Services, logger *log.Logger) (*Bot, error) {
	pref := tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	bot := &Bot{
		bot:      b,
		services: services,
		logger:   logger,
	}
	bot.registerHandlers()
	return bot, nil
}

// Start starts the bot polling loop (blocking)
func (b *Bot) Start() {
	b.logger.Info("Starting Telegram bot")
	b.bot.Start()
}

// Stop gracefully stops the bot
func (b *Bot) Stop() {
	b.bot.Stop()
}

func (b *Bot) registerHandlers() {
	b.bot.Handle("/start", b.handleStart)
	b.bot.Handle(tele.OnCallback, b.handleCallback)
	b.bot.Handle(tele.OnText, b.handleText)
}

// handleStart is the entry point for all conversations
func (b *Bot) handleStart(c tele.Context) error {
	b.clearState(c.Sender().ID)

	user, err := b.findUser(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error finding user", err)
	}

	if user == nil {
		return b.showRoleSelection(c)
	}

	// User exists but hasn't joined a family yet
	if user.FamilyUID == "" {
		return b.showFamilySetup(c, user.Role)
	}

	switch user.Role {
	case string(domain.RoleParent):
		return b.showParentMenu(c)
	case string(domain.RoleChild):
		return b.showChildMenu(c)
	}

	return b.internalError(c, "Unknown user role", fmt.Errorf("role: %s", user.Role))
}

// handleCallback routes all inline keyboard button presses
func (b *Bot) handleCallback(c tele.Context) error {
	data := c.Callback().Data
	_ = c.Respond()

	// Handle callbacks that depend on preserved input state (before clearState)
	switch data {
	case "review_approve":
		return b.onReviewApprove(c)
	case "review_reject":
		return b.onReviewReject(c)
	}
	// pick_penalty_child needs penalty name from state
	if strings.HasPrefix(data, "pick_penalty_child:") {
		num, err := parseNumber(data[len("pick_penalty_child:"):])
		if err != nil {
			return nil
		}
		return b.onApplyPenaltyChildPick(c, num)
	}

	b.clearState(c.Sender().ID)

	// Handle parameterized callbacks (format: "prefix:number")
	if idx := strings.IndexByte(data, ':'); idx > 0 {
		prefix := data[:idx]
		num, err := parseNumber(data[idx+1:])
		if err != nil {
			return nil
		}
		switch prefix {
		case "pick_edit_task":
			return b.onEditTaskPick(c, num)
		case "pick_del_task":
			return b.onDeleteTaskPick(c, num)
		case "pick_edit_reward":
			return b.onEditRewardPick(c, num)
		case "pick_del_reward":
			return b.onDeleteRewardPick(c, num)
		case "pick_review":
			return b.onReviewTaskPick(c, num)
		case "pick_task_done":
			return b.onTaskDonePick(c, num)
		case "pick_reward_claim":
			return b.onRewardClaimPick(c, num)
		case "pick_edit_penalty":
			return b.onEditPenaltyPick(c, num)
		case "pick_del_penalty":
			return b.onDeletePenaltyPick(c, num)
		case "pick_apply_penalty":
			return b.onApplyPenaltyPick(c, num)
		case "pick_manual_child":
			return b.onManualAdjustChildPick(c, num)
		case "pick_rename_member":
			return b.onRenameMemberPick(c, num)
		case "pick_delete_member":
			return b.onDeleteMemberPick(c, num)
		case "pick_history_child":
			return b.onHistoryChildPick(c, num)
		}
	}

	switch data {
	case "noop":
		return nil

	// Registration
	case "role_parent":
		return b.onSelectRole(c, string(domain.RoleParent))
	case "role_child":
		return b.onSelectRole(c, string(domain.RoleChild))
	case "family_create":
		return b.onFamilyCreate(c)
	case "family_join":
		return b.onFamilyJoinPrompt(c)

	// Parent navigation
	case "parent_tasks":
		return b.showTasks(c)
	case "parent_rewards":
		return b.showRewards(c)
	case "parent_review":
		return b.showPendingReview(c)
	case "back_parent":
		return b.showParentMenu(c)

	// Task CRUD
	case "task_add":
		return b.onAddTaskPrompt(c)
	case "task_add_bulk":
		return b.onAddTaskBulkPrompt(c)
	case "task_edit":
		return b.onEditTaskPrompt(c)
	case "task_delete":
		return b.onDeleteTaskPrompt(c)

	// Reward CRUD
	case "reward_add":
		return b.onAddRewardPrompt(c)
	case "reward_add_bulk":
		return b.onAddRewardBulkPrompt(c)
	case "reward_edit":
		return b.onEditRewardPrompt(c)
	case "reward_delete":
		return b.onDeleteRewardPrompt(c)

	// Penalty CRUD
	case "parent_penalties":
		return b.showPenalties(c)
	case "penalty_add":
		return b.onAddPenaltyPrompt(c)
	case "penalty_add_bulk":
		return b.onAddPenaltyBulkPrompt(c)
	case "penalty_edit":
		return b.onEditPenaltyPrompt(c)
	case "penalty_delete":
		return b.onDeletePenaltyPrompt(c)
	case "penalty_apply":
		return b.onApplyPenaltyPrompt(c)
	case "manual_adjust":
		return b.onManualAdjustPrompt(c)
	case "parent_family":
		return b.showFamilyMembers(c)
	case "parent_history":
		return b.showParentHistoryPrompt(c)
	case "family_rename":
		return b.onRenameMemberPrompt(c)
	case "family_delete":
		return b.onDeleteMemberPrompt(c)

	// Child actions
	case "child_penalties":
		return b.showChildPenalties(c)
	case "child_history":
		return b.showChildHistory(c)
	case "child_task_done":
		return b.onTaskDonePrompt(c)
	case "child_reward_claim":
		return b.onRewardClaimPrompt(c)
	case "back_child":
		return b.showChildMenu(c)
	}

	b.logger.Warn("Unknown callback data", "data", data)
	return nil
}

// handleText routes text input based on user's current input state
func (b *Bot) handleText(c tele.Context) error {
	user, err := b.findUser(c.Sender().ID)
	if err != nil {
		return b.internalError(c, "Error finding user", err)
	}
	if user == nil {
		return c.Send("❌ Для начала нажми /start")
	}

	text := strings.TrimSpace(c.Text())
	state := user.InputState
	inputCtx := user.InputContext

	switch state {
	// Registration
	case stateJoinFamily:
		return b.onFamilyJoinText(c, text)

	// Task management (parent)
	case stateAddTaskName:
		return b.onAddTaskName(c, text)
	case stateAddTaskBulk:
		return b.onAddTaskBulk(c, text)
	case stateAddTaskTokens:
		return b.onAddTaskTokens(c, text, inputCtx)
	case stateEditTaskTokens:
		return b.onEditTaskTokens(c, text, inputCtx)

	// Reward management (parent)
	case stateAddRewardBulk:
		return b.onAddRewardBulk(c, text)
	case stateAddRewardName:
		return b.onAddRewardName(c, text)
	case stateAddRewardTokens:
		return b.onAddRewardTokens(c, text, inputCtx)
	case stateEditRewardTokens:
		return b.onEditRewardTokens(c, text, inputCtx)

	// Penalty management (parent)
	case stateAddPenaltyName:
		return b.onAddPenaltyName(c, text)
	case stateAddPenaltyTokens:
		return b.onAddPenaltyTokens(c, text, inputCtx)
	case stateAddPenaltyBulk:
		return b.onAddPenaltyBulk(c, text)
	case stateEditPenaltyTokens:
		return b.onEditPenaltyTokens(c, text, inputCtx)

	// Family management
	case stateRenameMemberName:
		return b.onRenameMemberName(c, text, inputCtx)

	// Manual adjustment
	case stateManualAdjustReason:
		return b.onManualAdjustReason(c, text, inputCtx)
	case stateManualAdjustTokens:
		return b.onManualAdjustTokens(c, text, inputCtx)
	}

	return c.Send("❌ Не понимаю. Нажми /start для начала")
}

// --- Helpers ---

func bgCtx() context.Context {
	return context.Background()
}

func makeUserID(tgID int64) string {
	return fmt.Sprintf("user_%d", tgID)
}

func extractName(u *tele.User) string {
	if u.FirstName != "" && u.LastName != "" {
		return u.FirstName + " " + u.LastName
	}
	if u.FirstName != "" {
		return u.FirstName
	}
	if u.Username != "" {
		return u.Username
	}
	return fmt.Sprintf("User %d", u.ID)
}

func (b *Bot) findUser(tgID int64) (*models.Users, error) {
	user, err := b.services.UserService.FindUser(bgCtx(), makeUserID(tgID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

func (b *Bot) authCtx(tgID int64) (*domain.AuthContext, error) {
	return b.services.UserService.CreateAuthContext(bgCtx(), makeUserID(tgID))
}

func (b *Bot) clearState(tgID int64) {
	_ = b.services.UserService.SetInputState(bgCtx(), makeUserID(tgID), "", "")
}

func (b *Bot) setState(tgID int64, state, inputCtx string) error {
	return b.services.UserService.SetInputState(bgCtx(), makeUserID(tgID), state, inputCtx)
}

func (b *Bot) internalError(c tele.Context, msg string, err error) error {
	b.logger.Error(msg, "error", err, "tg_id", c.Sender().ID)
	return c.Send("❌ Внутренняя ошибка. Попробуй /start")
}

func parentMenuKeyboard() *tele.ReplyMarkup {
	return inlineKeyboard(btnRow(btn("📌 Меню", "back_parent")))
}

func childMenuKeyboard() *tele.ReplyMarkup {
	return inlineKeyboard(btnRow(btn("📌 Меню", "back_child")))
}

func (b *Bot) notifyParents(familyUID string, excludeTgID int64, message string) {
	users, err := b.services.UserService.FindFamilyUsersByRole(bgCtx(), familyUID, string(domain.RoleParent))
	if err != nil {
		b.logger.Error("Failed to find parents for notification", "error", err)
		return
	}
	for _, u := range users {
		tgIDStr := strings.TrimPrefix(u.UserID, "user_")
		tgID, err := strconv.ParseInt(tgIDStr, 10, 64)
		if err != nil || tgID == excludeTgID {
			continue
		}
		if _, err := b.bot.Send(&tele.User{ID: tgID}, message, parentMenuKeyboard()); err != nil {
			b.logger.Error("Failed to notify parent", "error", err, "tg_id", tgID)
		}
	}
}

func (b *Bot) notifyChildren(familyUID string, message string) {
	children, err := b.services.UserService.FindFamilyUsersByRole(bgCtx(), familyUID, string(domain.RoleChild))
	if err != nil {
		b.logger.Error("Failed to find children for notification", "error", err)
		return
	}
	for _, child := range children {
		b.notifyChild(child.UserID, message)
	}
}

func (b *Bot) notifyChild(childUserID, message string) {
	tgIDStr := strings.TrimPrefix(childUserID, "user_")
	tgID, err := strconv.ParseInt(tgIDStr, 10, 64)
	if err != nil {
		return
	}
	if _, err := b.bot.Send(&tele.User{ID: tgID}, message, childMenuKeyboard()); err != nil {
		b.logger.Error("Failed to notify child", "error", err, "tg_id", tgID)
	}
}

func formatList(header string, items []string) string {
	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString(":\n\n")
	if len(items) == 0 {
		sb.WriteString("Пока пусто 😔")
		return sb.String()
	}
	for i, item := range items {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, item))
	}
	return sb.String()
}

func formatHistory(prefix string, history []models.TokenHistory, limit int) string {
	var sb strings.Builder
	if prefix != "" {
		sb.WriteString(prefix + "\n\n")
	}
	sb.WriteString("📜 История:\n\n")
	if len(history) == 0 {
		sb.WriteString("Пока пусто 😔")
		return sb.String()
	}
	count := len(history)
	if limit > 0 && count > limit {
		count = limit
	}
	for i := 0; i < count; i++ {
		h := history[i]
		sign := "+"
		if h.Amount < 0 {
			sign = ""
		}
		sb.WriteString(fmt.Sprintf("%s%d 💎 %s\n", sign, h.Amount, h.Description))
	}
	return sb.String()
}

func inlineKeyboard(rows ...[]tele.InlineButton) *tele.ReplyMarkup {
	return &tele.ReplyMarkup{
		InlineKeyboard: rows,
	}
}

func btnRow(buttons ...tele.InlineButton) []tele.InlineButton {
	return buttons
}

func btn(text, data string) tele.InlineButton {
	return tele.InlineButton{Text: text, Data: data}
}

func parseNumber(text string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(text))
}

// isDuplicateKey checks if a DB error is a unique constraint violation (PostgreSQL SQLSTATE 23505)
func isDuplicateKey(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}

// numberGrid builds a grid of numbered buttons with placeholder padding.
// Buttons use callback data format "prefix:N" where N is 1-based.
func numberGrid(count int, callbackPrefix string) [][]tele.InlineButton {
	var rows [][]tele.InlineButton
	var row []tele.InlineButton
	for i := 1; i <= count; i++ {
		row = append(row, tele.InlineButton{
			Text: fmt.Sprintf("%d", i),
			Data: fmt.Sprintf("%s:%d", callbackPrefix, i),
		})
		if len(row) == gridMaxCols {
			rows = append(rows, row)
			row = nil
		}
	}
	// Pad last row with placeholders
	if len(row) > 0 {
		for len(row) < gridMaxCols {
			row = append(row, tele.InlineButton{Text: " ", Data: "noop"})
		}
		rows = append(rows, row)
	}
	return rows
}

// escapeMarkdownV2 escapes special characters for Telegram MarkdownV2
func escapeMarkdownV2(s string) string {
	replacer := strings.NewReplacer(
		"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]",
		"(", "\\(", ")", "\\)", "~", "\\~", "`", "\\`",
		">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-",
		"=", "\\=", "|", "\\|", "{", "\\{", "}", "\\}",
		".", "\\.", "!", "\\!",
	)
	return replacer.Replace(s)
}
