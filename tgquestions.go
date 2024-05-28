package tgquestions

import (
	"fmt"
	"errors"
	"strconv"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/dgraph-io/ristretto"
)

type BoolQuestionConfig struct {
	ID int64
	Question string
	TextTrue string
	TextFalse string
	TextAfterClick string
	CallbackTrue func()
	CallbackFalse func()
}

type QuestionConfig struct {
	ID int64
	Question string
	Result chan string
	Callback func()
	msgsent tgbotapi.Message
	Update chan tgbotapi.Update
	DoubleCheck bool
	DoubleCheckQuestion string
	DoubleCheckButtonYes string
	DoubleCheckButtonNo string
}

type Config struct {
	RistrettoConfig *ristretto.Config
	Bot *tgbotapi.BotAPI
}

type TGQ struct {
	bot *tgbotapi.BotAPI
	cache *ristretto.Cache
}

func Init(c Config) (*TGQ, error){
	var tgq TGQ
	var err error
	if (c.Bot == nil) {
		err = errors.New("Bot parameter is mandatory")
	} else {
		tgq.bot = c.Bot
	}
	if (c.RistrettoConfig == nil) {
		return &TGQ{}, errors.New("You need cache")
	}
	if (c.RistrettoConfig != nil) {
		tgq.cache, err = ristretto.NewCache(c.RistrettoConfig)
		if err != nil {
			return &TGQ{}, err
		}
	}
	return &tgq, err
}

func (tgq *TGQ) Update(update tgbotapi.Update) bool {
	if (update.Message != nil) {
		qct, found := tgq.cache.Get("WAITING-MESSAGE#"+strconv.FormatInt(update.Message.Chat.ID, 10))
		if found {
			qc := qct.(QuestionConfig)
			isTheMessage := false
			if (update.Message.Chat.Type == "private") {
				isTheMessage = true
			} else {
				if (update.Message.ReplyToMessage != nil) {
					if (qc.msgsent.MessageID == update.Message.ReplyToMessage.MessageID) {
						isTheMessage = true
					}
				}
			}
			if (isTheMessage == true) {
				if (qc.DoubleCheck == true) {
					returnBool := false
                                        tgq.DoBoolQuestion(BoolQuestionConfig{
                                                ID: update.Message.Chat.ID,
                                                Question: fmt.Sprintf(qc.DoubleCheckQuestion, update.Message.Text),
                                                TextTrue: qc.DoubleCheckButtonYes,
                                                CallbackTrue: func() {
							qc.Result <- update.Message.Text
							qc.Update <- update
							tgq.cache.Del("WAITING-MESSAGE#"+strconv.FormatInt(update.Message.Chat.ID, 10))
							qc.Callback()
							returnBool = true
                                                },
                                                TextFalse: qc.DoubleCheckButtonNo,
                                                CallbackFalse: func() {
							tgq.DoQuestion(qc)
                                                },
                                        })
					if (returnBool == true) {
						return true
					}
				} else {
					qc.Result <- update.Message.Text
					qc.Update <- update
					tgq.cache.Del("WAITING-MESSAGE#"+strconv.FormatInt(update.Message.Chat.ID, 10))
					qc.Callback()
					return true
				}
			}
		}
	}
	if (update.CallbackQuery != nil) {
		qct, found := tgq.cache.Get("WAITING-CALLBACK#"+strconv.FormatInt(update.CallbackQuery.Message.Chat.ID, 10))
		if found {
			qc := qct.(BoolQuestionConfig)
			if (update.CallbackQuery.Data == "BoolTrue") {
				var msgText string
				if (qc.TextAfterClick != "") {
					msgText = qc.TextAfterClick
				} else {
					msgText = qc.Question
				}
				edit := tgbotapi.EditMessageTextConfig{
					BaseEdit: tgbotapi.BaseEdit{
						ChatID:    update.CallbackQuery.Message.Chat.ID,
						MessageID: update.CallbackQuery.Message.MessageID,
					},
					Text: msgText,
				}
				tgq.bot.Send(edit)
				tgq.cache.Del("WAITING-CALLBACK#"+strconv.FormatInt(update.CallbackQuery.Message.Chat.ID, 10))
				qc.CallbackTrue()
				return true
			} else if (update.CallbackQuery.Data == "BoolFalse") {
				var msgText string
				if (qc.TextAfterClick != "") {
					msgText = qc.TextAfterClick
				} else {
					msgText = qc.Question
				}
				edit := tgbotapi.EditMessageTextConfig{
					BaseEdit: tgbotapi.BaseEdit{
						ChatID:    update.CallbackQuery.Message.Chat.ID,
						MessageID: update.CallbackQuery.Message.MessageID,
					},
					Text: msgText,
				}
				tgq.bot.Send(edit)
				tgq.cache.Del("WAITING-CALLBACK#"+strconv.FormatInt(update.CallbackQuery.Message.Chat.ID, 10))
				qc.CallbackFalse()
				return true
			}
		}
	} else {
		return false
	}
	return false
}

func (tgq *TGQ) DoBoolQuestion(qc BoolQuestionConfig) {
	tgq.cache.Set("WAITING-CALLBACK#"+strconv.FormatInt(qc.ID, 10), qc, 1)
	msg := tgbotapi.NewMessage(qc.ID, qc.Question)
	var keyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(qc.TextTrue,"BoolTrue"),
		tgbotapi.NewInlineKeyboardButtonData(qc.TextFalse,"BoolFalse"),
		),
	)
	msg.ReplyMarkup = keyboard
	tgq.bot.Send(msg)
}

func (tgq *TGQ) DoQuestion(qc QuestionConfig) error {
	var err error
	msg := tgbotapi.NewMessage(qc.ID, qc.Question)
	msgr, err := tgq.bot.Send(msg)
	if err == nil {
		qc.msgsent = msgr
	} else {
		return err
	}
	tgq.cache.Set("WAITING-MESSAGE#"+strconv.FormatInt(qc.ID, 10), qc, 1)
	return err
}
