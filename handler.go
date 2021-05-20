package main

import (
	"context"

	"github.com/yandex-cloud/examples/serverless/alice-shareable-todolist/app/config"
	"github.com/yandex-cloud/examples/serverless/alice-shareable-todolist/app/db"
	"github.com/yandex-cloud/examples/serverless/alice-shareable-todolist/app/model"
)

type app struct {
	ctx        context.Context
	conf       *config.Config
	repository db.Repository
	txMgr      db.TxManager
}

// Имплементируем db.Deps, чтобы менеджер транзакций мог забирать
// свои зависимости из структуры app
func (a *app) GetConfig() *config.Config { return a.conf }

func (a *app) GetContext() context.Context { return a.ctx }

// И добавляем getter-методы, чтобы компоненты, зависящие от БД,
// могли ссылаться на базу данных через свой интерфейс Deps
func (a *app) GetRepository() db.Repository { return a.repository }

func (a *app) GetTxManager() db.TxManager { return a.txMgr }

// initApp инициализирует наше приложение - читает конфигурацию и
// инициализирует все компоненты - для примера проинициализируем
// компоненты базы данных
func initApp() (*app, error) {
	appInstance := &app{
		ctx:  context.Background(),
		conf: config.LoadFromEnv(),
	}
	var err error
	appInstance.txMgr, err = db.NewTxManager(appInstance)
	if err != nil {
		return nil, err
	}
	appInstance.repository, err = db.NewRepository()
	if err != nil {
		return nil, err
	}
	return appInstance, nil
}

// Положим инстанс приложения в примитивный singleton
var appInstance *app

func getApp() (*app, error) {
	if appInstance == nil {
		return initApp()
	}
	return appInstance, nil
}

// Точка входа в функцию - именно в этот метод будет передано управление
// при вызове нашей функции
func Handler(ctx context.Context, req map[string]string) (interface{}, error) {
	appInstance, err := getApp()
	if err != nil {
		return nil, err
	}

	// для примера просто вернем информацию о пользователе из БД
	var user *model.User
	err = appInstance.GetTxManager().InTx(ctx).Do(func(ctx context.Context) error {
		var err error
		user, err = appInstance.GetRepository().GetUser(ctx, model.UserID(req["user"]))
		return err
	})
	return user, err
}
